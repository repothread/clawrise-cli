# Clawrise `spec` 子系统设计

## 1. 文档目的

这份文档定义 `clawrise spec` 的命令面、数据模型、状态模型与实现落点。

它要解决的问题不是“如何再写一份文档”，而是“如何让 Clawrise 自己成为当前能力的结构化事实源”。

整体推进节奏见 [roadmap.md](roadmap.md)。

## 2. 当前实现状态

当前已经实现：

- `clawrise spec list [path]`
- `clawrise spec get <operation>`
- 基于 registry 的层级浏览
- 基于 operation 元数据的详情查询

当前尚未实现：

- `clawrise spec status`
- `clawrise spec export`
- 基于 catalog 的漂移对账

## 3. 设计目标

`spec` 子系统需要同时满足以下目标：

- 让人和 agent 都能通过 CLI 自己发现当前支持的 operation
- 默认输出可控，避免 operation 数量增长后列表爆炸
- 让 registry、文档、completion、测试共享一份结构化元数据
- 明确区分“当前运行时真实能力”和“项目规划中的声明能力”
- 保持现有建模边界：统一运行时，不统一 provider-native 业务字段

## 4. 非目标

首版 `spec` 明确不做：

- 不做交互式 UI
- 不做 REPL
- 不解析 Markdown 反向生成结构化 spec
- 不引入完整 JSON Schema 体系
- 不把 `spec` 变成工作流或执行入口

## 5. 命令面设计

当前与规划中的 `spec` 子命令如下：

- `clawrise spec list [path]`
- `clawrise spec get <operation>`
- `clawrise spec status`
- `clawrise spec export`

其中：

- `list` 用于按层级浏览目录
- `get` 用于查看单个 operation 详情
- `status` 用于查看 runtime 与 catalog 的差异
- `export` 用于显式导出全量结构化数据，给脚本或 agent 缓存使用

### 5.1 `spec list`

`spec list` 的默认语义必须是“浏览目录”，不是“平铺所有 operation”。

原因是 Clawrise 的 operation 命名天然具备层级结构：

```text
<platform>.<resource-path>.<action>
```

推荐行为：

- `clawrise spec list`
  - 返回平台层，如 `feishu`、`notion`
- `clawrise spec list feishu`
  - 返回 `calendar`、`docs`、`wiki`、`contact`
- `clawrise spec list feishu.docs`
  - 返回 `document`、`block`
- `clawrise spec list feishu.docs.document`
  - 返回叶子 operation，如 `create`、`get`、`list_blocks`

推荐参数：

- `--depth`
- `--flat`
- `--implemented`
- `--subject`
- `--mutating`
- `--limit`
- `--cursor`

默认规则：

- 默认 `depth=1`
- 默认不递归
- 默认只返回摘要，不返回详细输入定义
- 默认不平铺全量 operation

### 5.2 `spec get`

`spec get` 只接收叶子 operation，例如：

```bash
clawrise spec get feishu.calendar.event.create
```

如果用户传入的是 group path，例如 `feishu.docs`，应返回明确错误，引导用户改用 `spec list`。

### 5.3 `spec status`

`spec status` 的职责不是浏览，而是治理。

它应回答这些问题：

- 当前 runtime 共注册了多少 operation
- 其中多少已实现，多少只是占位
- catalog 声明了多少 operation
- 哪些 operation 已声明但 runtime 中不存在
- 哪些 operation runtime 中存在但 catalog 未收录

### 5.4 `spec export`

`spec export` 用于显式导出完整结构化 spec，不应与 `list` 混用语义。

这能避免把 `list` 设计成既要做人类可读目录浏览，又要做机器可读全量导出的“双重接口”。

## 6. 层级模型

`spec list` 应返回树节点，而不是直接返回 operation 全表。

推荐节点类型：

- `root`
- `platform`
- `group`
- `operation`

这里不强制把中间层写死成 `resource`，因为不同平台未来的路径层级不一定完全对齐。

