# Widgets and cuespace:// links

## What exists

- **A desktop and Notification Center widget**, "Cues", in small, medium, large and extra-large sizes. It shows your Cues (pinned first, the same order as ⌘1–⌘9). Clicking one starts it.
- **`cuespace://` links**, which also work from Shortcuts ("Open URL"), Raycast, Alfred or a browser:
  - `cuespace://start/<cue-id>` starts a Cue
  - `cuespace://open/<cue-id>` shows a Cue
  - `cuespace://open` brings Cue Space forward

The widget never runs anything itself. The rule "only the runner opens things" holds:

```
Cue Space.app (Go)
 ├─ writes ~/Library/Application Support/CueSpace/widget.json   (names, icons, counts: no paths or URLs)
 ├─ runs Contents/MacOS/cuespace-widget-reload                   (asks WidgetKit to refresh)
 ├─ handles cuespace://start/<id>  →  the same runner as Start Cue
 └─ Contents/PlugIns/CueSpaceWidgets.appex (Swift, sandboxed)
      reads widget.json  →  SwiftUI views  →  each Cue links to cuespace://start/<id>
```

## Files

| File | What it is |
| --- | --- |
| `widgets/CueSpaceWidgets.swift` | The widget: timeline provider, views, brand mark |
| `widgets/reload.swift` | Tiny helper that calls `WidgetCenter.reloadAllTimelines()` (no C or Objective-C API exists) |
| `build/darwin/widgets/build.sh` | Compiles both with `swiftc` for arm64 and x86_64 and puts them in the app bundle |
| `build/darwin/widgets/Info.plist` | The extension's bundle info (`com.apple.widgetkit-extension`) |
| `build/darwin/widgets/widgets.entitlements` | Sandbox, plus read-only access to `~/Library/Application Support/CueSpace/` |
| `services/widget.go` | Writes the snapshot (debounced) when Cues or pins change |

`wails3 task darwin:package` builds everything. It needs only the Xcode command line tools, not full Xcode. The extension is linked with `-e _NSExtensionMain`, as Xcode does for extensions: with Swift's own `main` the process starts and immediately exits, and WidgetKit logs "query failed … connection invalidated". Signing goes inside-out: the helper, then the extension with its entitlements, then the app (no `--deep`, which would strip the extension's sandbox entitlements).

## Notes and limits

- **Data sharing without an App Group.** App Groups need an Apple Developer Team ID. Instead, the sandboxed extension has a read-only exception for the Cue Space data folder. That's fine outside the Mac App Store; an App Store build would switch to an App Group container.
- **Debugging.** Use `/usr/bin/log show --last 5m --predicate 'eventMessage CONTAINS "space.cue.app.widgets"'` (the full path matters: zsh has its own `log` command). "Reload success" from `chronod` means WidgetKit loaded the widget.
- **Development builds.** The app is ad-hoc signed. If the widget doesn't appear in the gallery after a rebuild, register the extension again:

  ```bash
  pluginkit -a bin/cuespace.app/Contents/PlugIns/CueSpaceWidgets.appex
  ```

  Moving the app to `/Applications` makes registration automatic.
- **`CUESPACE_DIR`.** The widget always reads the real data folder, so it shows your real Cues even when the app is pointed at a test folder.
- **Clicking brings Cue Space forward.** A widget link opens the app, which then starts the Cue and shows the run report.

## Still to do

- **Shortcuts-app actions** (App Intents: "Start Cue", "List Cues") with Siri and Spotlight phrases. Until then, Shortcuts can use "Open URL" with `cuespace://start/<id>`.
- **Start without bringing the app forward**, using an App Intent button (macOS 14+) that hands the Cue id to the app in the background.
- **Developer ID signing and notarization** for distribution, and an App Group once a Team ID exists.
