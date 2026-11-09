#!/usr/bin/env bash
# Build release files into build/release: for each platform, an archive of the
# server with the templates and editor bundled (-tags release), plus the pinned
# typst and its license; on macOS also the Mac app, as Resume-Builder.dmg to
# install and Resume-Builder.zip for its updates (scripts/package-macos.sh).
# Names carry no version, so
# https://github.com/dustinmichels/resume-builder/releases/latest/download/<name>
# always points at the newest release.
#
# Usage: scripts/package.sh [version]   (default: dev)
set -euo pipefail

cd "$(dirname "$0")/.."

version=${1:-dev}
out=build/release
cache=build/typst # downloaded typst archives, kept between runs

die() { echo "package: $*" >&2; exit 1; }

typst_version=$(mise current typst) || die "can't read the typst version from mise"
[[ "$typst_version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || die "unexpected typst version '$typst_version'"

# GOOS GOARCH typst-target
targets=(
  "darwin arm64 aarch64-apple-darwin"
  "darwin amd64 x86_64-apple-darwin"
  "linux amd64 x86_64-unknown-linux-musl"
  "linux arm64 aarch64-unknown-linux-musl"
  "windows amd64 x86_64-pc-windows-msvc"
  "windows arm64 aarch64-pc-windows-msvc"
)

rm -rf "$out" build/stage
mkdir -p "$out" build/stage "$cache"

echo "==> Building the editor"
(cd web && bun install --frozen-lockfile && bun run build)

for target in "${targets[@]}"; do
  read -r goos goarch triple <<<"$target"
  name=resume-builder-$goos-$goarch
  stage=build/stage/$name
  exe=""
  typst_archive=typst-$triple.tar.xz
  if [ "$goos" = windows ]; then
    exe=.exe
    typst_archive=typst-$triple.zip
  fi
  echo "==> $name"

  CGO_ENABLED=0 GOOS=$goos GOARCH=$goarch go build -trimpath -tags release \
    -ldflags "-s -w -X main.buildVersion=$version" -o "$stage/resume-builder$exe" ./server

  cached=$cache/$typst_version-$typst_archive
  if [ ! -f "$cached" ]; then
    curl -fsSL -o "$cached.part" "https://github.com/typst/typst/releases/download/v$typst_version/$typst_archive"
    mv "$cached.part" "$cached"
  fi
  unpacked=build/stage/typst-$triple
  mkdir -p "$unpacked"
  if [ "$goos" = windows ]; then
    unzip -q -o "$cached" -d "$unpacked"
  else
    tar -xJf "$cached" -C "$unpacked"
  fi
  cp "$unpacked/typst-$triple/typst$exe" "$stage/"
  mkdir -p "$stage/licenses/typst"
  cp "$unpacked/typst-$triple/LICENSE" "$unpacked/typst-$triple/NOTICE" "$stage/licenses/typst/"

  if [ "$goos" = windows ]; then
    (cd build/stage && zip -qr "../../$out/$name.zip" "$name")
  else
    tar -czf "$out/$name.tar.gz" -C build/stage "$name"
  fi
done

if [ "$(uname -s)" = Darwin ]; then
  scripts/package-macos.sh "$version" \
    build/stage/typst-aarch64-apple-darwin/typst-aarch64-apple-darwin \
    build/stage/typst-x86_64-apple-darwin/typst-x86_64-apple-darwin
else
  echo "==> Skipping Resume-Builder.dmg and .zip: the Mac app can only be built on macOS"
fi

(
  cd "$out"
  files=(resume-builder-*)
  for f in Resume-Builder.dmg Resume-Builder.zip; do
    [ ! -f "$f" ] || files+=("$f")
  done
  shasum -a 256 "${files[@]}" >checksums.txt
)
rm -rf build/stage
echo
echo "Release files for $version in $out:"
ls -lh "$out"
