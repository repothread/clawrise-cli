#!/usr/bin/env bash

set -euo pipefail

# 这组 live 测试专门跑 GitHub CI 的真实 Notion 联调。
# 所有资源都挂在一次运行专属的临时页面下，结束后统一归档，避免污染固定 sandbox 页面。

log() {
  printf '[notion-live] %s\n' "$*"
}

fail() {
  printf '[notion-live] 错误: %s\n' "$*" >&2
  exit 1
}

require_env() {
  local name="$1"
  if [[ -z "${!name:-}" ]]; then
    fail "缺少环境变量 ${name}"
  fi
}

require_command() {
  local name="$1"
  if ! command -v "${name}" >/dev/null 2>&1; then
    fail "缺少命令 ${name}"
  fi
}

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "${SCRIPT_DIR}/../.." && pwd)"

require_command go
require_command jq
require_env NOTION_TOKEN
require_env NOTION_PARENT_PAGE_ID

NOTION_VERSION="${NOTION_VERSION:-2026-03-11}"
LIVE_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/clawrise-notion-live.XXXXXX")"
CONFIG_PATH="${LIVE_ROOT}/config.yaml"
CLI_BIN="${LIVE_ROOT}/clawrise"
RUN_STAMP="$(date -u +%Y%m%d%H%M%S)"
RUN_KEY_BASE="notion-live-${GITHUB_RUN_ID:-local}-${RUN_STAMP}"
RUN_TITLE="Clawrise CI Live ${GITHUB_RUN_ID:-local}-${RUN_STAMP}"
RUN_PAGE_ID=""
RUN_DATA_SOURCE_ID=""
RUN_ROW_PAGE_ID=""

export CLAWRISE_CONFIG="${CONFIG_PATH}"

cleanup() {
  local status=$?

  if [[ -n "${RUN_PAGE_ID}" ]]; then
    local cleanup_input
    cleanup_input="$(mktemp "${LIVE_ROOT}/cleanup.XXXXXX.json")"
    cat > "${cleanup_input}" <<EOF
{
  "page_id": "${RUN_PAGE_ID}",
  "archived": true
}
EOF
    if (cd "${REPO_ROOT}" && "${CLI_BIN}" notion.page.update --idempotency-key "${RUN_KEY_BASE}-cleanup-page" --input "${cleanup_input}" >/dev/null); then
      log "已归档本次运行页面 ${RUN_PAGE_ID}"
    else
      log "警告：归档本次运行页面失败，请手动检查 ${RUN_PAGE_ID}"
      status=1
    fi
  fi

  if [[ -n "${GITHUB_STEP_SUMMARY:-}" ]]; then
    {
      echo "## Notion Live Test"
      echo
      echo "- run_page_id: \`${RUN_PAGE_ID:-未创建}\`"
      echo "- data_source_id: \`${RUN_DATA_SOURCE_ID:-未创建}\`"
      echo "- row_page_id: \`${RUN_ROW_PAGE_ID:-未创建}\`"
      echo "- cleanup: $(if [[ -n "${RUN_PAGE_ID}" ]]; then echo "attempted"; else echo "skipped"; fi)"
    } >> "${GITHUB_STEP_SUMMARY}"
  fi

  rm -rf "${LIVE_ROOT}"
  exit "${status}"
}

trap cleanup EXIT

cat > "${CONFIG_PATH}" <<EOF
defaults:
    platform: notion
    platform_accounts:
        notion: notion_ci
    account: notion_ci
    subject: integration
auth:
    secret_store:
        backend: auto
        fallback_backend: encrypted_file
    session_store:
        backend: file
runtime:
    retry:
        max_attempts: 1
        base_delay_ms: 200
        max_delay_ms: 1000
accounts:
    notion_ci:
        title: Notion CI
        platform: notion
        subject: integration
        auth:
            method: notion.internal_token
            public:
                notion_version: "${NOTION_VERSION}"
            secret_refs:
                token: env:NOTION_TOKEN
EOF

