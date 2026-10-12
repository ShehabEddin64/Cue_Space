# Plan: macOS widgets and Shortcuts actions

Status: planned, not built. Real widgets need tools this project doesn't have yet. This page lists what they need and how they fit the current architecture, so they can be built when those are available.

## What we want

- **Widgets** (desktop and Notification Center) that show your pinned Cues and start one with a click. Sizes: small (one Cue), medium (up to four), large (up to eight, with item counts).
- **Shortcuts app actions**: "Start Cue" and "List Cues", so people can start a Cue from Shortcuts, Siri, Spotlight, Focus filters or automations.

## Why this isn't built yet

Both features use Apple frameworks that only native Swift code can use:

| Feature | Framework | Needs |
| --- | --- | --- |
| Widgets | WidgetKit (SwiftUI) | A widget **app extension** (`.appex`) inside the app bundle |
| Shortcuts actions | App Intents | Swift intents compiled into the app or an extension |

Building and shipping them requires:

1. **Full Xcode** (not only the command line tools) for the WidgetKit and App Intents SDKs, `actool`, and extension targets. This also lets the Icon Composer icon in `build/appicon.icon` compile into `Assets.car`.
2. **An Apple Developer account** for a Team ID. Widget extensions must be sandboxed and signed, and sharing data between the app and the widget needs an **App Group** (`group.space.cue.app`), which is tied to the Team ID.
3. **Developer ID signing and notarization** to distribute outside the Mac App Store. Ad-hoc signed builds don't load widgets reliably.

## Architecture

The rule "the runner never talks to the AI" stays. Widgets and intents only list and start Cues.

```
┌────────────── Cue Space.app (Wails, Go) ──────────────┐
│  store ──writes──▶ App Group container                 │
│                    widget-snapshot.json                │
│  URL handler  ◀── cuespace://start/<cue-id>            │
│  runner (unchanged)                                    │
│  ┌──────── CueSpaceWidgets.appex (Swift) ──────────┐   │
│  │ reads widget-snapshot.json → SwiftUI timeline   │   │
│  │ button → opens cuespace://start/<cue-id>        │   │
│  │ AppIntents: StartCueIntent, ListCuesIntent      │   │
│  └─────────────────────────────────────────────────┘   │
└────────────────────────────────────────────────────────┘
```

### 1. A snapshot the widget can read (Go)

- After every Cue change (the store watcher already fires) and every pin change, write `widget-snapshot.json` to the App Group container: `~/Library/Group Containers/group.space.cue.app/`.
- Contents: the `ShortcutOrder` list (pinned first) of `{id, name, icon, itemCount, lastRunAt}`. No paths or URLs, so the widget never sees what a Cue opens.
- Then ask WidgetKit to reload timelines. From Go, call a tiny Objective-C helper (`WidgetCenter` has no Objective-C API, so the helper goes in the Swift extension target or in a small Swift static library linked into the app).

### 2. A URL scheme the widget can call (Go)

- Register `cuespace://` in `build/darwin/Info.plist` (`CFBundleURLTypes`).
- Handle `cuespace://start/<id>` and `cuespace://open/<id>` with Wails' application URL event, and run the same `CueService.Start` the UI uses.
- This also gives Raycast, Alfred and Shortcuts' "Open URL" action a way to start Cues before the native actions exist.

### 3. The widget extension (Swift)

- New Xcode target `CueSpaceWidgets` (Widget Extension), with the App Group entitlement and sandboxing.
- A `TimelineProvider` reads the snapshot. Entries refresh when the app reloads timelines, with a fallback refresh every hour.
- Views follow the identity: Cue Red only on the start control, Geist is not available in widgets so they use the system font, and light and dark come from the system.
- Every Cue is a `Link(destination: cuespace://start/<id>)`. On macOS 14 and later, an `AppIntent` button can start the Cue without bringing the app to the front.

### 4. Shortcuts actions (Swift, App Intents)

- `StartCueIntent(cue: CueEntity)` and `ListCuesIntent`, with `CueEntity` backed by the snapshot.
- `StartCueIntent.perform()` opens `cuespace://start/<id>`, so the Go runner stays the only code that opens things.
- Add `AppShortcutsProvider` phrases: "Start \(cue) in Cue Space".

### 5. Build integration

- Keep Wails building the Go binary. Add an Xcode project under `build/darwin/xcode/` that builds the `.appex`. A Taskfile step copies it into `Cue Space.app/Contents/PlugIns/` before signing.
- Sign inside-out: extension first, then the app, with the App Group entitlement on both. Then notarize.

## Order of work

1. URL scheme and handler (Go only). Useful on its own for Raycast, Alfred and Shortcuts "Open URL".
2. App Group snapshot writer (Go), behind a check that the container exists.
3. Widget extension (Xcode and Developer ID needed from here on).
4. App Intents.
5. Signing and notarization in CI.
