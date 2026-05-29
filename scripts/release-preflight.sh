#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

allow_dirty=0
allow_existing_tag=0
skip_verify=0

usage() {
  cat >&2 <<'EOF'
usage: scripts/release-preflight.sh [--allow-dirty] [--allow-existing-tag] [--skip-verify] vX.Y.Z

Runs the maintainer preflight for a core CLI release candidate:
  - validate release-version references
  - require a clean worktree by default
  - reject existing local/remote tags by default
  - run make verify by default
  - build release archives and verify SHA256SUMS
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --allow-dirty)
      allow_dirty=1
      shift
      ;;
    --allow-existing-tag)
      allow_existing_tag=1
      shift
      ;;
    --skip-verify)
      skip_verify=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    --*)
      echo "unknown option: $1" >&2
      usage
      exit 2
      ;;
    *)
      break
      ;;
  esac
done

if [[ $# -ne 1 ]]; then
  usage
  exit 2
fi

tag="$1"
version="${tag#v}"
if [[ "$tag" == "$version" || ! "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "release tag must look like vX.Y.Z" >&2
  exit 2
fi

python3 scripts/check-release-version.py "$tag"

if [[ "$allow_dirty" -eq 0 ]]; then
  if ! git diff --quiet || ! git diff --cached --quiet || [[ -n "$(git ls-files --others --exclude-standard)" ]]; then
    echo "release preflight requires a clean worktree; commit or stash changes first" >&2
    exit 1
  fi
fi

if [[ "$allow_existing_tag" -eq 0 ]]; then
  if git rev-parse -q --verify "refs/tags/$tag" >/dev/null; then
    echo "local tag already exists: $tag" >&2
    exit 1
  fi
  if git ls-remote --exit-code --tags origin "refs/tags/$tag" >/dev/null 2>&1; then
    echo "remote tag already exists: $tag" >&2
    exit 1
  fi
else
  tag_commit="$(git rev-list -n 1 "$tag")"
  head_commit="$(git rev-parse HEAD)"
  if [[ "$tag_commit" != "$head_commit" ]]; then
    echo "tag $tag points to $tag_commit, but HEAD is $head_commit" >&2
    exit 1
  fi
fi

if command -v gh >/dev/null && [[ -n "${GH_REPO:-${GITHUB_REPOSITORY:-}}" ]]; then
  repo="${GH_REPO:-${GITHUB_REPOSITORY:-}}"
  if gh release view "$tag" --repo "$repo" >/dev/null 2>&1; then
    echo "GitHub release already exists: $repo $tag" >&2
    exit 1
  fi
fi

if [[ "$skip_verify" -eq 0 ]]; then
  make verify
fi

scripts/release-core-cli.sh "$tag"
(
  cd "dist/release/$tag"
  shasum -a 256 -c SHA256SUMS
)

echo "release-preflight: ok ($tag)"
