# Clawrise

英文版 README 见 [README.md](README.md)。

## 项目概述

Clawrise 是一个面向 AI Agent 的 SaaS API CLI 执行层。

它的目标是让 AI 通过稳定的 CLI operation 调用第三方系统，而不是依赖较重的 MCP 风格工具 schema。

当前仓库仍处于设计阶段，Go 代码实现还没有开始。

## 当前 MVP 范围

当前 MVP 平台组合为：

- `feishu`
- `notion`

MVP 之后优先规划的平台为：

- `google`

## 文档入口

英文文档：

- [CLI Layer Design](docs/en/cli-layer-design.md)
- [Auth Model](docs/en/auth-model.md)
- [MVP Operation Spec](docs/en/mvp-operation-spec.md)
- [Feishu User Auth Setup](docs/en/feishu-user-auth-setup.md)

中文文档：

- [CLI Layer 架构设计](docs/zh/cli-layer-design.md)
- [授权模型](docs/zh/auth-model.md)
- [MVP Operation 规格](docs/zh/mvp-operation-spec.md)
- [飞书用户授权凭证获取说明](docs/zh/feishu-user-auth-setup.md)

## 当前设计范围

- CLI 命令模型
- adapter 架构
- 授权与 profile 模型
- 幂等与审计规则
- Feishu 与 Notion 的 MVP operation 契约

## 示例配置

- [examples/config.example.yaml](examples/config.example.yaml)

## 当前状态

当前仓库只包含架构与产品设计文档。

下一步实现工作是初始化 Go 项目骨架，并按文档约定搭建运行时内核。
