# Clawrise `spec` 子系统设计

英文版见 [../en/spec-design.md](../en/spec-design.md)。

## 1. 文档目的

`spec` 子系统的目标，是让 Clawrise 自己成为当前能力的结构化事实源。

它不是为了再写一份 Markdown 文档副本，而是为了回答这些问题：

- 当前有哪些 operation 可被发现
- 这些 operation 带有什么元数据
- 当前 runtime 真实可执行什么
- runtime 与 catalog 的差异是什么

## 2. 当前实现状态

当前已经实现：

- `clawrise spec list [path]`
- `clawrise spec get <operation>`
- `clawrise spec status`
- `clawrise spec export [path] [--format json|markdown]`
- `clawrise completion <bash|zsh|fish>`
- 基于当前 runtime registry 的层级发现
- 基于 catalog 的 runtime 漂移对账
- 基于同一元数据层的 Markdown 文档导出
- 元数据完整性测试

## 3. 当前运行时模型

Clawrise 现在已经转向 plugin-first 的 provider 架构。

这意味着 `spec` 当前读取的是两层聚合视图：

- 来自 provider runtime 的 runtime operation registry
- 来自 provider runtime 的结构化 catalog

在当前仓库中：

- Feishu / Notion 第一方 plugin 通过 provider runtime 接口暴露 operation
- core 聚合这些 provider runtime，形成统一 registry 视图
- `spec` 构建在这个聚合视图之上

## 4. 设计目标

`spec` 子系统当前应满足：

- 让人和 agent 都能直接通过 CLI 发现 operation
- 默认输出可控，避免 operation 增长后失控
- 让 runtime、docs、completion、tests 复用同一层元数据
- 明确区分 runtime 事实与 catalog 声明
- 保持建模边界：统一 runtime，不统一 provider-native 业务字段

## 5. 命令面

当前命令面：

- `clawrise spec list [path]`
- `clawrise spec get <operation>`
- `clawrise spec status`
- `clawrise spec export`

当前状态：

- `list` 已实现
- `get` 已实现
- `status` 已实现
- `export` 已实现

## 6. 层级浏览模型

`spec list` 仍然应是层级浏览器，而不是平铺导出接口。

operation 命名继续遵循：

```text
<platform>.<resource-path>.<action>
```

当前节点类型：

- `root`
- `platform`
- `group`
- `operation`

默认行为仍应保持：

- 层级浏览
- 输出可控
- 先摘要后细节
- 不承担机器全量导出职责

## 7. 详情模型

`spec get` 返回单个 operation 的详情记录。

当前字段包括：

- `operation`
- `platform`
- `resource_path`
- `action`
- `summary`
- `description`
- `allowed_subjects`
- `mutating`
- `implemented`
- `dry_run_supported`
- `default_timeout_ms`
- `idempotency`
- `input`
- `examples`
- `runtime_status`

当前详情页仍然以 runtime 元数据为主；runtime/catalog 漂移由 `spec status` 负责显式报告。

## 8. 事实源模型

当前模型明确拆成两层：

- `Runtime Registry`
- `Catalog`

### Runtime Registry

Runtime Registry 表示当前二进制通过已加载 provider runtime 能发现到的 operation 集。

它回答：

- 现在暴露了哪些 operation
- 这些 operation 现在带了哪些元数据
- 哪些 operation 当前真实已实现

### Catalog

Catalog 表示项目当前认领的结构化 operation 声明集。

它回答：

- 哪些 operation 被声明了
- 哪些 runtime operation 缺少 catalog 覆盖
- 哪些 catalog operation 在 runtime 中缺失

## 9. 状态模型

`spec status` 是治理接口，不是浏览接口。

它应报告：

- 当前注册 operation 总数
- implemented / stubbed 数量
- catalog 声明总数
- runtime 中存在但 catalog 未收录的 operation
- catalog 已声明但 runtime 缺失的 operation

当前状态标签：

- runtime:
  - `registered_and_implemented`
  - `registered_but_stubbed`
  - `runtime_missing`
- catalog:
  - `declared`
  - `catalog_missing`

## 10. 元数据模型

当前 operation 元数据仍然保持轻量。

它包括：

- 执行元数据：
  - operation name
  - platform
  - mutating flag
  - default timeout
  - allowed subjects
- 发现元数据：
  - summary
  - description
  - dry-run support
  - input fields
  - examples
  - idempotency behavior

当前仍然不打算把它直接扩成完整 JSON Schema 体系。

## 11. 当前文件落点

当前实现文件包括：

- `internal/spec/types.go`
- `internal/spec/service.go`
- `internal/spec/status.go`
- `internal/spec/catalog/*`
- `internal/cli/spec.go`

当前 provider 聚合层位于：

- `internal/plugin/runtime.go`
- `internal/plugin/process.go`
- `internal/plugin/registry_runtime.go`

当前最重要的架构事实是：

- `spec` 已不再假设 provider 永远硬编码在 core 中
- 它消费的是 provider runtime 聚合后的 registry 和 catalog 视图

## 12. 与其他子系统的关系

### 12.1 与 Completion

`completion` 应直接消费 `spec` 所在的同一层 provider 元数据，而不是维护另一套命令树。

### 12.2 与文档

operation 文档应逐步转向从 provider 元数据和 catalog 生成，而不是继续完全手工同步。

### 12.3 与测试

`spec` 应继续被当成契约层。

测试至少应覆盖：

- 层级浏览行为
- operation 详情行为
- runtime/catalog 漂移报告
- 元数据完整性

## 13. 下一步

在当前实现基础上，下一步主要是：

- 继续扩充 `spec export` 的消费场景
- 让更多文档直接复用导出元数据
- 保持 completion 与文档生成继续收敛到同一层事实源

## 14. 完成标志

可以认为近期 `spec` 工作完成的标志是：

- runtime 能力可通过 `list/get` 发现
- runtime/catalog 漂移可通过 `status` 查看
- `export` 可供机器消费者使用
- completion 和文档不再依赖另一套手工维护的命令知识
