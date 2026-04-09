# Phase 2: Context 传播 - Research

**Researched:** 2026-04-09
**Domain:** Go context 传播 / signal.NotifyContext / HTTP client context 集成
**Confidence:** HIGH

## Summary

本阶段需要将 4 个关键文件（root.go, runtime_options.go, store.go, install.go）中的 `context.Background()` 替换为调用方 context，使 CLI 操作执行路径支持 SIGINT 取消和超时中断。经过完整的代码审计，确认了所有需要修改的位置、调用链路以及接口约束。

核心挑战在于 `secretstore.Store` 接口的方法签名（`Get/Set/Delete`）不包含 `context.Context` 参数，而底层 `pluginSecretStore` 的 `ProcessSecretStore` 方法需要 ctx 进行 JSON-RPC 调用。需要在接口签名变更和最小影响范围之间做出权衡。

**Primary recommendation:** 在 root.go 入口处创建 `signal.NotifyContext`，沿调用链向下传递。对于 `secretstore.Store` 接口，由于改动接口签名影响面广（涉及 encryptedFileStore、commandSecretStore、pluginSecretStore 三个实现以及多处调用者），建议在 `pluginSecretStore` 实现层内部接受 context，而保持 `Store` 接口不变——将 `pluginSecretStore` 的方法签名改为接受 context 参数，并在 `pluginSecretStore` 内部传递给底层 client。

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** 严格限于 CTX-01~04（root.go, runtime_options.go, store.go, install.go），不扩展到其他文件
- **D-02:** 使用 `signal.NotifyContext(context.Background(), os.Interrupt)` 在 root.go 入口处创建根 context，支持 SIGINT（Ctrl+C）取消所有进行中的操作。不使用全局 WithTimeout，各 HTTP client 已有各自的超时设置。
- **D-03:** governance.go 和 audit_sink.go 的 context.Background() 不在本次范围，留到未来版本
- **D-04:** batch.go:75 不在本次范围，留到未来版本
- **D-05:** auth.go、config.go、account.go 等非关键路径的 context.Background() 不在本次范围

### Claude's Discretion
- root.go 中具体哪些函数需要创建 signal.NotifyContext（runOperation、runInspect、runAuthInspect 等入口函数的选择）
- context 参数在函数签名间的传递方式（参数名、位置等代码风格细节）
- 测试代码中 context.Background() 的使用不需要修改（测试中使用是正确的）

### Deferred Ideas (OUT OF SCOPE)
- batch.go:75 的 ExecuteContext(context.Background()) — 与 root.go:280 同模式，但不在 CTX-01~04 范围内
- governance.go 的 LoadIdempotency/SaveIdempotency/AppendAudit — 不在需求范围内
- audit_sink.go:102 的 Emit — Phase 1 已修复生命周期，context 传播留到未来
- auth.go 多处认证操作 — 非关键路径
- config.go, account.go — 配置初始化路径
- storage_backend_resolvers.go — 存储后端解析
- verify.go:106 — 插件验证
- server.go:195 — JSON-RPC 服务端
- executor.go:100 的 resolveExecutionIdentity(context.Background(), ...) — 不在 CTX-01~04 需求中，executor 本身已有 ctx 参数
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| CTX-01 | root.go 操作执行路径使用调用方 context 替代 context.Background()（lines 157, 280, 460, 555） | 4 处 context.Background() 已定位；root.go 是 CLI 入口，所有操作从 `Run()` 函数 dispatch；需在 Run() 或各子命令入口创建 signal.NotifyContext |
| CTX-02 | runtime_options.go HTTP 调用传递调用方 context（lines 35, 52, 72, 84） | 4 处 ctx = context.Background() 是 nil 防护；改为直接使用传入 ctx，仅在 ctx == nil 时 fallback 到 Background()；这些函数已有 ctx 参数，改动最小 |
| CTX-03 | secretstore/store.go 加解密操作使用调用方 context（lines 212, 226, 248, 262, 270） | 5 处 context.Background() 在 pluginSecretStore 方法中调用 ProcessSecretStore；Store 接口无 ctx 参数，需在实现层解决 |
| CTX-04 | plugin/install.go 下载/验证操作使用调用方 context（lines 373, 1004） | 2 处 context.Background()；line 373 在 InfoInstalledWithOptions 中调用 inspectRuntimeCapabilities；line 1004 在 resolveRegistrySource 中调用 runtime.Resolve；另需将 downloadRemoteSource 中 http.NewRequest 改为 http.NewRequestWithContext |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go stdlib context | 1.25.9 | Context 传播、取消信号 | Go 标准库原生支持，signal.NotifyContext 从 Go 1.16 起可用 [VERIFIED: go version] |
| Go stdlib os/signal | 1.25.9 | SIGINT 信号捕获 | 与 signal.NotifyContext 配合，标准做法 [VERIFIED: Go stdlib] |
| Go stdlib net/http | 1.25.9 | HTTP 请求 context 传递 | http.NewRequestWithContext 替代 http.NewRequest [VERIFIED: Go stdlib] |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| spf13/pflag | v1.0.5 | CLI 参数解析 | 已有依赖，无需新增 |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| signal.NotifyContext | 手动 signal.Notify + goroutine + channel | signal.NotifyContext 更简洁，自动清理 |
| 修改 Store 接口加 ctx | 在 pluginSecretStore 内部保存 ctx | 接口改动影响面广（3 个实现 + 多处调用者）；内部保存 ctx 更安全但不够 "clean" |

