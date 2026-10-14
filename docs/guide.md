# Cue Space guide

The app has a **Tutorial** tab (Help › Tutorial, ⇧⌘T) that walks through all of this inside Cue Space. This page is the same material for reading on GitHub.

## Cues

A Cue is a saved context for an activity: an ordered list of **links** (any scheme, including deep links such as `obsidian://`), **apps**, **files** and **folders**. Files and folders can open in a specific app, such as a project folder in VS Code. Any item can be switched off without deleting it.

Starting a Cue opens every item in order. It never calls an AI: it does the same thing every time, offline.

## Commands and shortcuts

Type `/` in any composer. Most commands run on your Mac and use no AI tokens at all:

| Command | Where | AI |
| --- | --- | --- |
| `/create <what it’s for>` | Home | Yes |
| `/new <name>` · `/start <cue>` · `/open <cue>` | Home | No |
| `/add figma, ~/Notes, github.com/me/repo` | A Cue, or while refining a proposal | Only for items it can’t find on your Mac |
| `/remove` · `/off` · `/on <item>` | A Cue, or while refining | No |
| `/rename` · `/icon` · `/describe` · `/pin` · `/start` | A Cue | No |

In the empty composer, press **Tab** to use the idea shown and **↑ ↓** to see others. AI requests are kept small: the model sees app names only, and an existing Cue is sent as a few compact lines.

Your first nine Cues get **⌘1–⌘9** (pinned first), also listed in the Cues menu. Press **⌘/** for every shortcut.

## Desktop layouts

A Cue can place its windows on desktops (macOS Spaces): an app in full screen on its own desktop, two apps side by side on a new desktop, quarters, thirds and more, with numbered slots and a Mission Control-style preview. It needs Accessibility permission (System Settings › Privacy & Security › Accessibility). Creating a new desktop automates Mission Control; if that fails, the windows are arranged on the current desktop and the run report says so.

## The newest file in a folder

Tick **Open the newest file inside, not the folder** on a folder item and starting the Cue opens the most recently added or changed file there: your latest download, the newest scan. **Only** limits it to file types such as `pdf`. Hidden files, subfolders and unfinished downloads are skipped.

## Widget and links

A **Cues** widget for the desktop and Notification Center starts a Cue in one click. `cuespace://start/<id>` links do the same from Shortcuts, Raycast or Alfred. See [widgets.md](widgets.md).

## Build Cues from Claude or ChatGPT

Cue Space ships with `cuespace-cli`, a command-line tool that is also an [MCP](https://modelcontextprotocol.io) server. Open **File › Connect AI Apps…**:

- **Claude Desktop:** one click adds Cue Space to Claude's settings (your other servers are kept, and the original file is saved next to it as `.before-cuespace`). Quit Claude with ⌘Q and open it again.
- **Claude Code:** run the `claude mcp add …` command shown in the sheet.
- **ChatGPT:** it connects to tools by web address, so run `cuespace-cli mcp --http` and expose port 8765 through a tunnel. Every request needs the secret in the address; keep it private.

The AI gets `list_cues`, `get_cue`, `create_cue`, `update_cue`, `delete_cue` (moves to the Trash), `start_cue`, `list_apps`, `search_files` and `check_paths`. Changes go through the same validation as the app, and starting a Cue is handed to the app, so it runs exactly like clicking Start.

From the terminal:

```bash
cuespace-cli list
cuespace-cli show math
cuespace-cli start math
cuespace-cli create cue.json        # or pipe JSON in
cuespace-cli apps code
cuespace-cli search "linear algebra"
```

Run `cuespace-cli help` for the JSON format.

## Your Cues are files

Each Cue is one JSON file in `~/Library/Application Support/CueSpace/cues/`. Read it, edit it in any editor (the app updates live), back it up, sync it, or put it in git. The format is defined by [`schema/cue.v1.json`](../schema/cue.v1.json).

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
    { "id": "a4", "type": "folder", "path": "~/Downloads", "newest": true, "extensions": ["pdf"] }
  ]
}
```

API keys are stored in the macOS Keychain, never in these files.

## Settings

- **AI:** Gemini (free tier), Anthropic, any OpenAI-compatible server, or a local model in Ollama or LM Studio. Optional: everything works by hand.
- **Appearance:** System, Light or Dark.
- **Language:** System, English or Français. The interface, menu bar, widget and messages follow it, and the AI writes its proposals in French when the app is in French.
- **Home:** View › Customize Home to change the greeting, reorder or hide sections, and pick a Small, Medium or Large size for each.
