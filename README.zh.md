# Clawrise

[![CI](https://github.com/repothread/clawrise-cli/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/repothread/clawrise-cli/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

简体中文 · [English](README.md)。

Clawrise 是一个命令行工具，用来访问第三方服务并执行稳定的操作命令。当前开箱即用的平台包括：

- `notion`
- `feishu`

## 安装

```bash
npm install -g @clawrise/clawrise-cli
clawrise doctor
```

## Notion 最小接入

前提：

- 在 Notion 中创建一个 integration
- 把你要访问的 page 或 data source 分享给这个 integration

生成账号骨架：

```bash
clawrise account add notion_docs --platform notion
```

写入 token 并检查状态：

```bash
export NOTION_TOKEN='secret_xxx'
clawrise auth secret set notion_docs token --from-env NOTION_TOKEN

# 检查账号配置是否完整
clawrise auth inspect notion_docs
# 确认这个账号是否已经可以执行操作
clawrise auth check notion_docs
```

先跑一条查询：

```bash
clawrise notion.search.query --json '{"query":"Demo","page_size":10}'
```

如果你要先验证写操作，可以用 dry-run：

```bash
clawrise notion.page.create --dry-run --json '{"parent":{"page_id":"page_demo"},"properties":{"title":[{"text":{"content":"Demo Page"}}]}}'
```

## Feishu 最小接入

前提：

- 创建一个飞书应用
- 拿到 `app_id` 和 `app_secret`

生成账号骨架：

```bash
clawrise account add feishu_bot --platform feishu
```

然后在配置文件里补上 `accounts.feishu_bot.auth.public.app_id`。

写入 secret 并检查状态：

```bash
export FEISHU_APP_SECRET='你的飞书应用密钥'
clawrise auth secret set feishu_bot app_secret --from-env FEISHU_APP_SECRET

# 检查账号配置是否完整
clawrise auth inspect feishu_bot
# 确认这个账号是否已经可以执行操作
clawrise auth check feishu_bot
```

先跑一条 dry-run：

```bash
clawrise feishu.calendar.event.create --dry-run --json '{"calendar_id":"cal_demo","summary":"Demo Event","start_at":"2026-03-30T10:00:00+08:00","end_at":"2026-03-30T11:00:00+08:00"}'
```

## 常用命令

```bash
clawrise doctor
clawrise auth methods --platform notion
clawrise auth methods --platform feishu
clawrise auth inspect <account>
clawrise auth check <account>
clawrise spec get <operation>
```

## 更多说明

- [Notion 页面更新 Playbook](docs/playbooks/zh/notion-page-update.md)
- [Notion Data Source 查询 Playbook](docs/playbooks/zh/notion-data-source-query.md)
- [飞书用户授权说明](docs/zh/feishu-user-auth-setup.md)
- [示例配置](examples/config.example.yaml)
- [支持说明](SUPPORT.zh.md)

## 许可证

本项目基于 [MIT License](LICENSE) 开源。
