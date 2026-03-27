# Clawrise

英文说明见 [README.md](README.md)。

## 项目简介

Clawrise 是一个面向智能体的第三方服务接口命令行执行层。

它的目标是让智能体通过稳定的操作命令调用第三方系统，而不是依赖较重的工具定义注入方式。

当前架构已经转向 plugin-first：

- `clawrise` 是 core runtime 和 CLI
- 平台能力通过外部 provider plugin 暴露
- Feishu 和 Notion 的第一方能力以 plugin binary 形式提供

当前仓库同时包含设计文档以及 core runtime 和第一方 plugin 的 Go 语言实现。

## 当前范围

当前第一方 plugin 平台为：

- `feishu`
- `notion`

最小可用阶段之后优先规划的平台为：

- `google`

## 文档入口

- [执行层架构设计](docs/zh/cli-layer-design.md)
- [插件系统设计](docs/zh/plugin-system-design.md)
- [近期路线图](docs/zh/roadmap.md)
- [本地 Playbooks 索引](docs/playbooks/index.yaml)
- [`spec` 子系统设计](docs/zh/spec-design.md)
- [授权模型](docs/zh/auth-model.md)
- [最小可用阶段操作规格](docs/zh/mvp-operation-spec.md)
- [飞书用户授权凭证获取说明](docs/zh/feishu-user-auth-setup.md)

## 当前设计范围

- 命令模型
- provider plugin 架构
- 授权与身份配置模型
- 幂等与审计规则
- 飞书与 Notion 的 operation 契约

## 建模边界

Clawrise 统一的是操作执行框架，不是所有服务资源的字段模型。

- 运行时契约应保持统一。
- 业务资源字段应尽量保留平台原生语义。
- 不应把飞书文档、Notion 页面、日历、多维表格、数据库记录等资源强行压成一套全局字段模型。
- 如果未来确实需要跨平台工作流抽象，应作为可选的上层能力存在，而不是替换底层平台操作契约。

## 示例配置

- [examples/config.example.yaml](examples/config.example.yaml)

## 当前状态

当前仓库已经具备：

- 外部进程 provider runtime 抽象
- Feishu / Notion 第一方 plugin binary
- 基于 plugin manifest 的插件发现
- 插件管理命令：
  - `clawrise plugin list`
  - `clawrise plugin install <source>`
  - `clawrise plugin info <name> <version>`
  - `clawrise plugin remove <name> <version>`
  - `clawrise plugin verify <name> <version>`
  - `clawrise plugin verify --all`
- 最小可用接入辅助命令：
  - `clawrise config init`
  - `clawrise auth list`
  - `clawrise auth inspect <profile>`
  - `clawrise auth check [profile]`
  - `clawrise doctor`
- 本地可搜索 playbooks：
  - `docs/playbooks/index.yaml`
  - `docs/playbooks/zh/*.md`
  - `docs/playbooks/en/*.md`
- 当前安装源支持：
  - 本地目录
  - `file://`
  - `https://`
  - `npm://`
- 当前自描述命令：
  - `clawrise spec list [path]`
  - `clawrise spec get <operation>`
  - `clawrise spec status`
- 当前运行时治理能力：
  - 写操作幂等状态本地持久化
  - 本地 JSONL 审计日志
  - 基于配置的自动重试策略
  - 审计输入输出脱敏

当前仍未完成：

- `clawrise spec export`
- `completion`
- plugin signature policy
- 第一方 plugin 的正式分发工作流
