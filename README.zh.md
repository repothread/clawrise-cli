# Clawrise

英文说明见 [README.md](README.md)。

## 项目简介

Clawrise 是一个面向智能体的第三方服务接口命令行执行层。

它的目标是让智能体通过稳定的操作命令调用第三方系统，而不是依赖较重的工具定义注入方式。

当前仓库同时包含设计文档和正在推进中的 Go 语言运行时实现。

## 当前范围

当前最小可用阶段的平台组合为：

- `feishu`
- `notion`

最小可用阶段之后优先规划的平台为：

- `google`

## 文档入口

- [执行层架构设计](docs/zh/cli-layer-design.md)
- [插件系统设计](docs/zh/plugin-system-design.md)
- [近期路线图](docs/zh/roadmap.md)
- [`spec` 子系统设计](docs/zh/spec-design.md)
- [授权模型](docs/zh/auth-model.md)
- [最小可用阶段操作规格](docs/zh/mvp-operation-spec.md)
- [飞书用户授权凭证获取说明](docs/zh/feishu-user-auth-setup.md)

## 当前设计范围

- 命令模型
- 适配层架构
- 授权与身份配置模型
- 幂等与审计规则
- 飞书与 Notion 的最小可用阶段操作契约

## 建模边界

Clawrise 统一的是操作执行框架，不是所有服务资源的字段模型。

- 运行时契约应保持统一。
- 业务资源字段应尽量保留平台原生语义。
- 不应把飞书文档、Notion 页面、日历、多维表格、数据库记录等资源强行压成一套全局字段模型。
- 如果未来确实需要跨平台工作流抽象，应作为可选的上层能力存在，而不是替换底层平台操作契约。

## 示例配置

- [examples/config.example.yaml](examples/config.example.yaml)

## 当前状态

当前仓库同时包含架构设计文档和正在推进中的 Go 语言运行时实现。

当前已提供基础自描述命令：

- `clawrise spec list [path]`
- `clawrise spec get <operation>`
