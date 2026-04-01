#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "$0")/../.." && pwd)"
temp_root="$(mktemp -d)"
trap 'rm -rf "${temp_root}"' EXIT

real_node="$(command -v node)"

assert_contains() {
  local file_path="$1"
  local expected="$2"
  if ! grep -Fq "${expected}" "${file_path}"; then
    echo "断言失败：${file_path} 中缺少 ${expected}" >&2
    exit 1
  fi
}

create_stub_npm() {
  local target_dir="$1"
  mkdir -p "${target_dir}"

  cat > "${target_dir}/npm" <<'EOF'
#!/usr/bin/env bash

set -euo pipefail

published_file="${CLAWRISE_TEST_PUBLISHED_FILE:?}"
publish_log="${CLAWRISE_TEST_PUBLISH_LOG:?}"
test_version="${CLAWRISE_TEST_VERSION:?}"
test_node="${CLAWRISE_TEST_NODE:?}"

args=("$@")
while [[ "${#args[@]}" -ge 2 && "${args[0]}" == "--cache" ]]; do
  args=("${args[@]:2}")
done

command_name="${args[0]:-}"
case "${command_name}" in
  view)
    spec="${args[1]:-}"
    if grep -Fxq "${spec}" "${published_file}"; then
      printf '%s\n' "${test_version}"
      exit 0
    fi
    exit 1
    ;;
  publish)
    package_dir="${args[$((${#args[@]} - 1))]}"
    package_name="$("${test_node}" -e "const fs=require('fs'); const data=JSON.parse(fs.readFileSync(process.argv[1], 'utf8')); process.stdout.write(data.name);" "${package_dir}/package.json")"
    printf '%s\n' "${package_name}" >> "${publish_log}"
    printf '%s@%s\n' "${package_name}" "${test_version}" >> "${published_file}"
    exit 0
    ;;
  *)
    echo "未预期的 npm 调用: $*" >&2
    exit 1
    ;;
esac
EOF

  chmod +x "${target_dir}/npm"
}

create_test_repo() {
  local root="$1"
  local version="$2"

  mkdir -p "${root}/scripts/release" "${root}/dist/release/npm"
  cp "${repo_root}/scripts/release/publish-npm.sh" "${root}/scripts/release/publish-npm.sh"
  cp "${repo_root}/scripts/release/resolve-version.sh" "${root}/scripts/release/resolve-version.sh"
  chmod +x "${root}/scripts/release/publish-npm.sh" "${root}/scripts/release/resolve-version.sh"

  cat > "${root}/dist/release/npm/release-metadata.json" <<EOF
{
  "version": "${version}",
  "dist_tag": "latest",
  "root_package": {
    "dir_name": "clawrise-cli",
    "package_name": "@clawrise/clawrise-cli"
  },
  "platform_packages": [
    {
      "dir_name": "clawrise-cli-linux-x64",
      "package_name": "@clawrise/clawrise-cli-linux-x64",
      "os": "linux",
      "cpu": "x64"
    },
    {
      "dir_name": "clawrise-cli-darwin-arm64",
      "package_name": "@clawrise/clawrise-cli-darwin-arm64",
      "os": "darwin",
      "cpu": "arm64"
    }
  ]
}
EOF

  mkdir -p "${root}/dist/release/npm/clawrise-cli" \
           "${root}/dist/release/npm/clawrise-cli-linux-x64" \
           "${root}/dist/release/npm/clawrise-cli-darwin-arm64"

  cat > "${root}/dist/release/npm/clawrise-cli/package.json" <<EOF
{ "name": "@clawrise/clawrise-cli", "version": "${version}" }
EOF
  cat > "${root}/dist/release/npm/clawrise-cli-linux-x64/package.json" <<EOF
{ "name": "@clawrise/clawrise-cli-linux-x64", "version": "${version}" }
EOF
  cat > "${root}/dist/release/npm/clawrise-cli-darwin-arm64/package.json" <<EOF
{ "name": "@clawrise/clawrise-cli-darwin-arm64", "version": "${version}" }
EOF
}

run_inconsistent_partial_state_test() {
  local case_root="${temp_root}/case-inconsistent"
  local npm_bin_dir="${case_root}/bin"
  local published_file="${case_root}/published.txt"
  local publish_log="${case_root}/publish.log"

  create_test_repo "${case_root}" "1.2.3"
  create_stub_npm "${npm_bin_dir}"
  cat > "${published_file}" <<'EOF'
@clawrise/clawrise-cli@1.2.3
@clawrise/clawrise-cli-linux-x64@1.2.3
EOF
  : > "${publish_log}"

  if PATH="${npm_bin_dir}:$PATH" \
    CLAWRISE_TEST_PUBLISHED_FILE="${published_file}" \
    CLAWRISE_TEST_PUBLISH_LOG="${publish_log}" \
    CLAWRISE_TEST_VERSION="1.2.3" \
    CLAWRISE_TEST_NODE="${real_node}" \
    "${case_root}/scripts/release/publish-npm.sh" 1.2.3 >"${case_root}/stdout.log" 2>"${case_root}/stderr.log"; then
    echo "期望在 root 已存在但平台包缺失时停止发布。" >&2
    exit 1
  fi

  assert_contains "${case_root}/stderr.log" "不一致的部分发布状态"
}

run_recoverable_partial_state_test() {
  local case_root="${temp_root}/case-recoverable"
  local npm_bin_dir="${case_root}/bin"
  local published_file="${case_root}/published.txt"
  local publish_log="${case_root}/publish.log"

  create_test_repo "${case_root}" "1.2.3"
  create_stub_npm "${npm_bin_dir}"
  cat > "${published_file}" <<'EOF'
@clawrise/clawrise-cli-linux-x64@1.2.3
EOF
  : > "${publish_log}"

  PATH="${npm_bin_dir}:$PATH" \
    CLAWRISE_TEST_PUBLISHED_FILE="${published_file}" \
    CLAWRISE_TEST_PUBLISH_LOG="${publish_log}" \
    CLAWRISE_TEST_VERSION="1.2.3" \
    CLAWRISE_TEST_NODE="${real_node}" \
    "${case_root}/scripts/release/publish-npm.sh" 1.2.3 >"${case_root}/stdout.log" 2>"${case_root}/stderr.log"

  assert_contains "${case_root}/stdout.log" "可恢复的部分发布状态"
  assert_contains "${publish_log}" "@clawrise/clawrise-cli-darwin-arm64"
  assert_contains "${publish_log}" "@clawrise/clawrise-cli"
}

run_inconsistent_partial_state_test
run_recoverable_partial_state_test

echo "publish-npm 脚本测试通过。"
