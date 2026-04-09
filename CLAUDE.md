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

<!-- GSD:project-start source:PROJECT.md -->
## Project

**Clawrise CLI — 发布前关键修复**

Clawrise 是一个插件驱动的 CLI 运行时，通过 JSON-RPC over stdio 与外部 provider 插件进程通信。本次工作是在 `develop` 分支合入 `main` 并发布新版本之前，修复已识别的关键 bug 和安全隐患。

**Core Value:** 修复审计 sink 和 context 传播问题后，CLI 可以安全地合并到 main 并发布新版本，不会在生产环境中出现审计丢失或无法取消操作的问题。

### Constraints

- **兼容性**: 不能破坏现有插件协议版本（`ProtocolVersion = 1`）
- **测试**: 所有现有测试必须继续通过，不能降低覆盖率
- **范围**: 仅修复发布阻塞项，不做额外重构
<!-- GSD:project-end -->

<!-- GSD:stack-start source:codebase/STACK.md -->
## Technology Stack

## Languages
- Go 1.25.9 - Core CLI runtime, all plugin binaries, adapter layer, config, auth, and execution engine
- JavaScript (Node.js) - npm distribution wrapper at `packaging/npm/root/`, skill install scripts, release tooling at `scripts/release/*.mjs`
- Shell (bash) - Build scripts, CI scripts, dev tooling at `scripts/`
- YAML - Config file format (`gopkg.in/yaml.v3`), GitHub Actions workflows at `.github/workflows/`
## Runtime
- Go 1.25.9 (pinned in `go.mod` line 3)
- Node.js 24 (CI uses `actions/setup-node@v4` with `node-version: '24'`)
- Go modules - primary dependency management (`go.mod`, `go.sum`)
- npm - distribution packaging only (no `package.json` in repo root; generated during release)
- Lockfile: `go.sum` present with 6 lines (minimal dependency footprint)
## Frameworks
- Go standard library only - No web framework; all HTTP calls use `net/http` directly
- `github.com/spf13/pflag` v1.0.5 - CLI flag parsing in `internal/cli/root.go`
- `gopkg.in/yaml.v3` v3.0.1 - Config file serialization in `internal/config/`
- Go `testing` package - All Go tests use standard `go test`
- Node.js `node --test` - npm wrapper tests at `packaging/npm/root/lib/*.test.js`
- No third-party test frameworks or assertion libraries
- Go cross-compilation via `GOOS`/`GOARCH` - 6 platform targets in `scripts/release/build-npm-bundles.sh`
- `CGO_ENABLED=0` - Static binaries for all release builds
- `-ldflags` injection for `buildinfo.Version`, `buildinfo.Commit`, `buildinfo.BuildDate`
## Key Dependencies
- `github.com/spf13/pflag` v1.0.5 - CLI argument parsing; used exclusively in `internal/cli/root.go`
- `gopkg.in/yaml.v3` v3.0.1 - Config file parse/serialize; used in `internal/config/store.go`
- Go `net/http` - All upstream API calls (Notion, Feishu) use standard library HTTP client
- Go `encoding/json` - JSON-RPC protocol between core and plugins, all API I/O
- Go `crypto/*` - SHA-256 for idempotency keys, SHA-1/SHA-256/SHA-512 for plugin checksum verification, random token generation
## Configuration
- YAML config file at `~/.clawrise/config.yaml` (resolved via `internal/locator/`)
- Runtime state directory at `~/.clawrise/runtime/` (sessions, auth flows, audit, idempotency)
- Plugin discovery paths: `CLAWRISE_PLUGIN_PATHS` env var, `.clawrise/plugins`, `~/.clawrise/plugins`
- `CLAWRISE_ROOT_PACKAGE_NAME` - Override npm package name during release
- `CLAWRISE_NPM_SCOPE`, `CLAWRISE_NPM_PACKAGE_PREFIX`, `CLAWRISE_NPM_DIST_TAG` - Release publishing config
- `NOTION_INTERNAL_TOKEN`, `FEISHU_APP_ID`, `FEISHU_APP_SECRET` - Setup-time credential env vars
- `go.mod` - Go module definition
- `scripts/release/build-npm-bundles.sh` - Cross-platform binary builds
- `scripts/release/prepare-npm-packages.mjs` - npm package assembly
- `scripts/release/prepare-skill-packages.mjs` - Skill package assembly
- `.github/workflows/ci.yml` - CI pipeline (test, hardening, build, smoke, release-smoke)
- `.github/workflows/release-npm.yml` - Release and npm publish pipeline
## Platform Requirements
- Go 1.25.9+
- Node.js 24+ (for npm wrapper tests and release tooling)
- macOS Keychain or Linux Secret Service (for default secret storage backend)
- Published as npm packages under `@clawrise` scope to npm registry
- Platform-specific binary packages: `darwin-arm64`, `darwin-x64`, `linux-arm64`, `linux-x64`, `win32-arm64`, `win32-x64`
- Supports agent clients: Codex, Claude Code, OpenClaw, OpenCode
- No server-side deployment; CLI is purely client-side
<!-- GSD:stack-end -->

