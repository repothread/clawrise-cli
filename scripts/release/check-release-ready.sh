#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "$0")/../.." && pwd)"
version="$("${repo_root}/scripts/release/resolve-version.sh" "${1:-}")"
allow_dirty="${CLAWRISE_RELEASE_ALLOW_DIRTY:-0}"
check_remote="${CLAWRISE_RELEASE_CHECK_REMOTE:-0}"
release_branch="${CLAWRISE_RELEASE_BASE_BRANCH:-main}"
release_remote="${CLAWRISE_RELEASE_REMOTE:-origin}"
allow_detached="${CLAWRISE_RELEASE_ALLOW_DETACHED:-0}"
pack_verify_root="${repo_root}/dist/release/pack-verify"
npm_cache_dir="${CLAWRISE_NPM_CACHE_DIR:-${repo_root}/.cache/npm}"

mkdir -p "${npm_cache_dir}"

npm_with_cache() {
  npm --cache "${npm_cache_dir}" "$@"
}

require_command() {
  local command_name="$1"
  if ! command -v "${command_name}" >/dev/null 2>&1; then
    echo "缺少必要命令: ${command_name}" >&2
    exit 1
  fi
}

check_git_state() {
  local current_branch
  current_branch="$(git -C "${repo_root}" branch --show-current 2>/dev/null || true)"

  if [[ -z "${current_branch}" ]]; then
    if [[ "${allow_detached}" != "1" ]]; then
      echo "当前处于 detached HEAD。标准发版前检查应在 ${release_branch} 分支执行；如确需跳过，请设置 CLAWRISE_RELEASE_ALLOW_DETACHED=1。" >&2
      exit 1
    fi
  elif [[ "${current_branch}" != "${release_branch}" ]]; then
    echo "标准发版前检查应在 ${release_branch} 分支执行，当前分支为 ${current_branch}。" >&2
    exit 1
  fi

  if [[ "${allow_dirty}" != "1" ]] && ! git -C "${repo_root}" diff --quiet --ignore-submodules HEAD --; then
    echo "当前工作区存在未提交修改，发布前检查中止。可设置 CLAWRISE_RELEASE_ALLOW_DIRTY=1 跳过。" >&2
    exit 1
  fi
}

check_tag_state() {
  local tag_name="v${version}"

  if git -C "${repo_root}" rev-parse -q --verify "refs/tags/${tag_name}" >/dev/null 2>&1; then
    echo "本地已存在同名 tag: ${tag_name}" >&2
    exit 1
  fi

  if [[ "${check_remote}" == "1" ]]; then
    if git -C "${repo_root}" ls-remote --tags "${release_remote}" "refs/tags/${tag_name}" | grep -q .; then
      echo "远端 ${release_remote} 已存在同名 tag: ${tag_name}" >&2
      exit 1
    fi
  fi
}

check_release_outputs() {
  local metadata_path="${repo_root}/dist/release/npm/release-metadata.json"
  local notes_path="${repo_root}/dist/release/release-notes.md"
  local checksum_path="${repo_root}/dist/release/archives/SHA256SUMS"

  for path in "${metadata_path}" "${notes_path}" "${checksum_path}"; do
    if [[ ! -f "${path}" ]]; then
      echo "缺少预期产物: ${path}" >&2
      exit 1
    fi
  done
}

verify_npm_packs() {
  local root_dir
  local current_platform_dir

  root_dir="$(
    node -e "const fs=require('fs'); const data=JSON.parse(fs.readFileSync(process.argv[1], 'utf8')); process.stdout.write(data.root_package.dir_name);" \
      "${repo_root}/dist/release/npm/release-metadata.json"
  )"
  current_platform_dir="$(
    node -e "const fs=require('fs'); const path=require('path'); const data=JSON.parse(fs.readFileSync(process.argv[1], 'utf8')); const key=process.platform + ':' + process.arch; const osMap={'darwin':'darwin','linux':'linux','win32':'win32'}; const cpuMap={'x64':'x64','arm64':'arm64'}; const targetOS=osMap[process.platform]; const targetCPU=cpuMap[process.arch]; const item=data.platform_packages.find((entry) => entry.os===targetOS && entry.cpu===targetCPU); if(!item){process.exit(1);} process.stdout.write(item.dir_name);" \
      "${repo_root}/dist/release/npm/release-metadata.json"
  )"

  rm -rf "${pack_verify_root}"
  mkdir -p "${pack_verify_root}"

  echo "校验 npm 根包打包: ${root_dir}"
  npm_with_cache pack "${repo_root}/dist/release/npm/${root_dir}" --pack-destination "${pack_verify_root}" >/dev/null

  echo "校验当前平台包打包: ${current_platform_dir}"
  npm_with_cache pack "${repo_root}/dist/release/npm/${current_platform_dir}" --pack-destination "${pack_verify_root}" >/dev/null
}

check_remote_auth() {
  if [[ "${check_remote}" != "1" ]]; then
    return 0
  fi

  require_command gh

  if ! gh auth status >/dev/null 2>&1; then
    echo "未通过 gh 完成 GitHub 认证，无法执行远端发布检查。" >&2
    exit 1
  fi

  if [[ -z "${NODE_AUTH_TOKEN:-${NPM_TOKEN:-}}" ]]; then
    echo "未设置 NODE_AUTH_TOKEN 或 NPM_TOKEN，无法执行 npm 远端发布检查。" >&2
    exit 1
  fi

  if ! npm_with_cache whoami >/dev/null 2>&1; then
    echo "npm 认证检查失败，请确认 token 可用。" >&2
    exit 1
  fi
}

echo "开始执行发布前检查: version=${version}"

require_command git
require_command go
require_command node
require_command npm

check_git_state
check_tag_state

echo "运行单元测试"
go test ./...

echo "构建多平台发布 bundle"
"${repo_root}/scripts/release/build-npm-bundles.sh" "${version}"

echo "生成 npm 发布目录"
node "${repo_root}/scripts/release/prepare-npm-packages.mjs" "${version}"

echo "生成 release notes"
"${repo_root}/scripts/release/generate-release-notes.sh" "${version}"

check_release_outputs
verify_npm_packs
check_remote_auth

echo "发布前检查通过。"
