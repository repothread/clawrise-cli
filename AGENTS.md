# Repository Guidelines

## Project Structure & Module Organization

- `cmd/clawrise/main.go`: CLI entrypoint.
- `internal/cli`: command parsing and local command handlers such as `platform`, `subject`, and `profile`.
- `internal/runtime`: operation parsing, execution flow, normalized output, and idempotency handling.
- `internal/adapter`: platform adapters. Feishu-specific code lives in `internal/adapter/feishu`.
- `internal/config`: config loading, validation, and default path resolution.
- `docs/en` and `docs/zh`: English and Chinese design docs.
- `examples/config.example.yaml`: reference config template.

## Build, Test, and Development Commands

- `go build ./...`: compile all packages.
- `go test ./...`: run all unit tests.
- `go run ./cmd/clawrise version`: run the CLI locally.
- `go run ./cmd/clawrise doctor`: inspect config path and runtime context.
- `go run ./cmd/clawrise feishu.calendar.event.create --dry-run --json '...'`: validate an operation without calling upstream APIs.

If local Go cache or module cache is restricted, use:

```bash
GOCACHE=/tmp/clawrise-go-build GOMODCACHE=/tmp/clawrise-gomodcache go test ./...
```

## Coding Style & Naming Conventions

- Use `gofmt` on every Go change.
- Keep code, comments, logs, and CLI-facing natural language in English.
- Prefer small packages with clear responsibilities; keep platform-specific logic inside adapter packages.
- Use `snake_case` for JSON/YAML fields and `camelCase`/`PascalCase` for Go identifiers.
- Operation names follow `<platform>.<resource-path>.<action>`, for example `feishu.wiki.node.create`.

## Architecture Guardrails

- Clawrise unifies the runtime contract, not the provider resource schema.
- Keep operation inputs and outputs provider-native when they describe business resources such as documents, calendars, tables, contacts, or records.
- Do not force Feishu, Notion, Google, or future platforms into one shared cross-platform resource field model.
- Shared structure should stay limited to runtime-level concerns such as the execution envelope, auth context, error model, timeout, retry, and idempotency behavior.
- If a future workflow needs a cross-platform abstraction, build it as an optional higher-level helper or workflow layer instead of changing the core operation contracts.

## Testing Guidelines

- Place tests next to implementation as `*_test.go`.
- Prefer table-free focused tests for core execution paths unless a table improves clarity.
- Cover operation parsing, config resolution, auth validation, and adapter request mapping.
- For HTTP integrations, prefer mocked `http.RoundTripper` tests over live network tests.

## Security & Configuration Tips

- Do not commit real credentials.
- Store secrets in environment variables referenced from `~/.clawrise/config.yaml`.
- Keep bot/user/integration identities separate through `subject` and `profile`.

## Commit & Pull Request Guidelines

- No Git history is available in this checkout, so there is no repository-specific convention to inherit.
- Use short, imperative commit messages going forward, preferably Conventional Commits, for example `feat: add feishu wiki node create`.
- PRs should include a concise summary, affected commands or operations, config changes, and test evidence (`go test ./...`, sample CLI output when relevant).
