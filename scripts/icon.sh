#!/usr/bin/env bash
# Turn macos/icon.svg into macos/AppIcon.icns for the app bundle (macOS only:
# sips and iconutil). Rerun after changing the artwork and commit both files.
set -euo pipefail

cd "$(dirname "$0")/.."

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
set=$work/AppIcon.iconset
mkdir "$set"

sips -s format png macos/icon.svg --out "$work/icon.png" >/dev/null
for size in 16 32 128 256 512; do
  sips -z $size $size "$work/icon.png" --out "$set/icon_${size}x${size}.png" >/dev/null
  sips -z $((size * 2)) $((size * 2)) "$work/icon.png" --out "$set/icon_${size}x${size}@2x.png" >/dev/null
done
iconutil -c icns "$set" -o macos/AppIcon.icns
echo "Wrote macos/AppIcon.icns"