log "构建 CLI 与第一方 provider plugin"
(cd "${REPO_ROOT}" && go build -o "${CLI_BIN}" ./cmd/clawrise)
(cd "${REPO_ROOT}" && ./scripts/dev-install-first-party-plugins.sh >/dev/null)

clawrise_json() {
  (cd "${REPO_ROOT}" && "${CLI_BIN}" "$@")
}

extract_json() {
  local expression="$1"
  local payload="$2"
  jq -re "${expression}" <<<"${payload}"
}

write_json_file() {
  local path="$1"
  local content="$2"
  printf '%s\n' "${content}" > "${path}"
}

log "验证 Notion CI 账号鉴权"
clawrise_json auth check notion_ci >/dev/null

create_run_page_input="${LIVE_ROOT}/create_run_page.json"
write_json_file "${create_run_page_input}" "$(cat <<EOF
{
  "title": "${RUN_TITLE}",
  "parent": {
    "type": "page_id",
    "id": "${NOTION_PARENT_PAGE_ID}"
  },
  "children": [
    {
      "type": "heading_1",
      "text": "Notion Live Smoke"
    },
    {
      "type": "paragraph",
      "text": "Initial paragraph"
    },
    {
      "type": "to_do",
      "text": "Initial todo",
      "checked": false
    },
    {
      "type": "toggle",
      "text": "Initial toggle",
      "children": [
        {
          "type": "paragraph",
          "text": "Toggle child"
        }
      ]
    },
    {
      "type": "code",
      "text": "fmt.Println(\"hello\")",
      "language": "go"
    }
  ]
}
EOF
)"

log "创建本次运行专属页面"
run_page_create_output="$(clawrise_json notion.page.create --idempotency-key "${RUN_KEY_BASE}-page-create" --input "${create_run_page_input}")"
RUN_PAGE_ID="$(extract_json '.data.page_id' "${run_page_create_output}")"
log "本次运行页面: ${RUN_PAGE_ID}"

log "验证基础只读能力"
user_me_output="$(clawrise_json notion.user.get --json '{"user_id":"me"}')"
extract_json '.data.user_id' "${user_me_output}" >/dev/null
user_list_output="$(clawrise_json notion.user.list --json '{"page_size":20}')"
extract_json '.data.items | length >= 1' "${user_list_output}" >/dev/null
page_get_output="$(clawrise_json notion.page.get --json "{\"page_id\":\"${RUN_PAGE_ID}\"}")"
extract_json '.data.page_id' "${page_get_output}" >/dev/null
page_property_output="$(clawrise_json notion.page.property_item.get --json "{\"page_id\":\"${RUN_PAGE_ID}\",\"property_id\":\"title\"}")"
extract_json '.data.items | length >= 1' "${page_property_output}" >/dev/null
page_markdown_output="$(clawrise_json notion.page.markdown.get --json "{\"page_id\":\"${RUN_PAGE_ID}\",\"include_transcript\":true}")"
extract_json '.data.markdown | length > 0' "${page_markdown_output}" >/dev/null
search_output="$(clawrise_json notion.search.query --json "{\"query\":\"${RUN_TITLE}\",\"page_size\":10}")"
extract_json '.ok' "${search_output}" >/dev/null

log "验证初始 block 读取"
block_list_output="$(clawrise_json notion.block.list_children --json "{\"block_id\":\"${RUN_PAGE_ID}\",\"page_size\":100}")"
initial_paragraph_id="$(extract_json '.data.items[] | select(.type == "paragraph") | .block_id' "${block_list_output}" | head -n 1)"
extract_json '.data.items | length >= 5' "${block_list_output}" >/dev/null
block_get_output="$(clawrise_json notion.block.get --json "{\"block_id\":\"${initial_paragraph_id}\"}")"
extract_json '.data.block_id' "${block_get_output}" >/dev/null
block_descendants_output="$(clawrise_json notion.block.get_descendants --json "{\"block_id\":\"${RUN_PAGE_ID}\",\"page_size\":100}")"
extract_json '.data.total_descendants >= 5' "${block_descendants_output}" >/dev/null

