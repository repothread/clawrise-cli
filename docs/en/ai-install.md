# Clawrise AI Install Guide

Use this guide when an AI assistant needs to install the published `clawrise` npm package and run the packaged `setup` wrapper for the current client.

## For Humans

Send this prompt to the AI assistant:

```text
Access https://raw.githubusercontent.com/repothread/clawrise-cli/main/docs/en/ai-install.md and follow the steps there to install the `clawrise` command and run setup for the current client.
```

## For AI Assistants

Use the raw file directly:

```bash
curl -fsSL https://raw.githubusercontent.com/repothread/clawrise-cli/main/docs/en/ai-install.md
```

Do not rely on rendered GitHub pages when executing the installation flow.

### Step 0: Confirm The Target Environment

Before installing anything, confirm these points with the user:

1. Which client is this for?
   - Codex
   - Claude Code
   - OpenClaw
   - OpenCode
2. Which platforms are needed right now?
   - `feishu`
   - `notion`
3. Should the skills be installed into the default shared directory for that client, or a project-local skills directory?

### Step 1: Run Preflight Checks

Check whether the required commands are available:

```bash
command -v npm
command -v npx
```

If `npm` is missing, stop and ask the user to install Node.js and npm first.

### Step 2: Choose The Setup Entry Point

The published npm root package exposes `clawrise setup ...`.

Raw `go run ./cmd/clawrise` executions and standalone development binaries expose the runtime commands directly and do not include this wrapper.

Use `clawrise setup ...` if the published package is already installed.

Otherwise use `npx @clawrise/clawrise-cli setup ...`.

Setup is responsible for:

- ensuring the published `clawrise` command is installed on the host unless `--skip-cli-install` is used
- installing `clawrise-core`
- installing any requested platform skills
- configuring default platform accounts when credentials are available

Default setup account names:

- `notion_bot`
- `feishu_bot`

If no platform is specified, setup installs only `clawrise-core` and does not initialize a platform account.

### Step 3: Run Setup

If `clawrise` is already installed and usable:

```bash
clawrise setup codex
clawrise setup codex feishu
```

If `clawrise` is not installed yet:

```bash
npx @clawrise/clawrise-cli setup codex
npx @clawrise/clawrise-cli setup codex feishu
```

For platform auth, prefer environment variables:

```bash
NOTION_INTERNAL_TOKEN=secret_xxx clawrise setup codex notion
FEISHU_APP_ID=cli_xxx FEISHU_APP_SECRET=cli_secret_xxx clawrise setup codex feishu
```

If the environment variables are missing and the shell is interactive, `setup` can prompt for the required credentials directly.

Client-specific examples:

```bash
clawrise setup codex
clawrise setup claude-code notion
clawrise setup openclaw feishu
clawrise setup opencode notion
```

Platform-only examples:

```bash
clawrise setup notion
clawrise setup feishu
```

Project-local examples:

```bash
clawrise setup claude-code notion --skills-dir ./.claude/skills
clawrise setup openclaw feishu --skills-dir ./skills
clawrise setup opencode notion --skills-dir ./.opencode/skills
```

### Step 4: Verify The Installation

Verify the CLI:

```bash
clawrise version
clawrise doctor
clawrise spec list
```

If platform setup was requested, also verify the default account:

```bash
clawrise auth check notion_bot
clawrise auth check feishu_bot
```

Also verify that the installed skill directories exist in the target location.

Expected skill names:

- `clawrise-core`
- `clawrise-feishu`
- `clawrise-notion`

### Step 5: Handle Common Issues

If installation fails because of write permissions:

- switch from a shared install path to a project-local install path

If the user only wants one platform skill:

- rerun setup with only the required platform names, for example:

```bash
clawrise setup claude-code feishu
```

If the user is running the repository directly with `go run ./cmd/clawrise` and expects `setup` to exist:

- switch back to the published npm package entrypoint or use `npx @clawrise/clawrise-cli setup ...`

### Step 6: Reload The Client

After setup completes, start a new session or restart the client so the new skills are loaded.