<!-- GSD:conventions-start source:CONVENTIONS.md -->
## Conventions

## Naming Patterns
- All lowercase, single word: `runtime`, `config`, `adapter`, `plugin`, `apperr`
- Import aliases resolve collisions: `authcache "github.com/clawrise/clawrise-cli/internal/auth"`, `pluginruntime "github.com/clawrise/clawrise-cli/internal/plugin"`
- Provider adapter packages use the provider name: `internal/adapter/feishu`, `internal/adapter/notion`
- `snake_case.go`: One file per domain concept: `operation.go`, `executor.go`, `governance.go`, `redact.go`
- Test files: Same name with `_test.go` suffix co-located in the same package
- Spec files: `<domain>_spec.go` (e.g., `calendar_spec.go`, `page_spec.go`) define operation metadata
- Registration: `register.go` in each adapter package wires operations into the registry
- Exported functions use PascalCase: `ParseOperation`, `NewExecutor`, `ResolveStore`
- Unexported helpers use camelCase: `buildRequestID`, `isSensitiveKey`, `cloneAnyMap`
- Constructor pattern: `New<Type>(...)` or `New<Type>WithOptions(...)`
- Factory helpers in tests: `newTestStore`, `newTestRegistry`, `newTestPluginManager`
- Exported structs use PascalCase: `Envelope`, `ExecuteOptions`, `Definition`
- Unexported structs for internal state: `runtimeGovernance`, `persistedIdempotencyRecord`
- Type aliases for function signatures: `type roundTripFunc func(*http.Request) (*http.Response, error)`
- Context key types use dedicated unexported types: `type callContextKey string`
- Constants use PascalCase (exported) or camelCase (unexported): `ProtocolVersion`, `policyDecisionAllow`
- Error codes use UPPER_SNAKE_CASE strings: `"CONFIG_LOAD_FAILED"`, `"OPERATION_NOT_FOUND"`
- Request prefixes: `"req_"`, `"idem_"`
## Code Style
- Standard `gofmt` / `goimports` formatting
- Indentation: tabs
- No explicit linter config files detected (relies on standard Go tooling)
## Error Handling
- UPPER_SNAKE_CASE: `CONFIG_LOAD_FAILED`, `ACCOUNT_NOT_FOUND`, `IDEMPOTENCY_KEY_CONFLICT`, `POLICY_DENIED`
- Consistent error codes across the runtime layer, all mapped into `Envelope.Error.Code`
- Runtime executor returns `(Envelope, error)` from `Execute` -- errors only for truly unexpected failures
- Business logic errors are wrapped in `apperr.AppError` and placed inside the `Envelope` with `OK: false`
- CLI layer returns `error` from `Run()`, with `ExitError{Code: 1}` for expected failures
## Logging
- All CLI output is JSON via `internal/output/json.go` `WriteJSON` function
- Human-readable help text uses `fmt.Fprintln`
- No log levels or log files for runtime operations
- Audit records written as JSONL files under `runtime/audit/`
## Comments
## Function Design
- Options structs for complex signatures: `ExecuteOptions`, `DiscoveryOptions`, `ManagerOptions`
- Context as first parameter: `func (e *Executor) Execute(ctx context.Context, opts ExecuteOptions)`
- Interface parameters: `io.Writer`, `io.Reader` for I/O
- Functions that can fail return `(result, error)` or `(result, *apperr.AppError)`
- Envelope-based return for execution layer: `(Envelope, error)` where Envelope captures both success and business-error state
- Pointer returns for optional values: `*IdempotencyState`, `*ErrorBody`, `*PolicyResult`
## Module Design
- Each package exports a clear public API surface
- Internal state is unexported: `runtimeGovernance`, `governanceStore` interfaces
- Provider packages export: `NewClient`, `RegisterOperations`, `NewAuthProvider`
## Context Usage Pattern
## JSON and YAML Conventions
## Security Conventions
## Operation Naming Convention
- `feishu.calendar.event.create`
- `notion.page.get`
- `feishu.docs.block.update`
- `notion.block.append`
## CLI Output Convention
<!-- GSD:conventions-end -->

