#!/bin/sh
set -eu

if [ "$#" -lt 4 ]; then
  echo "用法: $0 <plugin-source> <plugin-name> <plugin-version> <platform> [operation]" >&2
  echo "示例: $0 file:///tmp/clawrise-plugin-linear-0.1.0-darwin-arm64.tar.gz linear 0.1.0 linear linear.viewer.get" >&2
  exit 1
fi

PLUGIN_SOURCE="$1"
PLUGIN_NAME="$2"
PLUGIN_VERSION="$3"
PLATFORM="$4"
OPERATION="${5:-}"

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

TEMP_HOME=$(mktemp -d "${TMPDIR:-/tmp}/clawrise-plugin-verify-home.XXXXXX")
cleanup() {
  rm -rf "$TEMP_HOME"
}
trap cleanup EXIT

export HOME="$TEMP_HOME"

echo "检查生产态安装链路..."
run_clawrise plugin install "$PLUGIN_SOURCE" >/dev/null

echo "检查安装结果已进入插件目录..."
run_clawrise plugin info "$PLUGIN_NAME" "$PLUGIN_VERSION" >/dev/null

echo "检查安装后校验结果..."
run_clawrise plugin verify "$PLUGIN_NAME" "$PLUGIN_VERSION" >/dev/null

echo "检查 doctor 聚合结果..."
run_clawrise doctor >/dev/null

echo "检查鉴权方法暴露..."
run_clawrise auth methods --platform "$PLATFORM" >/dev/null

echo "检查 spec 列表暴露..."
run_clawrise spec list "$PLATFORM" >/dev/null

if [ -n "$OPERATION" ]; then
  echo "检查单个 operation spec 暴露..."
  run_clawrise spec get "$OPERATION" >/dev/null
fi

echo "外部 provider 插件生产态安装检查已通过。"
