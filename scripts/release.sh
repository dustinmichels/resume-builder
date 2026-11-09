#!/usr/bin/env bash
# Test, package, tag, and publish a GitHub release with the archives from
# scripts/package.sh.
#
# Usage: scripts/release.sh [patch|minor|major|X.Y.Z]   (default: patch)
#   Versions come from the vX.Y.Z git tags; with none yet, a bump releases 0.1.0.
set -euo pipefail

cd "$(dirname "$0")/.."

die() { echo "release: $*" >&2; exit 1; }
step() { printf '\n==> %s\n' "$*"; }

usage="usage: scripts/release.sh [patch|minor|major|X.Y.Z]"
[ $# -le 1 ] || die "$usage"
bump=${1:-patch}
[[ "$bump" =~ ^(patch|minor|major|[0-9]+\.[0-9]+\.[0-9]+)$ ]] || die "$usage"

# --- Preflight: everything that can fail before anything is published ---

step "Checking git state"
if [ -n "$(git status --porcelain)" ]; then
  git status --short >&2
  die "the working tree isn't clean; commit or stash first"
fi
branch=$(git symbolic-ref --short HEAD 2>/dev/null) || die "HEAD is detached"
[ "$branch" = main ] || die "on branch '$branch'; release from main"
git remote get-url origin >/dev/null 2>&1 || die "no 'origin' remote"
git fetch --quiet --tags origin main
[ "$(git rev-parse HEAD)" = "$(git rev-parse origin/main)" ] ||
  die "main and origin/main differ; push or pull first"
gh auth status >/dev/null 2>&1 || die "gh is not logged in; run 'gh auth login'"

latest=$(git tag --list 'v[0-9]*.[0-9]*.[0-9]*' --sort=-v:refname | awk 'NR == 1')
case "$bump" in
  patch | minor | major)
    if [ -z "$latest" ]; then
      version=0.1.0
    else
      IFS=. read -r major minor patch <<<"${latest#v}"
      case "$bump" in
        patch) version=$major.$minor.$((patch + 1)) ;;
        minor) version=$major.$((minor + 1)).0 ;;
        major) version=$((major + 1)).0.0 ;;
      esac
    fi
    ;;
  *) version=$bump ;;
esac
tag=v$version
git rev-parse -q --verify "refs/tags/$tag" >/dev/null && die "tag $tag already exists"
gh release view "$tag" >/dev/null 2>&1 && die "GitHub release $tag already exists"

echo "Releasing $tag (previous: ${latest:-none}) from $(git rev-parse --short HEAD)"

step "Testing"
go test ./server/

step "Packaging"
scripts/package.sh "$tag"
[ -z "$(git status --porcelain)" ] || die "packaging changed tracked files; commit them and rerun"

# --- Tag and publish ---

step "Tagging $tag and publishing the GitHub release"
git tag -a "$tag" -m "Resume Builder $version"
git push origin "$tag"
gh release create "$tag" build/release/* --verify-tag --title "Resume Builder $version" --generate-notes

echo
echo "Released $(gh release view "$tag" --json url --jq .url)"
