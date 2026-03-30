# Clawrise

[![CI](https://github.com/repothread/clawrise-cli/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/repothread/clawrise-cli/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

For the Chinese version, see [README.zh.md](README.zh.md).

## Overview

Clawrise is an agent-native CLI execution layer for SaaS APIs.

It is designed to let AI agents call third-party systems through stable CLI operations instead of heavyweight MCP-style tool schemas.

The current architecture is plugin-first:

- `clawrise` is the core runtime and CLI
- provider capabilities are exposed through external provider plugins
- first-party Feishu and Notion support are shipped as plugin binaries

The repository contains both the evolving design documents and the current Go implementation of the core runtime and first-party plugins.

## Open Source Status

Clawrise is developed as an open source project and now includes the baseline collaboration documents expected by external contributors:

- [MIT License](LICENSE)
- [Contributing Guide](CONTRIBUTING.md)
- [Code of Conduct](CODE_OF_CONDUCT.md)
- [Security Policy](SECURITY.md)
- [Support Guide](SUPPORT.md)
- [Chinese Contributing Guide](CONTRIBUTING.zh.md)
- [Chinese Security Policy](SECURITY.zh.md)
- [Public Roadmap](docs/en/roadmap.md)

## Current Scope

Current first-party plugin platforms:

- `feishu`
- `notion`

Candidate next platform after core hardening:

- `google`

Roadmap scope:

- [docs/en/roadmap.md](docs/en/roadmap.md) tracks forward-looking OSS core work only
- shipped capabilities are summarized in the `Status` section below

## Prerequisites

Before you try Feishu or Notion from a source checkout, prepare:

- Go `1.22.5` or newer
- one Feishu app or one Notion integration
- one secret-storage choice:
  - keep `auth.secret_store.backend: auto` if your system keychain works
  - or use `encrypted_file` with `CLAWRISE_MASTER_KEY` for a portable local-dev setup

## Quick Start From Source

### 1. Build the Core and Local Provider Plugins

```bash
go build ./...
./scripts/dev-install-first-party-plugins.sh

go run ./cmd/clawrise doctor
go run ./cmd/clawrise auth methods --platform feishu
go run ./cmd/clawrise auth methods --platform notion
```

Notes:

- `./scripts/dev-install-first-party-plugins.sh` rebuilds the first-party Feishu and Notion provider plugins into project-local `.clawrise/plugins/`
- project-local plugins are discovered automatically and ignored by Git
- `clawrise plugin list` currently reports globally installed packages under `~/.clawrise/plugins`; use `doctor` or `auth methods` to confirm project-local discovery

### 2. Choose a Secret Store Strategy

If your OS keychain works, you can keep the default:

```yaml
auth:
  secret_store:
    backend: auto
    fallback_backend: encrypted_file
```

If you want a portable source-based setup, or if `auth secret set` fails on Keychain / Secret Service, use:

```bash
export CLAWRISE_MASTER_KEY='replace-this-with-a-long-random-string'
```

Then update the generated config to:

```yaml
auth:
  secret_store:
    backend: encrypted_file
    fallback_backend: encrypted_file
```

### 3. Connect Feishu

#### Recommended First Path: Feishu Bot App Credentials

Create the account skeleton:

```bash
go run ./cmd/clawrise config init --platform feishu --preset bot --account feishu_bot_ops --force
```

`config init` creates the account skeleton, but it does not fill provider-specific public fields yet. Open the generated config file and set the Feishu `app_id`:

```yaml
accounts:
  feishu_bot_ops:
    auth:
      public:
        app_id: cli_your_feishu_app_id
```

Store the secret, inspect the account, and validate one dry-run call:

```bash
export FEISHU_APP_SECRET='your_feishu_app_secret'
go run ./cmd/clawrise auth secret set feishu_bot_ops app_secret --from-env FEISHU_APP_SECRET
go run ./cmd/clawrise auth inspect feishu_bot_ops
go run ./cmd/clawrise feishu.calendar.event.create --dry-run --json '{"calendar_id":"cal_demo","summary":"Demo Event","start_at":"2026-03-30T10:00:00+08:00","end_at":"2026-03-30T11:00:00+08:00"}'
```

If `auth inspect` still reports `missing_public_fields=["app_id"]`, the config file still needs the real Feishu App ID.

#### Need User Identity Instead of Bot Identity?

Use the interactive preset:

```bash
go run ./cmd/clawrise config init --platform feishu --preset user --account feishu_user_default --force
```

Then:

- fill `accounts.<name>.auth.public.client_id` in the config file
- store `client_secret` with `auth secret set`
- run `go run ./cmd/clawrise auth login <account>`
- finish with `go run ./cmd/clawrise auth complete <flow_id>`

For the full manual OAuth walkthrough, see [docs/en/feishu-user-auth-setup.md](docs/en/feishu-user-auth-setup.md).

### 4. Connect Notion

#### Recommended First Path: Notion Internal Integration Token

Create the account skeleton:

```bash
go run ./cmd/clawrise config init --platform notion --preset internal_token --account notion_team_docs --force
```

The default `notion_version` is already filled. Store the token, inspect the account, and validate one dry-run call:

```bash
export NOTION_TOKEN='secret_xxx'
go run ./cmd/clawrise auth secret set notion_team_docs token --from-env NOTION_TOKEN
go run ./cmd/clawrise auth inspect notion_team_docs
go run ./cmd/clawrise notion.page.create --dry-run --json '{"parent":{"page_id":"page_demo"},"properties":{"title":[{"text":{"content":"Demo Page"}}]}}'
```

#### Need Public OAuth Instead of an Internal Token?

Use the interactive preset:

```bash
go run ./cmd/clawrise config init --platform notion --preset public_oauth --account notion_public_default --force
```

Then:

- fill `accounts.<name>.auth.public.client_id` in the config file
- store `client_secret` with `auth secret set`
- run `go run ./cmd/clawrise auth login <account>`
- finish with `go run ./cmd/clawrise auth complete <flow_id>`

### 5. Recommended First-Run Workflow

For both platforms, the safest first-run sequence is:

```bash
go run ./cmd/clawrise auth inspect <account>
go run ./cmd/clawrise auth check <account>
go run ./cmd/clawrise <operation> --dry-run --json '<payload>'
```

Then use:

- `go run ./cmd/clawrise spec get <operation>` to inspect the operation contract
- `docs/playbooks/en/*.md` for task-oriented examples
- [examples/config.example.yaml](examples/config.example.yaml) for a multi-account config template

## Documentation

- [CLI Layer Design](docs/en/cli-layer-design.md)
- [Plugin System Design](docs/en/plugin-system-design.md)
- [Roadmap](docs/en/roadmap.md)
- [Local Playbooks Index](docs/playbooks/index.yaml)
- [`spec` Subsystem Design](docs/en/spec-design.md)
- [Auth Model](docs/en/auth-model.md)
- [MVP Operation Spec](docs/en/mvp-operation-spec.md)
- [Feishu User Auth Setup](docs/en/feishu-user-auth-setup.md)

## Design Focus

- CLI command model
- provider plugin architecture
- auth and account model
- idempotency and audit rules
- operation contracts for Feishu and Notion

## Modeling Boundary

Clawrise standardizes how operations are executed, not how every SaaS resource is modeled.

- The runtime contract should stay unified.
- Resource fields should remain provider-native.
- Feishu docs, Notion pages, calendars, sheets, databases, and future APIs must not be forced into one shared global schema.
- If a cross-platform workflow is useful later, it should be added as an optional higher-level layer rather than replacing provider-specific operation contracts.

## Example Config

- [examples/config.example.yaml](examples/config.example.yaml)

## Contributing

Contributions are welcome across runtime behavior, plugins, docs, playbooks, examples, and test coverage.

Before opening a pull request:

- read [CONTRIBUTING.md](CONTRIBUTING.md)
- check [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)
- follow [SECURITY.md](SECURITY.md) for responsible disclosure
- check [SUPPORT.md](SUPPORT.md) for the correct help path and issue expectations
- use the GitHub issue and pull request templates where relevant

## Support

If you need help using or extending Clawrise, start with [SUPPORT.md](SUPPORT.md).

For current project priorities and larger direction questions, see [docs/en/roadmap.md](docs/en/roadmap.md).

## Status

The current repository state includes:

- external-process provider runtime abstraction
- first-party plugin binaries for Feishu and Notion
- plugin discovery through plugin manifests
- plugin management commands:
  - `clawrise plugin list`
  - `clawrise plugin install <source>`
  - `clawrise plugin info <name> <version>`
  - `clawrise plugin remove <name> <version>`
  - `clawrise plugin verify <name> <version>`
  - `clawrise plugin verify --all`
- minimal onboarding helpers:
  - `clawrise config init`
  - `clawrise auth list`
  - `clawrise auth methods`
  - `clawrise auth presets`
  - `clawrise account add --platform <name> --preset <id>`
  - `clawrise auth inspect <account>`
  - `clawrise auth check [account]`
  - `clawrise auth login <account>`
  - `clawrise auth complete <flow_id>`
  - `clawrise doctor`
- local searchable playbooks:
  - `docs/playbooks/index.yaml`
  - `docs/playbooks/zh/*.md`
  - `docs/playbooks/en/*.md`
- current install sources:
  - local directory
  - `file://`
  - `https://`
  - `npm://`
- current discovery support:
  - `clawrise spec list [path]`
  - `clawrise spec get <operation>`
  - `clawrise spec status`
  - `clawrise spec export [path] [--format json|markdown]`
  - `clawrise docs generate [path] [--out-dir <dir>]`
  - `clawrise completion <bash|zsh|fish>`
- current runtime governance:
  - persisted local idempotency state for write operations
  - local JSONL audit logs
  - config-driven retry policy
  - redaction for audit input and output

Still not implemented:

- remote-source trust policy hardening beyond the current verify surface
- official packaged first-party plugin release workflow

## License

This project is licensed under the [MIT License](LICENSE).
