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

publish_one() {
  local package_dir_name="$1"
  local package_name="$2"
  local package_dir="${npm_root}/${package_dir_name}"

  if [[ ! -d "${package_dir}" ]]; then
    echo "缺少 npm 包目录: ${package_dir}" >&2
    exit 1
  fi

  if npm_with_cache view "${package_name}@${version}" version >/dev/null 2>&1; then
    echo "npm 包已存在，跳过发布: ${package_name}@${version}"
    return 0
  fi

  echo "发布 npm 包: ${package_name}@${version} (tag=${dist_tag})"
  npm_with_cache publish --access "${access_level}" --tag "${dist_tag}" "${package_dir}"
}

for spec in "${package_specs[@]}"; do
  package_dir_name="${spec%%|*}"
  package_name="${spec#*|}"
  publish_one "${package_dir_name}" "${package_name}"
done
