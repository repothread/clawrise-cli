# Phase 2: Context 传播 - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-09
**Phase:** 02-context
**Areas discussed:** Scope expansion, Root context source, Governance/audit paths, Batch mode

---

## Scope Expansion

| Option | Description | Selected |
|--------|-------------|----------|
| Strict | 仅修复 CTX-01~04 的 4 个文件 | ✓ |
| Expanded core | 修复 CTX-01~04 + executor + governance + audit + batch | |
| Full coverage | 修复所有 20+ 处生产代码 | |

**User's choice:** Strict — 最小风险，满足发布阻塞项
**Notes:** 其他 context.Background() 调用留到未来版本

---

## Root Context Source

| Option | Description | Selected |
|--------|-------------|----------|
| signal.NotifyContext | 在 root.go 入口创建，支持 SIGINT 取消 | ✓ |
| 透传空 context | 从 main.go 传入空 context | |
| WithTimeout + signal | 全局超时 + 信号处理 | |

**User's choice:** signal.NotifyContext
**Notes:** 用户明确确认使用 signal.NotifyContext。不需要全局 WithTimeout，各 HTTP client 已有各自超时设置。使用 os.Interrupt 即可。

---

## Governance/Audit Paths

| Option | Description | Selected |
|--------|-------------|----------|
| Skip | 不修复 governance.go 和 audit_sink.go | ✓ |
| Include | 一并修复 | |

**User's choice:** Skip — 严格限于 CTX-01~04 范围
**Notes:** governance.go 和 audit_sink.go 属于同一执行链但不在需求范围内

---

## Batch Mode

| Option | Description | Selected |
|--------|-------------|----------|
| Skip | 不修复 batch.go | ✓ |
| Include | 一并修复 | |

**User's choice:** Skip — 不在需求范围内
**Notes:** batch.go:75 与 root.go:280 同模式，但留到未来版本

---

## Claude's Discretion

- root.go 中具体哪些入口函数需要创建 signal.NotifyContext
- context 参数的传递方式（参数名、位置等代码风格细节）
- 测试代码中 context.Background() 不需要修改

## Deferred Ideas

- batch.go 的 context 传播
- governance.go/audit_sink.go 的 context 传播
- auth.go/config.go/account.go 等非关键路径
- storage_backend_resolvers.go
- verify.go/server.go
