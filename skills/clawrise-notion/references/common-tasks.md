# Notion Common Tasks

This reference assumes that `clawrise-core` and `clawrise-notion` have already been installed for the current client.

If the user still needs Notion support installed, run:

```bash
clawrise setup <client> notion
clawrise setup notion
```

Preferred one-line setup:

```bash
NOTION_INTERNAL_TOKEN=secret_xxx clawrise setup codex notion
```

## Auth Methods

Current Notion auth methods include:

- `notion.internal_token`
  - `subject=integration`
  - requires `token`
- `notion.oauth_public`
  - `subject=integration`
  - requires `client_id` and `client_secret`
  - supports interactive login

Check first:

```bash
clawrise spec list notion
clawrise auth methods --platform notion
clawrise auth check notion_bot
```

## High-Signal Tasks

### 1. Find The Right Page, Database, Or Data Source

Start with:

- `notion.search.query`
- `notion.task.database.resolve_target`
- `notion.database.get`
- `notion.data_source.get`

Use when:

- the user provides only a Notion URL, raw id, page id, or vague workspace location
- you need to resolve a row page back to its parent data source or database
- you need to know which data source sits under one database before row or schema work

### 2. Create, Update, Move, Or Inspect One Page

Start with:

- `notion.page.create`
- `notion.page.get`
- `notion.page.property_item.get`
- `notion.page.update`
- `notion.page.move`
- `notion.page.markdown.get`
- `notion.page.markdown.update`

Playbook:

- `docs/playbooks/en/notion-page-update.md`

Notes:

- use `notion.page.property_item.get` when one page property is paginated or only partially expanded
- use `notion.page.markdown.get` before rewriting a page body
- only `notion.page.create` and `notion.page.update` support `--verify` and `--debug-provider-payload`

### 3. Maintain Page Content From Markdown, Headings, Or Paths

Start with:

- `notion.task.page.import_markdown`
- `notion.task.page.upsert_markdown_child`
- `notion.task.page.patch_section`
- `notion.task.page.ensure_sections`
- `notion.task.page.append_under_heading`
- `notion.task.page.find_or_create_by_path`
- `notion.task.page.read_complete`
- `notion.task.page.read_graph`

Use when:

- the user talks in terms of importing a Markdown file, replacing one section, ensuring headings exist, appending under one heading, or finding a nested page path
- the task needs a fuller page read than `notion.page.get` can provide
- the task needs related pages discovered from relation properties

Notes:

- prefer `notion.task.page.patch_section`, `notion.task.page.ensure_sections`, and `notion.task.page.append_under_heading` over raw markdown replacement when the intent is section-scoped
- prefer `notion.task.page.read_complete` when an AI workflow needs full property items plus page markdown
- prefer `notion.task.page.read_graph` when the task is about relation traversal across pages

### 4. Append, Update, Or Inspect Blocks Safely

Start with:

- `notion.block.get`
- `notion.block.list_children`
- `notion.block.get_descendants`
- `notion.block.append`
- `notion.block.update`
- `notion.block.delete`

Notes:

- block writes support both shorthand top-level fields and provider-native nested block bodies
- when both input shapes are present on the same block, top-level fields win
- keep `--dry-run` in the loop until the payload shape is stable
- add `--verify` on `notion.block.append` or `notion.block.update` when you need immediate read-after-write confirmation
- add `--debug-provider-payload` on `notion.block.append` or `notion.block.update` when you need to inspect the final provider request and response
- use `notion.block.get_descendants` instead of repeated `notion.block.list_children` calls when you need the full subtree

Playbook:

- `docs/playbooks/en/notion-block-write.md`

### 5. Upload Or Attach Files

Start with:

- `notion.task.block.attach_file`
- `notion.file_upload.create`
- `notion.file_upload.send`
- `notion.file_upload.complete`
- `notion.file_upload.get`
- `notion.file_upload.list`

Use when:

- the user says "attach this screenshot or file to the page"
- you need manual multi-part upload control
- you need to inspect upload status or work with external upload URLs

Notes:

- prefer `notion.task.block.attach_file` for one-step upload and append flows
- use raw `notion.file_upload.*` when the task must control individual upload parts or manage upload lifecycle explicitly

### 6. Query Or Manage Databases And Data Sources

Start with:

- `notion.database.get`
- `notion.database.create`
- `notion.database.update`
- `notion.data_source.get`
- `notion.data_source.template.list`
- `notion.data_source.query`
- `notion.data_source.create`
- `notion.data_source.update`

Playbook:

- `docs/playbooks/en/notion-data-source-query.md`

Notes:

- use low-level `notion.database.*` and `notion.data_source.*` operations when the user already knows the provider-native payload they want
- `notion.database.get` is often the first stop before row or schema work because it exposes child data sources
- `notion.data_source.template.list` is the first stop when a create flow depends on a template

### 7. Sync Rows Or Schema Changes From External Systems

Start with:

- `notion.task.data_source.row.upsert`
- `notion.task.data_source.bulk_upsert`
- `notion.task.data_source.schema.ensure`

Use when:

- the user wants CRM-style row upserts
- the task is a batch sync from another system into Notion
- the task needs additive schema maintenance without rewriting the whole data source

Notes:

- `notion.task.data_source.row.upsert` matches one row with a provider-native Notion filter and updates or creates it
- `notion.task.data_source.bulk_upsert` is the better default for batch sync flows because it returns per-item results and can continue after partial failures
- `notion.task.data_source.schema.ensure` is additive by default and avoids rewriting existing non-option property definitions

### 8. Read Users, Comments, Or Meeting Notes Context

Start with:

- `notion.user.get`
- `notion.user.list`
- `notion.comment.get`
- `notion.comment.list`
- `notion.comment.create`
- `notion.task.meeting_notes.get`

Use when:

- the user needs visible workspace people, comment threads, or open comments under a page
- the task needs to add a comment or reply into an existing discussion
- the task needs summary, notes, or transcript data from Notion meeting notes blocks

Notes:

- `notion.comment.list` accepts a page id or block id through `block_id`
- `notion.comment.create` targets exactly one of `page_id`, `block_id`, or `discussion_id`
- `notion.task.meeting_notes.get` can discover `meeting_notes` blocks under a page and fetch summary, notes, and transcript sections

## Write Safety

- default to `clawrise spec get <operation>` before drafting JSON
- use `--dry-run` on every mutating operation until the input shape is stable
- read before write when replacing page bodies, patching sections, moving pages, or updating blocks
- use `--verify` only on `notion.page.create`, `notion.page.update`, `notion.block.append`, and `notion.block.update`
- use `--debug-provider-payload` only on the same supported operations when you need the final upstream request and response
- if the user intent is section-scoped, path-scoped, or sync-oriented, prefer the matching `notion.task.*` workflow over a lower-level primitive

## Suggested Flow

For any concrete task, prefer:

```bash
clawrise spec get <operation>
clawrise <operation> --dry-run --json '<payload>'
```

If the user describes a business intent without naming the operation, start with:

```bash
clawrise spec list notion
```
