# Testing Patterns

**Analysis Date:** 2026-04-09

## Test Framework

**Runner:**
- Go standard `testing` package (no external test frameworks like testify or ginkgo)
- Node.js built-in `node:test` for npm wrapper tests
- No `jest.config`, `vitest.config`, or other test runner config files

**Assertion Library:**
- Go: Standard `t.Fatalf`, `t.Fatal`, `t.Error` -- no assertion helpers
- Node.js: `node:assert/strict`

**Run Commands:**
```bash
go test ./...                                              # Run all Go tests
go test ./internal/runtime -run TestParseOperation         # Run one test target
node --test packaging/npm/root/lib/*.test.js               # Run npm wrapper tests
```

## Test File Organization

**Location:**
- Co-located: test files live in the same package directory as the code they test
- No separate `test/` or `tests/` directories
- All test files use `_test.go` suffix (Go) or `.test.js` suffix (Node.js)

**Naming:**
- Go: `TestXxx` pattern where `Xxx` describes the behavior being tested
- Descriptive names: `TestExecutorDryRunSuccess`, `TestExecutorRejectsSubjectMismatch`, `TestRunAuthCompleteNotionPublicWithCallbackURL`
- Some files are split by domain: `extended_test.go`, `p2_test.go`, `p1_p2_test.go`

**Structure:**
```
internal/
  runtime/
    executor.go
    executor_test.go          # Main executor tests
    operation.go
    operation_test.go         # Operation parsing tests
    governance.go
    governance_backend_test.go # Governance backend tests
  cli/
    root.go
    root_test.go              # CLI integration tests (~2900 lines)
  adapter/
    debug_redaction.go
    debug_redaction_test.go   # Redaction tests
    feishu/
      client.go
      client_test.go
      register.go
      extended_test.go
      p2_test.go              # P2 priority tests
      user_profile_test.go
    notion/
      client.go
      client_test.go
      auth_provider_test.go
      register_test.go
      ...
  plugin/
    runtime.go
    runtime_test.go
    ...
packaging/npm/root/lib/
  setup.js
  setup.test.js
  platform.js
  platform.test.js
```

## Test Structure

**Suite Organization:**
Tests use plain functions. No test suites, no table-driven test wrappers, no struct-based grouping. Each test is a standalone `func TestXxx(t *testing.T)`.

Example from `internal/runtime/operation_test.go`:
```go
func TestParseOperationWithFullPath(t *testing.T) {
    t.Parallel()

    operation, err := ParseOperation("feishu.calendar.event.create", "")
    if err != nil {
        t.Fatalf("ParseOperation returned error: %v", err)
    }
    if operation.Platform != "feishu" {
        t.Fatalf("unexpected platform: %s", operation.Platform)
    }
}
```

**Patterns:**
- `t.Parallel()` at the top of tests that are safe to parallelize
- `t.Helper()` in test helper functions
- `t.Setenv()` for environment variable isolation
- `t.TempDir()` for filesystem isolation
- `t.Cleanup()` for restoring global state
- `t.Fatalf()` for assertion failures (stops test immediately)
- No subtests (`t.Run`) detected in the codebase

## Mocking

**Framework:** No mocking framework. All mocking is hand-rolled.

**HTTP Mocking Pattern:**
Custom `roundTripFunc` type replacing `http.Transport`:
```go
type roundTripFunc func(request *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
    return f(request)
}
```

Used to construct mock HTTP clients:
```go
notionClient, err := notionadapter.NewClient(notionadapter.Options{
    BaseURL: "https://api.notion.com",
    HTTPClient: &http.Client{
        Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
            if request.URL.Path != "/v1/pages/page_demo" {
                t.Fatalf("unexpected request path: %s", request.URL.Path)
            }
            return jsonHTTPResponse(t, http.StatusOK, map[string]any{
                "id": "page_demo",
                ...
            }), nil
        }),
    },
})
```

**Interface Mocking Pattern:**
Struct-based stubs implementing plugin interfaces:
```go
type testAuthProvider struct {
    listMethods func(ctx context.Context) ([]pluginruntime.AuthMethodDescriptor, error)
    begin       func(ctx context.Context, params pluginruntime.AuthBeginParams) (pluginruntime.AuthBeginResult, error)
    complete    func(ctx context.Context, params pluginruntime.AuthCompleteParams) (pluginruntime.AuthCompleteResult, error)
    resolve     func(ctx context.Context, params pluginruntime.AuthResolveParams) (pluginruntime.AuthResolveResult, error)
}

func (p *testAuthProvider) Begin(ctx context.Context, params pluginruntime.AuthBeginParams) (pluginruntime.AuthBeginResult, error) {
    if p.begin == nil {
        return pluginruntime.AuthBeginResult{}, nil
    }
    return p.begin(ctx, params)
}
```

