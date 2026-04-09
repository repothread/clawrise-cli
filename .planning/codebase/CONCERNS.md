# Codebase Concerns

**Analysis Date:** 2026-04-09

## Tech Debt

**Monolithic `install.go` (1663 lines):**
- Issue: `internal/plugin/install.go` handles plugin source parsing, download, extraction, trust validation, NPM resolution, registry resolution, verification, and cleanup all in a single file. Functions exceed 50 lines regularly. The file mixes HTTP transport concerns, filesystem operations, and security validation without clear boundaries.
- Files: `internal/plugin/install.go`
- Impact: Hard to reason about plugin installation safety. Bug fixes in NPM resolution could silently break registry source handling. No clear test seam between download and verification.
- Fix approach: Split into focused files: `install_source.go` (source parsing), `install_download.go` (HTTP/materialization), `install_verify.go` (checksum/trust), `install_npm.go` (NPM-specific), `install_registry.go` (registry-specific). Keep `install.go` as the orchestrator.

**Monolithic `config.go` (1315 lines):**
- Issue: `internal/cli/config.go` contains all config subcommand handlers (init, secret-store, provider, auth-launcher, policy, audit) plus validation helpers and help text. The run function dispatches to ten-plus sub-handlers in one switch.
- Files: `internal/cli/config.go`
- Impact: Any config subcommand change requires reading the entire file. Help text is interleaved with logic.
- Fix approach: Extract each subcommand area (`config_audit.go`, `config_policy.go`, etc.) following the pattern already used by `cli/auth.go` and `cli/account.go`.

**Monolithic `root.go` (926 lines):**
- Issue: `internal/cli/root.go` mixes the main dispatch switch, subject/platform/version/doctor commands, and the doctor implementation. The `resolvePluginManager` helper is called with `_, _ =` return in multiple places, silently discarding errors.
- Files: `internal/cli/root.go` (lines 63, 75, 99, 113)
- Impact: Discarded plugin manager errors mean `doctor`, `account`, and `auth` commands may silently operate without plugin support.
- Fix approach: Split doctor into `cli/doctor.go`. Always surface the plugin manager error instead of discarding it, or use a clear fallback logging pattern.

**Duplicate `roundTripFunc` test helpers:**
- Issue: `roundTripFunc` is redefined in four separate test files with two incompatible signatures (function type vs struct with method).
- Files: `internal/runtime/executor_test.go`, `internal/plugin/install_test.go`, `internal/adapter/feishu/client_test.go`, `internal/adapter/notion/client_test.go`
- Impact: Each new adapter test file must choose or copy a pattern. The two signatures cannot be mixed.
- Fix approach: Create `internal/adapter/testutil/http.go` with a canonical `RoundTripFunc` type and shared `jsonResponse` / `mustJSON` helpers. Import in all test files.

**Pervasive `map[string]any` for structured data:**
- Issue: All adapter operations accept `map[string]any` input and return `map[string]any` output. No typed request/response structs exist at the adapter boundary. Input validation is ad-hoc (e.g., `requireIDField`, `asString` scattered in each method).
- Files: `internal/adapter/registry.go` (HandlerFunc signature), `internal/adapter/feishu/docx.go`, `internal/adapter/notion/block.go`, `internal/adapter/notion/page.go`, `internal/adapter/feishu/bitable.go`
- Impact: No compile-time safety for operation payloads. Typos in field names ("page_id" vs "pageId") are only caught at runtime by tests. IDE auto-completion is useless.
- Fix approach: For frequently used operations, define typed request structs with JSON tags and a single `Validate()` method. Keep the adapter handler signature as `map[string]any` but decode early.

**Duplicate redaction logic:**
- Issue: `internal/runtime/redact.go` and `internal/adapter/debug_redaction.go` both implement recursive `map[string]any` redaction with overlapping sensitive key lists and slightly different behavior (runtime version only redacts `Bearer` prefixes; adapter version also redacts content fields, URLs, and heuristically long tokens).
- Files: `internal/runtime/redact.go`, `internal/adapter/debug_redaction.go`
- Impact: Divergent redaction policies mean audit logs may leak data that debug output correctly masks, or vice versa.
- Fix approach: Unify into `internal/adapter/debug_redaction.go` as the authoritative implementation. Import it in `runtime` package.

## Known Bugs

**Plugin audit sink closes after first emit:**
- Symptoms: `pluginAuditSink.Emit` calls `defer s.runtime.Close()` at line 130 of `internal/runtime/audit_sink.go`. After the first audit event, the plugin process is terminated. Subsequent events from the same governance cycle will fail.
- Files: `internal/runtime/audit_sink.go` (lines 129-132)
- Trigger: Any execution with a plugin-backed audit sink configured. The first event succeeds; subsequent events in the same execution produce "failed to emit event to audit sink" warnings.
- Workaround: Use stdout or webhook sinks for multi-event scenarios.

