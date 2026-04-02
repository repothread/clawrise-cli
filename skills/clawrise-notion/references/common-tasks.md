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
clawrise auth methods --platform notion
clawrise auth check notion_bot
```

## High-Signal Tasks

### 1. Update A Page Title, Properties, Or Body

Start with:

- `notion.page.get`
- `notion.page.update`
- `notion.page.markdown.get`
- `notion.page.markdown.update`

Playbook:

- `docs/playbooks/en/notion-page-update.md`

### 2. Append Or Update Blocks Safely

Start with:

- `notion.page.create`
- `notion.block.append`
- `notion.block.update`
- `notion.block.get`
- `notion.page.markdown.get`

Notes:

- block writes support both shorthand top-level fields and provider-native nested block bodies
- when both input shapes are present on the same block, top-level fields win
- keep `--dry-run` in the loop until the payload shape is stable

Playbook:

- `docs/playbooks/en/notion-block-write.md`

### 3. Query A Data Source

Start with:

- `notion.data_source.get`
- `notion.data_source.query`

Playbook:

- `docs/playbooks/en/notion-data-source-query.md`

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
