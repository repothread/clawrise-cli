---
name: clawrise-notion
description: Use when the task is to access Notion through Clawrise, including Notion auth setup, page updates, markdown updates, data source queries, comments, blocks, or any other `notion.*` operation. Pair with clawrise-core.
---

# Clawrise Notion

This skill adds Notion-specific guidance. Use `clawrise-core` for the common execution workflow.

This skill assumes that the current client has already been prepared with:

```bash
clawrise setup <client> notion
clawrise setup notion
```

or:

```bash
npx @clawrise/clawrise-cli setup <client> notion
npx @clawrise/clawrise-cli setup notion
```

Preferred setup example:

```bash
NOTION_INTERNAL_TOKEN=secret_xxx clawrise setup codex notion
```

Default account name:

- `notion_bot`

## Usage

1. Start with the `clawrise-core` workflow to inspect the local environment and specs.
2. Add this skill only when the task is Notion-specific.
3. Do not use this skill to explain generic client setup unless the user is explicitly setting up Notion support.

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
- Notion block writes accept both shorthand top-level fields and provider-native nested block bodies
- When both block input shapes are present on the same block, top-level fields win
- When reusing Notion block output across tools, preserving the provider-native nested block body is safe

## Read This Reference Only When The Task Matches

- `references/common-tasks.md`
