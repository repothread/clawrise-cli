#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "${SCRIPT_DIR}/../.." && pwd)"

cd "${REPO_ROOT}"

if [[ -z "${GOCACHE:-}" ]]; then
  export GOCACHE="${TMPDIR:-/tmp}/clawrise-go-build"
fi

tmp_root="$(mktemp -d "${TMPDIR:-/tmp}/clawrise-first-party-smoke.XXXXXX")"
home_dir="${tmp_root}/home"
config_path="${tmp_root}/config.yaml"
docs_dir="${tmp_root}/docs"
cli_bin="${CLAWRISE_BIN:-${tmp_root}/clawrise}"

cleanup() {
  rm -rf "${tmp_root}" "${REPO_ROOT}/.clawrise/plugins"
}
trap cleanup EXIT

mkdir -p "${home_dir}"
export HOME="${home_dir}"

export FEISHU_APP_ID="${FEISHU_APP_ID:-demo-feishu-app-id}"
export FEISHU_APP_SECRET="${FEISHU_APP_SECRET:-demo-feishu-app-secret}"
export NOTION_TOKEN="${NOTION_TOKEN:-demo-notion-token}"
export CLAWRISE_CONFIG="${config_path}"

cat > "${config_path}" <<'YAML'
defaults:
  platform: feishu
  account: feishu_bot
  subject: bot
  platform_accounts:
    feishu: feishu_bot
    notion: notion_bot

auth:
  secret_store:
    backend: encrypted_file
  session_store:
    backend: file

runtime:
  retry:
    max_attempts: 1
    base_delay_ms: 100
    max_delay_ms: 200

accounts:
  feishu_bot:
    title: Feishu Bot
    platform: feishu
    subject: bot
    auth:
      method: feishu.app_credentials
      public:
        app_id: env:FEISHU_APP_ID
      secret_refs:
        app_secret: env:FEISHU_APP_SECRET

  notion_bot:
    title: Notion Bot
    platform: notion
    subject: integration
    auth:
      method: notion.internal_token
      public:
        notion_version: "2026-03-11"
      secret_refs:
        token: env:NOTION_TOKEN
YAML

if [[ ! -x "${cli_bin}" ]]; then
  go build -o "${cli_bin}" ./cmd/clawrise
fi

rm -rf "${REPO_ROOT}/.clawrise/plugins"

echo "Installing first-party provider plugins into the project plugin directory..."
bash "${REPO_ROOT}/scripts/dev-install-first-party-plugins.sh" >/dev/null

assert_contains() {
  local haystack="$1"
  local needle="$2"
  local label="$3"

  if ! grep -Fq -- "${needle}" <<<"${haystack}"; then
    echo "Expected ${label} to contain: ${needle}" >&2
    echo "--- ${label} ---" >&2
    printf '%s\n' "${haystack}" >&2
    exit 1
  fi
}

echo "Checking version output..."
version_output="$("${cli_bin}" version)"
assert_contains "${version_output}" '"version"' "version output"

echo "Checking doctor aggregation..."
doctor_output="$("${cli_bin}" doctor)"
assert_contains "${doctor_output}" '"config_path"' "doctor output"
assert_contains "${doctor_output}" '"checks"' "doctor output"
assert_contains "${doctor_output}" '".clawrise/plugins/feishu/0.1.0/plugin.json"' "doctor output"
assert_contains "${doctor_output}" '".clawrise/plugins/notion/0.1.0/plugin.json"' "doctor output"
assert_contains "${doctor_output}" '"name": "feishu"' "doctor output"
assert_contains "${doctor_output}" '"name": "notion"' "doctor output"

echo "Checking account and default-context commands..."
account_list_output="$("${cli_bin}" account list)"
assert_contains "${account_list_output}" '"name": "feishu_bot"' "account list output"
assert_contains "${account_list_output}" '"name": "notion_bot"' "account list output"

account_inspect_output="$("${cli_bin}" account inspect notion_bot)"
assert_contains "${account_inspect_output}" '"ok": true' "account inspect output"
assert_contains "${account_inspect_output}" '"method": "notion.internal_token"' "account inspect output"

platform_current_output="$("${cli_bin}" platform current)"
assert_contains "${platform_current_output}" '"platform": "feishu"' "platform current output"

subject_current_output="$("${cli_bin}" subject current)"
assert_contains "${subject_current_output}" '"subject": "bot"' "subject current output"

