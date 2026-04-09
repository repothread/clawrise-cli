---
name: clawrise-notion
description: Use when a task explicitly targets Notion through Clawrise, including `notion.page.*`, `notion.block.*`, `notion.database.*`, `notion.data_source.*`, `notion.file_upload.*`, `notion.comment.*`, `notion.user.*`, `notion.search.query`, and `notion.task.*` workflows such as markdown import, section patching, file attachment, target resolution, meeting notes extraction, or data-source sync. Pair with clawrise-core for generic setup, spec discovery, and auth inspection.
---

# Clawrise Notion

This skill adds Notion-specific routing and payload guidance on top of `clawrise-core`.

## Fast Path

1. Start with the `clawrise-core` workflow to inspect the local environment and specs.
2. Check these first:

```bash
clawrise spec list notion
clawrise auth methods --platform notion
```

3. Keep the subject fixed to `integration` for current Notion auth methods. Do not switch to `bot` or `user`.
4. For mutating tasks: run `clawrise spec get <operation>`, prefer `--dry-run`, and read before write when page or block content may be overwritten.
5. Use `--verify` and `--debug-provider-payload` only on `notion.page.create`, `notion.page.update`, `notion.block.append`, and `notion.block.update`.

## Auth Constraints

- `notion.internal_token`
  - for `integration`
- `notion.oauth_public`
  - for `integration`

Both current Notion auth methods use the `integration` subject.

## Route By Intent

- exact provider-native page, block, database, or data-source payloads: `notion.page.*`, `notion.block.*`, `notion.database.*`, `notion.data_source.*`
- markdown, heading, path, or section workflows: `notion.task.page.*`
- loose target discovery from a URL, id, or vague workspace context: `notion.search.query`, `notion.task.database.resolve_target`
- one-step upload and append: `notion.task.block.attach_file`; manual upload lifecycle or external upload URLs: `notion.file_upload.*`
- collaboration or meeting context: `notion.comment.*`, `notion.user.*`, `notion.task.meeting_notes.get`

## Read These References Only When The Task Matches

- `references/common-tasks.md` — decision guide, safety notes, and playbook links
- `references/operation-map.md` — exact operation lookup when the user intent is known but the command name is not
