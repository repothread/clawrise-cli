---
phase: 2
slug: context
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-09
---

# Phase 2 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing (stdlib) |
| **Config file** | none |
| **Quick run command** | `go test ./internal/cli/... ./internal/adapter/... ./internal/secretstore/... ./internal/plugin/... ./internal/runtime/... -count=1` |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/cli/... ./internal/adapter/... ./internal/secretstore/... ./internal/plugin/... ./internal/runtime/... -count=1`
- **After every plan wave:** Run `go test ./...`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 02-01-01 | 01 | 1 | CTX-01 | — | root.go 创建 signal.NotifyContext 并传递 ctx | unit | `go test ./internal/cli/... -run TestRun -count=1` | ✅ | ⬜ pending |
| 02-01-02 | 01 | 1 | CTX-01 | — | executor.ExecuteContext 接收调用方 ctx | unit | `go test ./internal/runtime/... -run TestExecutor -count=1` | ✅ | ⬜ pending |
| 02-02-01 | 02 | 1 | CTX-02 | — | runtime_options.go nil 防护保留，上游 ctx 正确传入 | unit | `go test ./internal/adapter/... -count=1` | ✅ | ⬜ pending |
| 02-03-01 | 03 | 2 | CTX-03 | — | pluginSecretStore 传递 ctx 给 ProcessSecretStore | unit | `go test ./internal/secretstore/... -count=1` | ✅ | ⬜ pending |
| 02-04-01 | 04 | 2 | CTX-04 | — | install.go HTTP 下载使用 NewRequestWithContext | unit | `go test ./internal/plugin/... -run TestInstall -count=1` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| SIGINT 取消正在执行的操作 | CTX-01 | 需要真实终端信号 | 运行 `go run ./cmd/clawrise feishu.calendar.event.create --json '...'`，按 Ctrl+C，验证操作终止 |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