**Installation:**
无新依赖需要安装。

**Version verification:**
```bash
$ go version
go version go1.25.9 darwin/arm64
```

## Architecture Patterns

### Recommended Project Structure
```
修改文件（严格限定 4 个）:
├── internal/cli/root.go              # CTX-01: 创建 signal.NotifyContext，传递给下游
├── internal/adapter/runtime_options.go  # CTX-02: 移除 context.Background() fallback
├── internal/secretstore/store.go     # CTX-03: pluginSecretStore 传递 context
└── internal/plugin/install.go        # CTX-04: HTTP 下载和插件验证传递 context
```

### Pattern 1: 根 Context 创建（root.go）
**What:** 在 CLI 入口创建可取消的 signal-aware context
**When to use:** 程序启动时，需要响应操作系统信号（SIGINT）取消进行中的操作
**Example:**
```go
// Source: Go stdlib context/signals [VERIFIED: Go 1.16+ API]
import (
    "context"
    "os"
    "os/signal"
)

// 在 Run() 函数或入口函数中:
ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
defer cancel()
// 传递 ctx 给下游函数...
```

### Pattern 2: HTTP 请求 context 传递
**What:** 使用 http.NewRequestWithContext 替代 http.NewRequest
**When to use:** 需要让 HTTP 请求响应 context 取消信号
**Example:**
```go
// Source: Go stdlib net/http [VERIFIED: Go 1.13+ API]
request, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
if err != nil {
    return ..., fmt.Errorf("failed to build request: %w", err)
}
response, err := httpClient.Do(request)
```

### Pattern 3: Context nil 防护模式
**What:** 保留 nil ctx 的安全 fallback
**When to use:** 函数接受 context 参数但调用方可能传 nil
**Example:**
```go
// 当前代码模式（runtime_options.go）：
func WithRuntimeOptions(ctx context.Context, options RuntimeOptions) context.Context {
    if ctx == nil {
        ctx = context.Background()
    }
    return context.WithValue(ctx, runtimeOptionsKey, options)
}

// 优化后：保留 nil 防护，这是防御性编程，不改这个模式
```

### Anti-Patterns to Avoid
- **Anti-pattern:** 在中间层创建新的 context.Background()，覆盖调用方的 context。下游操作永远无法被上游取消。
- **Anti-pattern:** 不调用 cancel() 导致 signal handler 泄漏。signal.NotifyContext 返回的 cancel 必须被 defer。
- **Anti-pattern:** 在测试代码中也使用 signal.NotifyContext。测试中使用 context.Background() 是正确的做法（已确认为 Claude's Discretion 范围）。

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| SIGINT 信号处理 | 手动 signal.Notify + goroutine + context.WithCancel | signal.NotifyContext | 标准库提供，自动清理 signal handler |
| HTTP 请求取消 | 自定义 Transport + channel 通知 | http.NewRequestWithContext + ctx.Done() | 标准库内置，net/http 自动监听 context 取消 |

**Key insight:** Go 1.16+ 的 `signal.NotifyContext` 和 Go 1.13+ 的 `http.NewRequestWithContext` 完全覆盖本阶段需求，无需任何第三方库。

## Runtime State Inventory

