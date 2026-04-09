# Codebase Structure

**Analysis Date:** 2026-04-09

## Directory Layout

```
clawrise-cli/
├── cmd/                            # Go binary entrypoints
│   ├── clawrise/                   # Main CLI binary
│   ├── clawrise-plugin-auth-browser/ # Auth launcher plugin binary
│   ├── clawrise-plugin-demo/       # Demo plugin binary
│   ├── clawrise-plugin-feishu/     # Feishu provider plugin binary
│   ├── clawrise-plugin-notion/     # Notion provider plugin binary
│   ├── clawrise-plugin-sample-audit/  # Sample audit sink plugin binary
│   └── clawrise-plugin-sample-policy/ # Sample policy plugin binary
├── internal/                       # Private Go packages (core implementation)
│   ├── account/                    # Account selection logic
│   ├── adapter/                    # Operation contract layer (registry, types, handler signature)
│   │   ├── feishu/                 # Feishu provider adapter (client, operations, auth)
│   │   └── notion/                 # Notion provider adapter (client, operations, auth)
│   ├── apperr/                     # Structured application error type
│   ├── auth/                       # Session store (file-based, pluggable)
│   ├── authflow/                   # Auth flow state management
│   ├── buildinfo/                  # Build version injection
│   ├── cli/                        # CLI subcommand handlers
│   ├── config/                     # Config loading, saving, types, resolution helpers
│   ├── locator/                    # Path resolution for config, state, runtime dirs
│   ├── metadata/                   # Unified metadata service (spec + playbook aggregation)
│   ├── output/                     # JSON output helpers
│   ├── paths/                      # Config path resolution
│   ├── plugin/                     # Plugin discovery, manifest, process management, protocol
│   ├── runtime/                    # Execution orchestrator, governance, policy, audit
│   ├── secretstore/                # Secret storage (file-based with encryption, pluggable)
│   └── spec/                       # Spec service, export, status, catalog
│       └── catalog/                # Structured operation catalog type
├── packaging/                      # Distribution packaging
│   ├── npm/root/                   # npm wrapper package (published entrypoint)
│   │   ├── bin/clawrise.js         # Node.js entrypoint
│   │   └── lib/                    # Platform resolution, setup, tests
│   ├── release/                    # Release artifact scripts
│   └── skills/                     # Skill package definitions
├── scripts/                        # Build, CI, and development scripts
│   ├── ci/                         # CI verification scripts
│   ├── plugin/                     # Plugin verification scripts
│   ├── release/                    # Release build, publish, verify scripts
│   └── skills/                     # Skill packaging scripts
├── skills/                         # Published skill packages
│   ├── clawrise-core/              # Core Clawrise skill
│   ├── clawrise-feishu/            # Feishu-specific skill
│   └── clawrise-notion/            # Notion-specific skill
├── docs/                           # Documentation
│   ├── en/                         # English docs
│   ├── zh/                         # Chinese docs
│   └── playbooks/                  # Playbook definitions
│       ├── en/                     # English playbooks
│       └── zh/                     # Chinese playbooks
├── examples/                       # Example plugins and config
│   └── plugins/                    # Example plugin packages with manifests
├── .clawrise/                      # Runtime state (not committed)
│   ├── runtime/                    # Idempotency and audit data
│   ├── scripts/                    # Runtime scripts
│   └── templates/                  # Runtime templates
├── .github/                        # GitHub CI/CD workflows and templates
├── .planning/                      # GSD planning documents
├── dist/                           # Generated release artifacts (not committed)
├── go.mod                          # Go module definition
├── go.sum                          # Go dependency checksums
└── CLAUDE.md                       # Claude Code instructions
```

## Directory Purposes

**`cmd/`:**
- Purpose: One subdirectory per Go binary entrypoint
- Contains: `main.go` files that bootstrap and serve plugin processes
- Key files: `cmd/clawrise/main.go` (CLI), `cmd/clawrise-plugin-feishu/main.go`, `cmd/clawrise-plugin-notion/main.go`

**`internal/cli/`:**
- Purpose: CLI subcommand handlers dispatched by `root.go`
- Contains: One file per command group: `root.go`, `auth.go`, `account.go`, `plugin.go`, `spec.go`, `docs.go`, `config.go`, `batch.go`, `completion.go`, plus helpers
- Key files: `internal/cli/root.go` (main dispatch), `internal/cli/auth.go` (auth flow commands), `internal/cli/plugin.go` (plugin install/remove)

**`internal/runtime/`:**
- Purpose: Operation execution orchestration and governance
- Contains: `executor.go` (main orchestrator), `operation.go` (parsing), `input.go` (input loading), `governance.go` (idempotency/audit/retry), `policy.go` (policy chain), `audit_sink.go` (audit fan-out), `types.go` (envelope types), `redact.go` (value redaction)
- Key files: `internal/runtime/executor.go`, `internal/runtime/types.go`, `internal/runtime/governance.go`

