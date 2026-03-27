# Clawrise OSS Roadmap

## 1. 文档目的

这份文档只跟踪未来一段时间的 OSS core 优先事项。

它刻意不再重复已经交付的内容。已落地能力应放在项目简介和 `README` 的 `当前状态` 中，而不是长期留在 roadmap 里反复占位。

## 2. 当前 OSS 产品边界

在当前仓库里，Clawrise 应继续聚焦：

- core runtime 和 CLI
- provider plugin 协议与第一方 plugin 基线
- `spec`、completion 和基于元数据的参考文档面
- 本地治理、诊断和 playbook 支持

## 3. 近期 Must-have

### 3.1 第一方 Plugin 的正式发布工作流

原因：

- plugin-first runtime 已经成立，但第一方 plugin 的官方交付方式仍然偏临时
- 用户仍缺一条清晰、可文档化的安装、升级和兼容性检查路径

交付物：

- 带版本的第一方 plugin 发布产物
- 明确的 release manifest 结构
- core 与 plugin 之间的兼容性字段
- 第一方 plugin 的安装与升级文档路径
- 能消费官方发布元数据的 `plugin verify` 行为

完成标志：

- 用户可以通过一条较短文档路径安装并升级第一方 plugin
- 兼容性不匹配可以在真实执行前被看见

### 3.2 远程安装源的 Trust 与 Verify 策略

原因：

- `https://` 和 `npm://` 安装源已经存在，但 trust policy 仍不完整
- 只有“能装”还不够，必须补齐“为什么可信、校验了什么、何时拒绝”

交付物：

- 面向远程插件安装源的 trust model 文档
- checksum policy 与更明确的 verify 语义
- 在 plugin inspect 或 verify 输出中可见的 trust / verify 结果
- 对篡改、兼容性不匹配、产物不完整等情况的明确失败行为
- 为后续更强签名策略预留扩展点

完成标志：

- 远程插件安装与 verify 可以解释：检查了什么、信任了什么、为什么拒绝某个插件

### 3.3 从安装到第一条成功调用的接入路径

原因：

- 当前命令面已经可用，但第一次真正跑通仍然偏手工
- 项目需要一条更短的路径，把用户从 fresh install 带到一次真实成功调用

交付物：

- 更紧凑的 core + 第一方 plugin quickstart
- 一条经过文档化的 `config init`、`auth check`、`doctor` 到真实调用的短路径
- 与当前 CLI 输入形状一致的样例输入
- playbooks 与第一批可执行 operation 之间更清晰的链接

完成标志：

- 新用户无需先读设计文档，就能通过一条较短路径完成一次真实调用

### 3.4 Plugin Authoring 与兼容性 DX

原因：

- 如果第三方插件作者需要反向阅读 core 才能接入，公开生态很难增长
- 插件协议已经存在，但 authoring path 还缺产品化整理

交付物：

- 精简的 plugin author guide
- manifest 与兼容性参考文档
- 面向插件作者的本地校验或兼容性检查路径
- 更明确的 handshake、catalog 和 execute 测试指引

完成标志：

- 第三方作者可以不逐行阅读 core 内部实现，就完成一个最小 plugin 的构建和校验

### 3.5 基于同一元数据层的 Operation Reference

原因：

- `spec export` 与 completion 已经存在，但下游文档仍有继续漂移的风险
- 元数据层应继续成为 runtime、docs 与 discovery 的共同事实源

交付物：

- 面向下游消费者的稳定导出元数据契约
- 基于与 `spec` 相同元数据层生成的 operation reference 材料
- runtime registry 事实、catalog 声明、completion 与生成文档之间更明确的对应关系

完成标志：

- operation reference 直接派生自 `spec export` 与 completion 所在的同一元数据层，而不是独立手工维护

## 4. Must-have 之后的 Should-have

### 4.1 更宽的第一方 Provider 覆盖面

原因：

- 增加 provider 很有价值，但应排在发布工作流、trust policy 和 onboarding 更稳定之后

说明：

- `google` 仍然是下一候选 provider
- 但它不应在 plugin-first core 仍待 hardening 时就变成最近里程碑

### 4.2 继续扩展本地可搜索 Playbooks

原因：

- 当前 playbooks 已经形成不错的起点，但仍应继续围绕现有第一方 provider 扩展任务覆盖面

范围：

- 为 Feishu 和 Notion 增加更多高信号任务 playbook
- 保持样例尽量贴近真实 CLI 输入形状与可验证路径

## 5. Can Wait

这些方向并不是无效，而是应明确排在上面的 Must-have 之后：

- 公共 plugin marketplace
- 不可信 plugin 的沙箱隔离
- REPL-first 交互壳
- 完整 JSON Schema 框架
- 跨平台 workflow engine

## 6. 推荐顺序

1. 第一方 plugin 的正式发布工作流
2. 远程安装源的 trust 与 verify 策略
3. 从安装到第一条成功调用的接入路径
4. plugin authoring 与兼容性 DX
5. 基于同一元数据层的 operation reference
6. 更宽的第一方 provider 覆盖面
7. 继续扩展本地可搜索 playbooks

## 7. 风险与注意事项

### 7.1 不要把平台重新硬编码回 Core

plugin-first 应继续保持为默认架构方向。

### 7.2 不要重新制造多套元数据事实源

runtime facts、`spec`、生成文档、completion 和 playbooks 应继续收敛到同一层元数据，而不是重新分叉。

### 7.3 不要把“支持远程安装”误判成“trust model 已完成”

远程安装源虽然已经可用，但 release 和 trust hardening 仍需明确推进。

### 7.4 不要在打包和接入路径未稳定前过早扩平台面

如果过早增加 provider，会在 first-run 和 release path 仍不完整时继续放大复杂度。

## 8. 下一阶段完成标志

可以认为下一阶段 OSS core 工作完成的标志是：

- 第一方 plugin 具备清晰的 release、install 和 upgrade 路径
- 远程安装具备明确的 trust 与 verify 行为
- 新用户可以通过较短路径完成一次真实调用
- 第三方插件作者可以根据文档化兼容性契约完成最小 plugin 的构建与校验
- 生成的 operation reference 复用与 `spec export` 和 completion 相同的元数据层
