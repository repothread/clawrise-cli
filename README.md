# Clawrise

For the Chinese version, see [README.zh.md](README.zh.md).

## Overview

Clawrise is an agent-native CLI execution layer for SaaS APIs.

It is designed to let AI agents call third-party systems through stable CLI operations instead of heavyweight MCP-style tool schemas.

The current architecture is plugin-first:

- `clawrise` is the core runtime and CLI
- provider capabilities are exposed through external provider plugins
- first-party Feishu and Notion support are shipped as plugin binaries

The repository contains both the evolving design documents and the current Go implementation of the core runtime and first-party plugins.

## Current Scope

Current first-party plugin platforms:

- `feishu`
- `notion`

Next planned platform after MVP:

- `google`

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
- auth and profile model
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
  - `clawrise auth inspect <profile>`
  - `clawrise auth check [profile]`
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
- current runtime governance:
  - persisted local idempotency state for write operations
  - local JSONL audit logs
  - config-driven retry policy
  - redaction for audit input and output

Still not implemented:

- `clawrise spec export`
- `completion`
- plugin signature policy
- official packaged first-party plugin distribution workflow