page_update_input="${LIVE_ROOT}/page_update.json"
write_json_file "${page_update_input}" "$(cat <<EOF
{
  "page_id": "${RUN_PAGE_ID}",
  "title": "${RUN_TITLE} Updated",
  "icon": "🧪"
}
EOF
)"
log "验证页面属性更新"
page_update_output="$(clawrise_json notion.page.update --idempotency-key "${RUN_KEY_BASE}-page-update" --input "${page_update_input}")"
extract_json '.data.title | endswith("Updated")' "${page_update_output}" >/dev/null

markdown_replace_input="${LIVE_ROOT}/markdown_replace.json"
write_json_file "${markdown_replace_input}" "$(cat <<EOF
{
  "page_id": "${RUN_PAGE_ID}",
  "type": "replace_content",
  "replace_content": {
    "new_str": "# Markdown Replace Test\n\nAlpha\n\nBeta"
  }
}
EOF
)"
log "验证 markdown replace"
markdown_replace_output="$(clawrise_json notion.page.markdown.update --idempotency-key "${RUN_KEY_BASE}-markdown-replace" --input "${markdown_replace_input}")"
extract_json '.data.markdown | contains("Alpha")' "${markdown_replace_output}" >/dev/null

markdown_update_input="${LIVE_ROOT}/markdown_update.json"
write_json_file "${markdown_update_input}" "$(cat <<EOF
{
  "page_id": "${RUN_PAGE_ID}",
  "type": "update_content",
  "update_content": {
    "content_updates": [
      {
        "old_str": "Alpha",
        "new_str": "Gamma"
      }
    ]
  }
}
EOF
)"
log "验证 markdown update_content"
markdown_update_output="$(clawrise_json notion.page.markdown.update --idempotency-key "${RUN_KEY_BASE}-markdown-update" --input "${markdown_update_input}")"
extract_json '.data.markdown | contains("Gamma")' "${markdown_update_output}" >/dev/null

markdown_insert_input="${LIVE_ROOT}/markdown_insert.json"
write_json_file "${markdown_insert_input}" "$(cat <<EOF
{
  "page_id": "${RUN_PAGE_ID}",
  "type": "insert_content",
  "insert_content": {
    "content": "## Inserted Section\n\nInserted from CI.",
    "after": "# Markdown Replace Test"
  }
}
EOF
)"
log "验证 markdown insert_content"
markdown_insert_output="$(clawrise_json notion.page.markdown.update --idempotency-key "${RUN_KEY_BASE}-markdown-insert" --input "${markdown_insert_input}")"
extract_json '.data.markdown | contains("Inserted Section")' "${markdown_insert_output}" >/dev/null

markdown_range_input="${LIVE_ROOT}/markdown_range.json"
write_json_file "${markdown_range_input}" "$(cat <<EOF
{
  "page_id": "${RUN_PAGE_ID}",
  "type": "replace_content_range",
  "replace_content_range": {
    "content": "Delta",
    "content_range": "Gamma...Beta"
  }
}
EOF
)"
log "验证 markdown replace_content_range"
markdown_range_output="$(clawrise_json notion.page.markdown.update --idempotency-key "${RUN_KEY_BASE}-markdown-range" --input "${markdown_range_input}")"
extract_json '.data.markdown | contains("Delta")' "${markdown_range_output}" >/dev/null

post_markdown_block_list="$(clawrise_json notion.block.list_children --json "{\"block_id\":\"${RUN_PAGE_ID}\",\"page_size\":100}")"
inserted_paragraph_id="$(extract_json '.data.items[] | select(.type == "paragraph" and .plain_text == "Inserted from CI.") | .block_id' "${post_markdown_block_list}" | head -n 1)"

