# Clawrise

[![CI](https://github.com/repothread/clawrise-cli/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/repothread/clawrise-cli/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

英文说明见 [README.md](README.md)。

## 项目简介

Clawrise 是一个面向智能体的第三方服务接口命令行执行层。

它的目标是让智能体通过稳定的操作命令调用第三方系统，而不是依赖较重的工具定义注入方式。

当前架构已经转向 plugin-first：

- `clawrise` 是 core runtime 和 CLI
- 平台能力通过外部 provider plugin 暴露
- Feishu 和 Notion 的第一方能力以 plugin binary 形式提供

当前仓库同时包含设计文档以及 core runtime 和第一方 plugin 的 Go 语言实现。

## 开源协作状态

仓库已经补齐面向公开协作的基础文档，方便外部贡献者快速判断项目是否可用、如何参与以及出现问题后如何处理：

- [MIT 开源协议](LICENSE)
- [贡献指南](CONTRIBUTING.zh.md)
- [行为准则](CODE_OF_CONDUCT.zh.md)
- [安全策略](SECURITY.zh.md)
- [支持说明](SUPPORT.zh.md)
- [英文贡献指南](CONTRIBUTING.md)
- [英文安全策略](SECURITY.md)
- [公开路线图](docs/zh/roadmap.md)

## 当前范围

当前第一方 plugin 平台为：

- `feishu`
- `notion`

在 core hardening 完成之后，下一候选平台为：

- `google`

路线图范围说明：

- [docs/zh/roadmap.md](docs/zh/roadmap.md) 只跟踪未来一段时间的 OSS core 工作
- 已经落地的能力统一放在下方 `当前状态` 中

## 快速开始

```bash
go build ./...
go test ./...
go run ./cmd/clawrise version
go run ./cmd/clawrise doctor
```

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
- 授权与 account 配置模型
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

## 参与贡献

欢迎围绕运行时能力、plugin、文档、playbook、示例和测试改进仓库。

提交 PR 前建议先阅读：

- [贡献指南](CONTRIBUTING.zh.md)
- [行为准则](CODE_OF_CONDUCT.zh.md)
- [安全策略](SECURITY.zh.md)
- [支持说明](SUPPORT.zh.md)
- 对应的 GitHub Issue / Pull Request 模板

## 获取支持

如果你在使用或扩展 Clawrise 时遇到问题，请先阅读 [SUPPORT.zh.md](SUPPORT.zh.md)。

如果你关心项目接下来的重点方向，请查看 [docs/zh/roadmap.md](docs/zh/roadmap.md)。

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
  - `clawrise auth methods`
  - `clawrise auth presets`
  - `clawrise account add --platform <name> --preset <id>`
  - `clawrise auth inspect <account>`
  - `clawrise auth check [account]`
  - `clawrise auth login <account>`
  - `clawrise auth complete <flow_id>`
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
  - `clawrise spec export [path] [--format json|markdown]`
  - `clawrise completion <bash|zsh|fish>`
- 当前运行时治理能力：
  - 写操作幂等状态本地持久化
  - 本地 JSONL 审计日志
  - 基于配置的自动重试策略
  - 审计输入输出脱敏

当前仍未完成：

- 远程安装源的 trust policy hardening 仍需继续完善
- 第一方 plugin 的正式打包发布工作流仍未落地

## 许可证

本项目基于 [MIT License](LICENSE) 开源。
