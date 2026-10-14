#!/bin/sh
# Builds cuespace-cli into an app bundle for the same CPU types as the app
# itself: Apple silicon only for everyday builds, both for universal releases.
#
#   sh scripts/build-cli.sh bin/cuespace.app
set -e
APP="$1"
OUT="$APP/Contents/MacOS/cuespace-cli"
export CGO_ENABLED=1 MACOSX_DEPLOYMENT_TARGET=12.0
export CGO_CFLAGS="-mmacosx-version-min=12.0" CGO_LDFLAGS="-mmacosx-version-min=12.0"

parts=""
for arch in $(lipo -archs "$APP/Contents/MacOS/cuespace"); do
	goarch="$arch"
	[ "$arch" = "x86_64" ] && goarch="amd64"
	GOARCH="$goarch" go build -trimpath -ldflags="-s -w" -o "$OUT-$arch" ./cmd/cuespace-cli
	parts="$parts $OUT-$arch"
done
# shellcheck disable=SC2086 # $parts is a list of paths without spaces in their arch suffixes
lipo -create -output "$OUT" $parts
rm -f $parts