### 6.1 `list` 输出示例

根层：

```json
{
  "ok": true,
  "data": {
    "path": "",
    "node_type": "root",
    "depth": 1,
    "items": [
      {
        "name": "feishu",
        "full_path": "feishu",
        "node_type": "platform",
        "child_count": 4,
        "operation_count": 16
      },
      {
        "name": "notion",
        "full_path": "notion",
        "node_type": "platform",
        "child_count": 5,
        "operation_count": 12
      }
    ]
  }
}
```

中间层：

```json
{
  "ok": true,
  "data": {
    "path": "feishu.docs",
    "node_type": "group",
    "depth": 1,
    "items": [
      {
        "name": "document",
        "full_path": "feishu.docs.document",
        "node_type": "group",
        "child_count": 5,
        "operation_count": 5
      },
      {
        "name": "block",
        "full_path": "feishu.docs.block",
        "node_type": "group",
        "child_count": 5,
        "operation_count": 5
      }
    ]
  }
}
```

叶子层：

```json
{
  "ok": true,
  "data": {
    "path": "feishu.docs.document",
    "node_type": "group",
    "depth": 1,
    "items": [
      {
        "name": "create",
        "full_path": "feishu.docs.document.create",
        "node_type": "operation",
        "implemented": false,
        "mutating": true,
        "summary": "Create an empty Feishu document."
      },
      {
        "name": "get",
        "full_path": "feishu.docs.document.get",
        "node_type": "operation",
        "implemented": true,
        "mutating": false,
        "summary": "Get Feishu document metadata."
      }
    ]
  }
}
```

## 7. 详情模型

`spec get` 返回单个 operation 的完整契约信息。

建议字段：

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

### 7.1 `get` 输出示例

```json
{
  "ok": true,
  "data": {
    "operation": "feishu.calendar.event.create",
    "platform": "feishu",
    "resource_path": "calendar.event",
    "action": "create",
    "summary": "Create a Feishu calendar event.",
    "allowed_subjects": ["bot"],
    "mutating": true,
    "implemented": true,
    "dry_run_supported": true,
    "default_timeout_ms": 10000,
    "idempotency": {
      "required": true,
      "auto_generated": true
    },
    "input": {
      "required": ["calendar_id", "summary", "start_at", "end_at"],
      "optional": ["description", "location", "reminders", "timezone"],
      "notes": [
        "Time fields use RFC3339.",
        "`attendees` is not supported in the current implementation."
      ],
      "sample": {
        "calendar_id": "cal_demo",
        "summary": "Weekly sync",
        "start_at": "2026-03-30T10:00:00+08:00",
        "end_at": "2026-03-30T11:00:00+08:00"
      }
    },
    "examples": [
      {
        "title": "Create an event",
        "command": "clawrise feishu.calendar.event.create --json '{...}'"
      }
    ],
    "runtime_status": "registered_and_implemented"
  }
}
```

## 8. 事实源模型

`spec` 不应只有一份事实源，而应拆成两层：

- `Runtime Registry`
- `Catalog`

二者职责不同。

### 7.1 Runtime Registry

Runtime Registry 表示当前二进制真实可见的 operation。

它的特点是：

- 由 adapter 注册产生
- 表示“现在能发现到什么”
- 不一定代表“现在都已经真正实现”

### 7.2 Catalog

Catalog 表示项目当前认领的 operation 集。

它的特点是：

- 用结构化方式声明“项目认为应该支持哪些 operation”
- 可以包含已实现、占位、规划中
- 用于驱动文档、状态治理与测试约束

### 7.3 为什么需要两层

如果只有 runtime registry，就无法表达“文档已认领但尚未实现”的 operation。

如果只有 catalog，就无法知道当前二进制真实暴露了哪些能力。

因此 `spec status` 必须基于两层做差异分析。

## 9. 状态模型

建议定义以下 runtime 状态：

- `registered_and_implemented`
- `registered_but_stubbed`
- `runtime_missing`

