# Clawrise Roadmap

## 1. 文档目的

这份文档用于收敛 Clawrise 近期的产品推进重点，明确：

- 当前最需要补齐的能力是什么
- 各阶段的目标边界是什么
- 每个阶段的交付物和验收结果是什么

这份文档不展开具体命令设计与数据结构细节。

`spec` 子系统的详细设计见 [spec-design.md](spec-design.md)。

## 2. 当前问题

当前仓库已经具备统一运行时、配置模型、基础 adapter 注册与部分真实 operation 实现，但距离“可大规模被 agent 稳定消费”的状态还有明显差距。

当前最核心的问题有四类：

- 自描述能力仍未闭环：`spec list/get` 已可用，但 `status`、`export`、`completion`、`auth`、`batch`、`config` 仍未补齐，agent 还无法完整发现和校验当前能力。
- 文档与实现存在漂移：设计文档里声明了部分 operation 和能力，但当前二进制中并不一定已实现或已注册。
- 运行时治理能力未闭环：幂等、重试、审计、限流仍主要停留在设计层，运行时只实现了部分框架。
- 扩展机制仍偏手工：新增平台和 operation 还需要修改核心注册逻辑，尚未形成稳定的元数据驱动流程。

## 3. 近期目标

近期目标不是把 Clawrise 做成 REPL、工作流平台或 MCP 替代品，而是把它补齐为一层更完整的 Agent-Native CLI Execution Layer。

近期必须达到的状态：

- 当前二进制可用能力可被 CLI 自身发现
- operation 元数据成为单一事实源
- 文档声明、运行时注册、测试校验三者可以自动对账
- 写操作具备更真实的幂等与审计基础
- 后续 `completion`、文档生成、agent skill 都能复用同一份结构化元数据

## 4. 路线分期

### 4.1 M1：补齐 `spec` 最小闭环

状态：

- 已完成

目标：

- 让当前二进制具备基础自描述能力
- 让用户和 agent 可以直接查看 operation 目录和详情

交付物：

- `clawrise spec list [path]`
- `clawrise spec get <operation>`
- adapter registry 扩展描述性元数据
- 已注册 operation 补齐摘要、输入字段、示例、实现状态

验收标准：

- 根层可以列出当前平台
- 可以按层级浏览 operation 路径
- 可以查看单个 operation 的完整摘要信息
- `spec` 不依赖真实凭证即可运行

### 4.2 M2：补齐状态治理与漂移对账

目标：

- 明确区分“已实现”“已注册但未实现”“已声明但未落地”
- 避免文档、代码、测试继续分叉

交付物：

- `clawrise spec status`
- 结构化 operation catalog
- registry 与 catalog 的 diff 逻辑
- 元数据完整性测试

验收标准：

- 可以识别 stubbed operation
- 可以识别 catalog 声明但 runtime 缺失的 operation
- 新增 operation 时若未补齐元数据或 catalog，会被测试拦下

### 4.3 M3：补齐运行时治理能力

目标：

- 把幂等、审计、重试从“字段存在”推进到“能力可用”

交付物：

- 幂等状态持久化存储
- 基础审计记录
- 可配置的重试策略与错误分类
- 更明确的敏感信息脱敏规则

验收标准：

- 写操作可查询幂等状态
- 审计记录不泄露明文凭证
- 重试次数和最终状态能进入标准输出元信息

### 4.4 M4：衔接 completion 与文档生成

目标：

- 让 `spec` 成为多个消费方的统一底座

交付物：

- `completion` 改为消费 `spec` 层级树
- 中英文 operation 文档逐步由结构化元数据生成
- 面向外部脚本或 agent 的 `spec export`

验收标准：

- completion 不再维护独立命令树
- 结构化元数据可以导出完整 operation 清单
- 文档更新路径从“手工同步”转为“结构化生成优先”

## 5. 范围边界

近期明确不做：

- REPL-first 的交互产品
- 复杂跨平台工作流引擎
- 强制统一各平台业务资源 schema
- 一上来就做动态插件加载
- 一上来就做完整 JSON Schema 生成与校验体系

## 6. 依赖关系

几个阶段之间有明显依赖关系：

- `M2` 依赖 `M1` 的结构化元数据
- `M3` 不完全依赖 `M2`，但会受益于 `status` 输出的实现现状
- `M4` 明确依赖 `M1` 和 `M2`

因此近期建议优先级如下：

1. `spec status` + catalog
2. 幂等 / 审计 / 重试
3. completion / 文档生成

## 7. 风险与注意事项

### 7.1 不要把 `spec` 做成文档副本

`spec` 的首要职责是提供结构化事实源，而不是复制 Markdown 描述。

### 7.2 不要默认平铺全量 operation

随着平台和 operation 增长，`spec list` 的默认输出必须按层级浏览，否则很快会失控。

### 7.3 不要把 catalog 和 runtime 混为一层

如果只保留一份“理想中的 operation 列表”，就无法准确表达当前二进制真正可执行的能力。

### 7.4 不要过早引入过重的插件系统

当前更紧迫的问题是让元数据、文档、测试、completion 闭环，而不是立即追求动态加载能力。

## 8. 完成标志

可以认为近期路线图完成的标志是：

- 当前二进制支持的 operation 可以通过 `spec` 逐层发现
- `spec status` 可以准确指出实现漂移
- mutating operation 具备更真实的幂等与审计能力
- completion 与文档开始复用结构化元数据，而不是继续手工维护
