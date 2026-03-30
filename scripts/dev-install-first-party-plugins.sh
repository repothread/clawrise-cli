#!/usr/bin/env bash

set -euo pipefail

# 这个脚本用于源码开发场景：
# 重新构建第一方 provider plugin，并安装到项目级 .clawrise/plugins 目录。

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
plugin_version="${CLAWRISE_PLUGIN_VERSION:-0.1.0}"

build_provider_plugin() {
  local name="$1"
  local cmd_path="$2"
  local binary_name="$3"
  local plugin_dir="$repo_root/.clawrise/plugins/$name/$plugin_version"

  mkdir -p "$plugin_dir/bin"

  echo "构建 provider plugin: $name@$plugin_version"
  go build -o "$plugin_dir/bin/$binary_name" "$cmd_path"

  cat > "$plugin_dir/plugin.json" <<EOF
{
  "schema_version": 1,
  "name": "$name",
  "version": "$plugin_version",
  "kind": "provider",
  "protocol_version": 1,
  "platforms": ["$name"],
  "entry": {
    "type": "binary",
    "command": ["./bin/$binary_name"]
  }
}
EOF
}

cd "$repo_root"

build_provider_plugin "feishu" "./cmd/clawrise-plugin-feishu" "clawrise-plugin-feishu"
build_provider_plugin "notion" "./cmd/clawrise-plugin-notion" "clawrise-plugin-notion"

echo "已安装项目级 provider plugins 到: $repo_root/.clawrise/plugins"
echo "可继续运行:"
echo "  go run ./cmd/clawrise auth methods --platform feishu"
echo "  go run ./cmd/clawrise auth methods --platform notion"
