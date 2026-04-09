# Clawrise CLI — 发布前关键修复

## What This Is

Clawrise 是一个插件驱动的 CLI 运行时，通过 JSON-RPC over stdio 与外部 provider 插件进程通信。本次工作是在 `develop` 分支合入 `main` 并发布新版本之前，修复已识别的关键 bug 和安全隐患。

## Core Value

修复审计 sink 和 context 传播问题后，CLI 可以安全地合并到 main 并发布新版本，不会在生产环境中出现审计丢失或无法取消操作的问题。

## Requirements

### Validated

<!-- 已有的经过验证的能力 -->

- ✓ 插件驱动的 CLI 运行时架构 — core + plugin 二层分离，JSON-RPC 协议通信
- ✓ Feishu 和 Notion 两大 provider 适配器 — 完整的 CRUD 操作、文件上传、数据源查询
- ✓ 治理层（idempotency, audit, policy）— 基于 JSON 文件的幂等性记录和审计日志
- ✓ 插件安装/升级流程 — 支持 NPM、registry、本地源三种安装方式
- ✓ CI 流水线 — 包括单元测试、冒烟测试、release 脚本回归
- ✓ npm 发行包装器 — 跨平台分发

### Active

<!-- 本次需要修复的范围 -->

- [ ] 审计 sink 插件进程在首次 emit 后被错误关闭（`defer s.runtime.Close()` 导致后续审计事件失败）
- [ ] 关键生产路径使用 `context.Background()` 替换为调用方 context 传播（CLI dispatch、adapter options、secret store、plugin install）
- [ ] 所有修复后 `go test ./...` 和 `node --test` 全部通过
- [ ] `go vet ./...` 无新增警告
- [ ] 工作区干净，可以合入 main

### Out of Scope

<!-- 明确不做的 -->

- `install.go` / `config.go` / `root.go` 文件拆分重构 — 代码质量问题，不阻塞发布
- `map[string]any` 类型安全改造 — 架构改进，独立处理
- 结构化日志框架引入 — 基础设施改进，不阻塞发布
- `roundTripFunc` 测试辅助函数去重 — 测试质量改进，不阻塞发布
- 红action 逻辑统一 — 代码质量，不阻塞发布
- NPM SHA-1 fallback 强制 SRI — 安全加固，当前已有 SHA-256/SHA-512 优先
- 插件下载大小限制 — 安全加固，有 60s 超时兜底

## Context

- 代码库已有 codebase map：`.planning/codebase/`
- 最新版本标签：`v0.1.1`，develop 分支有 41 个未发布提交
- 发布流程是 main-only：需先合并到 main 才能触发正式发布
- 插件审计 sink bug 有临时 workaround（使用 stdout/webhook），但正式修复应在发布前完成
- `context.Background()` 问题影响 SIGINT 响应和超时取消，在 batch 模式下风险更高

## Constraints

- **兼容性**: 不能破坏现有插件协议版本（`ProtocolVersion = 1`）
- **测试**: 所有现有测试必须继续通过，不能降低覆盖率
- **范围**: 仅修复发布阻塞项，不做额外重构

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| 仅修复审计 sink bug 和 context 传播 | 这两个问题直接影响生产可靠性和安全性，其他技术债不阻塞发布 | — Pending |
| 审计 sink 修复策略待研究 | 需要评估是复用插件进程还是改用其他生命周期管理方式 | — Pending |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd-transition`):
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via `/gsd-complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-04-09 after initialization*