**`context.Background()` used in non-test execution paths:**
- Symptoms: Multiple production code paths use `context.Background()` instead of propagating the caller context. This breaks cancellation propagation and timeout enforcement.
- Files: `internal/cli/root.go` (lines 157, 280, 460, 555), `internal/adapter/runtime_options.go` (lines 35, 52, 72, 84), `internal/secretstore/store.go` (lines 212, 226, 248, 262, 270), `internal/plugin/install.go` (lines 373, 1004)
- Trigger: When a user sends SIGINT during plugin discovery or a context timeout fires during execution, background-context operations will not be cancelled.
- Workaround: None currently.

## Security Considerations

**SHA-1 used for NPM artifact verification:**
- Risk: `internal/plugin/install.go` uses `crypto/sha1` at line 1421 (`sha1.New`) to verify NPM package shasums. SHA-1 is collision-vulnerable. While npm registry still publishes SHA-1 shasums, the code should at minimum enforce that Subresource Integrity (SHA-256/SHA-512) is verified first and fail when only SHA-1 is available.
- Files: `internal/plugin/install.go` (lines 7, 1421)
- Current mitigation: `verifySubresourceIntegrity` at line 1431 checks SHA-256/SHA-512 integrity tokens when present. SHA-1 is only used as a fallback. The `verifyDownloadedNPMArtifact` function at line 1408 checks integrity first, then falls back to SHA-1.
- Recommendations: Log a warning when falling back to SHA-1 verification. Consider making SRI mandatory for production installs.

**No download size limit for plugin archives:**
- Risk: `io.Copy(file, response.Body)` at line 901 of `internal/plugin/install.go` has no size bound. A malicious or compromised registry could serve an arbitrarily large download, exhausting disk space.
- Files: `internal/plugin/install.go` (line 901)
- Current mitigation: HTTP client has a 60-second timeout (line 28), but a fast connection can still download multi-GB files.
- Recommendations: Wrap `response.Body` with `io.LimitReader` using a configurable maximum (e.g., 500MB default).

