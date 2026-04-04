#!/bin/sh
set -eu

if [ "$#" -lt 2 ]; then
  echo "用法: $0 <plugin-root> <platform> [operation]" >&2
  exit 1
fi

PLUGIN_ROOT="$1"
PLATFORM="$2"
OPERATION="${3:-}"

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
REPO_ROOT=$(CDPATH= cd -- "$SCRIPT_DIR/../.." && pwd)

cd "$REPO_ROOT"

if [ -z "${GOCACHE:-}" ]; then
  export GOCACHE="${TMPDIR:-/tmp}/clawrise-go-build"
fi

run_clawrise() {
  if [ -n "${CLAWRISE_BIN:-}" ]; then
    "$CLAWRISE_BIN" "$@"
    return
  fi
  go run ./cmd/clawrise "$@"
}

export CLAWRISE_PLUGIN_PATHS="$PLUGIN_ROOT"

echo "检查开发态 discovery 插件发现与健康状态..."
run_clawrise doctor >/dev/null

echo "检查鉴权方法暴露..."
run_clawrise auth methods --platform "$PLATFORM" >/dev/null

echo "检查 spec 列表暴露..."
run_clawrise spec list "$PLATFORM" >/dev/null

if [ -n "$OPERATION" ]; then
  echo "检查单个 operation spec 暴露..."
  run_clawrise spec get "$OPERATION" >/dev/null
fi

echo "外部 provider 插件开发态接入检查已通过。"