> 本阶段为代码修改（refactor），不涉及数据迁移、服务配置或 OS 注册状态。

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | 无 — 代码修改不影响任何存储数据 | 无 |
| Live service config | 无 — 无外部服务配置变更 | 无 |
| OS-registered state | 无 — 不涉及 OS 级注册 | 无 |
| Secrets/env vars | 无 — 不涉及密钥或环境变量变更 | 无 |
| Build artifacts | 无 — 纯 Go 代码修改，`go build` 自动反映 | 无 |

## Common Pitfalls

### Pitfall 1: signal.NotifyContext 的 cancel 未调用导致 goroutine 泄漏
**What goes wrong:** signal.NotifyContext 注册了 signal handler，如果不调用 cancel()，handler 永远不会清理
**Why it happens:** 忘记 defer cancel()
**How to avoid:** 严格使用 `ctx, cancel := signal.NotifyContext(...); defer cancel()` 模式
**Warning signs:** 程序退出后进程仍挂起，或在测试中 signal handler 残留

### Pitfall 2: Store 接口签名变更导致编译错误扩散
**What goes wrong:** 如果给 `secretstore.Store` 接口的方法添加 ctx 参数，需要修改所有实现（encryptedFileStore、commandSecretStore、pluginSecretStore）和所有调用者（auth_secret.go、auth.go、executor.go 等）
**Why it happens:** 接口变更需要所有实现和调用者同步更新
**How to avoid:** 在 pluginSecretStore 实现内部解决 context 传递问题，保持 Store 接口不变。如果未来需要全链路 context 传播，再统一修改接口。
**Warning signs:** 编译错误出现在未预期的文件中

### Pitfall 3: http.NewRequestWithContext 传入 nil context
**What goes wrong:** 如果 ctx 传递链路中某处创建了新的 context.Background()，NewRequestWithContext 仍然 "有效" 但取消了信号
**Why it happens:** 某个中间层函数没有正确传递 ctx
**How to avoid:** 确保从 root.go 到最终 HTTP 调用的完整链路都传递同一 ctx
**Warning signs:** SIGINT 后操作不终止

### Pitfall 4: runtime_options.go 的 nil ctx 防护被误删
**What goes wrong:** 删除 `if ctx == nil { ctx = context.Background() }` 后，传入 nil ctx 会导致 panic
**Why it happens:** 误认为调用方一定传非 nil ctx
**How to avoid:** 保留 nil ctx 防护代码，这是防御性编程的正确做法
**Warning signs:** 测试中出现 nil pointer panic

### Pitfall 5: install.go 中 downloadRemoteSource 和 resolveNPMSource 函数签名变更链过长
**What goes wrong:** downloadRemoteSource 是内部函数，被 resolveNPMSource、resolveRegistrySource、materializeSource 调用；如果给所有中间函数添加 ctx 参数，修改范围会扩散到函数签名链
**Why it happens:** 这些函数当前不接受 ctx 参数，需要从上游传递
**How to avoid:** 按调用链从最上层函数（InstallWithOptions、resolveInstallCandidate、materializeSource、downloadRemoteSource 等）逐层添加 ctx 参数，确保链路完整
**Warning signs:** 中间层函数签名缺少 ctx，下游仍使用 context.Background()

## Code Examples

### CTX-01: root.go 创建 signal.NotifyContext 并传递
```go
// 当前代码（root.go:157）：
return pluginruntime.NewDiscoveredManagerWithOptions(context.Background(), buildPluginDiscoveryOptions(cfg))

// 修改方案：在 Run() 函数中创建根 ctx，传递给需要的子命令
// 注意：Run() 函数签名不变（不添加 ctx 参数），
// 在 Run() 内部创建 signal.NotifyContext
func Run(args []string, deps Dependencies) error {
    ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
    defer cancel()
    // ... switch 分支传递 ctx ...
}

// 或者在各入口函数中分别创建（Claude's Discretion 范围）
```

### CTX-01: root.go:280 runOperation 传递 ctx
```go
// 当前代码：
envelope, err := executor.ExecuteContext(context.Background(), runtime.ExecuteOptions{...})

// 修改为：
envelope, err := executor.ExecuteContext(ctx, runtime.ExecuteOptions{...})
```

### CTX-02: runtime_options.go 保持 nil 防护
```go
// 当前代码已经合理——nil 防护不应删除
// WithRuntimeOptions 等函数已有 ctx 参数，只需确保上游传入正确的 ctx
// 4 处 `ctx = context.Background()` 是 nil fallback，保留
// 实际上这里不需要修改——这些函数是被调用方传入 ctx 的，
// 关键是调用方（如 executor.go:200-206）已经传入正确的 ctx
```

