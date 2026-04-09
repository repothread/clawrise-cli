---
phase: 02-context
plan: 03
subsystem: plugin-install
tags: [context-propagation, http, plugin-install, ctx]
dependency_graph:
  requires: ["02-01"]
  provides: ["CTX-04"]
  affects: ["internal/plugin/install.go"]
tech_stack:
  added: []
  patterns: ["context-propagation-chain"]
key_files:
  created: []
  modified:
    - internal/plugin/install.go
decisions: []
metrics:
  duration_seconds: 294
  completed_date: "2026-04-09"
---

# Phase 2 Plan 03: install.go context 传播 Summary

install.go 插件安装流程的 context 传播改造，使所有 HTTP 下载请求和 registry source 解析调用支持 context 取消。

## Objective

修改 install.go 的插件安装流程，将 context 从公开 API 方法沿调用链传递到 HTTP 下载和 runtime.Resolve 调用，使插件下载和验证操作可被 context 取消。

## Tasks Completed

| Task | Name | Commit | Status |
|------|------|--------|--------|
| 1 | install.go 内部函数链添加 ctx 参数，替换 context.Background() 和 http.NewRequest | `2cc6f25` | Done |

## Changes Made

### internal/plugin/install.go

**函数签名修改（6 个内部函数）：**

- `downloadRemoteSource(ctx context.Context, ...)` — 新增 ctx 参数
- `resolveNPMSource(ctx context.Context, ...)` — 新增 ctx 参数
- `resolveRegistrySource(ctx context.Context, ...)` — 新增 ctx 参数
- `materializeSource(ctx context.Context, ...)` — 新增 ctx 参数
- `resolveInstallCandidate(ctx context.Context, ...)` — 新增 ctx 参数
- `checkUpgradeCandidate(ctx context.Context, ...)` — 新增 ctx 参数

**HTTP 请求改造（2 处）：**

- `downloadRemoteSource` 第 886 行：`http.NewRequest` -> `http.NewRequestWithContext(ctx, ...)`
- `resolveNPMSource` 第 935 行：`http.NewRequest` -> `http.NewRequestWithContext(ctx, ...)`

**context.Background() 替换（1 处）：**

- `resolveRegistrySource` 第 1029 行：`runtime.Resolve(context.Background(), ...)` -> `runtime.Resolve(ctx, ...)`

**公开 API fallback（3 处）：**

- `InstallWithOptions` — 内部创建 `ctx := context.Background()`
- `UpgradeInstalled` — 内部创建 `ctx := context.Background()`
- `InfoInstalledWithOptions` — 内部创建 `ctx := context.Background()`，`inspectRuntimeCapabilities(context.Background(), ...)` 改为 `inspectRuntimeCapabilities(ctx, ...)`

## Verification

- `grep -n 'http.NewRequest(' internal/plugin/install.go` — 返回空（全部替换为 NewRequestWithContext）
- `grep -n 'NewRequestWithContext' internal/plugin/install.go` — 2 处匹配
- `grep -n 'context.Background()' internal/plugin/install.go` — 仅出现在 3 个公开 API 的内部 fallback
- 所有 6 个内部函数签名包含 `ctx context.Context` 参数
- `go build ./internal/plugin/...` — 编译通过
- `go test ./internal/plugin/... -count=1` — 全部通过（29s）

## Deviations from Plan

### Auto-fixed Issues

None - plan executed exactly as written.

## Known Stubs

None.

## Threat Flags

No new threat surface introduced. Changes are purely context propagation — the same HTTP requests and registry calls are made, now with proper cancellation support.

## Self-Check: PASSED

- Commit `2cc6f25` found in git log
- SUMMARY.md exists at `.planning/phases/02-context/02-03-SUMMARY.md`
- `internal/plugin/install.go` exists and contains all expected changes
