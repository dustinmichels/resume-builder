#!/usr/bin/env bash
# Turn macos/icon.svg into macos/AppIcon.icns for the app bundle and
# server/icon.png, which the desktop app sets as its Dock icon at run time (so
# `mise run app` shows it too). Typst renders the SVG: sips ignores its
# filters. macOS only (sips and iconutil). Rerun after changing the artwork and
# commit all three files.
set -euo pipefail

cd "$(dirname "$0")/.."

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
set=$work/AppIcon.iconset
mkdir "$set"

cp macos/icon.svg "$work/"
cat >"$work/icon.typ" <<'EOF'
#set page(width: 1024pt, height: 1024pt, margin: 0pt, fill: none)
#image("icon.svg", width: 100%)
EOF
typst compile --format png --ppi 72 "$work/icon.typ" "$work/icon.png"

for size in 16 32 128 256 512; do
  sips -z $size $size "$work/icon.png" --out "$set/icon_${size}x${size}.png" >/dev/null
  sips -z $((size * 2)) $((size * 2)) "$work/icon.png" --out "$set/icon_${size}x${size}@2x.png" >/dev/null
done
iconutil -c icns "$set" -o macos/AppIcon.icns
cp "$set/icon_256x256@2x.png" server/icon.png
echo "Wrote macos/AppIcon.icns and server/icon.png"
