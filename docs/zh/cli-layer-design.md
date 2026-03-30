# Clawrise CLI Layer 架构设计

英文版见 [../en/cli-layer-design.md](../en/cli-layer-design.md)。

## 1. 产品定义

Clawrise 不是单个平台的 SDK 包装器，而是一层面向 AI Agent 的统一执行层。

目标不是“把 API 暴露成更多工具”，而是把第三方 SaaS API 统一成 AI 可以稳定调用的系统命令。

核心定位：

- `CLI = AI 的系统调用接口`
- `Clawrise = Agent-Native CLI Layer`
- `MCP` 更适合作为治理层，`CLI` 更适合作为执行层

## 2. 设计目标

Clawrise 必须满足：

- 低 token：默认不依赖 MCP schema 注入上下文
- 高稳定：命令格式固定，输出结构固定，减少 prompt 解释空间
- 高扩展：平台能力通过 provider plugin 增量扩展
- 高可控：支持认证、限流、重试、超时、审计、幂等
- AI 友好：输入简单、输出稳定、错误可判定、可直接串联

当前阶段明确不做：

- GUI
- 复杂工作流编排器
- 大规模手写动态子命令树
- MCP-first 的主执行路径

## 3. 核心原则

### 3.1 稳定优先

AI 需要的是可预测接口，不是“看起来更聪明”的命令系统。

### 3.2 运行时简单，生成时复杂

复杂度尽量放在生成阶段和适配层，不要堆在运行时。

也就是说：

- 运行时负责解析、认证、幂等、执行、输出
- 生成器负责把 OpenAPI 或其他 API 描述转成 operation 清单与映射代码

### 3.3 标准化优先于平台特性

各平台差异很大，但执行层必须统一：

- 成功结构统一
- 失败结构统一
- 元信息统一
- 可重试信息统一

### 3.4 自动生成只做 70%，保留 overlay

OpenAPI 或类似描述文件适合生成基础代码，但真实平台总会有文档之外的约束。

因此必须支持：

- `generated`
- `overlay`

### 3.5 只统一运行时，不统一资源 schema

Clawrise 统一的是执行语义，不是所有平台的业务资源字段。

- 运行时外壳应统一，例如 envelope、认证上下文、错误模型、超时、重试、幂等。
- 业务资源字段应尽量保留平台原生语义。
- 不应把飞书文档、Notion 页面、日历、多维表格、数据库记录等资源强行压成一套全局 schema。
- 如果未来需要跨平台工作流抽象，应放在可选的上层能力里，而不是改写核心 operation 契约。

## 4. 命令模型

主执行入口：

```bash
clawrise <operation> [flags]
```

示例：

```bash
clawrise feishu.calendar.event.create --input @event.json
clawrise notion.page.create --json '{"title":"项目记录"}'

clawrise platform use feishu
clawrise calendar.event.create --input @event.json
```

operation 路径格式：

```text
<platform>.<resource-path>.<action>
```

示例：

- `feishu.calendar.event.create`
- `feishu.docs.document.create`
- `notion.page.create`

如果已设置默认平台，也可以接受：

```text
<resource-path>.<action>
```

示例：

- `calendar.event.create`
- `docs.document.create`

保留管理命令：

- `clawrise platform ...`
- `clawrise account ...`
- `clawrise subject ...`
- `clawrise plugin ...`
- `clawrise auth ...`
- `clawrise config ...`
- `clawrise batch ...`
- `clawrise spec ...`
- `clawrise doctor`
- `clawrise version`
- `clawrise completion`

当前实现状态：

- Feishu / Notion 第一方能力已经通过外部 provider plugin binary 暴露
- `clawrise plugin list`
- `clawrise plugin install <source>`
- `clawrise plugin info <name> <version>`
- `clawrise plugin remove <name> <version>`
- `clawrise spec list [path]`
- `clawrise spec get <operation>`
- `clawrise spec status`
- `clawrise spec export`
- `clawrise completion <bash|zsh|fish>`

其中当前：

- `plugin list/install/info/remove` 已实现
- `spec list/get/status/export` 已实现
- `completion` 已实现

## 5. 输入输出规范

推荐输入方式：

- `--json '<json>'`
- `--input @file.json`
- `stdin`

通用运行时 flags：

- `--account`
- `--json`
- `--input`
- `--timeout`
- `--dry-run`
- `--idempotency-key`
- `--output`
- `--quiet`

所有命令统一输出标准 JSON 包络。

成功输出示例：

```json
{
  "ok": true,
  "operation": "notion.page.create",
  "request_id": "req_01HYTEST",
  "context": {
    "platform": "notion",
    "subject": "integration",
    "account": "notion_team_docs"
  },
  "data": {},
  "error": null,
  "meta": {
    "platform": "notion",
    "duration_ms": 184,
    "retry_count": 0,
    "dry_run": false
  },
  "idempotency": {
    "key": "idem_xxx",
    "status": "executed"
  }
}
```

失败输出示例：

```json
{
  "ok": false,
  "operation": "feishu.calendar.event.create",
  "request_id": "req_01HXXX",
  "data": null,
  "error": {
    "code": "RATE_LIMITED",
    "message": "飞书接口限流",
    "retryable": true,
    "upstream_code": "99991400",
    "http_status": 429
  },
  "meta": {
    "platform": "feishu",
    "duration_ms": 512,
    "retry_count": 2,
    "dry_run": false
  },
  "idempotency": {
    "key": "idem_xxx",
    "status": "rejected"
  }
}
```