**`internal/plugin/`:**
- Purpose: Plugin lifecycle management, protocol, and capability routing
- Contains: `runtime.go` (Manager), `process.go` (ProcessRuntime / JSON-RPC client), `server.go` (JSON-RPC server), `protocol.go` (protocol types), `manifest.go` (manifest parsing), `discovery.go` (file system discovery), `capability.go` (capability types), `install.go` (plugin install logic)
- Key files: `internal/plugin/runtime.go`, `internal/plugin/process.go`, `internal/plugin/server.go`, `internal/plugin/protocol.go`

**`internal/adapter/`:**
- Purpose: Operation contract definition and runtime option propagation
- Contains: `registry.go` (Definition, Registry, HandlerFunc, Call, Identity), `runtime_options.go` (context-based option passing, debug capture), `debug_redaction.go`
- Key files: `internal/adapter/registry.go`, `internal/adapter/runtime_options.go`

**`internal/adapter/feishu/`:**
- Purpose: Feishu/Lark platform API client and operation implementations
- Contains: `client.go` (HTTP client), `register.go` (operation registration), `auth.go`/`auth_provider.go` (auth flows), operation files (`calendar_ops.go`, `docx.go`, etc.), spec files (`calendar_spec.go`, `docx_spec.go`, etc.)
- Key files: `internal/adapter/feishu/register.go`, `internal/adapter/feishu/client.go`, `internal/adapter/feishu/auth_provider.go`

**`internal/adapter/notion/`:**
- Purpose: Notion platform API client and operation implementations
- Contains: `client.go` (HTTP client), `register.go` (operation registration), `auth_provider.go` (auth flows), operation files (`page.go`, `block.go`, `database.go`, etc.), spec files, task workflow files
- Key files: `internal/adapter/notion/register.go`, `internal/adapter/notion/client.go`, `internal/adapter/notion/auth_provider.go`

**`internal/config/`:**
- Purpose: YAML config file management and resolution helpers
- Contains: Config types, Store (load/save), validation, secret reference resolution, policy/audit mode resolution, storage binding resolution, legacy auth bridge logic
- Key files: `internal/config/config.go`

**`internal/spec/` and `internal/metadata/`:**
- Purpose: Unified fact source for spec, docs, completion, and playbook validation
- Contains: `spec/service.go` (spec aggregation), `spec/export.go` (Markdown export), `spec/status.go` (spec status), `metadata/service.go` (top-level aggregator), `metadata/playbooks.go` (playbook validation)
- Key files: `internal/spec/service.go`, `internal/metadata/service.go`

**`packaging/npm/root/`:**
- Purpose: Published npm package that wraps the Go binary
- Contains: `bin/clawrise.js` (entrypoint), `lib/` (platform resolution, setup, tests)
- Key files: `packaging/npm/root/bin/clawrise.js`

**`scripts/`:**
- Purpose: Build automation, CI verification, release management
- Contains: `ci/` (smoke tests), `plugin/` (plugin verification), `release/` (build/publish/verify), `dev-install-first-party-plugins.sh`
- Key files: `scripts/dev-install-first-party-plugins.sh`, `scripts/ci/verify-first-party-provider-smoke.sh`

## Key File Locations

**Entry Points:**
- `cmd/clawrise/main.go`: Go CLI entrypoint
- `packaging/npm/root/bin/clawrise.js`: Published npm wrapper entrypoint
- `cmd/clawrise-plugin-feishu/main.go`: Feishu plugin process entrypoint
- `cmd/clawrise-plugin-notion/main.go`: Notion plugin process entrypoint

**Configuration:**
- `internal/config/config.go`: Config types, loading, saving, resolution
- `internal/paths/paths.go`: Config path resolution
- `internal/locator/locator.go`: Config/state/runtime directory resolution
- `examples/config.example.yaml`: Example config file

**Core Logic:**
- `internal/cli/root.go`: Command dispatch (all CLI behavior)
- `internal/runtime/executor.go`: Execution orchestrator
- `internal/runtime/types.go`: Envelope and execution types
- `internal/runtime/operation.go`: Operation name parsing
- `internal/runtime/governance.go`: Idempotency, audit, retry
- `internal/runtime/policy.go`: Policy chain evaluation
- `internal/runtime/audit_sink.go`: Audit fan-out
- `internal/plugin/runtime.go`: Plugin Manager (aggregator)
- `internal/plugin/process.go`: JSON-RPC client for plugin processes
- `internal/plugin/server.go`: JSON-RPC server for plugins
- `internal/plugin/protocol.go`: Protocol type definitions
- `internal/plugin/manifest.go`: Plugin manifest parsing
- `internal/plugin/discovery.go`: Plugin file system discovery
- `internal/adapter/registry.go`: Operation registry and handler contract

**Provider Implementations:**
- `internal/adapter/feishu/register.go`: Feishu operation registration (~30 operations)
- `internal/adapter/feishu/client.go`: Feishu HTTP client
- `internal/adapter/notion/register.go`: Notion operation registration
- `internal/adapter/notion/client.go`: Notion HTTP client

