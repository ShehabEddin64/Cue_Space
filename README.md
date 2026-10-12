# Cue Space

**Reduce the distance between intention and action.**

Cue Space is an open-source, local-first Mac app. Computers are organized around apps, files and websites; people think in activities: *“I want to work on my project.” “I want to study math.”* Cue Space turns those intentions into **Cues**.

A Cue is a saved context with everything an activity needs. Start it and everything opens, in order.

```
Atlas                         Math
  GitHub                        Textbook
  IDE                           Exercise list
  Claude                        Notes
  Documentation                 Calculator
  localhost                     Course website
  Start Cue →                   Start Cue →
```

Build Cues by hand (links, apps, files, folders, a project folder opened in your IDE), or describe one to your own AI:

> “Create a Cue for working on Atlas. I need my GitHub repo, IDE, Claude, localhost and documentation.”

Cue Space proposes the Cue using the apps you actually have installed. You review it, fill in anything it couldn’t know (such as where the repo lives), and save.

## Principles

- **Activities, not apps.** Organize around what you want to do.
- **Simplicity above automation.** Making a Cue should feel like making a playlist, not programming a workflow.
- **AI creates. Cue Space executes.** AI helps write Cues. Starting a Cue never calls an AI: it is deterministic and works offline.
- **Bring your own AI.** Anthropic, or any OpenAI-compatible server, including local models in Ollama or LM Studio. Optional.
- **Local first.** No account and no cloud. Your Cues are plain files you own.
- **Predictability over magic.** You stay in control of what your Cues do.

*Your computer. Your Cues. Your AI.*

## Your Cues are files

Each Cue is one JSON file in `~/Library/Application Support/CueSpace/cues/`. Read it, edit it in any editor (the app updates live), back it up, sync it, or put it in git. The format is defined by [`schema/cue.v1.json`](schema/cue.v1.json).

```json
{
  "schemaVersion": 1,
  "id": "4f1c2a9be0d3c871",
  "name": "Atlas",
  "icon": "🛰️",
  "resources": [
    { "id": "a1", "type": "url", "label": "GitHub", "url": "https://github.com/me/atlas" },
    { "id": "a2", "type": "folder", "label": "Code", "path": "~/Developer/atlas",
      "openWith": { "bundleId": "com.microsoft.VSCode", "name": "Visual Studio Code" } },
    { "id": "a3", "type": "app", "app": { "bundleId": "com.anthropic.claudefordesktop", "name": "Claude" } },
    { "id": "a4", "type": "url", "url": "http://localhost:3000", "disabled": true }
  ]
}
```

Resource types are `url` (any scheme, including deep links such as `obsidian://`), `app`, `file` and `folder`. Files and folders can name an `openWith` app. Shell commands are deliberately not supported yet.

API keys are stored in the macOS Keychain, never in these files.

## Design

Cue Space follows Apple’s Human Interface Guidelines for structure and its own identity for character. A design reviewer skill grounded in the guidelines lives in `.claude/skills/apple-design`.

- **Identity.** Cue Red `#D93025` for the mark and primary actions only, Ink `#18181A`, Canvas `#F6F5F3`, Night `#111113` in dark mode. Geist for the interface, Geist Mono for paths and URLs. Fonts are bundled, so nothing loads from the network. Tokens are in [`frontend/src/index.css`](frontend/src/index.css), with contrast ratios noted.
- **The mark.** A ring (the intention) and a dot (the cue that closes it). While the AI works, the dot travels the ring.
- **AI first, never required.** Home leads with the composer. Proposals are reviewed in place and can be fixed, refined, retried, undone or discarded. The app always says which model is used and where requests go. Building a Cue by hand is always one click away.
- **Mac conventions.** A sidebar you can hide (⌃⌘S), every command in the menu bar, Settings at ⌘,, and light and dark appearance that follow the system.
- **App icon.** `build/appicon.icon` is the Icon Composer source for macOS 26. Compiling it into `Assets.car` needs full Xcode. Without Xcode, the build uses `build/darwin/icons.icns`, generated from `build/appicon.png`.

## Development

Requirements: macOS, Go 1.25+, Node 20+, and the Wails v3 CLI:

```bash
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.28
```

Run with hot reload:

```bash
wails3 dev
```

Build `bin/cuespace.app`:

```bash
wails3 task darwin:package
```

Run the Go tests:

```bash
go test ./internal/...
```

To keep your real Cues out of the way while developing, point the app at another folder with `CUESPACE_DIR=/tmp/cues`.

### Layout

```
main.go              app bootstrap: data folder, services, window
internal/cue         the Cue model, validation (JSON Schema + rules)
internal/store       one .cue.json per Cue, atomic writes, folder watcher
internal/runner      starts a Cue: opens enabled resources in order, reports each step
internal/platform    the only OS-specific code: `open`, installed-app scan, Trash (macOS)
internal/ai          providers (Anthropic, OpenAI-compatible) and the Cue proposal prompt
internal/settings    settings.json (provider, model, base URL)
internal/secrets     Keychain access
services/            the thin layer bound to the frontend (TypeScript bindings are generated)
frontend/            React + TypeScript + Tailwind UI
schema/cue.v1.json   the public file format
```

Two rules keep the product predictable:

1. **The runner never talks to the AI**, and the AI never touches the runner or the store. An AI proposal is just an unsaved Cue that the user reviews.
2. **Only `internal/platform` knows about the OS.** Windows and Linux support means adding files there.

## Roadmap

Shortcut → Workspace → Context → Continue.

- **Now:** Start a Cue to open everything you need.
- **Next:** global-hotkey quick launcher; a `cue` CLI and `cuespace://start/atlas` deep links (for Raycast, Shortcuts and the terminal); Windows and Linux.
- **Then:** window layouts and a dedicated browser window per Cue.
- **Eventually:** *Continue Atlas* puts you back where you left off: restored tabs, files and editor state.

## License

[MIT](LICENSE)