<!-- GSD:architecture-start source:ARCHITECTURE.md -->
## Architecture

## Pattern Overview
- Core CLI dispatches management commands internally and delegates operation execution to external plugin processes
- Plugins communicate with the core via a JSON-RPC 2.0 protocol over stdin/stdout pipes
- Each provider plugin is a standalone Go binary that registers operations, handles auth, and executes API calls
- The runtime layer normalizes all execution into a uniform envelope with governance (idempotency, audit, policy, retry)
- Provider models are kept provider-native -- no shared business object schema across platforms
## Layers
- Purpose: Argument parsing, command dispatch, and user-facing output
- Location: `internal/cli/`
- Contains: Subcommand handlers (`root.go`, `auth.go`, `account.go`, `plugin.go`, `spec.go`, `docs.go`, `config.go`, `batch.go`, `completion.go`)
- Depends on: `internal/runtime`, `internal/plugin`, `internal/config`, `internal/metadata`, `internal/output`
- Used by: `cmd/clawrise/main.go`
- Purpose: Operation parsing, input loading, account resolution, execution orchestration, governance
- Location: `internal/runtime/`
- Contains: `Executor` (main orchestrator), operation parsing, input loading, policy evaluation, idempotency, audit, retry
- Depends on: `internal/adapter`, `internal/plugin`, `internal/config`, `internal/apperr`, `internal/auth`, `internal/secretstore`
- Used by: `internal/cli/root.go` via `runOperation()` and `runBatch()`
- Purpose: Operation contract definition (types, registry, handler signature) and runtime option propagation
- Location: `internal/adapter/`
- Contains: `Registry`, `Definition`, `HandlerFunc`, `Call`, `Identity`, `RuntimeOptions`, `ProviderDebugCapture`
- Depends on: `internal/apperr`, `internal/auth` (for `Session` type)
- Used by: `internal/runtime/` (for dispatch), `internal/plugin/` (for operation registration), `internal/adapter/feishu/`, `internal/adapter/notion/`
- Purpose: Plugin discovery, manifest parsing, process lifecycle, capability routing, protocol types
- Location: `internal/plugin/`
- Contains: `Manager` (aggregator), `ProcessRuntime` (JSON-RPC client), `ServeRuntime` (JSON-RPC server), manifest loading, discovery, capability inspection
- Depends on: `internal/adapter` (for `Definition` type), `internal/apperr`, `internal/auth`, `internal/authflow`, `internal/spec/catalog`
- Used by: `internal/cli/` (for manager creation), `internal/runtime/` (for auth resolution and execution delegation)
- Purpose: Platform-specific API clients, operation registration, and auth provider implementations
- Location: `internal/adapter/feishu/`, `internal/adapter/notion/`
- Contains: API clients, `RegisterOperations()` functions, spec definitions, auth providers, session management
- Depends on: `internal/adapter`, `internal/apperr`
- Used by: Plugin binaries in `cmd/clawrise-plugin-feishu/`, `cmd/clawrise-plugin-notion/`
- Purpose: YAML config file loading, saving, and resolution helpers
- Location: `internal/config/`
- Contains: `Config` struct, `Store`, account/auth/plugin/runtime config types, validation, secret reference resolution
- Depends on: `internal/paths`
- Used by: Nearly all layers
- Purpose: Unified fact source for spec, docs, completion, and playbook validation
- Location: `internal/metadata/`, `internal/spec/`
- Contains: `Service` (aggregator), `spec.Service` (spec/export/status), `spec/catalog.Entry`, playbook validation
- Depends on: `internal/adapter` (for registry), `internal/spec/catalog`
- Used by: `internal/cli/` for `spec`, `docs`, `completion` commands
- Purpose: Published user-facing entrypoint; resolves platform-specific binary and injects plugin paths
- Location: `packaging/npm/root/`
- Contains: `bin/clawrise.js` (entrypoint), `lib/platform.js` (binary resolution), `lib/setup.js` (setup command)
- Depends on: Go binary at install time
- Used by: End users via `npx clawrise` or global install
## Data Flow
- Config stored as YAML at `.clawrise/config.yaml` (path resolved via `internal/paths`)
- Secrets stored in secret store (file-based default, pluggable backend)
- Sessions stored in session store (file-based default, pluggable backend)
- Auth flow state stored in authflow store
- Idempotency records stored as JSON files in `.clawrise/runtime/idempotency/`
- Audit records stored as JSONL files in `.clawrise/runtime/audit/`
## Key Abstractions
- Purpose: Represents one registered operation with metadata, handler, spec, and timeout
- Examples: `internal/adapter/feishu/register.go`, `internal/adapter/notion/register.go`
- Pattern: Value struct registered into `adapter.Registry` by provider packages
- Purpose: Contract for communicating with an external plugin process
- Examples: `internal/plugin/process.go` (`ProcessRuntime` implements it)
- Pattern: JSON-RPC 2.0 over stdin/stdout with method dispatch in `internal/plugin/server.go`
- Purpose: Aggregates multiple plugin runtimes into a unified execution and discovery view
- Examples: `internal/plugin/runtime.go`
- Pattern: Holds `adapter.Registry`, catalogs, platform-to-runtime mappings, and auth launcher registrations
- Purpose: Orchestrates the full execution pipeline from input to output envelope
- Examples: `internal/runtime/executor.go`
- Pattern: Accepts `ExecuteOptions`, runs parse/resolve/auth/policy/execute/audit pipeline, returns `Envelope`
- Purpose: Manages idempotency, audit, retry, and policy evaluation for each execution
- Examples: `internal/runtime/governance.go`, `internal/runtime/policy.go`, `internal/runtime/audit_sink.go`
- Pattern: Internal struct used by `Executor`, delegates storage to pluggable backends
- Purpose: Describes one capability exposed by a plugin (provider, auth_launcher, storage_backend, policy, audit_sink, workflow, registry_source)
- Examples: `internal/plugin/capability.go`
- Pattern: V2 manifest feature; plugins declare capabilities in `plugin.json`
## Entry Points
- Location: `cmd/clawrise/main.go`
- Triggers: User execution via npm wrapper or direct binary
- Responsibilities: Minimal bootstrap; delegates to `cli.Run()`
- `cmd/clawrise-plugin-feishu/main.go`: Creates Feishu client, registers operations, serves JSON-RPC
- `cmd/clawrise-plugin-notion/main.go`: Same pattern for Notion
- `cmd/clawrise-plugin-auth-browser/main.go`: Auth launcher plugin
- `cmd/clawrise-plugin-demo/main.go`: Demo/example plugin
- `cmd/clawrise-plugin-sample-audit/main.go`: Sample audit sink plugin
- `cmd/clawrise-plugin-sample-policy/main.go`: Sample policy plugin
- Triggers: Spawned by `ProcessRuntime.ensureStarted()` when core needs to communicate with a provider
- Responsibilities: Each binary calls `pluginruntime.NewRegistryRuntimeWithOptions()` then `pluginruntime.ServeRuntime()` to serve JSON-RPC over stdio
- Location: `packaging/npm/root/bin/clawrise.js`
- Triggers: User runs `clawrise` after npm install
- Responsibilities: Resolves platform-specific Go binary, sets `CLAWRISE_PLUGIN_PATHS`, spawns binary, handles `setup` command before Go binary is invoked
## Error Handling
- `apperr.AppError` in `internal/apperr/apperr.go` carries `Code`, `Message`, `Retryable`, `HTTPStatus`, `UpstreamCode`
- All errors in the execution pipeline are converted to `*apperr.AppError` and surfaced in the `Envelope.Error` field
- Error codes are uppercase snake_case strings (e.g., `CONFIG_LOAD_FAILED`, `OPERATION_NOT_FOUND`, `ACCOUNT_NOT_FOUND`, `AUTH_RESOLVE_FAILED`, `POLICY_DENIED`, `IDEMPOTENCY_KEY_CONFLICT`)
- Retryable errors trigger the retry loop in `Executor.Execute()` with exponential backoff
- Plugin process errors are wrapped with `PROVIDER_RUNTIME_FAILED` code
- `cli.ExitError` carries a process exit code for non-zero exits without printing additional error text
## Cross-Cutting Concerns
- Operation format validation in `runtime.ParseOperationWithPlatforms()`
- JSON input parsing in `runtime.ReadInput()`
- Account shape validation in `config.ValidateAccountShape()`
- Full account validation in `config.ValidateAccount()`
- Subject allow-list checking against `definition.AllowedSubjects`
- Account config declares auth method and public/secret fields
- Secret references resolved via `config.ResolveSecret()` through the secret store
- Session data loaded from session store
- Plugin `ResolveAuth()` called before execution to get fresh execution credentials
- Auth patches (session updates, secret updates) persisted after resolution
- Key derived from operation + input hash if not explicitly provided
- Records persisted as JSON files in `.clawrise/runtime/idempotency/`
- Duplicate requests with matching keys replay previous result
- Conflicting requests (different input, same key) rejected
- Local deny/approval/annotate rules in config
- External policy plugins discovered and evaluated in order
- Decisions: allow, deny, require_approval, annotate
- Deny and require_approval block execution; annotate adds warnings
- File-based JSONL (daily rotation)
- Webhook HTTP POST
- Stdout output
- Plugin audit sinks (external processes)
<!-- GSD:architecture-end -->

<!-- GSD:skills-start source:skills/ -->
## Project Skills

No project skills found. Add skills to any of: `.claude/skills/`, `.agents/skills/`, `.cursor/skills/`, or `.github/skills/` with a `SKILL.md` index file.
<!-- GSD:skills-end -->

<!-- GSD:workflow-start source:GSD defaults -->
## GSD Workflow Enforcement

Before using Edit, Write, or other file-changing tools, start work through a GSD command so planning artifacts and execution context stay in sync.

Use these entry points:
- `/gsd-quick` for small fixes, doc updates, and ad-hoc tasks
- `/gsd-debug` for investigation and bug fixing
- `/gsd-execute-phase` for planned phase work

Do not make direct repo edits outside a GSD workflow unless the user explicitly asks to bypass it.
<!-- GSD:workflow-end -->

<!-- GSD:profile-start -->
## Developer Profile

> Profile not yet configured. Run `/gsd-profile-user` to generate your developer profile.
> This section is managed by `generate-claude-profile` -- do not edit manually.
<!-- GSD:profile-end -->
