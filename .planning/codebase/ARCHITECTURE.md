# Architecture

**Analysis Date:** 2026-04-09

## Pattern Overview

**Overall:** Plugin-driven CLI runtime with out-of-process provider execution via JSON-RPC over stdio

**Key Characteristics:**
- Core CLI dispatches management commands internally and delegates operation execution to external plugin processes
- Plugins communicate with the core via a JSON-RPC 2.0 protocol over stdin/stdout pipes
- Each provider plugin is a standalone Go binary that registers operations, handles auth, and executes API calls
- The runtime layer normalizes all execution into a uniform envelope with governance (idempotency, audit, policy, retry)
- Provider models are kept provider-native -- no shared business object schema across platforms

## Layers

**CLI Layer:**
- Purpose: Argument parsing, command dispatch, and user-facing output
- Location: `internal/cli/`
- Contains: Subcommand handlers (`root.go`, `auth.go`, `account.go`, `plugin.go`, `spec.go`, `docs.go`, `config.go`, `batch.go`, `completion.go`)
- Depends on: `internal/runtime`, `internal/plugin`, `internal/config`, `internal/metadata`, `internal/output`
- Used by: `cmd/clawrise/main.go`

**Runtime Layer:**
- Purpose: Operation parsing, input loading, account resolution, execution orchestration, governance
- Location: `internal/runtime/`
- Contains: `Executor` (main orchestrator), operation parsing, input loading, policy evaluation, idempotency, audit, retry
- Depends on: `internal/adapter`, `internal/plugin`, `internal/config`, `internal/apperr`, `internal/auth`, `internal/secretstore`
- Used by: `internal/cli/root.go` via `runOperation()` and `runBatch()`

**Adapter Layer:**
- Purpose: Operation contract definition (types, registry, handler signature) and runtime option propagation
- Location: `internal/adapter/`
- Contains: `Registry`, `Definition`, `HandlerFunc`, `Call`, `Identity`, `RuntimeOptions`, `ProviderDebugCapture`
- Depends on: `internal/apperr`, `internal/auth` (for `Session` type)
- Used by: `internal/runtime/` (for dispatch), `internal/plugin/` (for operation registration), `internal/adapter/feishu/`, `internal/adapter/notion/`

**Plugin Layer:**
- Purpose: Plugin discovery, manifest parsing, process lifecycle, capability routing, protocol types
- Location: `internal/plugin/`
- Contains: `Manager` (aggregator), `ProcessRuntime` (JSON-RPC client), `ServeRuntime` (JSON-RPC server), manifest loading, discovery, capability inspection
- Depends on: `internal/adapter` (for `Definition` type), `internal/apperr`, `internal/auth`, `internal/authflow`, `internal/spec/catalog`
- Used by: `internal/cli/` (for manager creation), `internal/runtime/` (for auth resolution and execution delegation)

**Provider Adapter Layer:**
- Purpose: Platform-specific API clients, operation registration, and auth provider implementations
- Location: `internal/adapter/feishu/`, `internal/adapter/notion/`
- Contains: API clients, `RegisterOperations()` functions, spec definitions, auth providers, session management
- Depends on: `internal/adapter`, `internal/apperr`
- Used by: Plugin binaries in `cmd/clawrise-plugin-feishu/`, `cmd/clawrise-plugin-notion/`

**Config Layer:**
- Purpose: YAML config file loading, saving, and resolution helpers
- Location: `internal/config/`
- Contains: `Config` struct, `Store`, account/auth/plugin/runtime config types, validation, secret reference resolution
- Depends on: `internal/paths`
- Used by: Nearly all layers

**Metadata Layer:**
- Purpose: Unified fact source for spec, docs, completion, and playbook validation
- Location: `internal/metadata/`, `internal/spec/`
- Contains: `Service` (aggregator), `spec.Service` (spec/export/status), `spec/catalog.Entry`, playbook validation
- Depends on: `internal/adapter` (for registry), `internal/spec/catalog`
- Used by: `internal/cli/` for `spec`, `docs`, `completion` commands