## 6. 运行时核心模块

建议的运行时职责：

1. 命令解析器
2. 配置加载器
3. 认证管理器
4. provider runtime 管理层
5. operation 元信息解析器
6. 输入校验器
7. 幂等控制器
8. 执行器
9. 输出编码器
10. 审计记录器

执行链路：

```text
CLI 输入
  -> 解析 operation 和 flags
  -> 加载 config / account
  -> 解析 provider runtime 和 operation 元信息
  -> 读取 JSON 输入
  -> 本地校验与标准化
  -> 处理幂等
  -> 装配超时 / 重试 / 限流策略
  -> 调用 provider runtime
  -> 标准化输出
  -> 写审计日志
```

## 7. Provider Runtime 模型

core runtime 不应直接知道 provider 的实现细节。

推荐 Go 接口：

```go
type ProviderRuntime interface {
    Name() string
    Handshake(ctx context.Context) (HandshakeResult, error)
    ListOperations(ctx context.Context) ([]Definition, error)
    GetCatalog(ctx context.Context) ([]CatalogEntry, error)
    Execute(ctx context.Context, req ExecuteRequest) (ExecuteResult, error)
    Health(ctx context.Context) (HealthResult, error)
}
```

operation 元信息建议覆盖：

- operation 名称
- platform
- 是否写操作
- 是否幂等
- 是否支持 dry-run
- 默认超时
- 重试策略
- 限流 key
- 授权约束

## 8. 授权边界

完整授权模型见 [auth-model.md](auth-model.md)。

关键规则：

- `platform` 决定请求发往哪里
- `account` 决定请求以哪个身份执行
- 同一平台必须支持多个 account
- operation 必须声明允许的主体类型
- 运行时不能静默切换 account 或降级主体类型

推荐配置结构：

```yaml
defaults:
  platform: feishu
  subject: bot
  account: feishu_bot_ops

accounts:
  feishu_bot_ops:
    platform: feishu
    subject: bot
    auth:
      method: feishu.app_credentials
      public:
        app_id: cli_xxx
      secret_refs:
        app_secret: secret:feishu_bot_ops:app_secret

  notion_team_docs:
    platform: notion
    subject: integration
    auth:
      method: notion.internal_token
      public:
        notion_version: "2026-03-11"
      secret_refs:
        token: secret:notion_team_docs:token
```

## 9. OpenAPI 与生成策略

自动生成不应该追求一步到位生成最终产品。

建议输出：

- manifest 文件
- 请求 / 响应结构
- 基础映射代码

仍然必须保留人工 overlay，用于覆盖：

- 鉴权细节
- 幂等策略
- 错误归一化
- 文档外行为
- 字段语义补充

推荐原则：

`generated + overlay`

## 10. 资源可见性与共享模型

“调用成功” 不等于 “用户可见”。

这是 Clawrise 在办公类平台里必须单独建模的一层，尤其适用于飞书文档、表格、日历等资源。

### 10.1 核心原则

资源可见性由资源权限决定，而不是由调用是否成功决定。

也就是说：

- bot 可以成功创建资源
- 但目标用户未必自动可见
- bot 可以成功修改资源
- 但前提通常是 bot 本身已经被授权访问该资源

### 10.2 典型场景

场景 A：编辑已有共享资源

- 如果 bot 对该资源已有编辑权限
- 且用户本来就有查看权限
- 那么 bot 的修改对用户可见

场景 B：bot 新建资源

- 如果资源创建在应用自己的可控空间或应用创建的文件夹下
- 那么资源不会自动出现在用户的个人列表中
- 需要后续再做共享、授权，或创建在用户本来就可见的位置

### 10.3 对飞书文档与表格的影响

在飞书文档与表格场景下，应用身份通常只能操作：

- 自己创建的资源
- 或者明确授权给应用的资源

因此：

- `create` 类 operation 不能默认承诺 “用户马上能看到”
- `update` 类 operation 需要区分 “修改已有共享资源” 和 “修改应用自有资源”
- 后续必须补充资源共享或授权相关 operation

### 10.4 对 Clawrise 命令设计的要求

operation 的语义需要明确区分：

- `create`
- `update`
- `share`
- `grant`

不能把 “创建资源” 和 “让某人可见” 混成一个默认行为。

### 10.5 MVP 约束

MVP 阶段默认遵循以下约束：

- `create` 只保证创建成功
- 不默认保证目标用户可见
- 如果资源可见性对业务结果很关键，必须在文档和输出中明确提示

## 11. 幂等、重试与审计

写操作默认应走幂等保护。

建议幂等状态：

- `executed`
- `replayed`
- `in_progress`
- `rejected`

建议存储演进：

- MVP：本地 SQLite
- 服务化阶段：Redis 或 PostgreSQL

只有在以下场景才允许自动重试：

- 读操作
- 明确声明为幂等的写操作
- 上游错误被归类为临时性错误

审计日志建议包含：

- request_id
- operation
- profile
- 输入摘要
- 输出摘要
- duration
- retry_count
- 最终状态

## 12. MVP 范围

MVP 平台组合：

- `feishu`
- `notion`

推荐实施顺序：

1. 实现运行时内核
2. 定义 provider runtime 边界
3. 交付 Feishu / Notion 第一方 plugin
4. 实现插件发现与安装
5. 后续扩展 Google 和其他 provider

具体 operation 契约见 [mvp-operation-spec.md](mvp-operation-spec.md)。