**`safeFilename` uses simple string replacement:**
- Risk: The `safeFilename` function at line 358 of `internal/runtime/governance.go` only replaces `/`, `\`, `:`, and `..`. It does not handle null bytes, control characters, or Unicode tricks that could cause unexpected filesystem behavior.
- Files: `internal/runtime/governance.go` (line 358)
- Current mitigation: Input comes from idempotency keys which are typically hashes, limiting exposure.
- Recommendations: Use `filepath.Base` or a stricter allowlist (alphanumeric + limited punctuation).

**Webhook audit sink sends full audit records without TLS validation config:**
- Risk: `internal/runtime/audit_sink.go` lines 164-194 send full audit records (including operation names, account names, input summaries) to user-configured webhook URLs. Custom headers can include auth tokens stored in the config file. There is no option to enforce HTTPS-only or validate certificates.
- Files: `internal/runtime/audit_sink.go` (lines 160-195)
- Current mitigation: Default timeout is 5 seconds. HTTP client is created fresh per emission.
- Recommendations: Add config validation that rejects non-HTTPS webhook URLs by default with an explicit opt-in flag.

**`CLAWRISE_MASTER_KEY` environment variable:**
- Risk: When `CLAWRISE_MASTER_KEY` is set (line 375 of `internal/secretstore/store.go`), it is SHA-256 hashed and used directly as the AES key. The env var value is readable by any process with the same user. If the key is weak or leaked, all stored secrets are compromised.
- Files: `internal/secretstore/store.go` (lines 375-383)
- Current mitigation: Key file is written with 0o600 permissions. The env var approach is intended for CI/CD.
- Recommendations: Document the security implications clearly. Consider requiring a minimum key length.

## Performance Bottlenecks

**File-based idempotency store with no concurrency control:**
- Problem: `internal/runtime/governance.go` stores idempotency records as individual JSON files in a flat directory. Each execution reads and writes one file. Under concurrent CLI invocations, two processes can both see no existing record and both proceed, violating idempotency.
- Files: `internal/runtime/governance.go` (lines 166-229, 417-451)
- Cause: No file locking. `LoadIdempotencyRecord` followed by `SaveIdempotencyRecord` is not atomic.
- Improvement path: Use `fcntl`/`flock` file locks, or switch to an embedded store (e.g., SQLite) for the governance backend.

**Audit log append with per-record file open:**
- Problem: `AppendAuditRecord` at line 453 opens the audit file, appends one line, and closes it for every single execution. Under batch mode (`clawrise batch`), this creates N file open/close operations.
- Files: `internal/runtime/governance.go` (lines 453-474)
- Cause: `fileGovernanceStore` has no buffering or batching.
- Improvement path: Buffer audit records and flush periodically or at end-of-batch.

**Plugin process startup on every operation:**
- Problem: `internal/runtime/audit_sink.go` line 131 closes the plugin runtime after every audit emit (`defer s.runtime.Close()`). This means for each operation execution, a new plugin subprocess is started, does its work, and is killed.
- Files: `internal/runtime/audit_sink.go` (lines 129-132), `internal/plugin/process.go` (lines 245-276)
- Cause: The `pluginAuditSink` is constructed per-sink-per-emission and immediately closed.
- Improvement path: Cache running plugin processes across audit emissions within the same CLI invocation. Add a lifecycle manager.

**`map[string]any` serialization in idempotency hash:**
- Problem: `calculateInputHash` at line 349 of `internal/runtime/governance.go` marshals the entire input map to JSON for hashing. For large inputs (e.g., markdown content in `notion.page.create`), this allocates and serializes the full payload.
- Files: `internal/runtime/governance.go` (line 349)
- Cause: Go's `json.Marshal` for `map[string]any` is not stream-based.
- Improvement path: For large payloads, hash the raw input JSON string directly instead of re-marshaling.

## Fragile Areas

**Plugin protocol version handshake:**
- Files: `internal/plugin/protocol.go` (line 16, `ProtocolVersion = 1`), `internal/plugin/process.go` (lines 245-276)
- Why fragile: Any change to the JSON-RPC message shapes or protocol version breaks backward compatibility with all installed plugins. The protocol version is a single constant with no migration path.
- Safe modification: When adding fields, make them optional. Never remove or rename existing fields. Increment `ProtocolVersion` only when breaking changes are intentional, and add a compatibility matrix.
- Test coverage: `internal/plugin/sample_plugin_compat_test.go` covers basic compatibility but does not test version mismatch scenarios.

**Feishu client response parsing:**
- Files: `internal/adapter/feishu/docx.go` (1055 lines), `internal/adapter/feishu/bitable.go` (709 lines)
- Why fragile: Each operation manually unmarshals nested Feishu API response structures into local types. Any upstream API change (field rename, nested structure change) silently produces zero-value fields rather than errors.
- Safe modification: Add response validation checks after unmarshaling (verify critical fields are non-empty). The code already does this partially (e.g., `docx.go` line 47 checks `DocumentID == ""`), but coverage is inconsistent.
- Test coverage: `internal/adapter/feishu/client_test.go` (1244 lines) covers most operations but relies on manually constructed response JSON that may drift from actual API responses.

**Notion block normalization:**
- Files: `internal/adapter/notion/block.go` (894 lines)
- Why fragile: `normalizeBlockData` handles 15+ Notion block types via a large type switch. Adding a new block type requires finding and updating this switch. Missing block types silently pass through with no indication.
- Safe modification: Add a log or annotation when an unrecognized block type is encountered. Centralize the block type string constants.
- Test coverage: `internal/adapter/notion/task_block_test.go` covers some block types but not all 15+.

## Scaling Limits

**File-based config store:**
- Current capacity: Single YAML file loaded/saved per command invocation.
- Limit: Config file is fully loaded, modified in memory, and rewritten for every config mutation. With hundreds of accounts or plugins, the file grows but remains manageable. Concurrent writes from parallel CLI invocations will lose data.
- Scaling path: For concurrent access, add file locking to `internal/config/store.go` (line 71 `Save` method).

**Plugin discovery via directory walk:**
- Current capacity: `filepath.WalkDir` in `internal/plugin/discovery.go` (line 69) scans the entire plugin directory on every CLI invocation.
- Limit: With many installed plugins (50+), discovery adds noticeable startup latency. Each plugin directory requires reading and parsing a `plugin.json` manifest.
- Scaling path: Add a plugin cache/manifest index that is refreshed only on install/upgrade.

**Single-process architecture:**
- Current capacity: Each CLI invocation is a separate process. No daemon mode.
- Limit: Repeated invocations pay full startup cost (config load, plugin discovery, process handshake) each time.
- Scaling path: Not currently needed for CLI usage, but batch mode (`clawrise batch`) reuses the executor within one process, which is the right pattern.

## Dependencies at Risk

**Minimal dependency surface:**
- The project has only two external dependencies: `github.com/spf13/pflag v1.0.5` and `gopkg.in/yaml.v3 v3.0.1`. Both are stable, well-maintained libraries with no known vulnerabilities.
- Risk: Low. The minimal dependency surface is a strength.

**Custom JSON-RPC protocol for plugins:**
- Risk: The plugin communication protocol at `internal/plugin/protocol.go` is hand-rolled JSON-RPC over stdin/stdout. While simple, it lacks features like request multiplexing, backpressure, or graceful shutdown signaling.
- Impact: Plugin crashes are detected but recovery is limited. A hung plugin process blocks the CLI indefinitely until timeout.
- Migration plan: If needed, could migrate to gRPC or a more robust framing protocol, but the current approach is adequate for the single-request-per-plugin-invocation pattern.

## Missing Critical Features

**No structured logging framework:**
- Problem: The codebase uses no logging framework. `fmt.Fprintf(os.Stderr, ...)` is used in plugin binaries (`cmd/clawrise-plugin-feishu/main.go`, `cmd/clawrise-plugin-notion/main.go`). The CLI itself produces no diagnostic output beyond the JSON envelope. There is no way to enable verbose/debug logging for troubleshooting.
- Files: All `cmd/clawrise-plugin-*/main.go` files
- Blocks: Troubleshooting production issues, understanding plugin lifecycle events, debugging auth flows.

**No graceful plugin process shutdown:**
- Problem: `internal/plugin/process.go` kills plugin processes via `cmd.Process.Kill()` on close (line ~290). There is no signal to the plugin to flush state or clean up.
- Files: `internal/plugin/process.go` (lines 288-295)
- Blocks: Plugins that need to clean up temporary resources or flush buffers on shutdown.

**No config file migration/versioning:**
- Problem: `internal/config/config.go` has no config schema version field. Adding new config fields requires either breaking existing configs or adding nil checks everywhere (current approach via `Ensure()`).
- Files: `internal/config/config.go` (lines 155-182)
- Blocks: Clean evolution of configuration schema.

## Test Coverage Gaps

**No test coverage for `internal/apperr`:**
- What's not tested: The entire `apperr` package (`internal/apperr/apperr.go`, 42 lines) has no test file. While small, the `WithRetryable`, `WithHTTPStatus`, and `WithUpstreamCode` builder methods are used throughout the codebase for error classification.
- Files: `internal/apperr/apperr.go`
- Risk: A regression in error builder chaining would break retry logic and upstream error reporting silently.
- Priority: Medium

**No test coverage for `internal/output`:**
- What's not tested: `internal/output/json.go` handles JSON output formatting. No test file exists.
- Files: `internal/output/json.go`
- Risk: Low -- the package is thin, but JSON output formatting regressions would affect every command.
- Priority: Low

**No test coverage for `internal/buildinfo`:**
- What's not tested: `internal/buildinfo/buildinfo.go` provides version metadata. No test file exists.
- Files: `internal/buildinfo/buildinfo.go`
- Risk: Low -- static metadata, unlikely to regress.
- Priority: Low

**No test coverage for `internal/paths`:**
- What's not tested: `internal/paths/paths.go` provides path resolution constants. No test file exists.
- Files: `internal/paths/paths.go`
- Risk: Low.
- Priority: Low

**No test coverage for plugin binary entry points:**
- What's not tested: All `cmd/clawrise-plugin-*/main.go` files lack test coverage. The main functions are thin (create adapter + call `ServeRuntime`), but error handling paths (`os.Exit(1)`) are untested.
- Files: `cmd/clawrise-plugin-feishu/main.go`, `cmd/clawrise-plugin-notion/main.go`, `cmd/clawrise-plugin-demo/main.go`, `cmd/clawrise-plugin-sample-policy/main.go`, `cmd/clawrise-plugin-sample-audit/main.go`, `cmd/clawrise-plugin-auth-browser/main.go`
- Risk: Medium -- binary startup regressions would not be caught until integration testing.
- Priority: Medium

**Limited `t.Parallel()` usage:**
- What's not tested: Only `internal/runtime/operation_test.go` uses `t.Parallel()` (3 tests). The remaining 361 test functions run sequentially.
- Files: All `*_test.go` files
- Risk: Test suite takes longer than necessary. No concurrency bugs are caught by test parallelism.
- Priority: Low (test speed, not correctness)

**No end-to-end test with real provider APIs:**
- What's not tested: `internal/adapter/notion/live_test.go` exists but is gated behind `NOTION_ACCESS_TOKEN` env var. No CI integration. No equivalent for Feishu.
- Files: `internal/adapter/notion/live_test.go`
- Risk: Upstream API changes are only caught by manual testing or production failures.
- Priority: High (for production reliability)

---

*Concerns audit: 2026-04-09*
