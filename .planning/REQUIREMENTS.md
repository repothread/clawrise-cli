# Requirements: Clawrise CLI — 发布前关键修复

**Defined:** 2026-04-09
**Core Value:** 修复审计 sink 和 context 传播问题后，CLI 可以安全发布新版本

## v1 Requirements

Requirements for initial release. Each maps to roadmap phases.

### Bug Fix

- [ ] **BUG-01**: 插件审计 sink 不再在首次 emit 后关闭插件进程 — `internal/runtime/audit_sink.go` 中 `defer s.runtime.Close()` 导致后续审计事件失败
- [ ] **BUG-02**: 审计 sink 支持在同一 CLI 调用中多次 emit 而无需重启插件进程

### Context Propagation

- [ ] **CTX-01**: `internal/cli/root.go` 的操作执行路径使用调用方 context 替代 `context.Background()`（lines 157, 280, 460, 555）
- [ ] **CTX-02**: `internal/adapter/runtime_options.go` 的 HTTP 调用传递调用方 context（lines 35, 52, 72, 84）
- [ ] **CTX-03**: `internal/secretstore/store.go` 的加解密操作使用调用方 context（lines 212, 226, 248, 262, 270）
- [ ] **CTX-04**: `internal/plugin/install.go` 的下载/验证操作使用调用方 context（lines 373, 1004）

### Validation

- [ ] **VAL-01**: 所有修复后 `go test ./...` 全部通过
- [ ] **VAL-02**: `go vet ./...` 无新增警告
- [ ] **VAL-03**: `node --test packaging/npm/root/lib/*.test.js` 全部通过
- [ ] **VAL-04**: 工作区干净，可合入 main

## v2 Requirements

Deferred to future release.

### Code Quality

- **QUAL-01**: 拆分 `internal/plugin/install.go`（1663 行）为多个聚焦文件
- **QUAL-02**: 拆分 `internal/cli/config.go`（1315 行）为多个子命令文件
- **QUAL-03**: 拆分 `internal/cli/root.go` doctor 实现到独立文件
- **QUAL-04**: 统一 `roundTripFunc` 测试辅助函数到 `internal/adapter/testutil/http.go`
- **QUAL-05**: 统一 redaction 逻辑到 `internal/adapter/debug_redaction.go`

### Security Hardening

- **SEC-01**: NPM 下载 fallback 到 SHA-1 时输出警告日志
- **SEC-02**: 插件下载添加可配置大小限制（`io.LimitReader`）
- **SEC-03**: `safeFilename` 使用 `filepath.Base` 或更严格的 allowlist
- **SEC-04**: Webhook 审计 sink 默认拒绝非 HTTPS URL

### Infrastructure

- **INFRA-01**: 引入结构化日志框架
- **INFRA-02**: 配置文件 schema 版本化
- **INFRA-03**: 插件进程优雅关闭（signal-based）

## Out of Scope

| Feature | Reason |
|---------|--------|
| `map[string]any` 类型安全改造 | 架构改进，影响范围大，独立处理 |
| 端到端 provider API 测试 | 需要 CI 集成 API token，独立任务 |
| 插件协议版本迁移 | 无 breaking change，不需要 |
| 性能优化（idempotency 并发、审计批写） | 不阻塞发布，当前规模可接受 |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| BUG-01 | — | Pending |
| BUG-02 | — | Pending |
| CTX-01 | — | Pending |
| CTX-02 | — | Pending |
| CTX-03 | — | Pending |
| CTX-04 | — | Pending |
| VAL-01 | — | Pending |
| VAL-02 | — | Pending |
| VAL-03 | — | Pending |
| VAL-04 | — | Pending |

**Coverage:**
- v1 requirements: 10 total
- Mapped to phases: 0
- Unmapped: 10 ⚠️

---
*Requirements defined: 2026-04-09*
*Last updated: 2026-04-09 after initial definition*