### CTX-03: secretstore/store.go pluginSecretStore
```go
// 当前代码（store.go:248）：
func (s *pluginSecretStore) Get(connectionName string, field string) (string, error) {
    result, err := s.client.Get(context.Background(), pluginruntime.SecretStoreGetParams{
        AccountName: connectionName,
        Field:       field,
    })
    ...
}

// 方案 A：修改 Store 接口（影响面广）
// 方案 B（推荐）：pluginSecretStore 内部使用默认 context
//   — 但这没有解决 context 传递问题
// 方案 C（折中）：pluginSecretStore 存储 context 引用，
//   在构造时从调用方获取 context
```

### CTX-04: install.go HTTP 请求传递 context
```go
// 当前代码（install.go:871）：
request, err := http.NewRequest(http.MethodGet, source, nil)

// 修改为（需要函数签名添加 ctx 参数）：
request, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)

// 当前代码（install.go:919）：
request, err := http.NewRequest(http.MethodGet, packageURL, nil)

// 修改为：
request, err := http.NewRequestWithContext(ctx, http.MethodGet, packageURL, nil)
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| http.NewRequest + 手动 context 检查 | http.NewRequestWithContext | Go 1.13 (2019) | 本项目应使用 NewRequestWithContext |
| signal.Notify + goroutine | signal.NotifyContext | Go 1.16 (2021) | 本项目应使用 NotifyContext |

**Deprecated/outdated:**
- 无。Go context API 稳定，本项目使用的 Go 1.25.9 完全支持所需 API。

## Detailed Call Chain Analysis

### CTX-01: root.go 调用链

```
cmd/clawrise/main.go:main()
  -> cli.Run(os.Args[1:], deps)    // 无 ctx 参数
       -> switch 分支:
           case default:
             -> resolvePluginManager(deps, store)  // 创建 plugin manager
                  -> newDefaultPluginManager(store)  // line 147
                       -> pluginruntime.NewDiscoveredManagerWithOptions(context.Background(), ...)  // line 157 [CTX-01-1]
             -> runOperation(args, stdout, stderr, executor)  // line 136
                  -> executor.ExecuteContext(context.Background(), ...)  // line 280 [CTX-01-2]
           case "doctor":
             -> runDoctor(store, stdout, manager)
                  -> pluginruntime.InspectDiscoveryWithOptions(context.Background(), ...)  // line 460 [CTX-01-3]
                  -> manager.InspectAuth(context.Background(), ...)  // line 555 [CTX-01-4]
```

**修改策略：** 在 `Run()` 函数开头创建 `signal.NotifyContext`，传递给 `newDefaultPluginManager`、`runOperation`、`runDoctor` 等需要 ctx 的函数。需要修改这些函数的签名以接受 `ctx context.Context` 参数。

### CTX-02: runtime_options.go 调用链

```
executor.go:200:  adapter.WithRuntimeOptions(ctx, ...)   // 已传入 ctx
executor.go:204:  adapter.WithRequestID(ctx, ...)         // 已传入 ctx
executor.go:206:  adapter.WithProviderDebugCapture(ctx)   // 已传入 ctx
```

**关键发现：** `runtime_options.go` 中的 4 处 `context.Background()` 是 **nil 防护代码**（`if ctx == nil { ctx = context.Background() }`），不是主动创建 Background context。这些函数已经接受 `ctx context.Context` 参数，上游（executor.go）已经传入正确的 ctx。**因此 CTX-02 的修改实际上非常小——只需确保上游传入非 nil ctx 即可。** 如果上游正确传入 ctx（从 root.go 传播下来），这些 nil fallback 不会触发。

**重新评估：** executor.go:200-206 已经使用 `ctx` 变量调用这些函数，而 `ctx` 来自 `Execute()` 方法的参数。只要 root.go:280 传入正确的 ctx（signal.NotifyContext 创建的），CTX-02 就自动解决了。runtime_options.go 的 nil 防护代码保持不变是正确的做法。

### CTX-03: secretstore/store.go 调用链

```
root.go:280 -> executor.ExecuteContext(ctx, ...)
  -> executor.Execute(ctx, opts)
    -> executor.resolveExecutionIdentity(context.Background(), ...)  // line 100 [NOT IN SCOPE per D-05]
      -> executor.buildPluginAuthAccount(...)
        -> secretstore.Open(...)  // 创建 Store
      -> persistAuthPatches(...)
        -> secretStore.Set(accountName, field, value)  // line 670
          -> pluginSecretStore.Set(...)  // 无 ctx 参数
            -> s.client.Set(context.Background(), ...)  // line 262 [CTX-03-2]