block_append_input="${LIVE_ROOT}/block_append.json"
write_json_file "${block_append_input}" "$(cat <<EOF
{
  "block_id": "${RUN_PAGE_ID}",
  "children": [
    {
      "type": "callout",
      "text": "Append callout",
      "emoji": "💡"
    },
    {
      "type": "table",
      "has_column_header": true,
      "table_width": 2,
      "rows": [
        {
          "type": "table_row",
          "cells": ["H1", "H2"]
        },
        {
          "type": "table_row",
          "cells": ["R1C1", "R1C2"]
        }
      ]
    },
    {
      "type": "code",
      "text": "fmt.Println(\"before update\")",
      "language": "go"
    },
    {
      "type": "to_do",
      "text": "Todo before update",
      "checked": false
    },
    {
      "type": "image",
      "url": "https://example.com/delete-me.png"
    }
  ]
}
EOF
)"
log "验证 block append"
block_append_output="$(clawrise_json notion.block.append --idempotency-key "${RUN_KEY_BASE}-block-append" --input "${block_append_input}")"
code_block_id="$(extract_json '.data.child_ids[2]' "${block_append_output}")"
todo_block_id="$(extract_json '.data.child_ids[3]' "${block_append_output}")"
image_block_id="$(extract_json '.data.child_ids[4]' "${block_append_output}")"
extract_json '.data.appended_count == 5' "${block_append_output}" >/dev/null

log "验证 block update"
block_text_update_output="$(clawrise_json notion.block.update --idempotency-key "${RUN_KEY_BASE}-block-text-update" --json "{\"block_id\":\"${inserted_paragraph_id}\",\"type\":\"paragraph\",\"text\":\"Inserted from CI updated.\"}")"
extract_json '.data.plain_text == "Inserted from CI updated."' "${block_text_update_output}" >/dev/null
block_code_update_output="$(clawrise_json notion.block.update --idempotency-key "${RUN_KEY_BASE}-block-code-update" --json "{\"block_id\":\"${code_block_id}\",\"type\":\"code\",\"text\":\"fmt.Println(\\\"after update\\\")\",\"language\":\"go\"}")"
extract_json '.data.language == "go"' "${block_code_update_output}" >/dev/null
block_todo_update_output="$(clawrise_json notion.block.update --idempotency-key "${RUN_KEY_BASE}-block-todo-update" --json "{\"block_id\":\"${todo_block_id}\",\"type\":\"to_do\",\"text\":\"Todo after update\",\"checked\":true}")"
extract_json '.data.checked == true' "${block_todo_update_output}" >/dev/null

log "验证 block delete"
block_delete_output="$(clawrise_json notion.block.delete --idempotency-key "${RUN_KEY_BASE}-block-delete" --json "{\"block_id\":\"${image_block_id}\"}")"
extract_json '.data.deleted == true' "${block_delete_output}" >/dev/null

data_source_create_input="${LIVE_ROOT}/data_source_create.json"
write_json_file "${data_source_create_input}" "$(cat <<EOF
{
  "body": {
    "parent": {
      "page_id": "${RUN_PAGE_ID}"
    },
    "title": [
      {
        "type": "text",
        "text": {
          "content": "Clawrise Live Data Source ${RUN_STAMP}"
        }
      }
    ],
    "properties": {
      "Name": {
        "title": {}
      }
    }
  }
}
EOF
)"
log "验证 data_source create"
data_source_create_output="$(clawrise_json notion.data_source.create --idempotency-key "${RUN_KEY_BASE}-data-source-create" --input "${data_source_create_input}")"
RUN_DATA_SOURCE_ID="$(extract_json '.data.data_source_id' "${data_source_create_output}")"

log "验证 data_source get"
data_source_get_output="$(clawrise_json notion.data_source.get --json "{\"data_source_id\":\"${RUN_DATA_SOURCE_ID}\"}")"
extract_json '.data.data_source_id' "${data_source_get_output}" >/dev/null

row_create_input="${LIVE_ROOT}/row_create.json"
write_json_file "${row_create_input}" "$(cat <<EOF
{
  "title": "Clawrise CI Row ${RUN_STAMP}",
  "title_property": "Name",
  "parent": {
    "type": "data_source_id",
    "id": "${RUN_DATA_SOURCE_ID}"
  }
}
EOF
)"
log "验证 data_source 下的 page create"
row_create_output="$(clawrise_json notion.page.create --idempotency-key "${RUN_KEY_BASE}-row-create" --input "${row_create_input}")"
RUN_ROW_PAGE_ID="$(extract_json '.data.page_id' "${row_create_output}")"

