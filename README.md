# Clawrise

For the Chinese version of this README, see [README.zh.md](README.zh.md).

## Overview

Clawrise is an agent-native CLI execution layer for SaaS APIs.

It is designed to let AI agents call third-party systems through stable CLI operations instead of heavyweight MCP-style tool schemas.

The current repository is still in the design phase. The Go implementation has not started yet.

## Current MVP Scope

The current MVP platform set is:

- `feishu`
- `notion`

The next major platform planned after MVP is:

- `google`

## Documentation

English docs:

- [CLI Layer Design](docs/en/cli-layer-design.md)
- [Auth Model](docs/en/auth-model.md)
- [MVP Operation Spec](docs/en/mvp-operation-spec.md)
- [Feishu User Auth Setup](docs/en/feishu-user-auth-setup.md)

Chinese docs:

- [CLI Layer 架构设计](docs/zh/cli-layer-design.md)
- [授权模型](docs/zh/auth-model.md)
- [MVP Operation 规格](docs/zh/mvp-operation-spec.md)
- [飞书用户授权凭证获取说明](docs/zh/feishu-user-auth-setup.md)

## Current Design Areas

- CLI command model
- adapter architecture
- auth and profile model
- idempotency and audit rules
- MVP operation contracts for Feishu and Notion

## Example Config

- [examples/config.example.yaml](examples/config.example.yaml)

## Status

This repository currently contains architecture and product design documents only.

The next implementation step is to initialize the Go project skeleton and build the runtime core around the documented contracts.
