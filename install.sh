#!/bin/sh
set -eu

repo="${FBT_INSTALL_REPO:-nyuta01/fbt}"
version="${FBT_VERSION:-latest}"
install_dir="${FBT_INSTALL_DIR:-$HOME/.local/bin}"
download_base="${FBT_DOWNLOAD_BASE_URL:-}"

usage() {
  cat >&2 <<'EOF'
usage: install.sh [--version vX.Y.Z] [--dir DIR] [--repo OWNER/REPO] [--base-url URL]

Installs the fbt core CLI from GitHub Releases and verifies SHA256SUMS.

Environment variables:
  FBT_VERSION            Release tag to install. Defaults to latest.
  FBT_INSTALL_DIR        Install directory. Defaults to $HOME/.local/bin.
  FBT_INSTALL_REPO       GitHub repo. Defaults to nyuta01/fbt.
  FBT_DOWNLOAD_BASE_URL  Override release asset base URL for tests/mirrors.
EOF
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --version)
      version="${2:-}"
      shift 2
      ;;
    --dir)
      install_dir="${2:-}"
      shift 2
      ;;
    --repo)
      repo="${2:-}"
      shift 2
      ;;
    --base-url)
      download_base="${2:-}"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      usage
      exit 2
      ;;
  esac
done

command_exists() {
  command -v "$1" >/dev/null 2>&1
}

download() {
  url="$1"
  dest="$2"
  if command_exists curl; then
    curl -fsSL "$url" -o "$dest"
  elif command_exists wget; then
    wget -q "$url" -O "$dest"
  else
    echo "curl or wget is required" >&2
    exit 1
  fi
}

resolve_latest_tag() {
  if [ -n "$download_base" ]; then
    echo "FBT_VERSION must be explicit when FBT_DOWNLOAD_BASE_URL is set" >&2
    exit 2
  fi
  if ! command_exists curl; then
    echo "curl is required to resolve the latest release; set FBT_VERSION=vX.Y.Z to use wget" >&2
    exit 1
  fi
  latest_url="$(curl -fsSIL -o /dev/null -w '%{url_effective}' "https://github.com/$repo/releases/latest")"
  tag="$(printf '%s\n' "$latest_url" | sed 's#/$##; s#.*/##')"
  if ! printf '%s\n' "$tag" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+$'; then
    echo "could not resolve latest release tag from $latest_url" >&2
    exit 1
  fi
  printf '%s\n' "$tag"
}

case "$version" in
  latest)
    tag="$(resolve_latest_tag)"
    ;;
  v[0-9]*)
    tag="$version"
    ;;
  [0-9]*)
    tag="v$version"
    ;;
  *)
    echo "version must be latest, vX.Y.Z, or X.Y.Z" >&2
    exit 2
    ;;
esac

version_no_v="${tag#v}"
if ! printf '%s\n' "$tag" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+$'; then
  echo "version must be latest, vX.Y.Z, or X.Y.Z" >&2
  exit 2
fi

case "$(uname -s)" in
  Darwin)
    os="darwin"
    ;;
  Linux)
    os="linux"
    ;;
  MINGW*|MSYS*|CYGWIN*)
    os="windows"
    ;;
  *)
    echo "unsupported OS: $(uname -s)" >&2
    exit 1
    ;;
esac

case "$(uname -m)" in
  x86_64|amd64)
    arch="amd64"
    ;;
  arm64|aarch64)
    arch="arm64"
    ;;
  *)
    echo "unsupported architecture: $(uname -m)" >&2
    exit 1
    ;;
esac

if [ "$os" = "windows" ]; then
  archive="fbt_${version_no_v}_${os}_${arch}.zip"
  binary="fbt.exe"
  target="fbt.exe"
else
  archive="fbt_${version_no_v}_${os}_${arch}.tar.gz"
  binary="fbt"
  target="fbt"
fi

if [ -z "$download_base" ]; then
  download_base="https://github.com/$repo/releases/download/$tag"
fi

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT HUP INT TERM

download "$download_base/$archive" "$tmpdir/$archive"
download "$download_base/SHA256SUMS" "$tmpdir/SHA256SUMS"

grep "  $archive\$" "$tmpdir/SHA256SUMS" >"$tmpdir/SHA256SUMS.current" || {
  echo "SHA256SUMS does not contain $archive" >&2
  exit 1
}

(
  cd "$tmpdir"
  if command_exists shasum; then
    shasum -a 256 -c SHA256SUMS.current
  elif command_exists sha256sum; then
    sha256sum -c SHA256SUMS.current
  else
    echo "shasum or sha256sum is required" >&2
    exit 1
  fi
)

if [ "$os" = "windows" ]; then
  command_exists unzip || {
    echo "unzip is required for Windows archives" >&2
    exit 1
  }
  (cd "$tmpdir" && unzip -q "$archive")
else
  tar -C "$tmpdir" -xzf "$tmpdir/$archive"
fi

mkdir -p "$install_dir"
if command_exists install; then
  install -m 0755 "$tmpdir/$binary" "$install_dir/$target"
else
  cp "$tmpdir/$binary" "$install_dir/$target"
  chmod 0755 "$install_dir/$target"
fi

echo "installed fbt $tag to $install_dir/$target"
"$install_dir/$target" version

case ":$PATH:" in
  *":$install_dir:"*)
    ;;
  *)
    echo "add $install_dir to PATH to run fbt from any directory"
    ;;
esac