cli/auth_secret.go:120 -> secretStore.Set(...)
cli/auth.go:478 -> secretStore.Set(...)
```

**核心问题：** `secretstore.Store` 接口的 `Get/Set/Delete` 方法没有 `context.Context` 参数。调用者通过接口调用，无法传递 ctx。有三种方案：

1. **方案 A：修改 Store 接口添加 ctx 参数** — 影响面最广（3 个实现 + 7+ 处调用者），不在本次范围内更安全
2. **方案 B：仅在 pluginSecretStore 中使用默认 context** — 不解决根本问题
3. **方案 C：在 `pluginSecretStore` 构造时或通过 context value 机制传递** — 过于复杂

**推荐方案：** 鉴于 CONTEXT.md 锁定了 CTX-03 的范围是 `store.go` 文件内的 5 处 `context.Background()`，最合理的做法是修改 `pluginSecretStore` 的方法，让它们接受 context，同时修改 `Store` 接口以添加 ctx 参数。这需要同步修改 `encryptedFileStore` 和 `commandSecretStore` 实现（它们的 ctx 可以传 `context.Background()`），以及所有调用处（在调用处传入适当的 ctx 或 `context.Background()`）。

**影响评估：**
- 接口 `Store` 的 3 个方法需要加 ctx 参数
- `encryptedFileStore` 的 3 个方法签名加 ctx（内部不使用，直接传 Background）
- `commandSecretStore` 的 3 个方法签名加 ctx（内部不使用，直接传 Background）
- `pluginSecretStore` 的 5 个方法修改（Backend、Status、Get、Set、Delete）
- 调用者约 7 处需要传 ctx：executor.go:670, auth_secret.go:120/144, auth.go:395/478, root_test.go:2023, store_test.go 多处

### CTX-04: install.go 调用链

```
plugin.go -> runPlugin() -> install/upgrade 命令
  -> plugin.InstallWithOptions(source, options)
    -> resolveInstallCandidate(source, options)
      -> materializeSource(reference, tempDir, options)
        -> downloadRemoteSource(source, tempDir, options)    // http.NewRequest [line 871]
        -> resolveNPMSource(reference, tempDir, options)
          -> http.NewRequest [line 919]
          -> downloadRemoteSource(...)                       // 二次调用
        -> resolveRegistrySource(reference, tempDir, options)
          -> runtime.Resolve(context.Background(), ...) [line 1004]
          -> downloadRemoteSource(...)                       // 二次调用
  -> InfoInstalledWithOptions(name, version, options)
    -> inspectRuntimeCapabilities(context.Background(), manifest) [line 373]
