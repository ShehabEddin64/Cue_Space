#!/bin/sh
# Builds Cue Space, registers its widget and opens the new build.
#
#   ./scripts/build.sh          build, then open the app
#   ./scripts/build.sh --no-run build only
set -e
cd "$(dirname "$0")/.."
export PATH="$(go env GOPATH)/bin:$PATH"

command -v wails3 >/dev/null || {
	echo "wails3 not found. Install it with: go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.28" >&2
	exit 1
}

# Quit the running copy so the new build replaces it.
pkill -x cuespace 2>/dev/null && sleep 0.5 || true

wails3 generate bindings -ts -d frontend/bindings >/dev/null
# A universal binary left by scripts/release.sh can't be overwritten by go build.
rm -f bin/cuespace
wails3 task darwin:package

# Let WidgetKit pick up the rebuilt widget.
for ext in bin/cuespace.app/Contents/PlugIns/*.appex; do
	[ -e "$ext" ] && pluginkit -a "$ext" 2>/dev/null || true
done

[ "$1" = "--no-run" ] || open bin/cuespace.app
echo "Built bin/cuespace.app"