**Auth Launcher Mocking:**
```go
type testAuthLauncherRuntime struct {
    descriptor pluginruntime.AuthLauncherDescriptor
    launch     func(params pluginruntime.AuthLaunchParams) (pluginruntime.AuthLaunchResult, error)
}
```

**What to Mock:**
- HTTP transport (roundTripFunc) for testing adapter operations against provider APIs
- Plugin interfaces (testAuthProvider, testAuthLauncherRuntime) for testing plugin manager behavior
- Time injection: `Executor.now func() time.Time` for deterministic timestamps

**What NOT to Mock:**
- Config loading: use real `config.Store` backed by `t.TempDir()`
- Registry: use real `adapter.Registry` with actual adapter registration
- File system: use `t.TempDir()` for real file I/O

## Fixtures and Factories

**Test Data:**
Helper functions construct test fixtures inline. No external fixture files or JSON fixtures.

Key factory functions in `internal/runtime/executor_test.go`:
```go
func newTestStore(t *testing.T, cfg *config.Config) *config.Store {
    t.Helper()
    store := config.NewStore(t.TempDir() + "/config.yaml")
    if err := store.Save(cfg); err != nil {
        t.Fatalf("failed to save test config: %v", err)
    }
    return store
}

func newTestRegistry(t *testing.T, feishuClient *feishuadapter.Client, notionClient *notionadapter.Client) *adapter.Registry {
    t.Helper()
    registry := adapter.NewRegistry()
    feishuadapter.RegisterOperations(registry, feishuClient)
    notionadapter.RegisterOperations(registry, notionClient)
    return registry
}
```

**Config Seeding Pattern:**
```go
store := newTestStore(t, &config.Config{
    Defaults: config.Defaults{
        Platform: "feishu",
        Account:  "feishu_bot_ops",
    },
    Accounts: accountsFromLegacyAccounts(map[string]legacyTestAccount{
        "feishu_bot_ops": {
            Platform: "feishu",
            Subject:  "bot",
            LegacyAuth: legacyTestAuth{
                Type:      "client_credentials",
                AppID:     "env:FEISHU_BOT_OPS_APP_ID",
                AppSecret: "env:FEISHU_BOT_OPS_APP_SECRET",
            },
        },
    }),
})
```

**Example Config Copy Pattern:**
```go
func copyExampleConfig(t *testing.T) string {
    t.Helper()
    configPath := filepath.Join(t.TempDir(), "config.yaml")
    configBytes, err := os.ReadFile("../../examples/config.example.yaml")
    ...
}
```

**Location:**
- All test helpers defined in the same `_test.go` file that uses them
- Shared within files only (no cross-package test helper imports)
- Plugin fixture helpers: `writePolicyCapabilityFixture`, `writeAuditSinkCapabilityFixture`

## Coverage

**Requirements:** No enforced coverage target detected.

**View Coverage:**
```bash
go test -cover ./...
go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out
```

## Test Types

**Unit Tests:**
- Pure logic tests: `TestParseOperationWithFullPath` in `internal/runtime/operation_test.go`
- Redaction tests: `TestRedactDebugValueRedactsSensitiveAndContentFields` in `internal/adapter/debug_redaction_test.go`
- Config resolution tests: `internal/config/resolve_test.go`
- Store backend tests: `internal/plugin/storage_backend_lookup_test.go`

**Integration Tests:**
- Full executor pipeline: `TestExecutorDryRunSuccess`, `TestExecutorExecutesNotionPageGet` in `internal/runtime/executor_test.go`
- CLI end-to-end: `TestRunOperationDryRun`, `TestRunAuthLoginAndCompleteWithDeviceCode` in `internal/cli/root_test.go`
- Plugin lifecycle: `TestManagerRegistersOperationsThroughRuntimeBoundary` in `internal/plugin/runtime_test.go`
- Plugin install/upgrade: `TestRunPluginUpgradeCommand` in `internal/cli/root_test.go`
- Live external API tests: `internal/adapter/notion/live_test.go` (skipped unless env vars set)

