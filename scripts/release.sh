#!/bin/sh
# Builds a signed, notarized dist/CueSpace.dmg for GitHub Releases.
#
#   ./scripts/release.sh 0.2.0                 sign and notarize
#   ./scripts/release.sh 0.2.0 --no-notarize   sign only (for testing)
#
# Needs a "Developer ID Application" certificate in the keychain, and for
# notarization either a notarytool keychain profile (NOTARY_PROFILE, default
# "cuespace-notary") or an App Store Connect API key in NOTARY_KEY_PATH,
# NOTARY_KEY_ID and NOTARY_ISSUER_ID (used by CI).
#
# To try the packaging without a certificate, sign locally instead:
#   SIGN_IDENTITY=- ./scripts/release.sh 0.1.0 --no-notarize
set -eu
VERSION="${1:?usage: release.sh <version> [--no-notarize]}"
NOTARIZE=1
[ "${2:-}" = "--no-notarize" ] && NOTARIZE=0
IDENTITY="${SIGN_IDENTITY:-Developer ID Application}"
cd "$(dirname "$0")/.."
export PATH="$(go env GOPATH)/bin:$PATH"
APP=bin/cuespace.app
mkdir -p dist

# 1. A universal (Apple silicon + Intel) build, with the release version.
wails3 generate bindings -ts -d frontend/bindings >/dev/null
wails3 task darwin:package:universal
plutil -replace CFBundleShortVersionString -string "$VERSION" "$APP/Contents/Info.plist"
plutil -replace CFBundleVersion -string "$VERSION" "$APP/Contents/Info.plist"

# 2. Sign inside-out with Developer ID, the hardened runtime and a timestamp.
TIMESTAMP=--timestamp
[ "$IDENTITY" = "-" ] && TIMESTAMP=--timestamp=none # local signatures can't be timestamped
sign() { codesign --force $TIMESTAMP --options runtime --sign "$IDENTITY" "$@"; }
sign "$APP/Contents/MacOS/cuespace-widget-reload"
sign "$APP/Contents/MacOS/cuespace-cli"
sign --entitlements build/darwin/widgets/widgets.entitlements "$APP/Contents/PlugIns/CueSpaceWidgets.appex"
sign --entitlements build/darwin/app.entitlements "$APP"
codesign --verify --deep --strict --verbose=2 "$APP"

notarize() {
	if [ -n "${NOTARY_KEY_PATH:-}" ]; then
		set -- "$1" --key "$NOTARY_KEY_PATH" --key-id "$NOTARY_KEY_ID" --issuer "$NOTARY_ISSUER_ID"
	else
		set -- "$1" --keychain-profile "${NOTARY_PROFILE:-cuespace-notary}"
	fi
	out=$(xcrun notarytool submit "$@" --wait --output-format json)
	echo "$out"
	echo "$out" | grep -q '"status":"Accepted"' || { echo "Notarization failed" >&2; exit 1; }
}

# 3. Notarize and staple the app itself, so it opens offline too.
if [ "$NOTARIZE" = 1 ]; then
	ditto -c -k --keepParent "$APP" dist/app.zip
	notarize dist/app.zip
	xcrun stapler staple "$APP"
	rm dist/app.zip
fi

# 4. The disk image: the app next to a link to Applications, on the Cue
#    Space background (scripts/make-dmg-background.py).
STAGE=$(mktemp -d)
ditto "$APP" "$STAGE/Cue Space.app"
ln -s /Applications "$STAGE/Applications"
mkdir "$STAGE/.background"
tiffutil -cathidpicheck build/darwin/dmg-background.png build/darwin/dmg-background@2x.png \
	-out "$STAGE/.background/background.tiff" >/dev/null
rm -f dist/CueSpace.dmg dist/rw.dmg
hdiutil create -volname "Cue Space" -srcfolder "$STAGE" -fs HFS+ -format UDRW -ov dist/rw.dmg >/dev/null
rm -rf "$STAGE"
MOUNT=$(hdiutil attach -readwrite -noverify -noautoopen dist/rw.dmg | sed -n 's|.*\(/Volumes/.*\)|\1|p' | tail -1)
DISK=$(basename "$MOUNT")
# Lay out the window. Finder scripting can be refused (no Automation
# permission); the image still works then, just without the layout.
osascript <<OSA || echo "note: Finder layout skipped; the disk image has the default look"
tell application "Finder"
	tell disk "$DISK"
		open
		set current view of container window to icon view
		set toolbar visible of container window to false
		set statusbar visible of container window to false
		set the bounds of container window to {200, 120, 800, 548}
		set opts to the icon view options of container window
		set arrangement of opts to not arranged
		set icon size of opts to 112
		set text size of opts to 13
		set background picture of opts to file ".background:background.tiff"
		set position of item "Cue Space.app" of container window to {160, 190}
		set position of item "Applications" of container window to {440, 190}
		update without registering applications
		delay 1
		close
	end tell
end tell
OSA
sync
hdiutil detach "$MOUNT" -quiet || hdiutil detach "$MOUNT" -force -quiet
hdiutil convert dist/rw.dmg -format UDZO -imagekey zlib-level=9 -o dist/CueSpace.dmg >/dev/null
rm -f dist/rw.dmg
codesign --force $TIMESTAMP --sign "$IDENTITY" dist/CueSpace.dmg

# 5. Notarize and staple the disk image, then check what Gatekeeper sees.
if [ "$NOTARIZE" = 1 ]; then
	notarize dist/CueSpace.dmg
	xcrun stapler staple dist/CueSpace.dmg
	spctl -a -vvv -t install dist/CueSpace.dmg
fi
echo "Built dist/CueSpace.dmg ($VERSION)"
