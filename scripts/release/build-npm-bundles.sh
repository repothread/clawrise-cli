#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "$0")/../.." && pwd)"
version="$("${repo_root}/scripts/release/resolve-version.sh" "${1:-}")"
commit="${CLAWRISE_RELEASE_COMMIT:-$(git -C "${repo_root}" rev-parse --short HEAD 2>/dev/null || printf 'unknown')}"
build_date="${CLAWRISE_RELEASE_BUILD_DATE:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"
dist_root="${repo_root}/dist/release"
bundles_root="${dist_root}/bundles"
archives_root="${dist_root}/archives"

# 平台映射同时兼顾 Go 交叉编译参数、npm 的 os/cpu 命名，以及对外展示更直观的归档命名。
targets=(
  "darwin arm64 darwin arm64 darwin arm64"
  "darwin amd64 darwin x64 darwin amd64"
  "linux arm64 linux arm64 linux arm64"
  "linux amd64 linux x64 linux amd64"
  "windows arm64 win32 arm64 windows arm64"
  "windows amd64 win32 x64 windows amd64"
)

build_binary() {
  local goos="$1"
  local goarch="$2"
  local output_path="$3"
  local package_path="$4"

  echo "构建 ${goos}/${goarch}: ${package_path}"
  CGO_ENABLED=0 GOOS="${goos}" GOARCH="${goarch}" \
    go build \
    -trimpath \
    -ldflags "-s -w -X github.com/clawrise/clawrise-cli/internal/buildinfo.Version=${version} -X github.com/clawrise/clawrise-cli/internal/buildinfo.Commit=${commit} -X github.com/clawrise/clawrise-cli/internal/buildinfo.BuildDate=${build_date}" \
    -o "${output_path}" \
    "${package_path}"
}

write_provider_manifest() {
  local manifest_path="$1"
  local name="$2"
  local binary_name="$3"

  cat > "${manifest_path}" <<EOF
{
  "schema_version": 2,
  "name": "${name}",
  "version": "${version}",
  "protocol_version": 1,
  "min_core_version": "${version}",
  "capabilities": [
    {
      "type": "provider",
      "platforms": ["${name}"]
    }
  ],
  "entry": {
    "type": "binary",
    "command": ["./bin/${binary_name}"]
  }
}
EOF
}

archive_bundle() {
  local bundle_dir="$1"
  local archive_path="$2"

  tar -C "${bundle_dir}" -czf "${archive_path}" .
}

write_checksums() {
  local target_dir="$1"

  if command -v sha256sum >/dev/null 2>&1; then
    (
      cd "${target_dir}"
      sha256sum ./*.tar.gz > SHA256SUMS
    )
    return 0
  fi

  if command -v shasum >/dev/null 2>&1; then
    (
      cd "${target_dir}"
      shasum -a 256 ./*.tar.gz > SHA256SUMS
    )
    return 0
  fi

  echo "未找到 sha256sum 或 shasum，无法生成归档校验文件。" >&2
  exit 1
}

rm -rf "${bundles_root}" "${archives_root}"
mkdir -p "${bundles_root}" "${archives_root}"

for target in "${targets[@]}"; do
  read -r goos goarch npm_os npm_cpu archive_os archive_arch <<<"${target}"

  bundle_id="${npm_os}-${npm_cpu}"
  archive_id="${archive_os}-${archive_arch}"
  bundle_dir="${bundles_root}/${bundle_id}"
  core_bin_name="clawrise"
  feishu_bin_name="clawrise-plugin-feishu"
  notion_bin_name="clawrise-plugin-notion"

  if [[ "${goos}" == "windows" ]]; then
    core_bin_name="${core_bin_name}.exe"
    feishu_bin_name="${feishu_bin_name}.exe"
    notion_bin_name="${notion_bin_name}.exe"
  fi

  core_bin_path="${bundle_dir}/bin/${core_bin_name}"
  feishu_plugin_dir="${bundle_dir}/plugins/feishu/${version}"
  notion_plugin_dir="${bundle_dir}/plugins/notion/${version}"

  mkdir -p "${bundle_dir}/bin" "${feishu_plugin_dir}/bin" "${notion_plugin_dir}/bin"

  build_binary "${goos}" "${goarch}" "${core_bin_path}" "./cmd/clawrise"
  build_binary "${goos}" "${goarch}" "${feishu_plugin_dir}/bin/${feishu_bin_name}" "./cmd/clawrise-plugin-feishu"
  build_binary "${goos}" "${goarch}" "${notion_plugin_dir}/bin/${notion_bin_name}" "./cmd/clawrise-plugin-notion"

  write_provider_manifest "${feishu_plugin_dir}/plugin.json" "feishu" "${feishu_bin_name}"
  write_provider_manifest "${notion_plugin_dir}/plugin.json" "notion" "${notion_bin_name}"

  archive_bundle "${bundle_dir}" "${archives_root}/clawrise-cli_${version}_${archive_id}.tar.gz"
done

write_checksums "${archives_root}"

echo "已生成发布产物目录: ${dist_root}"