**Testing:**
- `internal/runtime/executor_test.go`: Executor tests
- `internal/runtime/operation_test.go`: Operation parsing tests
- `internal/plugin/runtime_test.go`: Plugin manager tests
- `internal/adapter/feishu/client_test.go`: Feishu client tests
- `internal/adapter/notion/client_test.go`: Notion client tests
- `internal/cli/root_test.go`: CLI dispatch tests

## Naming Conventions

**Files:**
- Go packages: `lowercase_snake_case.go` (e.g., `audit_sink.go`, `auth_launcher_runtime.go`)
- Test files: `*_test.go` (e.g., `executor_test.go`, `runtime_test.go`)
- Spec definitions: `*_spec.go` (e.g., `calendar_spec.go`, `page_spec.go`) in provider adapter dirs
- Plugin manifests: Always `plugin.json` in plugin package root

**Directories:**
- Go packages: `lowercase` (e.g., `authflow`, `secretstore`)
- Plugin binaries: `cmd/clawrise-plugin-{name}/`
- Provider adapters: `internal/adapter/{platform}/`
- Example plugins: `examples/plugins/{name}/{version}/`

**Types and Functions:**
- Go exported types: `PascalCase` (e.g., `Executor`, `Envelope`, `Manager`)
- Go exported functions: `PascalCase` (e.g., `NewExecutor`, `ParseOperation`, `ServeRuntime`)
- Go unexported: `camelCase` (e.g., `buildRequestID`, `resolveAccountSelection`)
- Error codes: `UPPER_SNAKE_CASE` (e.g., `OPERATION_NOT_FOUND`, `POLICY_DENIED`)
- Config YAML keys: `lowercase_snake_case` (e.g., `max_attempts`, `base_delay_ms`)
- JSON-RPC methods: `lowercase.dot.separated` (e.g., `clawrise.handshake`, `clawrise.execute`)

## Where to Add New Code

**New Provider Platform:**
1. Create `internal/adapter/{platform}/` directory with `client.go`, `register.go`, `auth_provider.go`, operation files, spec files
2. Create `cmd/clawrise-plugin-{platform}/main.go` binary
3. Follow pattern from `cmd/clawrise-plugin-feishu/main.go`: create adapter client, register operations, create `RegistryRuntime`, serve
4. Add `plugin.json` manifest for distribution
5. Add `internal/adapter/{platform}/*_spec.go` files for operation specs

**New Operation (existing provider):**
1. Add operation implementation method to `internal/adapter/{platform}/client.go` (or dedicated file)
2. Add spec definition in `internal/adapter/{platform}/{resource}_spec.go`
3. Add `registry.Register()` call in `internal/adapter/{platform}/register.go`
4. Operation name follows `<platform>.<resource-path>.<action>` pattern

**New Plugin Capability Type:**
1. Add capability type constant in `internal/plugin/capability.go`
2. Add validation case in `CapabilityDescriptor.Validate()`
3. Add runtime interface and process-backed implementation in `internal/plugin/`
4. Add server handler in `internal/plugin/server.go` or dedicated `*_server.go`
5. Add discovery/resolution logic as needed

**New CLI Subcommand:**
1. Add handler file in `internal/cli/` (e.g., `internal/cli/newcommand.go`)
2. Add `case "newcommand":` in `internal/cli/root.go` `Run()` switch
3. Add help line in `printRootHelp()`
4. Add tests in `internal/cli/root_test.go` or dedicated test file

**New Governance Backend:**
1. Implement `governanceStore` interface from `internal/runtime/governance.go`
2. Register via `RegisterGovernanceStoreBackend()` or create plugin with `storage_backend` capability targeting `governance`

**New Audit Sink:**
1. Implement `auditSink` interface from `internal/runtime/audit_sink.go`
2. Add resolution logic in `resolveSelectedAuditSinks()` or create plugin with `audit_sink` capability

**Utilities:**
- Shared helpers: Add to appropriate `internal/` package
- Cross-cutting types: Consider `internal/apperr/` for errors, `internal/output/` for output formatting

## Special Directories

**`.clawrise/`:**
- Purpose: Runtime state directory (config, plugins, runtime data)
- Generated: Yes (created by CLI at runtime)
- Committed: No (in `.gitignore`)
- Subdirs: `plugins/` (installed plugins), `runtime/audit/` (audit logs), `runtime/idempotency/` (idempotency records)

**`dist/`:**
- Purpose: Generated release artifacts
- Generated: Yes (by release scripts)
- Committed: No

**`.cache/`:**
- Purpose: npm cache for packaging operations
- Generated: Yes
- Committed: No

**`docs/playbooks/`:**
- Purpose: YAML playbook definitions with `index.yaml` as the entry point
- Generated: No
- Committed: Yes
- Validated by `internal/metadata/playbooks.go` via `clawrise doctor`

**`skills/`:**
- Purpose: Published skill packages for agent integration
- Each skill has a `references/` subdirectory with reference docs
- Generated: Partially (assembled by `scripts/skills/`)
- Committed: Yes

**`examples/plugins/`:**
- Purpose: Example plugin packages showing manifest structure and capability types
- Structure: `examples/plugins/{name}/{version}/plugin.json`
- Committed: Yes

---

*Structure analysis: 2026-04-09*