建议定义以下 catalog 状态：

- `declared`
- `catalog_missing`

`status` 重点需要识别的问题项：

- `registered_but_stubbed`
- `catalog_declared_but_runtime_missing`
- `runtime_present_but_catalog_missing`

## 10. 元数据模型

当前 adapter registry 的定义只覆盖执行相关元数据，无法支撑 `spec` 层；见 `internal/adapter/registry.go`。

建议把 operation 定义扩展为“执行元数据 + 描述元数据”。

建议结构：

```go
type Definition struct {
    Operation       string
    Platform        string
    Mutating        bool
    DefaultTimeout  time.Duration
    AllowedSubjects []string
    Handler         HandlerFunc
    Spec            OperationSpec
}

type OperationSpec struct {
    Summary          string
    Description      string
    DryRunSupported  bool
    Input            InputSpec
    Idempotency      IdempotencySpec
    Examples         []ExampleSpec
    Stability        string
    UnsupportedNotes []string
}

type InputSpec struct {
    Required []string
    Optional []string
    Notes    []string
    Sample   map[string]any
}

type IdempotencySpec struct {
    Required      bool
    AutoGenerated bool
}

type ExampleSpec struct {
    Title   string
    Command string
}
```

首版不建议直接上完整 JSON Schema，原因有三点：

- 当前实现里很多校验逻辑还散落在 adapter 中
- 首版最重要的是先建立统一事实源
- 过早引入完整 schema 会显著扩大实现范围

## 11. 当前实现落点

当前已经存在：

- `internal/spec/types.go`
- `internal/spec/service.go`
- `internal/cli/spec.go`
- `internal/adapter/feishu/calendar_spec.go`
- `internal/adapter/feishu/docx_spec.go`
- `internal/adapter/feishu/wiki_spec.go`
- `internal/adapter/notion/page_spec.go`
- `internal/adapter/notion/block_spec.go`

已完成的关键修改：

- `internal/adapter/registry.go`
  - 扩展 `Definition`
  - 增加只读遍历接口
- 各平台 `register.go`
  - 为每个 operation 补 `Spec`
- `internal/cli/root.go`
  - 把 `spec` 从占位命令替换为真实子命令

下一阶段计划新增：

- `internal/spec/status.go`
- `internal/spec/catalog/feishu.go`
- `internal/spec/catalog/notion.go`

## 12. 与其他子系统的关系

### 12.1 与 completion

`completion` 不应维护另一套独立命令树，而应直接消费 `spec` 生成的层级节点。

也就是说：

- 平台补全来自 root 层
- group 补全来自 path 节点
- operation 补全来自叶子节点

### 12.2 与文档

中长期应让中英文 operation 文档逐步从 catalog 和 registry 元数据生成，而不是继续完全手工同步。

但首版不建议反向解析现有 Markdown。

### 12.3 与测试

测试需要把 `spec` 当成契约层，而不是可选辅助功能。

至少应覆盖：

- `list` 路径行为
- `get` 对叶子与非叶子的处理
- `status` 的 diff 逻辑
- operation 元数据完整性

## 13. 分阶段实施建议

### 13.1 第一阶段

- 扩展 registry 元数据
- 实现 `spec list`
- 实现 `spec get`
- 为当前已注册 operation 补齐基础 `Spec`

状态：

- 已完成

### 13.2 第二阶段

- 引入 catalog
- 实现 `spec status`
- 加入 metadata completeness test

### 13.3 第三阶段

- 实现 `spec export`
- 让 `completion` 复用 `spec`
- 开始把 operation 文档迁移为结构化生成优先

## 14. 验收标准

当前 `M1` 已达到的验收结果：

- `clawrise spec list` 可以逐层浏览当前 operation 目录
- `clawrise spec get <operation>` 可以返回完整详情
- `spec` 输出不依赖真实凭证与外部网络

下一阶段需要补齐：

- `clawrise spec status`
- catalog 对账
- metadata completeness test
