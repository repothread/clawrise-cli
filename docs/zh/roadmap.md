# Clawrise Roadmap

## 1. 文档目的

这份文档用于收敛 Clawrise 近期的推进重点，并按优先级区分：

- 哪些是近期必做
- 哪些是应该做但不必最先做
- 哪些可以明确后置

`spec` 子系统的详细设计见 [spec-design.md](spec-design.md)。

## 2. 判断原则

一个事项是否进入近期 roadmap，主要看三个问题：

- 它是不是项目能否被真实使用的关键阻塞
- 它是不是其他能力的前置依赖
- 它会不会引入第二套知识源，导致后续漂移

## 3. 当前状态

当前仓库已经具备：

- 统一 runtime 和配置模型
- Feishu / Notion 的一批真实 operation
- `clawrise spec list [path]`
- `clawrise spec get <operation>`

但距离“让人和 agent 都能稳定使用”的状态还有明显差距。

当前最主要的问题有：

- 自描述能力未闭环：`spec status`、`spec export`、`completion` 仍未实现
- 文档与实现仍可能漂移：尚无 catalog 与状态对账
- 接入友好度不足：首次安装、配置、认证、样例调用、故障排查路径仍偏长
- AI 使用材料不足：尚无官方 skill，也没有明确的 agent 操作指南
- 常见任务缺少本地可检索 playbooks / recipes
- 幂等、审计、重试等运行时治理能力仍未闭环

## 4. Must-have

这些事项应进入近期主线，而且优先级最高。

### 4.1 `spec status + catalog`

原因：

- 这是后续文档、recipes、skills、completion 的事实源基础
- 没有它，仓库会继续在“代码 / 文档 / AI 使用材料”之间漂移

交付物：

- `clawrise spec status`
- 结构化 operation catalog
- registry 与 catalog 的 diff 逻辑
- 元数据完整性测试

完成标志：

- 能区分 implemented / stubbed / declared-but-missing
- 新增 operation 时如果漏补 metadata 或 catalog，会被测试发现

### 4.2 接入友好度优化

原因：

- 这直接决定项目是否能从“可运行”进入“可顺利用”

交付物：

- `config init` 或等价的初始化引导
- 更短的 quickstart
- 可直接复用的样例配置 / 样例输入
- 更强的 `doctor`
- 最小可用的 `auth` 辅助命令，例如检查当前认证状态

完成标志：

- 新用户可以按最短路径完成一次真实调用
- 常见错误可以通过 `doctor` 或接入文档快速定位

### 4.3 本地 recipes / playbooks

原因：

- 这是“能力”到“任务”的桥梁
- 同时服务人和 AI
- 比官方 skill 更轻，也更适合先沉淀

建议形态：

- `docs/recipes` 或 `docs/playbooks`
- 一个可搜索的小型索引，例如 `index.yaml`
- 每篇 recipe 聚焦单个任务：
  - 更新飞书指定文档
  - 更新飞书指定表格或记录
  - 创建飞书日历事件
  - 更新 Notion 页面内容
  - 查询 Notion data source

完成标志：

- 常见任务可以通过本地检索快速找到
- recipe 中的命令模板和输入样例是可复用、可验证的

### 4.4 运行时治理基本闭环

原因：

- 如果写操作仍缺幂等 / 审计 / 重试基础，项目在真实使用中风险仍然偏高

交付物：

- 幂等状态持久化
- 基础审计记录
- 可配置重试策略
- 更明确的敏感信息脱敏规则

完成标志：

- 写操作可查询幂等状态
- 审计记录不泄露明文凭证
- 重试次数和最终状态进入标准元信息

## 5. Should-have

这些事项有价值，但应建立在 Must-have 基础上推进。

### 5.1 官方 `clawrise-operator` skill

原因：

- 有助于 agent 更快学会如何稳定使用 Clawrise
- 但它应该建立在 `spec`、catalog、recipes 之上

建议内容：

- 如何选择 `platform / profile / subject`
- 如何优先用 `spec` 探索能力
- 如何查找对应 recipe
- 如何处理常见失败

### 5.2 `completion` / 文档生成 / `spec export`

原因：

- 很重要，但更像元数据体系成熟后的放大器

建议交付物：

- `completion` 复用 `spec`
- operation 文档逐步由结构化元数据生成
- `spec export` 提供机器可读导出

### 5.3 开发者向 `clawrise-builder` skill

原因：

- 适合帮助 AI 参与平台扩展和 adapter 开发
- 但优先级低于 `operator` skill

## 6. Can wait

这些事项可以明确后置，不应挤占近期主线。

- 动态插件系统
- REPL-first 交互产品
- 完整 JSON Schema 体系
- 复杂跨平台 workflow engine
- 过早做外部分发生态

## 7. 推荐顺序

1. `spec status + catalog`
2. 接入友好度优化
3. 本地 recipes / playbooks
4. 运行时治理基本闭环
5. 官方 `clawrise-operator` skill
6. `completion` / 文档生成 / `spec export`
7. 开发者向 `clawrise-builder` skill

## 8. 风险与注意事项

### 8.1 不要把 `spec` 做成文档副本

`spec` 的首要职责是提供结构化事实源，而不是复制 Markdown 描述。

### 8.2 不要默认平铺全量 operation

随着平台和 operation 增长，`spec list` 的默认输出必须按层级浏览，否则很快会失控。

### 8.3 不要把 catalog 和 runtime 混为一层

如果只保留一份“理想中的 operation 列表”，就无法准确表达当前二进制真正可执行的能力。

### 8.4 不要把 skill 和 recipe 写成另一套事实源

官方 skill、本地 recipes、接入文档都应尽量复用 `spec`、catalog、结构化样例，而不是重新维护另一套命令知识。

## 9. 完成标志

可以认为近期 roadmap 完成的标志是：

- 当前二进制支持的 operation 可以通过 `spec` 逐层发现
- `spec status` 可以准确指出实现漂移
- 新用户可以更顺滑地完成首次接入
- 常见任务拥有本地可检索、可复用的 recipes / playbooks
- 写操作具备更真实的幂等与审计基础
- 官方 skill 与本地 playbooks 开始复用结构化元数据，而不是继续手工维护
