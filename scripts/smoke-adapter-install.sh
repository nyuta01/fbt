#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
commit="$(git rev-parse HEAD)"

if ! git -C "$repo_root" diff --quiet || ! git -C "$repo_root" diff --cached --quiet; then
  echo "adapter-install-smoke requires a clean committed working tree" >&2
  exit 2
fi

tmp="$(mktemp -d)"
cleanup() {
  chmod -R u+w "$tmp" 2>/dev/null || true
  rm -rf "$tmp"
}
trap cleanup EXIT

git clone --bare "$repo_root" "$tmp/fbt.git" >/dev/null 2>&1

cat >"$tmp/gitconfig" <<EOF
[url "file://$tmp/fbt.git"]
	insteadOf = https://github.com/nyuta01/fbt
EOF

export GIT_CONFIG_GLOBAL="$tmp/gitconfig"
export GOBIN="$tmp/bin"
export GOCACHE="$tmp/gocache"
export GOMODCACHE="$tmp/gomodcache"
export GOPRIVATE="github.com/nyuta01/fbt"
export GONOSUMDB="github.com/nyuta01/fbt"
export GONOPROXY="github.com/nyuta01/fbt"
export GOPROXY="direct"
export GOWORK="off"

install_cmd() {
  local module_path="$1"
  local binary_name="$2"
  go install "${module_path}@${commit}"
  test -x "$GOBIN/$binary_name"
}

install_cmd "github.com/nyuta01/fbt/adapters/command/cmd/fbt-runner-command" "fbt-runner-command"
install_cmd "github.com/nyuta01/fbt/adapters/openai/cmd/fbt-runner-openai" "fbt-runner-openai"
install_cmd "github.com/nyuta01/fbt/adapters/codex-cli/cmd/fbt-runner-codex-cli" "fbt-runner-codex-cli"
install_cmd "github.com/nyuta01/fbt/adapters/claude-code/cmd/fbt-runner-claude-code" "fbt-runner-claude-code"

echo "adapter-install-smoke: ok"
