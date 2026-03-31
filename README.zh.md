# Clawrise

[![CI](https://github.com/clawrise/clawrise-cli/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/clawrise/clawrise-cli/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

简体中文 · [English](README.md)。

Clawrise 是一个面向 AI 的命令行工具，用稳定的 operation 调用第三方平台。它适合两种使用方式：

- 先把 `clawrise` 接入 Codex、Claude Code、OpenClaw、OpenCode 这类 AI 客户端，再让 AI 通过 skills 调用第三方平台
- 人类直接在终端里执行 `clawrise` 命令

当前开箱即用的平台包括：

- `notion`
- `feishu`

## AI 接入

把下面这段 prompt 原样发给 AI 助手：

```text
Access https://raw.githubusercontent.com/clawrise/clawrise-cli/main/docs/en/ai-install.md and follow the steps there to install the `clawrise` command and run setup for the current client.
```

## 人类接入

### 1. 安装

```bash
npm install -g @clawrise/clawrise-cli
```

### 2. 接入 Notion

1. 打开 [Notion 开发集成页面](https://www.notion.so/profile/integrations)并创建或选择一个 integration
2. 复制这个 integration 的 `Internal Integration Token`
3. 如果你要让 AI 客户端直接加载对应 skills，可以执行下面任意一种：

```bash
NOTION_INTERNAL_TOKEN=secret_xxx clawrise setup codex notion
NOTION_INTERNAL_TOKEN=secret_xxx clawrise setup claude-code notion
NOTION_INTERNAL_TOKEN=secret_xxx clawrise setup openclaw notion
NOTION_INTERNAL_TOKEN=secret_xxx clawrise setup opencode notion
```

如果你不想通过环境变量传值，也可以运行：

```bash
clawrise setup codex notion
clawrise setup claude-code notion
clawrise setup openclaw notion
clawrise setup opencode notion
```

然后按提示输入 `Internal Integration Token`。交互式终端下，`clawrise setup codex notion` 这类命令会直接进入交互式输入状态。

设置完成后，可以执行下面的命令验证当前默认账号是否可用：

```bash
clawrise auth check
clawrise doctor
```

### 3. 接入 Feishu

1. 打开[飞书开发者后台应用中心](https://open.feishu.cn/app)并创建或选择一个应用
2. 记下这个应用的 `App ID` 和 `App Secret`
3. 如果你要让 AI 客户端直接加载对应 skills，可以执行下面任意一种：

```bash
FEISHU_APP_ID=cli_xxx FEISHU_APP_SECRET=cli_secret_xxx clawrise setup codex feishu
FEISHU_APP_ID=cli_xxx FEISHU_APP_SECRET=cli_secret_xxx clawrise setup claude-code feishu
FEISHU_APP_ID=cli_xxx FEISHU_APP_SECRET=cli_secret_xxx clawrise setup openclaw feishu
FEISHU_APP_ID=cli_xxx FEISHU_APP_SECRET=cli_secret_xxx clawrise setup opencode feishu
```

如果你不想通过环境变量传值，也可以运行：

```bash
clawrise setup codex feishu
clawrise setup claude-code feishu
clawrise setup openclaw feishu
clawrise setup opencode feishu
```

然后按提示输入 `App ID` 和 `App Secret`。

设置完成后，可以执行下面的命令验证当前默认账号是否可用：

```bash
clawrise auth check
clawrise doctor
```

## 相关文档

- [AI 安装引导](docs/en/ai-install.md)

## 许可证

本项目基于 [MIT License](LICENSE) 开源。
