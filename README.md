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

## Quick Start

```bash
go build ./...
go test ./...
go run ./cmd/clawrise version
go run ./cmd/clawrise doctor
```

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
