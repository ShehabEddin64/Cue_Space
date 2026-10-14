# Developing Cue Space

## Setup

Requirements: macOS, Go 1.25+, Node 20+, and the Wails v3 CLI:

```bash
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.28
```

Run with hot reload:

```bash
wails3 dev
```

Build `bin/cuespace.app` and open it (quits the running copy and refreshes the widget first; add `--no-run` to only build):

```bash
./scripts/build.sh
```

Checks, as CI runs them:

```bash
go vet ./... && go test ./...
cd frontend && npm run build && npm run check:i18n
```

To keep your real Cues out of the way while developing, point the app at another folder with `CUESPACE_DIR=/tmp/cues`.

Development builds are ad-hoc signed with a designated requirement on the bundle id (`build/darwin/designated.req`), so macOS privacy permissions such as Accessibility survive rebuilds.

Releases (a signed, notarized `CueSpace.dmg`) are described in [releasing.md](releasing.md).

## Layout

```
main.go              app bootstrap: data folder, services, menu bar, window
cmd/cuespace-cli     the command-line tool and MCP server
internal/cue         the Cue model and validation (JSON Schema + rules)
internal/store       one .cue.json per Cue, atomic writes, folder watcher
internal/runner      starts a Cue: opens enabled items in order, reports each step
internal/platform    the only OS-specific code: opening things, window layouts, app scan, Trash
internal/ai          providers (Anthropic, Gemini, OpenAI-compatible), retries, backup models, the proposal prompt
internal/tools       the operations behind the CLI and the MCP server
internal/i18n        translated messages (English text is the key)
internal/settings    settings.json
internal/secrets     Keychain access
services/            the layer bound to the frontend (TypeScript bindings are generated)
frontend/            React + TypeScript + Tailwind interface
widgets/             the WidgetKit widget (Swift)
schema/cue.v1.json   the public file format
```

Two rules keep the product predictable:

1. **The runner never talks to the AI**, and the AI never touches the runner or the store. An AI proposal is an unsaved Cue the user reviews.
2. **Only `internal/platform` knows about the OS.** Windows and Linux support means adding files there.

## Design

Cue Space follows Apple’s Human Interface Guidelines for structure and its own identity for character. A design reviewer skill grounded in the guidelines lives in `.claude/skills/apple-design`.

- **Identity.** Cue Red `#D93025` for the mark and primary actions only, Ink `#18181A`, Canvas `#F6F5F3`, Night `#111113` in dark mode. Geist for the interface, Geist Mono for paths and URLs, bundled so nothing loads from the network. Tokens are in [`frontend/src/index.css`](../frontend/src/index.css).
- **The mark.** A ring (the intention) and a dot (the cue that closes it). While the AI works, the dot travels the ring.
- **AI first, never required.** Home leads with the composer; proposals are reviewed in place. The app always says which model is used and where requests go.
- **Mac conventions.** A sidebar you can hide (⌃⌘S), every command in the menu bar, Settings at ⌘,.
- **App icon.** `build/appicon.icon` is the Icon Composer source; builds use `build/darwin/icons.icns`, generated from `build/appicon.png`.

## Translations

The interface uses `t("English text")`; French lives in `frontend/src/i18n.fr.ts`. Go messages use `i18n.T(...)` with French in `internal/i18n/fr.go`. `npm run check:i18n` and `go test ./internal/i18n` fail when a string has no translation or its placeholders don’t match.

## Media

The screenshots in `docs/media` are the real interface, rendered in a browser with mock data, and the demo video was made with the [/brag](https://github.com/latent-spaces/brag) skill.
