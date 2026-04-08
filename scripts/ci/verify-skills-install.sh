#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "${SCRIPT_DIR}/../.." && pwd)"

cd "${REPO_ROOT}"

version="${1:-0.0.0-skills-smoke.1}"
tmp_root="$(mktemp -d "${TMPDIR:-/tmp}/clawrise-skills-smoke.XXXXXX")"
serve_root="${tmp_root}/serve"
codex_home="${tmp_root}/codex-home"
server_log="${tmp_root}/http.log"

cleanup() {
  if [[ -n "${server_pid:-}" ]]; then
    kill "${server_pid}" >/dev/null 2>&1 || true
    wait "${server_pid}" >/dev/null 2>&1 || true
  fi
  rm -rf "${tmp_root}"
}
trap cleanup EXIT

mkdir -p "${serve_root}" "${codex_home}"

echo "Preparing skill release artifacts..."
node "${REPO_ROOT}/scripts/release/prepare-skill-packages.mjs" "${version}" >/dev/null
cp -R "${REPO_ROOT}/dist/release/skills/." "${serve_root}/"

port="$(
  python3 - <<'PY'
import socket

sock = socket.socket()
sock.bind(("127.0.0.1", 0))
print(sock.getsockname()[1])
sock.close()
PY
)"

echo "Serving generated skill artifacts over HTTP..."
python3 -m http.server "${port}" --bind 127.0.0.1 --directory "${serve_root}" >"${server_log}" 2>&1 &
server_pid=$!
sleep 1

echo "Installing released skills through the generated installer..."
bash "${REPO_ROOT}/dist/release/skills/install.sh" \
  --base-url "http://127.0.0.1:${port}" \
  --codex-home "${codex_home}" \
  clawrise-core \
  clawrise-feishu \
  clawrise-notion >/dev/null

assert_file() {
  local path="$1"
  if [[ ! -f "${path}" ]]; then
    echo "Expected file was not installed: ${path}" >&2
    exit 1
  fi
}

assert_file "${codex_home}/skills/clawrise-core/SKILL.md"
assert_file "${codex_home}/skills/clawrise-core/references/install-and-layout.md"
assert_file "${codex_home}/skills/clawrise-feishu/SKILL.md"
assert_file "${codex_home}/skills/clawrise-feishu/references/common-tasks.md"
assert_file "${codex_home}/skills/clawrise-notion/SKILL.md"
assert_file "${codex_home}/skills/clawrise-notion/references/operation-map.md"

echo "Skills install smoke checks passed."
