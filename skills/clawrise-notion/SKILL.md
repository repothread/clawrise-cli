---
name: clawrise-notion
description: Use when the task is to access Notion through Clawrise, including Notion auth setup, page and block editing, markdown workflows, database and data source management, file uploads, comments, users, search, or any other `notion.*` operation. Pair with clawrise-core.
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

## Use This Skill For

- raw Notion operations such as `notion.page.*`, `notion.block.*`, `notion.database.*`, `notion.data_source.*`, `notion.comment.*`, `notion.user.*`, `notion.file_upload.*`, and `notion.search.query`
- higher-level Notion workflows such as `notion.task.page.*`, `notion.task.block.attach_file`, `notion.task.database.resolve_target`, `notion.task.data_source.*`, and `notion.task.meeting_notes.get`

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
- Keep pages, blocks, comments, databases, and data sources in Notion-native fields
- Notion block writes accept both shorthand top-level fields and provider-native nested block bodies
- When both block input shapes are present on the same block, top-level fields win
- When reusing Notion block output across tools, preserving the provider-native nested block body is safe
- Use `--verify` and `--debug-provider-payload` only on `notion.page.create`, `notion.page.update`, `notion.block.append`, and `notion.block.update`
- Prefer the matching `notion.task.page.*` workflow over raw `notion.page.markdown.update` when the intent is section-scoped, heading-scoped, path-scoped, or markdown import oriented
- Prefer `notion.task.block.attach_file` for one-step upload and append flows; use raw `notion.file_upload.*` only when the task needs manual multi-part control or external upload URLs

## Route By Intent

- find a page, database, or data source from loose context: `notion.search.query`, `notion.task.database.resolve_target`
- inspect or edit one page: `notion.page.*`, `notion.page.markdown.*`, `notion.page.property_item.get`
- inspect or edit blocks: `notion.block.*`
- maintain page structure from markdown or headings: `notion.task.page.*`
- upload or attach files: `notion.task.block.attach_file`, `notion.file_upload.*`
- create or update databases, data sources, or schema: `notion.database.*`, `notion.data_source.*`, `notion.task.data_source.*`
- read collaboration context: `notion.comment.*`, `notion.user.*`, `notion.task.meeting_notes.get`

## Read These References Only When The Task Matches

- `references/common-tasks.md`
- `references/operation-map.md`
