#!/bin/sh
# Builds the Cue Space widget extension and puts it in the app bundle.
# Usage: build.sh path/to/App.app
# Needs only the Xcode command line tools (swiftc and the macOS SDK).
set -eu

APP="$1"
HERE="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "$HERE/../../.." && pwd)"
APPEX="$APP/Contents/PlugIns/CueSpaceWidgets.appex"

if ! command -v swiftc >/dev/null 2>&1; then
  echo "swiftc not found: skipping the widget extension"
  exit 0
fi

SDK="$(xcrun --show-sdk-path)"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

for arch in arm64 x86_64; do
  swiftc -parse-as-library -application-extension \
    -target "$arch-apple-macos14.0" -sdk "$SDK" -O \
    -module-name CueSpaceWidgets \
    "$ROOT/widgets/CueSpaceWidgets.swift" \
    -o "$WORK/CueSpaceWidgets-$arch"
done

for arch in arm64 x86_64; do
  swiftc -target "$arch-apple-macos14.0" -sdk "$SDK" -O \
    "$ROOT/widgets/reload.swift" -o "$WORK/reload-$arch"
done
lipo -create "$WORK/reload-arm64" "$WORK/reload-x86_64" -output "$APP/Contents/MacOS/cuespace-widget-reload"

rm -rf "$APPEX"
mkdir -p "$APPEX/Contents/MacOS"
lipo -create "$WORK/CueSpaceWidgets-arm64" "$WORK/CueSpaceWidgets-x86_64" -output "$APPEX/Contents/MacOS/CueSpaceWidgets"
cp "$HERE/Info.plist" "$APPEX/Contents/Info.plist"
echo "built $APPEX"