**npm Wrapper Layer:**
- Purpose: Published user-facing entrypoint; resolves platform-specific binary and injects plugin paths
- Location: `packaging/npm/root/`
- Contains: `bin/clawrise.js` (entrypoint), `lib/platform.js` (binary resolution), `lib/setup.js` (setup command)
- Depends on: Go binary at install time
- Used by: End users via `npx clawrise` or global install

## Data Flow

**Operation Execution (primary flow):**

1. User invokes `clawrise notion.page.create --json '{"title":"Hello"}'` via npm wrapper
2. `packaging/npm/root/bin/clawrise.js` spawns the Go binary with `CLAWRISE_PLUGIN_PATHS` set
3. `cmd/clawrise/main.go` calls `cli.Run(args, deps)`
4. `internal/cli/root.go` `Run()` dispatches to `runOperation()` (default case in switch)
5. `runOperation()` parses flags (--json, --account, --dry-run, etc.) and creates `runtime.Executor`
6. `runtime.Executor.Execute()` executes the full pipeline:
   - Loads config via `config.Store`
   - Parses operation string into normalized `Operation` struct (e.g., `notion.page.create`)
   - Resolves operation definition from `adapter.Registry`
   - Resolves account selection based on platform, subject, and defaults
   - Resolves execution identity via plugin auth (`manager.ResolveAuth()`)
   - Reads input from --json, --input, or stdin
   - Evaluates local policy rules then plugin policy chain
   - For mutating operations: builds idempotency key, checks/replays existing records
   - Delegates to `definition.Handler()` which routes through `plugin.Manager` to the external plugin process
   - Retries on retryable errors with exponential backoff
   - Writes audit record (file, webhook, or plugin sink)
   - Returns normalized `Envelope`

**Plugin Discovery Flow:**

1. `DefaultDiscoveryRoots()` returns roots from `CLAWRISE_PLUGIN_PATHS`, `.clawrise/plugins`, and `~/.clawrise/plugins`
2. `DiscoverManifests()` walks roots looking for `plugin.json` files
3. Each manifest declares capabilities (provider, auth_launcher, storage_backend, policy, audit_sink, workflow, registry_source)
4. `Manager` handshakes each plugin process, lists operations, builds registry, loads catalog
5. Provider bindings and enabled-plugins config filter which plugins participate

**State Management:**
- Config stored as YAML at `.clawrise/config.yaml` (path resolved via `internal/paths`)
- Secrets stored in secret store (file-based default, pluggable backend)
- Sessions stored in session store (file-based default, pluggable backend)
- Auth flow state stored in authflow store
- Idempotency records stored as JSON files in `.clawrise/runtime/idempotency/`
- Audit records stored as JSONL files in `.clawrise/runtime/audit/`

## Key Abstractions

**`adapter.Definition`:**
- Purpose: Represents one registered operation with metadata, handler, spec, and timeout
- Examples: `internal/adapter/feishu/register.go`, `internal/adapter/notion/register.go`
- Pattern: Value struct registered into `adapter.Registry` by provider packages

**`plugin.Runtime` (interface):**
- Purpose: Contract for communicating with an external plugin process
- Examples: `internal/plugin/process.go` (`ProcessRuntime` implements it)
- Pattern: JSON-RPC 2.0 over stdin/stdout with method dispatch in `internal/plugin/server.go`

**`plugin.Manager`:**
- Purpose: Aggregates multiple plugin runtimes into a unified execution and discovery view
- Examples: `internal/plugin/runtime.go`
- Pattern: Holds `adapter.Registry`, catalogs, platform-to-runtime mappings, and auth launcher registrations

**`runtime.Executor`:**
- Purpose: Orchestrates the full execution pipeline from input to output envelope
- Examples: `internal/runtime/executor.go`
- Pattern: Accepts `ExecuteOptions`, runs parse/resolve/auth/policy/execute/audit pipeline, returns `Envelope`

**`runtimeGovernance`:**
- Purpose: Manages idempotency, audit, retry, and policy evaluation for each execution
- Examples: `internal/runtime/governance.go`, `internal/runtime/policy.go`, `internal/runtime/audit_sink.go`
- Pattern: Internal struct used by `Executor`, delegates storage to pluggable backends

**`plugin.CapabilityDescriptor`:**
- Purpose: Describes one capability exposed by a plugin (provider, auth_launcher, storage_backend, policy, audit_sink, workflow, registry_source)
- Examples: `internal/plugin/capability.go`
- Pattern: V2 manifest feature; plugins declare capabilities in `plugin.json`

