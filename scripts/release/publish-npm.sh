#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "$0")/../.." && pwd)"
version="$("${repo_root}/scripts/release/resolve-version.sh" "${1:-}")"
npm_root="${repo_root}/dist/release/npm"
metadata_path="${npm_root}/release-metadata.json"
npm_cache_dir="${CLAWRISE_NPM_CACHE_DIR:-${repo_root}/.cache/npm}"

mkdir -p "${npm_cache_dir}"

npm_with_cache() {
  npm --cache "${npm_cache_dir}" "$@"
}

if [[ ! -f "${metadata_path}" ]]; then
  echo "缺少 npm 发布元数据文件: ${metadata_path}" >&2
  exit 1
fi

dist_tag="${CLAWRISE_NPM_DIST_TAG:-$(
  node -e "const fs=require('fs'); const data=JSON.parse(fs.readFileSync(process.argv[1], 'utf8')); process.stdout.write(data.dist_tag);" "${metadata_path}"
)}"
access_level="${CLAWRISE_NPM_ACCESS:-public}"
allow_inconsistent_partial="${CLAWRISE_RELEASE_ALLOW_INCONSISTENT_PUBLISH:-0}"
root_package_name="$(
  node -e "const fs=require('fs'); const data=JSON.parse(fs.readFileSync(process.argv[1], 'utf8')); process.stdout.write(data.root_package.package_name);" "${metadata_path}"
)"

mapfile -t package_specs < <(
  node -e "
    const fs = require('fs');
    const data = JSON.parse(fs.readFileSync(process.argv[1], 'utf8'));
    for (const item of data.platform_packages) {
      console.log(item.dir_name + '|' + item.package_name);
    }
    console.log(data.root_package.dir_name + '|' + data.root_package.package_name);
  " "${metadata_path}"
)

existing_specs=()
missing_specs=()
platform_existing_count=0
platform_missing_count=0
root_exists=0

publish_one() {
  local package_dir_name="$1"
  local package_name="$2"
  local package_dir="${npm_root}/${package_dir_name}"

  if [[ ! -d "${package_dir}" ]]; then
    echo "缺少 npm 包目录: ${package_dir}" >&2
    exit 1
  fi

  echo "发布 npm 包: ${package_name}@${version} (tag=${dist_tag})"
  npm_with_cache publish --access "${access_level}" --tag "${dist_tag}" "${package_dir}"
}

for spec in "${package_specs[@]}"; do
  package_dir_name="${spec%%|*}"
  package_name="${spec#*|}"

  if npm_with_cache view "${package_name}@${version}" version >/dev/null 2>&1; then
    existing_specs+=("${spec}")
    if [[ "${package_name}" == "${root_package_name}" ]]; then
      root_exists=1
    else
      platform_existing_count=$((platform_existing_count + 1))
    fi
  else
    missing_specs+=("${spec}")
    if [[ "${package_name}" != "${root_package_name}" ]]; then
      platform_missing_count=$((platform_missing_count + 1))
    fi
  fi
done

echo "npm 发布预检: total=${#package_specs[@]} existing=${#existing_specs[@]} missing=${#missing_specs[@]} dist-tag=${dist_tag}"

if [[ "${root_exists}" == "1" && "${platform_missing_count}" -gt 0 && "${allow_inconsistent_partial}" != "1" ]]; then
  echo "检测到不一致的部分发布状态：root package 已存在，但仍有平台包缺失。" >&2
  echo "这通常意味着之前的发布顺序被破坏，默认停止后续发布。若确认需要强制继续，可设置 CLAWRISE_RELEASE_ALLOW_INCONSISTENT_PUBLISH=1。" >&2
  exit 1
fi

if [[ "${root_exists}" == "0" && "${platform_existing_count}" -gt 0 ]]; then
  echo "检测到可恢复的部分发布状态：部分平台包已存在，但 root package 尚未发布。脚本将继续只发布缺失项。"
fi

if [[ "${#missing_specs[@]}" -eq 0 ]]; then
  echo "所有 npm 包都已存在，本次无需发布。"
  exit 0
fi

for spec in "${package_specs[@]}"; do
  package_dir_name="${spec%%|*}"
  package_name="${spec#*|}"
  if npm_with_cache view "${package_name}@${version}" version >/dev/null 2>&1; then
    echo "npm 包已存在，跳过发布: ${package_name}@${version}"
    continue
  fi
  publish_one "${package_dir_name}" "${package_name}"
done
