---
phase: 01-audit-sink
plan: 01
subsystem: runtime
tags: [audit, plugin, lifecycle, bug-fix, governance]

# Dependency graph
requires: []
provides:
  - "pluginAuditSink.Emit() 不再在首次 emit 后关闭插件进程"
  - "runtimeGovernance.closeSinks() 统一关闭所有 pluginAuditSink"
  - "executor.Execute() defer closeSinks 确保插件进程清理"
  - "多次 emit 测试覆盖（TestExecutorPluginAuditSinkSupportsMultipleEmitsInOneCall）"
  - "close 生命周期测试覆盖（TestPluginAuditSinkCloseLifecycle）"
affects: [01-audit-sink]

# Tech tracking
tech-stack:
  added: []
  patterns: [lifecycle-managed-plugin-sinks, deferred-cleanup-on-execute]

key-files:
  created: []
  modified:
    - internal/runtime/audit_sink.go
    - internal/runtime/governance.go
    - internal/runtime/executor.go
    - internal/runtime/executor_test.go

key-decisions:
  - "将插件进程关闭职责从单次 Emit 提升到 Execute 方法级别（defer governance.closeSinks），确保 CLI 调用期间插件进程始终可用"

patterns-established:
  - "lifecycle-managed-plugin-sinks: 插件 audit sink 的 Close 由 runtimeGovernance 统一管理，而非每次 Emit 后自动关闭"

requirements-completed: [BUG-01, BUG-02]

# Metrics
duration: 10min
completed: 2026-04-09
---

# Phase 01 Plan 01: 修复审计 sink 插件进程生命周期 Summary

**移除 pluginAuditSink.Emit() 中的 defer Close()，由 runtimeGovernance.closeSinks() 在 Execute 返回时统一清理插件进程，修复连续审计写入失败**

## Performance

- **Duration:** 10 min
- **Started:** 2026-04-09T04:32:35Z
- **Completed:** 2026-04-09T04:42:10Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments

- pluginAuditSink.Emit() 不再在首次审计写入后终止插件子进程（BUG-01/BUG-02 修复）
- runtimeGovernance 新增 closeSinks() 方法，在 Execute 生命周期结束时统一关闭所有 plugin sink 进程
- executor.go Execute 方法通过 `defer governance.closeSinks()` 确保所有返回路径都正确清理
- 新增 TestExecutorPluginAuditSinkSupportsMultipleEmitsInOneCall 验证同一插件进程可接收两次 audit emit
- 新增 TestPluginAuditSinkCloseLifecycle 验证 closePluginAuditSinks 仅影响 plugin sink，不影响 stdout/webhook

## Task Commits

Each task was committed atomically:

1. **Task 1: 移除 pluginAuditSink.Emit() 中的 defer Close() 并添加 lifecycle Close 支持** - `d95f449` (fix)
2. **Task 2: 添加多次 emit 和 close 生命周期的测试用例** - `3fe06a3` (test)

## Files Created/Modified

- `internal/runtime/audit_sink.go` - 移除 Emit 中 defer Close()，新增 closePluginAuditSinks 辅助函数
- `internal/runtime/governance.go` - 新增 closeSinks() 方法
- `internal/runtime/executor.go` - 在 Execute 方法中添加 defer governance.closeSinks()
- `internal/runtime/executor_test.go` - 新增两个测试验证多次 emit 和 close 生命周期

## Decisions Made

- 将插件进程关闭职责从单次 Emit 提升到 Execute 方法级别（defer governance.closeSinks），确保 CLI 调用期间插件进程始终可用。CLI 是短生命周期工具，单次调用结束后即退出，进程自然释放，延长存活窗口的风险可接受（T-01-01 accept）。

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] closePluginAuditSinks 缺少 runtime nil 检查导致 panic**
- **Found during:** Task 2（TestPluginAuditSinkCloseLifecycle 测试运行时）
- **Issue:** closePluginAuditSinks 中 `pluginSink.runtime.Close()` 在 runtime 为 nil 时 panic
- **Fix:** 在条件中增加 `pluginSink.runtime != nil` 检查
- **Files modified:** internal/runtime/audit_sink.go
- **Verification:** TestPluginAuditSinkCloseLifecycle 通过
- **Committed in:** 3fe06a3（Task 2 commit）

---

**Total deviations:** 1 auto-fixed (1 bug)
**Impact on plan:** 必要的正确性修复，nil runtime 检查与 Emit 方法中的防护逻辑保持一致。无范围蔓延。

## Issues Encountered

- worktree 初始基点为旧版代码（缺少 audit_sink.go、sinks 字段等），需要从 develop HEAD checkout 相关文件后才能开始修改。这是分支管理的预期行为，不影响最终结果。

## User Setup Required

None - 无需外部配置。

## Next Phase Readiness

- 审计 sink 生命周期修复完成，插件进程在整个 CLI 调用期间保持可用
- 所有 runtime 包测试通过，无回归
- 可继续执行下一个 context propagation 相关计划

---
*Phase: 01-audit-sink*
*Completed: 2026-04-09*

## Self-Check: PASSED

- FOUND: internal/runtime/audit_sink.go
- FOUND: internal/runtime/governance.go
- FOUND: internal/runtime/executor.go
- FOUND: internal/runtime/executor_test.go
- FOUND: 01-01-SUMMARY.md
- FOUND: d95f449 (Task 1 commit)
- FOUND: 3fe06a3 (Task 2 commit)
