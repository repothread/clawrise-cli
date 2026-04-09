# Coding Conventions

**Analysis Date:** 2026-04-09

## Naming Patterns

**Packages:**
- All lowercase, single word: `runtime`, `config`, `adapter`, `plugin`, `apperr`
- Import aliases resolve collisions: `authcache "github.com/clawrise/clawrise-cli/internal/auth"`, `pluginruntime "github.com/clawrise/clawrise-cli/internal/plugin"`
- Provider adapter packages use the provider name: `internal/adapter/feishu`, `internal/adapter/notion`

**Files:**
- `snake_case.go`: One file per domain concept: `operation.go`, `executor.go`, `governance.go`, `redact.go`
- Test files: Same name with `_test.go` suffix co-located in the same package
- Spec files: `<domain>_spec.go` (e.g., `calendar_spec.go`, `page_spec.go`) define operation metadata
- Registration: `register.go` in each adapter package wires operations into the registry

**Functions:**
- Exported functions use PascalCase: `ParseOperation`, `NewExecutor`, `ResolveStore`
- Unexported helpers use camelCase: `buildRequestID`, `isSensitiveKey`, `cloneAnyMap`
- Constructor pattern: `New<Type>(...)` or `New<Type>WithOptions(...)`
- Factory helpers in tests: `newTestStore`, `newTestRegistry`, `newTestPluginManager`

**Types:**
- Exported structs use PascalCase: `Envelope`, `ExecuteOptions`, `Definition`
- Unexported structs for internal state: `runtimeGovernance`, `persistedIdempotencyRecord`
- Type aliases for function signatures: `type roundTripFunc func(*http.Request) (*http.Response, error)`
- Context key types use dedicated unexported types: `type callContextKey string`

**Variables and Constants:**
- Constants use PascalCase (exported) or camelCase (unexported): `ProtocolVersion`, `policyDecisionAllow`
- Error codes use UPPER_SNAKE_CASE strings: `"CONFIG_LOAD_FAILED"`, `"OPERATION_NOT_FOUND"`
- Request prefixes: `"req_"`, `"idem_"`

## Code Style

**Formatting:**
- Standard `gofmt` / `goimports` formatting
- Indentation: tabs
- No explicit linter config files detected (relies on standard Go tooling)

**Import Groups:**
Three groups separated by blank lines:
1. Standard library (`context`, `fmt`, `strings`, `time`, etc.)
2. Third-party (`github.com/spf13/pflag`, `gopkg.in/yaml.v3`)
3. Internal packages (`github.com/clawrise/clawrise-cli/internal/...`)

Example from `internal/runtime/executor.go`:
```go
import (
    "context"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "io"
    "sort"
    "strings"
    "time"

    "github.com/clawrise/clawrise-cli/internal/adapter"
    "github.com/clawrise/clawrise-cli/internal/apperr"
    authcache "github.com/clawrise/clawrise-cli/internal/auth"
    "github.com/clawrise/clawrise-cli/internal/config"
    pluginruntime "github.com/clawrise/clawrise-cli/internal/plugin"
    "github.com/clawrise/clawrise-cli/internal/secretstore"
)
```

## Error Handling

**Application Error Type:**
All domain errors use `*apperr.AppError` from `internal/apperr/apperr.go`:
```go
apperr.New("CODE", "message")
apperr.New("CODE", "message").WithRetryable(true)
apperr.New("CODE", "message").WithHTTPStatus(404).WithUpstreamCode("not_found")
```

**Error Code Convention:**
- UPPER_SNAKE_CASE: `CONFIG_LOAD_FAILED`, `ACCOUNT_NOT_FOUND`, `IDEMPOTENCY_KEY_CONFLICT`, `POLICY_DENIED`
- Consistent error codes across the runtime layer, all mapped into `Envelope.Error.Code`

**Error Propagation:**
- Runtime executor returns `(Envelope, error)` from `Execute` -- errors only for truly unexpected failures
- Business logic errors are wrapped in `apperr.AppError` and placed inside the `Envelope` with `OK: false`
- CLI layer returns `error` from `Run()`, with `ExitError{Code: 1}` for expected failures

**Guard Pattern:**
Early returns on error, wrapping with context:
```go
if err != nil {
    return e.auditEnvelope(governance, e.buildFatalEnvelope(requestID, opts.DryRun, "", "", apperr.New("CONFIG_LOAD_FAILED", err.Error())), input), nil
}
```

## Logging

**Framework:** No structured logging library. Output goes through `output.WriteJSON` to stdout.

**Patterns:**
- All CLI output is JSON via `internal/output/json.go` `WriteJSON` function
- Human-readable help text uses `fmt.Fprintln`
- No log levels or log files for runtime operations
- Audit records written as JSONL files under `runtime/audit/`

## Comments