echo "Checking auth method discovery..."
auth_feishu_output="$("${cli_bin}" auth methods --platform feishu)"
assert_contains "${auth_feishu_output}" '"id": "feishu.app_credentials"' "feishu auth methods output"
assert_contains "${auth_feishu_output}" '"id": "feishu.oauth_user"' "feishu auth methods output"

auth_notion_output="$("${cli_bin}" auth methods --platform notion)"
assert_contains "${auth_notion_output}" '"id": "notion.internal_token"' "notion auth methods output"
assert_contains "${auth_notion_output}" '"id": "notion.oauth_public"' "notion auth methods output"

echo "Checking runtime catalog status..."
spec_status_output="$("${cli_bin}" spec status)"
assert_contains "${spec_status_output}" '"ok": true' "spec status output"
assert_contains "${spec_status_output}" '"registered_count"' "spec status output"

echo "Checking spec discovery and single-operation lookups..."
spec_feishu_output="$("${cli_bin}" spec list feishu.calendar.event)"
assert_contains "${spec_feishu_output}" '"full_path": "feishu.calendar.event.create"' "feishu spec list output"

spec_notion_output="$("${cli_bin}" spec list notion.page)"
assert_contains "${spec_notion_output}" '"full_path": "notion.page.get"' "notion spec list output"

spec_get_feishu_output="$("${cli_bin}" spec get feishu.calendar.event.create)"
assert_contains "${spec_get_feishu_output}" '"runtime_status": "registered_and_implemented"' "feishu spec get output"
assert_contains "${spec_get_feishu_output}" '"dry_run_supported": true' "feishu spec get output"

spec_get_notion_output="$("${cli_bin}" spec get notion.page.get)"
assert_contains "${spec_get_notion_output}" '"runtime_status": "registered_and_implemented"' "notion spec get output"
assert_contains "${spec_get_notion_output}" '"dry_run_supported": true' "notion spec get output"

echo "Checking docs and completion generators..."
"${cli_bin}" docs generate notion --out-dir "${docs_dir}" >/dev/null
if [[ ! -f "${docs_dir}/index.md" ]]; then
  echo "Expected docs index to be generated at ${docs_dir}/index.md" >&2
  exit 1
fi

bash_completion_output="$("${cli_bin}" completion bash)"
assert_contains "${bash_completion_output}" '# bash completion for clawrise' "bash completion output"

zsh_completion_output="$("${cli_bin}" completion zsh)"
assert_contains "${zsh_completion_output}" '#compdef clawrise' "zsh completion output"

fish_completion_output="$("${cli_bin}" completion fish)"
assert_contains "${fish_completion_output}" '# fish completion for clawrise' "fish completion output"

echo "Checking dry-run operation paths..."
"${cli_bin}" account use notion_bot >/dev/null

notion_read_output="$("${cli_bin}" notion.page.get --dry-run --json '{"page_id":"page_demo"}')"
assert_contains "${notion_read_output}" '"ok": true' "notion page get output"
assert_contains "${notion_read_output}" '"operation": "notion.page.get"' "notion page get output"
assert_contains "${notion_read_output}" '"account": "notion_bot"' "notion page get output"

notion_write_output="$("${cli_bin}" notion.page.create --dry-run --verify --debug-provider-payload --json '{"title":"Dry Run","parent":{"type":"page_id","id":"page_demo"}}')"
assert_contains "${notion_write_output}" '"ok": true' "notion page create output"
assert_contains "${notion_write_output}" '"status": "dry_run"' "notion page create output"
assert_contains "${notion_write_output}" 'skipped --verify because --dry-run does not execute mutating operations' "notion page create output"

"${cli_bin}" account use feishu_bot >/dev/null

feishu_write_output="$("${cli_bin}" feishu.calendar.event.create --dry-run --json '{"calendar_id":"cal_demo","summary":"Smoke Event","start_at":"2026-03-30T10:00:00+08:00","end_at":"2026-03-30T11:00:00+08:00"}')"
assert_contains "${feishu_write_output}" '"ok": true' "feishu calendar create output"
assert_contains "${feishu_write_output}" '"operation": "feishu.calendar.event.create"' "feishu calendar create output"
assert_contains "${feishu_write_output}" '"account": "feishu_bot"' "feishu calendar create output"

echo "First-party provider smoke checks passed."
