# Roadmap: Clawrise CLI — 发布前关键修复

## Overview

本次里程碑修复两个发布阻塞项：审计 sink 进程生命周期 bug 和关键路径上的 context 传播缺失。修复分三个阶段：先修复审计 sink（独立、可控），再修复 context 传播（4 个文件的同类型改造），最后全量验证确保代码库处于可合入 main 的状态。

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [ ] **Phase 1: Audit Sink 修复** - 修复插件审计 sink 进程在首次 emit 后被错误关闭的 bug
- [ ] **Phase 2: Context 传播** - 将关键路径上的 context.Background() 替换为调用方 context
- [ ] **Phase 3: 发布验证** - 全量测试验证，确保工作区可安全合入 main

## Phase Details

### Phase 1: Audit Sink 修复
**Goal**: 审计 sink 在整个 CLI 调用生命周期中保持可用，支持多次审计事件写入
**Depends on**: Nothing (first phase)
**Requirements**: BUG-01, BUG-02
**Success Criteria** (what must be TRUE):
  1. 单次 CLI 调用中连续触发两次以上审计事件，所有事件均成功写入且无错误日志
  2. 审计 sink 插件进程仅在 CLI 主进程退出时关闭，而非首次 emit 后关闭
  3. 现有审计相关测试全部通过，无回归
**Plans**: 1 plan

Plans:
- [ ] 01-01-PLAN.md — 移除 Emit 中 defer Close，新增 runtimeGovernance.closeSinks 统一清理，添加多次 emit 测试

### Phase 2: Context 传播
**Goal**: 所有关键生产路径尊重调用方 context，支持超时取消和 SIGINT 中断
**Depends on**: Phase 1
**Requirements**: CTX-01, CTX-02, CTX-03, CTX-04
**Success Criteria** (what must be TRUE):
  1. CLI 操作执行路径（root.go dispatch）的 HTTP 调用接受并传递调用方 context，超时或取消时操作能正确终止
  2. adapter 层 HTTP 调用（runtime_options.go）传递调用方 context，不再使用 context.Background()
  3. secret store 的加解密操作传递调用方 context，长时间加密操作可被取消
  4. 插件安装的下载/验证操作传递调用方 context，下载中断时能正确清理
**Plans**: TBD

Plans:
- [ ] 02-01: 修复 root.go 操作执行路径的 context 传播
- [ ] 02-02: 修复 runtime_options.go HTTP 调用的 context 传播
- [ ] 02-03: 修复 store.go 加解密操作的 context 传播
- [ ] 02-04: 修复 install.go 下载/验证操作的 context 传播

### Phase 3: 发布验证
**Goal**: 代码库处于可安全合入 main 的状态，所有测试通过且无新增警告
**Depends on**: Phase 2
**Requirements**: VAL-01, VAL-02, VAL-03, VAL-04
**Success Criteria** (what must be TRUE):
  1. `go test ./...` 全部通过，无失败用例
  2. `go vet ./...` 零警告
  3. `node --test packaging/npm/root/lib/*.test.js` 全部通过
  4. `git status` 显示工作区干净（或仅有 .planning/ 目录变更）
**Plans**: TBD

Plans:
- [ ] 03-01: 运行全部测试套件并修复任何失败项
- [ ] 03-02: 运行 go vet 和 npm 测试，确认工作区状态

## Progress

**Execution Order:**
Phases execute in numeric order: 1 → 2 → 3

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Audit Sink 修复 | 0/1 | Planning complete | - |
| 2. Context 传播 | 0/4 | Not started | - |
| 3. 发布验证 | 0/2 | Not started | - |
