# Feishu Common Tasks

This is a deprecated legacy compatibility reference for the currently documented Feishu-through-Clawrise flows.

Do not use it as the default path for new Feishu automation. Prefer the official Feishu command path for new work, and use this reference only when an existing Clawrise Feishu flow still needs maintenance.

This reference assumes that `clawrise-core` and `clawrise-feishu` have already been installed for the current client.

If the user still needs Feishu support installed, run:

```bash
clawrise setup <client> feishu
clawrise setup feishu
```

Preferred one-line setup:

```bash
FEISHU_APP_ID=cli_xxx FEISHU_APP_SECRET=cli_secret_xxx clawrise setup codex feishu
```

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
clawrise auth check feishu_bot
```

## High-Signal Tasks

If a Feishu task falls outside the flows below, rely on:

```bash
clawrise spec list feishu
clawrise spec get <operation>
```

If the task is greenfield rather than legacy maintenance, stop and switch away from this skill.

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