**GoDoc Convention:**
Exported types and functions have GoDoc comments starting with the type/function name:
```go
// Executor runs the normalized operation execution flow.
type Executor struct { ... }

// ParseOperation converts user input into the normalized operation shape.
func ParseOperation(raw, defaultPlatform string) (Operation, error) { ... }
```

**Inline Comments:**
Used sparingly for non-obvious logic. Comments are written in Chinese when explaining intent:
```go
// 显式绑定到 127.0.0.1，避免在 IPv6 受限环境下默认监听 [::1] 造成测试不稳定。
```

**No TODO/FIXME markers detected** in the current codebase.

## Function Design

**Size:** Functions range from small helpers (5-10 lines) to medium domain functions (30-80 lines). The `Execute` method in `internal/runtime/executor.go` is the largest at ~180 lines, handling the full execution pipeline.

**Parameters:**
- Options structs for complex signatures: `ExecuteOptions`, `DiscoveryOptions`, `ManagerOptions`
- Context as first parameter: `func (e *Executor) Execute(ctx context.Context, opts ExecuteOptions)`
- Interface parameters: `io.Writer`, `io.Reader` for I/O

**Return Values:**
- Functions that can fail return `(result, error)` or `(result, *apperr.AppError)`
- Envelope-based return for execution layer: `(Envelope, error)` where Envelope captures both success and business-error state
- Pointer returns for optional values: `*IdempotencyState`, `*ErrorBody`, `*PolicyResult`

## Module Design

**Exports:**
- Each package exports a clear public API surface
- Internal state is unexported: `runtimeGovernance`, `governanceStore` interfaces
- Provider packages export: `NewClient`, `RegisterOperations`, `NewAuthProvider`

**Barrel / Registration Pattern:**
Provider adapters use a `register.go` file that calls `registry.Register(adapter.Definition{...})` for each operation. Example from `internal/adapter/feishu/register.go`:
```go
func RegisterOperations(registry *adapter.Registry, client *Client) {
    registry.Register(adapter.Definition{
        Operation:       "feishu.calendar.event.create",
        Platform:        "feishu",
        Mutating:        true,
        DefaultTimeout:  10 * time.Second,
        AllowedSubjects: []string{"bot", "user"},
        Spec:            calendarEventCreateSpec(),
        Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
            return client.CreateCalendarEvent(ctx, executionProfileFromCall(call), call.Input, call.IdempotencyKey)
        },
    })
}
```

**No `init()` functions detected.** All wiring is explicit.

## Context Usage Pattern

Context carries runtime metadata via typed keys with unexported key types:
```go
type runtimeOptionsContextKey string
const runtimeOptionsKey runtimeOptionsContextKey = "runtime_options"

func WithRuntimeOptions(ctx context.Context, options RuntimeOptions) context.Context { ... }
func RuntimeOptionsFromContext(ctx context.Context) RuntimeOptions { ... }
```

Nil-safe: All context readers handle `nil` context gracefully.

## JSON and YAML Conventions

**JSON Tags:**
Struct fields use `json:"name,omitempty"` consistently:
```go
type Envelope struct {
    OK          bool              `json:"ok"`
    Operation   string            `json:"operation"`
    RequestID   string            `json:"request_id"`
    Context     *Context          `json:"context,omitempty"`
    Error       *ErrorBody        `json:"error"`
}
```

**YAML Tags:**
Config structs use `yaml:"name,omitempty"`:
```go
type Config struct {
    Defaults Defaults           `yaml:"defaults"`
    Auth     AuthConfig         `yaml:"auth,omitempty"`
    Runtime  RuntimeConfig      `yaml:"runtime,omitempty"`
    Plugins  PluginsConfig      `yaml:"plugins,omitempty"`
    Accounts map[string]Account `yaml:"accounts,omitempty"`
}
```

## Security Conventions

**Secret Redaction:**
Two redaction layers:
1. `internal/runtime/redact.go` for audit log redaction
2. `internal/adapter/debug_redaction.go` for provider debug output

Both scan for sensitive key fragments (`token`, `secret`, `password`, `authorization`, etc.) and replace values with `"***"`.

**Secret References:**
Config never stores raw secrets; uses references like `"env:ENV_VAR_NAME"` or `"secret:account:field"` in `AccountAuth.SecretRefs`.

## Operation Naming Convention

Operations follow `<platform>.<resource-path>.<action>`:
- `feishu.calendar.event.create`
- `notion.page.get`
- `feishu.docs.block.update`
- `notion.block.append`

When a default platform is set, users can omit the platform prefix: `calendar.event.create` resolves to `feishu.calendar.event.create`.

## CLI Output Convention

All CLI output uses JSON via `output.WriteJSON`. Successful responses include `"ok": true`. Error responses include `"ok": false` with an `Error` body containing `code`, `message`, and optional `retryable`/`upstream_code` fields.

---

*Convention analysis: 2026-04-09*
