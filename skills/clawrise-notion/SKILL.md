---
name: clawrise-notion
description: Use when the task is to access Notion through Clawrise, including Notion auth setup, page updates, markdown updates, data source queries, comments, blocks, or any other `notion.*` operation. Pair with clawrise-core.
---

# Clawrise Notion

This skill adds Notion-specific guidance. Use `clawrise-core` for the common execution workflow.

## Usage

1. Start with the `clawrise-core` workflow to inspect the local environment and specs.
2. Add this skill when the task is Notion-specific.

## Check These First

```bash
clawrise spec list notion
clawrise auth methods --platform notion
```

## Auth Constraints

- `notion.internal_token`
  - for `integration`
- `notion.oauth_public`
  - for `integration`

Both current Notion auth methods use the `integration` subject. Do not switch to `bot` or `user`.

## Task Rules

- Run `clawrise spec get <operation>` before building JSON
- Prefer `--dry-run` for write operations
- Read before write to avoid overwriting page or block content
- Keep pages, blocks, comments, and data sources in Notion-native fields

## Read This Reference Only When The Task Matches

- `references/common-tasks.md`