data_source_update_input="${LIVE_ROOT}/data_source_update.json"
write_json_file "${data_source_update_input}" "$(cat <<EOF
{
  "data_source_id": "${RUN_DATA_SOURCE_ID}",
  "body": {
    "title": [
      {
        "type": "text",
        "text": {
          "content": "Clawrise Live Data Source ${RUN_STAMP} Updated"
        }
      }
    ]
  }
}
EOF
)"
log "验证 data_source update"
data_source_update_output="$(clawrise_json notion.data_source.update --idempotency-key "${RUN_KEY_BASE}-data-source-update" --input "${data_source_update_input}")"
extract_json '.data.title | endswith("Updated")' "${data_source_update_output}" >/dev/null

log "验证 data_source query"
data_source_query_output="$(clawrise_json notion.data_source.query --json "{\"data_source_id\":\"${RUN_DATA_SOURCE_ID}\",\"page_size\":20}")"
extract_json '.data.items | length >= 1' "${data_source_query_output}" >/dev/null

comment_page_input="${LIVE_ROOT}/comment_page.json"
write_json_file "${comment_page_input}" "$(cat <<EOF
{
  "page_id": "${RUN_PAGE_ID}",
  "text": "Clawrise live comment"
}
EOF
)"
log "验证 page comment create"
comment_page_output="$(clawrise_json notion.comment.create --idempotency-key "${RUN_KEY_BASE}-comment-page" --input "${comment_page_input}")"
page_comment_id="$(extract_json '.data.comment_id' "${comment_page_output}")"
page_discussion_id="$(extract_json '.data.discussion_id' "${comment_page_output}")"

log "验证 page comment get/list"
comment_get_output="$(clawrise_json notion.comment.get --json "{\"comment_id\":\"${page_comment_id}\"}")"
extract_json '.data.comment_id' "${comment_get_output}" >/dev/null
comment_list_page_output="$(clawrise_json notion.comment.list --json "{\"block_id\":\"${RUN_PAGE_ID}\",\"page_size\":20}")"
extract_json '.data.items | length >= 1' "${comment_list_page_output}" >/dev/null

comment_reply_input="${LIVE_ROOT}/comment_reply.json"
write_json_file "${comment_reply_input}" "$(cat <<EOF
{
  "discussion_id": "${page_discussion_id}",
  "text": "Clawrise live reply"
}
EOF
)"
log "验证 discussion reply create"
comment_reply_output="$(clawrise_json notion.comment.create --idempotency-key "${RUN_KEY_BASE}-comment-reply" --input "${comment_reply_input}")"
extract_json '.data.discussion_id == "'"${page_discussion_id}"'"' "${comment_reply_output}" >/dev/null

comment_block_input="${LIVE_ROOT}/comment_block.json"
write_json_file "${comment_block_input}" "$(cat <<EOF
{
  "block_id": "${inserted_paragraph_id}",
  "text": "Clawrise block comment"
}
EOF
)"
log "验证 block comment create/get/list"
comment_block_output="$(clawrise_json notion.comment.create --idempotency-key "${RUN_KEY_BASE}-comment-block" --input "${comment_block_input}")"
block_comment_id="$(extract_json '.data.comment_id' "${comment_block_output}")"
comment_get_block_output="$(clawrise_json notion.comment.get --json "{\"comment_id\":\"${block_comment_id}\"}")"
extract_json '.data.comment_id' "${comment_get_block_output}" >/dev/null
comment_list_block_output="$(clawrise_json notion.comment.list --json "{\"block_id\":\"${inserted_paragraph_id}\",\"page_size\":20}")"
extract_json '.data.items | length >= 1' "${comment_list_block_output}" >/dev/null

log "最终回读页面 markdown"
final_markdown_output="$(clawrise_json notion.page.markdown.get --json "{\"page_id\":\"${RUN_PAGE_ID}\",\"include_transcript\":true}")"
extract_json '.data.markdown | contains("Inserted from CI updated.")' "${final_markdown_output}" >/dev/null

log "Notion live 联调完成"
