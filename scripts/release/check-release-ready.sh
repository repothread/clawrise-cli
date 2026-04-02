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
    echo "Missing required command: ${command_name}" >&2
    exit 1
  fi
}

check_git_state() {
  local current_branch
  current_branch="$(git -C "${repo_root}" branch --show-current 2>/dev/null || true)"

  if [[ -z "${current_branch}" ]]; then
    if [[ "${allow_detached}" != "1" ]]; then
      echo "The repository is currently in detached HEAD state. Standard release checks must run on ${release_branch}; set CLAWRISE_RELEASE_ALLOW_DETACHED=1 only if you intentionally want to bypass this." >&2
      exit 1
    fi
  elif [[ "${current_branch}" != "${release_branch}" ]]; then
    echo "Standard release checks must run on ${release_branch}; the current branch is ${current_branch}." >&2
    exit 1
  fi

  if [[ "${allow_dirty}" != "1" ]] && ! git -C "${repo_root}" diff --quiet --ignore-submodules HEAD --; then
    echo "The working tree has uncommitted changes. Release checks stopped. Set CLAWRISE_RELEASE_ALLOW_DIRTY=1 only if you intentionally want to bypass this." >&2
    exit 1
  fi
}

check_tag_state() {
  local tag_name="v${version}"

  if git -C "${repo_root}" rev-parse -q --verify "refs/tags/${tag_name}" >/dev/null 2>&1; then
    echo "A local tag with the same name already exists: ${tag_name}" >&2
    exit 1
  fi

  if [[ "${check_remote}" == "1" ]]; then
    if git -C "${repo_root}" ls-remote --tags "${release_remote}" "refs/tags/${tag_name}" | grep -q .; then
      echo "A remote tag with the same name already exists on ${release_remote}: ${tag_name}" >&2
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
      echo "Missing expected release artifact: ${path}" >&2
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

  echo "Verifying npm root package pack output: ${root_dir}"
  npm_with_cache pack "${repo_root}/dist/release/npm/${root_dir}" --pack-destination "${pack_verify_root}" >/dev/null

  echo "Verifying current platform package pack output: ${current_platform_dir}"
  npm_with_cache pack "${repo_root}/dist/release/npm/${current_platform_dir}" --pack-destination "${pack_verify_root}" >/dev/null
}

verify_npm_runtime() {
  echo "Verifying platform package resolution after npm installation"
  CLAWRISE_PACK_VERIFY_ROOT="${pack_verify_root}" \
    CLAWRISE_NPM_CACHE_DIR="${npm_cache_dir}" \
    node "${repo_root}/scripts/release/verify-npm-runtime.mjs"
}

verify_release_artifacts() {
  echo "Verifying release artifact consistency"
  CLAWRISE_NPM_DIST_TAG="${CLAWRISE_NPM_DIST_TAG:-}" \
    node "${repo_root}/scripts/release/verify-release-artifacts.mjs" "${version}"
}

check_remote_auth() {
  if [[ "${check_remote}" != "1" ]]; then
    return 0
  fi

  require_command gh

  if ! gh auth status >/dev/null 2>&1; then
    echo "GitHub authentication is not available through gh; remote release checks cannot continue." >&2
    exit 1
  fi

  if [[ -z "${NODE_AUTH_TOKEN:-${NPM_TOKEN:-}}" ]]; then
    echo "NODE_AUTH_TOKEN or NPM_TOKEN is not set. npm publish authentication is skipped locally because Trusted Publishing depends on GitHub Actions OIDC."
    return 0
  fi

  if ! npm_with_cache whoami >/dev/null 2>&1; then
    echo "npm authentication check failed. Confirm that the token is valid." >&2
    exit 1
  fi
}

echo "Starting release readiness checks: version=${version}"

require_command git
require_command go
require_command node
require_command npm

check_git_state
check_tag_state

echo "Running unit tests"
go test ./...

echo "Building multi-platform release bundles"
"${repo_root}/scripts/release/build-npm-bundles.sh" "${version}"

echo "Preparing npm release directories"
node "${repo_root}/scripts/release/prepare-npm-packages.mjs" "${version}"

echo "Generating release notes"
"${repo_root}/scripts/release/generate-release-notes.sh" "${version}"

check_release_outputs
verify_release_artifacts
verify_npm_packs
verify_npm_runtime
check_remote_auth

echo "Release readiness checks passed."
