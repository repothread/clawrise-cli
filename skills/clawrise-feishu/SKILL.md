---
name: clawrise-feishu
description: Use when the task is to access Feishu through Clawrise, including Feishu auth setup, calendar events, docs updates, bitable records, wiki nodes, contacts, or any other `feishu.*` operation. Pair with clawrise-core.
---

# Clawrise Feishu

This skill adds Feishu-specific guidance. Use `clawrise-core` for the common execution workflow.

## Usage

1. Start with the `clawrise-core` workflow to inspect the local environment and specs.
2. Add this skill when the task is Feishu-specific.

## Check These First

```bash
clawrise spec list feishu
clawrise auth methods --platform feishu
```

## Auth Constraints

- `feishu.app_credentials`
  - for `bot`
- `feishu.oauth_user`
  - for `user`

If the user does not explicitly say whether the task should use a bot or a user identity, do not guess. Inspect the account config and auth method first.

## Task Rules

- Run `clawrise spec get <operation>` before building JSON
- Prefer `--dry-run` for write operations
- Read before write to avoid overwriting existing data
- Prefer RFC3339 for time fields

## Read This Reference Only When The Task Matches

- `references/common-tasks.md`
