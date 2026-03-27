# Clawrise Roadmap

## 1. 文档目的

这份文档用于描述 Clawrise 在转向 plugin-first 架构之后的近期推进重点。

它区分三类事项：

- 已经完成的基础能力
- 接下来最应该做的事项
- 仍然可以明确后置的事项

## 2. 当前方向

Clawrise 当前的目标方向是：

- `clawrise` 作为 core runtime 和 CLI
- 平台能力通过外部 provider plugin 交付
- Feishu 和 Notion 作为第一方 provider plugin 交付
- 运行时外壳、`spec` 能力和治理模型保持统一
- 业务资源字段继续保持 provider-native

## 3. 已完成基础

当前仓库已经具备：

- 统一 runtime 和配置模型
- 一批真实可执行的 Feishu / Notion operation
- `clawrise spec list [path]`
- `clawrise spec get <operation>`
- `clawrise spec status`
- 结构化 operation catalog 与 runtime/catalog 对账
- 元数据完整性测试
- core 内的 provider runtime 抽象
- 基于 `stdio + JSON-RPC` 的外部进程 plugin runtime
- Feishu / Notion 第一方 plugin binary
- 插件管理命令：
  - `clawrise plugin list`
  - `clawrise plugin install <source>`
  - `clawrise plugin info <name> <version>`
  - `clawrise plugin remove <name> <version>`
- 当前安装源支持：
  - 本地目录
  - `file://`
  - `https://`
  - `npm://`

## 3.1 最新进度

截至 2026-03-27，近期 roadmap 的推进情况是：

- 4.1 接入友好度和第一方 Plugin UX：
  - 已落地 `plugin verify`
  - 已落地 `config init`
  - 已落地 `auth list|inspect|check`
  - `doctor` 已补充 plugin 发现、profile 检查、runtime 存储路径与 next steps
- 4.2 本地 Recipes / Playbooks：
  - 已落地 `docs/playbooks/index.yaml`
  - 已补充首批 Feishu / Notion 常用 playbooks
- 4.3 运行时治理：
  - 已落地写操作幂等状态本地持久化
  - 已落地基础审计记录
  - 已落地可配置自动重试
  - 已落地审计输入输出脱敏
- 4.4 `spec export`、completion 与文档生成：
  - 仍未完成

## 4. 近期 Must-have

这些是近期 must-have 事项，其中 4.1 到 4.3 已完成第一轮交付，4.4 仍待实现。

### 4.1 接入友好度和第一方 Plugin UX

状态：

- 已完成第一轮交付

原因：

- 现在架构已经转向 plugin-first，安装和首次使用体验比之前更关键
- 当前实现已经可用，但对非开发者仍然偏手工

交付物：

- 更清晰的 core + plugin quickstart
- 第一方 plugin 的官方打包约定
- 更强的 `doctor`
- `plugin verify` 或等价的 checksum / trust 能力
- 最小可用的 `auth` 辅助命令
- `config init` 或等价初始化引导

完成标志：

- 新用户可以按短路径安装一个官方 plugin 并完成一次真实调用
- 常见接入失败可以不依赖阅读源码进行定位

### 4.2 本地 Recipes / Playbooks

状态：

- 已完成第一批交付

原因：

- 现在已经有能力发现，但还缺从 capability 到 task 的桥梁
- 这同时服务人和 agent

交付物：

- `docs/recipes` 或 `docs/playbooks`
- 可搜索索引，例如 `index.yaml`
- 可复用的任务 recipes，例如：
  - 更新飞书文档
  - 更新飞书多维表格记录
  - 创建或更新飞书日历事件
  - 更新 Notion 页面内容
  - 查询 Notion data source

完成标志：

- 常见任务可以通过本地搜索快速找到
- recipe 中的命令模板和输入样例可复用、可验证

### 4.3 运行时治理

状态：

- 已完成第一轮交付

原因：

- 写操作在广泛使用前仍需更强的运行时保障

交付物：

- 幂等状态持久化
- 基础审计记录
- 可配置重试策略
- 更明确的敏感信息脱敏规则

完成标志：

- 写操作可查询持久化幂等状态
- 审计输出不泄露敏感信息
- 重试行为进入标准元信息

### 4.4 `spec export`、Completion 与文档生成

状态：

- 当前仍未完成

原因：

- `spec` 发现能力已经到位，但机器可读导出和消费层还没补齐

交付物：

- `clawrise spec export`
- 复用同一套 provider 元数据的 `completion`
- 基于 registry/catalog 元数据逐步生成 operation 文档

完成标志：

- 机器消费者可以获取完整结构化导出
- completion 不再维护另一套独立命令树
- operation 文档逐步从结构化元数据生成，而不是手工漂移

## 5. Should-have

这些事项有价值，但应排在 Must-have 之后。

### 5.1 官方 `clawrise-operator` skill

原因：

- 有助于 agent 更稳定地使用 Clawrise
- 应建立在 `spec`、catalog、recipes 和 plugin-aware onboarding 之上

### 5.2 开发者向 `clawrise-builder` skill

原因：

- 适合帮助 AI 参与 provider 扩展和 adapter/plugin 开发
- 但优先级低于 operator-facing 材料

### 5.3 Plugin Hardening 与分发运营

原因：

- plugin 架构已经存在，但 release、trust 和升级策略还没有完全产品化

建议范围：

- plugin release manifest
- checksum policy
- signature policy
- upgrade strategy
- 官方分发渠道

## 6. Can Wait

这些事项仍然可以明确后置：

- 公共 plugin marketplace
- 不可信 plugin 的沙箱隔离
- REPL-first 交互壳
- 完整 JSON Schema 框架
- 跨平台 workflow engine

## 7. 推荐顺序

1. 接入友好度和第一方 plugin UX
2. 本地 recipes / playbooks
3. 运行时治理
4. `spec export`、completion 与文档生成
5. 官方 `clawrise-operator` skill
6. plugin hardening 与分发运营
7. 开发者向 `clawrise-builder` skill

## 8. 风险与注意事项

### 8.1 不要把平台重新硬编码回 Core

plugin-first 应保持为默认架构方向。

### 8.2 不要重新制造多套元数据事实源

`spec`、catalog、docs、completion、recipes 应继续收敛到同一层结构化元数据。

### 8.3 不要在缺少信任策略时把远程安装当成“完成”

`https://` 和 `npm://` 已经可用，但 release 与 trust policy 仍需产品化。

### 8.4 不要把“安装成功”当成“接入完成”

plugin 安装只是第一步。认证、profile 选择、样例输入和诊断路径仍然决定实际可用性。

## 9. 完成标志

可以认为当前近期 roadmap 完成的标志是：

- 用户可以方便地安装并校验官方第一方 plugin
- 常见任务具备本地可搜索 playbooks
- 写操作具备更强的幂等与审计保证
- `spec export` 与 completion 复用同一套 provider 元数据
- operator-facing 材料开始复用结构化元数据，而不是继续手工维护命令知识
