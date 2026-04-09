# Phase 2: Context 传播 - Context

**Gathered:** 2026-04-09
**Status:** Ready for planning

<domain>
## Phase Boundary

将 4 个关键文件（root.go, runtime_options.go, store.go, install.go）中的 `context.Background()` 替换为调用方 context，使 CLI 操作执行路径支持 SIGINT 取消和超时中断。

严格限于 CTX-01~04 需求范围，不扩展到其他生产代码中的 context.Background() 调用。

</domain>

<decisions>
## Implementation Decisions

### 修复范围
- **D-01:** 严格限于 CTX-01~04（root.go, runtime_options.go, store.go, install.go），不扩展到其他文件

### 根 Context 创建方式
- **D-02:** 使用 `signal.NotifyContext(context.Background(), os.Interrupt)` 在 root.go 入口处创建根 context，支持 SIGINT（Ctrl+C）取消所有进行中的操作。不使用全局 WithTimeout，各 HTTP client 已有各自的超时设置。

### 明确排除的范围
- **D-03:** governance.go 和 audit_sink.go 的 context.Background() 不在本次范围，留到未来版本
- **D-04:** batch.go:75 不在本次范围，留到未来版本
- **D-05:** auth.go、config.go、account.go 等非关键路径的 context.Background() 不在本次范围

### Claude's Discretion
- root.go 中具体哪些函数需要创建 signal.NotifyContext（runOperation、runInspect、runAuthInspect 等入口函数的选择）
- context 参数在函数签名间的传递方式（参数名、位置等代码风格细节）
- 测试代码中 context.Background() 的使用不需要修改（测试中使用是正确的）

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### 核心需求文件
- `.planning/REQUIREMENTS.md` — CTX-01~04 的具体行号和描述
- `.planning/ROADMAP.md` §Phase 2 — 成功标准和验收条件

### 待修改文件
- `internal/cli/root.go` — CTX-01: 操作执行路径的 context 传播（lines 157, 280, 460, 555）
- `internal/adapter/runtime_options.go` — CTX-02: HTTP 调用的 context 传播（lines 35, 52, 72, 84）
- `internal/secretstore/store.go` — CTX-03: 加解密操作的 context 传播（lines 212, 226, 248, 262, 270）
- `internal/plugin/install.go` — CTX-04: 下载/验证操作的 context 传播（lines 373, 1004）

### 项目指南
- `./CLAUDE.md` — 项目编码规范和架构约束

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `context.Context` 已作为参数存在于 `Executor.ExecuteContext()` 签名中 — 只需在调用处传入正确的 ctx
- `adapter.WithRuntimeOptions()` 和 `adapter.WithAccountName()` 已有 context value 传递模式，runtime_options.go 可以直接复用

### Established Patterns
- Go 标准 context 传播模式：`ctx, cancel := signal.NotifyContext(...)` + `defer cancel()`
- HTTP 调用使用 `http.NewRequestWithContext(ctx, ...)` 替换 `http.NewRequest(...)`
- 函数签名中 context 作为第一个参数：`func (s *Store) Get(ctx context.Context, ...)`

### Integration Points
- root.go 是 CLI 入口，所有操作执行路径的起点
- runtime_options.go 被 adapter 层 HTTP client 调用链引用
- store.go 被 secret store 加解密流程引用
- install.go 被插件安装/验证流程引用

</code_context>

<specifics>
## Specific Ideas

- signal.NotifyContext 使用 `os.Interrupt` 即可，不需要 syscall.SIGTERM（CLI 工具通常只接收 SIGINT）
- root.go 中需要修改的入口函数：`runOperation()`（line 280）、`NewDiscoveredManagerWithOptions`（line 157）、`InspectDiscoveryWithOptions`（line 460）、`InspectAuth`（line 555）
- runtime_options.go 的 4 处 `ctx = context.Background()` 应改为使用传入的 ctx 参数，仅在 ctx 为 nil 时 fallback 到 Background()

</specifics>

<deferred>
## Deferred Ideas

- batch.go:75 的 ExecuteContext(context.Background()) — 与 root.go:280 同模式，但不在 CTX-01~04 范围内
- governance.go 的 LoadIdempotency/SaveIdempotency/AppendAudit — 不在需求范围内
- audit_sink.go:102 的 Emit — Phase 1 已修复生命周期，context 传播留到未来
- auth.go 多处认证操作 — 非关键路径
- config.go, account.go — 配置初始化路径
- storage_backend_resolvers.go — 存储后端解析
- verify.go:106 — 插件验证
- server.go:195 — JSON-RPC 服务端

</deferred>

---

*Phase: 02-context*
*Context gathered: 2026-04-09*
