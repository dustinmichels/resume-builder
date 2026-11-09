#!/usr/bin/env bash
# Build build/release/Resume-Builder.dmg: a universal (arm64 + x86_64)
# "Resume Builder.app" with the desktop server (-tags "release desktop") and
# typst inside, signed with the Developer ID Application identity in the
# Keychain, notarized and stapled. Called by scripts/package.sh, which has
# built web/dist and unpacked both macOS typst releases.
#
# Usage: scripts/package-macos.sh <version> <typst arm64 dir> <typst x86_64 dir>
#   Each typst dir holds typst, LICENSE and NOTICE.
#
# Without a Developer ID identity the app is signed ad hoc and not notarized:
# fine on this Mac, blocked by Gatekeeper elsewhere. scripts/release.sh refuses
# to release in that case.
#
# Environment:
#   CODESIGN_IDENTITY  signing identity (default: the first "Developer ID Application" one)
#   NOTARY_PROFILE     notarytool keychain profile (default: resume-builder)
set -euo pipefail

cd "$(dirname "$0")/.."

die() { echo "package-macos: $*" >&2; exit 1; }

[ $# -eq 3 ] || die "usage: scripts/package-macos.sh <version> <typst arm64 dir> <typst x86_64 dir>"
version=$1
typst_arm64=$2
typst_x86_64=$3

name="Resume Builder"
bundle_id=com.dustinmichels.resumebuilder
min_macos=13.0 # Go 1.27 targets macOS 13 (Ventura); typst is checked below
out=build/release
dmg=$out/Resume-Builder.dmg
work=build/macos
app="$work/$name.app"
notary_profile=${NOTARY_PROFILE:-resume-builder}

identity=${CODESIGN_IDENTITY:-$(security find-identity -v -p codesigning |
  sed -n 's/^ *[0-9]*) [0-9A-F]* "\(Developer ID Application: .*\)"$/\1/p' | awk 'NR == 1')}

# CFBundleShortVersionString must be numeric: v1.2.3 → 1.2.3, dev → 0.0.0.
bundle_version=${version#v}
[[ "$bundle_version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || bundle_version=0.0.0

rm -rf "$work" "$dmg"
mkdir -p "$app/Contents/MacOS" "$app/Contents/Resources/licenses/typst"

echo "==> $name.app"
for arch in arm64 amd64; do
  CGO_ENABLED=1 GOOS=darwin GOARCH=$arch \
    CGO_CFLAGS="-O2 -mmacosx-version-min=$min_macos" CGO_LDFLAGS="-mmacosx-version-min=$min_macos" \
    go build -trimpath -tags "release desktop production" \
    -ldflags "-s -w -X main.buildVersion=$version" -o "$work/resume-builder-$arch" ./server
done
lipo -create -output "$app/Contents/MacOS/resume-builder" "$work/resume-builder-arm64" "$work/resume-builder-amd64"
lipo -create -output "$app/Contents/MacOS/typst" "$typst_arm64/typst" "$typst_x86_64/typst"
rm "$work"/resume-builder-*
for exe in resume-builder typst; do
  archs=$(lipo -archs "$app/Contents/MacOS/$exe")
  [[ " $archs " == *" x86_64 "* && " $archs " == *" arm64 "* ]] || die "$exe is not universal: $archs"
done
# typst must run on every macOS the app claims to. Its minimum is "minos" in
# LC_BUILD_VERSION or "version" in the older LC_VERSION_MIN_MACOSX.
for arch in arm64 x86_64; do
  typst_min=$(vtool -arch "$arch" -show-build "$app/Contents/MacOS/typst" |
    awk '$1 == "minos" || $1 == "version" { print $2; exit }')
  [ -n "$typst_min" ] || die "can't read the macOS typst ($arch) needs"
  [ "$(printf '%s\n%s\n' "$typst_min" "$min_macos" | sort -V | tail -1)" = "$min_macos" ] ||
    die "typst ($arch) needs macOS $typst_min, newer than LSMinimumSystemVersion $min_macos"
done

cp macos/AppIcon.icns "$app/Contents/Resources/"
cp "$typst_arm64/LICENSE" "$typst_arm64/NOTICE" "$app/Contents/Resources/licenses/typst/"
cat >"$app/Contents/Info.plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleDevelopmentRegion</key><string>en</string>
  <key>CFBundleDisplayName</key><string>$name</string>
  <key>CFBundleExecutable</key><string>resume-builder</string>
  <key>CFBundleIconFile</key><string>AppIcon</string>
  <key>CFBundleIdentifier</key><string>$bundle_id</string>
  <key>CFBundleInfoDictionaryVersion</key><string>6.0</string>
  <key>CFBundleName</key><string>$name</string>
  <key>CFBundlePackageType</key><string>APPL</string>
  <key>CFBundleShortVersionString</key><string>$bundle_version</string>
  <key>CFBundleVersion</key><string>$bundle_version</string>
  <key>LSApplicationCategoryType</key><string>public.app-category.productivity</string>
  <key>LSMinimumSystemVersion</key><string>$min_macos</string>
  <key>NSHighResolutionCapable</key><true/>
</dict>
</plist>
EOF
plutil -lint -s "$app/Contents/Info.plist"

# Inside-out: typst, then the app (which seals the main executable and resources).
# No App Sandbox: it would block arbitrary data folders and running typst.
if [ -n "$identity" ]; then
  echo "==> Signing as $identity"
  sign=(codesign --force --options runtime --timestamp --sign "$identity")
else
  echo "==> Signing ad hoc: no Developer ID Application identity in the Keychain"
  sign=(codesign --force --options runtime --sign -)
fi
"${sign[@]}" "$app/Contents/MacOS/typst"
"${sign[@]}" "$app"
codesign --verify --deep --strict --verbose=2 "$app"

echo "==> Resume-Builder.dmg"
stage=$work/dmg
mkdir -p "$stage"
cp -R "$app" "$stage/"
ln -s /Applications "$stage/Applications"
hdiutil create -quiet -volname "$name" -srcfolder "$stage" -fs HFS+ -format UDZO "$dmg"
rm -rf "$stage"

if [ -z "$identity" ]; then
  echo "package-macos: warning: $dmg is not notarized; Gatekeeper blocks it on other Macs" >&2
  exit 0
fi

codesign --force --timestamp --sign "$identity" "$dmg"

echo "==> Notarizing (this can take a few minutes)"
result=$(xcrun notarytool submit "$dmg" --keychain-profile "$notary_profile" --wait --output-format json) ||
  die "notarytool submit failed: $result"
id=$(plutil -extract id raw - <<<"$result")
status=$(plutil -extract status raw - <<<"$result")
if [ "$status" != Accepted ]; then
  xcrun notarytool log "$id" --keychain-profile "$notary_profile" >&2 || true
  die "notarization $id: $status"
fi

xcrun stapler staple "$dmg"
spctl -a -t open --context context:primary-signature -vv "$dmg"
spctl -a -t exec -vv "$app"