```

**修改策略：** 需要给 `InstallWithOptions`、`resolveInstallCandidate`、`materializeSource`、`downloadRemoteSource`、`resolveNPMSource`、`resolveRegistrySource` 等函数添加 `ctx context.Context` 参数。`InfoInstalledWithOptions` 同理。HTTP 调用改用 `http.NewRequestWithContext`。

**注意：** `pluginDownloadHTTPClient` 有自己的 60s 超时设置（line 28），这与 context timeout 是叠加关系。使用 `NewRequestWithContext` 后，任一超时先触发都会取消请求。

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | signal.NotifyContext 使用 `os.Interrupt` 即可满足 CLI 工具的需求，不需要 syscall.SIGTERM | CTX-01 | 低 — CLI 工具通常只接收 SIGINT，用户已确认 |
| A2 | runtime_options.go 的 nil 防护不需要修改，上游传入正确 ctx 即可解决 CTX-02 | CTX-02 | 低 — 代码分析已确认 executor.go 已传入 ctx |
| A3 | CTX-03 需要修改 `Store` 接口签名，影响面包括 3 个实现和约 7+ 处调用者 | CTX-03 | 中 — 如果选择不修改接口，需要在 pluginSecretStore 中另找方案 |
| A4 | 测试代码中的 `context.Background()` 不需要修改 | 全局 | 低 — CONTEXT.md 已明确归入 Claude's Discretion |
| A5 | install.go 中的 `pluginDownloadHTTPClient` 的 60s 超时与 context timeout 叠加不会产生意外行为 | CTX-04 | 低 — 两者独立，先到先触发 |

## Open Questions

1. **CTX-03 接口变更策略**
   - What we know: `secretstore.Store` 接口方法无 ctx 参数，`pluginSecretStore` 底层需要 ctx
   - What's unclear: 是否应该修改 `Store` 接口（影响面广）还是在 `pluginSecretStore` 内部另想办法
   - Recommendation: 修改接口是更 clean 的方案，且影响可控（3 个实现、约 7 处调用者）

## Environment Availability

> 本阶段仅涉及 Go 代码修改，无外部依赖。

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go 1.25.9 | 编译和测试 | ✓ | 1.25.9 darwin/arm64 | — |
| signal.NotifyContext | context 传播 | ✓ | Go stdlib 1.16+ | — |
| http.NewRequestWithContext | HTTP 请求取消 | ✓ | Go stdlib 1.13+ | — |

**Missing dependencies with no fallback:**
- 无

**Missing dependencies with fallback:**
- 无

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) |
| Config file | none |
| Quick run command | `go test ./internal/cli/... ./internal/adapter/... ./internal/secretstore/... ./internal/plugin/... ./internal/runtime/... -count=1` |
| Full suite command | `go test ./...` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| CTX-01 | root.go 使用 signal.NotifyContext 传递 ctx | unit | `go test ./internal/cli/... -run TestRun -count=1` | ✅ root_test.go |
| CTX-01 | executor.ExecuteContext 接收并使用传入 ctx | unit | `go test ./internal/runtime/... -run TestExecutor -count=1` | ✅ executor_test.go |
| CTX-02 | runtime_options.go 保留 nil 防护，上游 ctx 正确传入 | unit | `go test ./internal/adapter/... -count=1` | ✅ adapter tests |
| CTX-03 | pluginSecretStore 传递 ctx 给 ProcessSecretStore | unit | `go test ./internal/secretstore/... -count=1` | ✅ store_test.go |
| CTX-04 | install.go HTTP 下载使用 NewRequestWithContext | unit | `go test ./internal/plugin/... -run TestInstall -count=1` | ✅ install_test.go |
| VAL-01 | 所有测试通过 | regression | `go test ./...` | ✅ |

### Sampling Rate
- **Per task commit:** `go test ./internal/cli/... ./internal/adapter/... ./internal/secretstore/... ./internal/plugin/... ./internal/runtime/... -count=1`
- **Per wave merge:** `go test ./...`
- **Phase gate:** `go test ./...` 全部通过 + `go vet ./...` 无新增警告

### Wave 0 Gaps
- None — existing test infrastructure covers all phase requirements.

## Security Domain

> security_enforcement 未显式禁用，包含此部分。

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | 不涉及认证逻辑变更 |
| V3 Session Management | no | 不涉及 session 管理 |
| V4 Access Control | no | 不涉及访问控制 |
| V5 Input Validation | no | 不涉及输入验证 |
| V6 Cryptography | no | 不涉及密码学实现 |

### Known Threat Patterns for Context Propagation

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Context 取消导致操作半完成 | Tampering | 幂等性机制（idempotency）已存在，可安全重试 |
| HTTP 请求取消后资源泄露 | Denial of Service | Go net/http 自动关闭连接，defer response.Body.Close() 已存在 |

## Sources

### Primary (HIGH confidence)
- Go stdlib `context` 包 — signal.NotifyContext API, context 传播模式
- Go stdlib `net/http` 包 — NewRequestWithContext API
- Codebase grep/audit — 所有 context.Background() 位置和调用链路
- CONTEXT.md — 用户锁定决策和排除范围

### Secondary (MEDIUM confidence)
- Go 官方博客 "Go Concurrency Patterns: Context" — context 最佳实践 [CITED: go.dev/blog/context]

### Tertiary (LOW confidence)
- 无

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — 纯 Go stdlib，无第三方依赖
- Architecture: HIGH — 完整代码审计已确认所有调用链路
- Pitfalls: HIGH — 基于代码分析和 Go context 使用经验

**Research date:** 2026-04-09
**Valid until:** 2026-05-09（稳定领域，30 天有效期）
