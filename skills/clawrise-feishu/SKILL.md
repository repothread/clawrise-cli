---
name: clawrise-feishu
description: Deprecated legacy skill. Use only when maintaining or operating existing Feishu-through-Clawrise workflows (`feishu.*`) such as calendar, docs, bitable, wiki, or contact work. Pair with clawrise-core for generic setup, spec discovery, and auth inspection. Do not use this skill for new Feishu integrations; prefer the official Feishu command path instead.
---

# Clawrise Feishu (Deprecated)

Status: deprecated for new work.

Use this skill only for legacy Feishu-through-Clawrise workflows that still need maintenance or operation. Use `clawrise-core` for the common execution workflow.

## Fast Path

1. Start with the `clawrise-core` workflow to inspect the local environment and specs.
2. Use this skill only when the task explicitly names `feishu.*` operations or legacy Feishu objects that are still handled by Clawrise.
3. Check these first:

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
- Do not start brand-new Feishu automation here; prefer the official Feishu command path instead.
- Keep this skill focused on existing Clawrise Feishu flows until they are retired.

## Read This Reference Only When The Task Matches

- `references/common-tasks.md` — currently documented legacy Feishu task families and safety reminders
