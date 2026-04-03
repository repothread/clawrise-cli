# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Common commands

- Build: `go build ./...`
- Test all Go packages: `go test ./...`
- Test npm wrapper code used by CI: `node --test packaging/npm/root/lib/*.test.js`
- Run one Go test target: `go test ./internal/runtime -run TestParseOperation`
- Run the CLI locally: `go run ./cmd/clawrise version`
- Inspect config, plugin, and runtime state: `go run ./cmd/clawrise doctor`
- List operation specs: `go run ./cmd/clawrise spec list`
- Validate an operation without calling the upstream API: `go run ./cmd/clawrise feishu.calendar.event.create --dry-run --json '{"calendar_id":"cal_demo"}'`
- Rebuild and install first-party plugins into the project-local discovery path for source development: `./scripts/dev-install-first-party-plugins.sh`

## Key operational context

- The repository is mainly Go, but the published user-facing entrypoint is the npm wrapper at `packaging/npm/root/bin/clawrise.js`.
- `clawrise setup ...` belongs to the published npm wrapper flow. Raw source execution with `go run ./cmd/clawrise ...` does not expose that setup behavior.
- Plugin discovery checks `CLAWRISE_PLUGIN_PATHS`, then `.clawrise/plugins`, then `~/.clawrise/plugins`.
- `dist/release` contains generated release artifacts. Treat `packaging/npm/root` and `scripts/release` as the implementation source of truth.

## Architecture

Clawrise is split into three layers:

1. Core CLI/runtime
2. External provider plugins
3. npm distribution wrapper

### Core CLI/runtime

- `cmd/clawrise/main.go` is the Go entrypoint.
- `internal/cli/root.go` dispatches management commands (`platform`, `account`, `plugin`, `auth`, `spec`, `docs`, `completion`, `doctor`) and sends everything else through operation execution.
- `internal/runtime` owns operation parsing, input loading, account resolution, retry/timeout/idempotency, normalized output envelopes, and audit/policy flow.

### Plugin layer

- `internal/plugin` owns manifest discovery, external plugin process startup, capability routing, and registry/catalog aggregation.
- First-party providers are shipped as standalone plugin binaries such as `cmd/clawrise-plugin-feishu` and `cmd/clawrise-plugin-notion`.
- `internal/adapter` is the contract layer for operation registration. Each operation carries metadata such as platform, mutating behavior, allowed subjects, spec, and handler.

### Metadata layer

- `internal/spec` and `internal/metadata` provide the shared fact source for `spec`, `docs`, `completion`, and playbook validation.
- When changing operation documentation or discoverability, prefer updating registry/spec metadata instead of introducing a second metadata path.

## Architectural guardrails

- Keep provider models provider-native. Do not force Notion, Feishu, or future platforms into one shared business object schema.
- Shared abstractions belong only at the runtime layer: execution envelope, auth context, retry/timeout behavior, error model, and idempotency.
- Operation names follow `<platform>.<resource-path>.<action>`.
