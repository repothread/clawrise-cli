# Clawrise

[![CI](https://github.com/repothread/clawrise-cli/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/repothread/clawrise-cli/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

English · [简体中文](README.zh.md)

Clawrise is a CLI for calling third-party services through stable operations. Current built-in platforms:

- `notion`
- `feishu`

## Install

```bash
npm install -g @clawrise/clawrise-cli
clawrise doctor
```

## Notion Quick Start

Prerequisites:

- create a Notion integration
- share the target page or data source with that integration

Create an account:

```bash
clawrise account add notion_docs --platform notion
```

Store the token and check the account:

```bash
export NOTION_TOKEN='secret_xxx'
clawrise auth secret set notion_docs token --from-env NOTION_TOKEN

# Check whether the account configuration is complete
clawrise auth inspect notion_docs
# Confirm that the account is ready to execute operations
clawrise auth check notion_docs
```

Run a simple query:

```bash
clawrise notion.search.query --json '{"query":"Demo","page_size":10}'
```

If you want to validate a write operation first, use dry-run:

```bash
clawrise notion.page.create --dry-run --json '{"parent":{"page_id":"page_demo"},"properties":{"title":[{"text":{"content":"Demo Page"}}]}}'
```

## Feishu Quick Start

Prerequisites:

- create a Feishu app
- get `app_id` and `app_secret`

Create an account:

```bash
clawrise account add feishu_bot --platform feishu
```

Then set `accounts.feishu_bot.auth.public.app_id` in the config file.

Store the secret and check the account:

```bash
export FEISHU_APP_SECRET='your_feishu_app_secret'
clawrise auth secret set feishu_bot app_secret --from-env FEISHU_APP_SECRET

# Check whether the account configuration is complete
clawrise auth inspect feishu_bot
# Confirm that the account is ready to execute operations
clawrise auth check feishu_bot
```

Validate one write call with dry-run:

```bash
clawrise feishu.calendar.event.create --dry-run --json '{"calendar_id":"cal_demo","summary":"Demo Event","start_at":"2026-03-30T10:00:00+08:00","end_at":"2026-03-30T11:00:00+08:00"}'
```

## Common Commands

```bash
clawrise doctor
clawrise auth methods --platform notion
clawrise auth methods --platform feishu
clawrise auth inspect <account>
clawrise auth check <account>
clawrise spec get <operation>
```

## More

- [Notion Page Update Playbook](docs/playbooks/en/notion-page-update.md)
- [Notion Data Source Query Playbook](docs/playbooks/en/notion-data-source-query.md)
- [Feishu User Auth Setup](docs/en/feishu-user-auth-setup.md)
- [Example Config](examples/config.example.yaml)
- [Support](SUPPORT.md)

## License

This project is licensed under the [MIT License](LICENSE).
