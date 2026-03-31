# Feishu Common Tasks

## Auth Methods

Current Feishu auth methods include:

- `feishu.app_credentials`
  - `subject=bot`
  - requires `app_id` and `app_secret`
- `feishu.oauth_user`
  - `subject=user`
  - requires `client_id` and `client_secret`
  - supports interactive login

Check first:

```bash
clawrise auth methods --platform feishu
```

## High-Signal Tasks

### 1. Create Or Update A Feishu Calendar Event

Start with:

- `feishu.calendar.event.create`
- `feishu.calendar.event.get`
- `feishu.calendar.event.update`

Playbook:

- `docs/playbooks/en/feishu-calendar-event-upsert.md`

### 2. Update A Feishu Document

Start with:

- `feishu.docs.document.get`
- `feishu.docs.document.get_raw_content`
- `feishu.docs.document.edit`

Playbook:

- `docs/playbooks/en/feishu-document-update.md`

### 3. Create Or Update A Feishu Bitable Record

Start with:

- `feishu.bitable.record.list`
- `feishu.bitable.record.create`
- `feishu.bitable.record.update`

Playbook:

- `docs/playbooks/en/feishu-bitable-record-upsert.md`

## Suggested Flow

For any concrete task, prefer:

```bash
clawrise spec get <operation>
clawrise <operation> --dry-run --json '<payload>'
```

If the user describes a business intent without naming the operation, start with:

```bash
clawrise spec list feishu
```
