# Clawrise

For the Chinese version, see [README.zh.md](README.zh.md).

## Overview

Clawrise is an agent-native CLI execution layer for SaaS APIs.

It is designed to let AI agents call third-party systems through stable CLI operations instead of heavyweight MCP-style tool schemas.

The repository contains design documents and an in-progress Go implementation of the runtime core.

## MVP Scope

Current MVP platforms:

- `feishu`
- `notion`

Next planned platform after MVP:

- `google`

## Documentation

- [CLI Layer Design](docs/en/cli-layer-design.md)
- [Plugin System Design](docs/en/plugin-system-design.md)
- [Roadmap](docs/en/roadmap.md)
- [`spec` Subsystem Design](docs/en/spec-design.md)
- [Auth Model](docs/en/auth-model.md)
- [MVP Operation Spec](docs/en/mvp-operation-spec.md)
- [Feishu User Auth Setup](docs/en/feishu-user-auth-setup.md)

## Design Focus

- CLI command model
- adapter architecture
- auth and profile model
- idempotency and audit rules
- MVP operation contracts for Feishu and Notion

## Modeling Boundary

Clawrise standardizes how operations are executed, not how every SaaS resource is modeled.

- The runtime contract should stay unified.
- Resource fields should remain provider-native.
- Feishu docs, Notion pages, calendars, sheets, databases, and future APIs must not be forced into one shared global schema.
- If a cross-platform workflow is useful later, it should be added as an optional higher-level layer rather than replacing provider-specific operation contracts.

## Example Config

- [examples/config.example.yaml](examples/config.example.yaml)

## Status

This repository contains both design documents and an in-progress Go implementation of the runtime core.

Current discovery support:

- `clawrise spec list [path]`
- `clawrise spec get <operation>`