**E2E Tests:**
- No separate E2E test suite. The CLI tests in `root_test.go` effectively serve as E2E tests by calling `cli.Run()` with real arguments.

## Common Patterns

**Testing Execution Flow:**
```go
executor := NewExecutor(store, registry)
envelope, err := executor.ExecuteContext(context.Background(), ExecuteOptions{
    OperationInput: "calendar.event.create",
    DryRun:         true,
    InputJSON:      `{"calendar_id":"cal_demo","summary":"Demo Event",...}`,
})
if err != nil {
    t.Fatalf("ExecuteContext returned error: %v", err)
}
if !envelope.OK {
    t.Fatalf("expected dry-run success, got error: %+v", envelope.Error)
}
```

**Testing CLI Commands:**
```go
var stdout bytes.Buffer
var stderr bytes.Buffer
err := Run([]string{"subject", "use", "bot"}, Dependencies{
    Version:       "test",
    Stdout:        &stdout,
    Stderr:        &stderr,
    PluginManager: newTestPluginManager(t),
})
if err != nil {
    t.Fatalf("Run returned error: %v", err)
}
if !bytes.Contains(stdout.Bytes(), []byte(`"subject": "bot"`)) {
    t.Fatalf("expected subject output, got: %s", stdout.String())
}
```

**Async Testing:**
No async test patterns detected. All tests are synchronous.

**Error Testing:**
```go
// Testing expected failure
envelope, err := executor.ExecuteContext(context.Background(), ExecuteOptions{...})
if err != nil {
    t.Fatalf("ExecuteContext returned error: %v", err)
}
if envelope.OK {
    t.Fatal("expected execution to fail because of subject mismatch")
}
if envelope.Error == nil || envelope.Error.Code != "ACCOUNT_PLATFORM_MISMATCH" {
    t.Fatalf("unexpected error payload: %+v", envelope.Error)
}
```

**Testing Exit Codes:**
```go
var exitErr ExitError
if !errors.As(err, &exitErr) || exitErr.Code != 1 {
    t.Fatalf("expected exit code 1, got: %v", err)
}
```

**Testing Filesystem Side Effects:**
```go
auditFile := filepath.Join(filepath.Dir(store.Path()), "runtime", "audit", time.Now().UTC().Format("2006-01-02")+".jsonl")
data, err := os.ReadFile(auditFile)
if err != nil {
    t.Fatalf("failed to read audit file: %v", err)
}
content := string(data)
if strings.Contains(content, "very-secret") {
    t.Fatalf("expected audit log to redact secrets, got: %s", content)
}
```

**Testing Plugin Processes:**
Shell scripts written to temp dirs act as mock plugin binaries:
```go
script := `#!/bin/sh
while IFS= read -r line; do
  case "$line" in
    *'"method":"clawrise.policy.evaluate"'*)
      printf '{"jsonrpc":"2.0","id":"1","result":{"decision":"annotate",...}}'"\n"
      ;;
  esac
done
`
os.WriteFile(filepath.Join(pluginDir, "policy-demo.sh"), []byte(script), 0o755)
```

## Test Isolation

**Environment:**
- `t.Setenv()` for environment variable isolation (auto-cleanup)
- `t.TempDir()` for filesystem isolation (auto-cleanup)
- `t.Setenv("CLAWRISE_CONFIG", ...)` for config path isolation
- `t.Setenv("HOME", t.TempDir())` to prevent reading user home config

**Global State:**
- Restore via `t.Cleanup()`: `builtinAuditStdoutWriter` is replaced and restored
- No use of `os.Setenv` directly (always `t.Setenv`)

**IPv4 Binding:**
Tests that start HTTP servers explicitly bind to `127.0.0.1`:
```go
func newIPv4TestServer(t *testing.T, handler http.Handler) *httptest.Server {
    listener, err := net.Listen("tcp4", "127.0.0.1:0")
    ...
}
```

## Node.js Test Patterns

**Framework:** Node.js built-in `node:test` with `node:assert/strict`

**Pattern:**
```javascript
const { test } = require('node:test');
const assert = require('node:assert/strict');

test('描述', () => {
  assert.equal(actual, expected);
  assert.throws(() => fn(), /regex/);
});
```

**Files:**
- `packaging/npm/root/lib/setup.test.js` -- setup argument parsing and validation
- `packaging/npm/root/lib/platform.test.js` -- platform detection

---

*Testing analysis: 2026-04-09*
