# Contributing to Cue Space

Thanks for helping. Cue Space aims to stay small, predictable and easy to read, so please keep changes in that spirit.

## Before you start

- For anything larger than a bug fix, open an issue first so we can agree on the approach.
- Read the two architecture rules in the README: the runner never calls the AI, and only `internal/platform` talks to the OS.

## Making a change

1. Set up the tools listed under *Development* in the README.
2. Run `go test ./internal/...` and `cd frontend && npm run build` before opening a pull request.
3. If you change a Go service signature, regenerate the TypeScript bindings with `wails3 generate bindings -ts -d frontend/bindings` and commit the result.
4. If you change the Cue file format, update `schema/cue.v1.json`, the Go model in `internal/cue`, and the tests together. Existing `.cue.json` files must keep loading.

## Style

- Go: `gofmt`, small packages, errors that read well to a user (the UI shows them as they are).
- Frontend: function components, Tailwind with the design tokens in `frontend/src/index.css`, no new UI libraries without discussion.
- New resource types or anything that runs code (scripts, shell commands) need a design discussion first. Predictability is a feature.
