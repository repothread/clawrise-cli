# Clawrise

[![CI](https://github.com/clawrise/clawrise-cli/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/clawrise/clawrise-cli/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

English · [简体中文](README.zh.md)

Clawrise is an AI-oriented CLI that calls third-party platforms through stable operations. It fits two main workflows:

- connect `clawrise` to AI clients such as Codex, Claude Code, OpenClaw, and OpenCode, then let the AI use platform skills
- run `clawrise` directly in your terminal

Current built-in platforms:

- `notion`
- `feishu`

## AI Setup

Send the following prompt to the AI assistant as-is:

```text
Access https://raw.githubusercontent.com/clawrise/clawrise-cli/main/docs/en/ai-install.md and follow the steps there to install the `clawrise` command and run setup for the current client.
```

## Human Setup

### 1. Install

```bash
npm install -g @clawrise/clawrise-cli
```

### 2. Connect Notion

1. Open the [Notion integrations page](https://www.notion.so/profile/integrations) and create or choose an integration
2. Copy the integration `Internal Integration Token`
3. If you want an AI client to load the matching skills directly, run any of these:

```bash
NOTION_INTERNAL_TOKEN=secret_xxx clawrise setup codex notion
NOTION_INTERNAL_TOKEN=secret_xxx clawrise setup claude-code notion
NOTION_INTERNAL_TOKEN=secret_xxx clawrise setup openclaw notion
NOTION_INTERNAL_TOKEN=secret_xxx clawrise setup opencode notion
```

If you do not want to pass the token through an environment variable, you can also run:

```bash
clawrise setup codex notion
clawrise setup claude-code notion
clawrise setup openclaw notion
clawrise setup opencode notion
```

Then enter the `Internal Integration Token` when prompted. In an interactive terminal, commands such as `clawrise setup codex notion` will enter interactive input mode automatically.

After setup completes, you can verify that the current default account is usable:

```bash
clawrise auth check
clawrise doctor
```

### 3. Connect Feishu

1. Open the [Feishu developer app console](https://open.feishu.cn/app) and create or choose an app
2. Record the app `App ID` and `App Secret`
3. If you want an AI client to load the matching skills directly, run any of these:

```bash
FEISHU_APP_ID=cli_xxx FEISHU_APP_SECRET=cli_secret_xxx clawrise setup codex feishu
FEISHU_APP_ID=cli_xxx FEISHU_APP_SECRET=cli_secret_xxx clawrise setup claude-code feishu
FEISHU_APP_ID=cli_xxx FEISHU_APP_SECRET=cli_secret_xxx clawrise setup openclaw feishu
FEISHU_APP_ID=cli_xxx FEISHU_APP_SECRET=cli_secret_xxx clawrise setup opencode feishu
```

If you do not want to pass credentials through environment variables, you can also run:

```bash
clawrise setup codex feishu
clawrise setup claude-code feishu
clawrise setup openclaw feishu
clawrise setup opencode feishu
```

Then enter `App ID` and `App Secret` when prompted.

After setup completes, you can verify that the current default account is usable:

```bash
clawrise auth check
clawrise doctor
```

## Related Docs

- [AI Installation Guide](docs/en/ai-install.md)

## License

This project is licensed under the [MIT License](LICENSE).