## Entry Points

**CLI Main:**
- Location: `cmd/clawrise/main.go`
- Triggers: User execution via npm wrapper or direct binary
- Responsibilities: Minimal bootstrap; delegates to `cli.Run()`

**Provider Plugin Binaries:**
- `cmd/clawrise-plugin-feishu/main.go`: Creates Feishu client, registers operations, serves JSON-RPC
- `cmd/clawrise-plugin-notion/main.go`: Same pattern for Notion
- `cmd/clawrise-plugin-auth-browser/main.go`: Auth launcher plugin
- `cmd/clawrise-plugin-demo/main.go`: Demo/example plugin
- `cmd/clawrise-plugin-sample-audit/main.go`: Sample audit sink plugin
- `cmd/clawrise-plugin-sample-policy/main.go`: Sample policy plugin
- Triggers: Spawned by `ProcessRuntime.ensureStarted()` when core needs to communicate with a provider
- Responsibilities: Each binary calls `pluginruntime.NewRegistryRuntimeWithOptions()` then `pluginruntime.ServeRuntime()` to serve JSON-RPC over stdio

**npm Wrapper:**
- Location: `packaging/npm/root/bin/clawrise.js`
- Triggers: User runs `clawrise` after npm install
- Responsibilities: Resolves platform-specific Go binary, sets `CLAWRISE_PLUGIN_PATHS`, spawns binary, handles `setup` command before Go binary is invoked

## Error Handling

**Strategy:** Structured error codes with normalized error envelope

**Patterns:**
- `apperr.AppError` in `internal/apperr/apperr.go` carries `Code`, `Message`, `Retryable`, `HTTPStatus`, `UpstreamCode`
- All errors in the execution pipeline are converted to `*apperr.AppError` and surfaced in the `Envelope.Error` field
- Error codes are uppercase snake_case strings (e.g., `CONFIG_LOAD_FAILED`, `OPERATION_NOT_FOUND`, `ACCOUNT_NOT_FOUND`, `AUTH_RESOLVE_FAILED`, `POLICY_DENIED`, `IDEMPOTENCY_KEY_CONFLICT`)
- Retryable errors trigger the retry loop in `Executor.Execute()` with exponential backoff
- Plugin process errors are wrapped with `PROVIDER_RUNTIME_FAILED` code
- `cli.ExitError` carries a process exit code for non-zero exits without printing additional error text

## Cross-Cutting Concerns

**Logging:** No structured logging framework. Errors propagate through return values and the normalized envelope. Audit records are the primary observability mechanism.

**Validation:** Input validation occurs at multiple points:
- Operation format validation in `runtime.ParseOperationWithPlatforms()`
- JSON input parsing in `runtime.ReadInput()`
- Account shape validation in `config.ValidateAccountShape()`
- Full account validation in `config.ValidateAccount()`
- Subject allow-list checking against `definition.AllowedSubjects`

**Authentication:** Multi-layer auth resolution:
- Account config declares auth method and public/secret fields
- Secret references resolved via `config.ResolveSecret()` through the secret store
- Session data loaded from session store
- Plugin `ResolveAuth()` called before execution to get fresh execution credentials
- Auth patches (session updates, secret updates) persisted after resolution

**Idempotency:** Automatic for mutating operations:
- Key derived from operation + input hash if not explicitly provided
- Records persisted as JSON files in `.clawrise/runtime/idempotency/`
- Duplicate requests with matching keys replay previous result
- Conflicting requests (different input, same key) rejected

**Policy Chain:** Evaluated before execution:
- Local deny/approval/annotate rules in config
- External policy plugins discovered and evaluated in order
- Decisions: allow, deny, require_approval, annotate
- Deny and require_approval block execution; annotate adds warnings

**Audit:** Fan-out to multiple sinks after execution:
- File-based JSONL (daily rotation)
- Webhook HTTP POST
- Stdout output
- Plugin audit sinks (external processes)

**Redaction:** Sensitive values in input/output summaries are redacted before audit persistence (`internal/runtime/redact.go`)

---

*Architecture analysis: 2026-04-09*
