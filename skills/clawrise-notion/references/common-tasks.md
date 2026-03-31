# Notion Common Tasks

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

### 2. Query A Data Source

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
