#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "$0")/../.." && pwd)"
version="$("${repo_root}/scripts/release/resolve-version.sh" "${1:-}")"
metadata_path="${repo_root}/dist/release/npm/release-metadata.json"
template_path="${repo_root}/packaging/release/release-notes.md.tmpl"
output_path="${2:-${CLAWRISE_RELEASE_NOTES_PATH:-${repo_root}/dist/release/release-notes.md}}"
repository="${CLAWRISE_GITHUB_REPOSITORY:-${GITHUB_REPOSITORY:-repothread/clawrise-cli}}"
git_sha="${CLAWRISE_RELEASE_GIT_SHA:-${GITHUB_SHA:-$(git -C "${repo_root}" rev-parse HEAD 2>/dev/null || printf 'HEAD')}}"

if [[ ! -f "${metadata_path}" ]]; then
  echo "缺少 npm 发布元数据文件，请先执行 prepare-npm-packages.mjs: ${metadata_path}" >&2
  exit 1
fi

if [[ ! -f "${template_path}" ]]; then
  echo "缺少 release notes 模板文件: ${template_path}" >&2
  exit 1
fi

install_package="$(
  node -e "const fs=require('fs'); const data=JSON.parse(fs.readFileSync(process.argv[1], 'utf8')); process.stdout.write(data.root_package.package_name);" "${metadata_path}"
)"
dist_tag="$(
  node -e "const fs=require('fs'); const data=JSON.parse(fs.readFileSync(process.argv[1], 'utf8')); process.stdout.write(data.dist_tag);" "${metadata_path}"
)"

archives_dir="${repo_root}/dist/release/archives"
mapfile -t archives < <(find "${archives_dir}" -maxdepth 1 -type f -name '*.tar.gz' -exec basename {} \; | sort)
if [[ "${#archives[@]}" -eq 0 ]]; then
  echo "未找到 release 归档文件，请先执行 build-npm-bundles.sh。" >&2
  exit 1
fi

asset_list=""
for archive_name in "${archives[@]}"; do
  asset_list+="- \`${archive_name}\`"$'\n'
done

release_type="stable"
release_type_note="- This release is published on the stable channel."
if [[ "${version}" == *-* ]]; then
  release_type="prerelease"
  release_type_note="- This release is a prerelease build. Validate the dist-tag and installation path before promoting it."
fi

rendered="$(cat "${template_path}")"
rendered="${rendered//\{\{VERSION\}\}/${version}}"
rendered="${rendered//\{\{INSTALL_PACKAGE\}\}/${install_package}}"
rendered="${rendered//\{\{DIST_TAG\}\}/${dist_tag}}"
rendered="${rendered//\{\{RELEASE_TYPE\}\}/${release_type}}"
rendered="${rendered//\{\{RELEASE_TYPE_NOTE\}\}/${release_type_note}}"
rendered="${rendered//\{\{GIT_SHA\}\}/${git_sha}}"
rendered="${rendered//\{\{ASSET_LIST\}\}/${asset_list}}"
rendered="${rendered//\{\{DOCS_EN_URL\}\}/https:\/\/github.com\/${repository}\/blob\/${git_sha}\/docs\/en\/npm-release-workflow.md}"
rendered="${rendered//\{\{DOCS_ZH_URL\}\}/https:\/\/github.com\/${repository}\/blob\/${git_sha}\/docs\/zh\/npm-release-workflow.md}"
rendered="${rendered//\{\{RUNBOOK_EN_URL\}\}/https:\/\/github.com\/${repository}\/blob\/${git_sha}\/docs\/en\/npm-release-runbook.md}"
rendered="${rendered//\{\{RUNBOOK_ZH_URL\}\}/https:\/\/github.com\/${repository}\/blob\/${git_sha}\/docs\/zh\/npm-release-runbook.md}"

mkdir -p "$(dirname "${output_path}")"
printf '%s\n' "${rendered}" > "${output_path}"

echo "已生成 release notes: ${output_path}"
