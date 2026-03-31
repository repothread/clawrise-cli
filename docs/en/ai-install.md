# Clawrise AI Install Guide

Use this guide when an AI assistant needs to install the `clawrise` command and the related skills.

## For Humans

Send this prompt to the AI assistant:

```text
Access https://raw.githubusercontent.com/repothread/clawrise-cli/main/docs/en/ai-install.md and follow the steps there to install the `clawrise` command and the skills for the current client.
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
2. Should skills be installed globally for that client, or only for the current project/workspace?
3. Is a global npm install allowed on this machine, or should `npx` be used instead?

### Step 1: Run Preflight Checks

Check whether the required commands are available:

```bash
command -v npm
command -v npx
command -v clawrise
```

If `npm` is missing, stop and ask the user to install Node.js and npm first.

### Step 2: Install Or Reuse The CLI

If `clawrise` is already installed and usable, keep using it:

```bash
clawrise version
```

If `clawrise` is not installed and global npm installs are allowed:

```bash
npm install -g @clawrise/clawrise-cli
clawrise version
```

If global npm installs are not appropriate, use `npx`:

```bash
npx @clawrise/clawrise-cli version
```

### Step 3: Install Skills For The Target Client

Choose the command that matches the current client.

If `clawrise` is globally installed, use `clawrise ...`.

If the user wants to avoid a global install, replace `clawrise` with:

```bash
npx @clawrise/clawrise-cli
```

#### Codex

Global/shared install:

```bash
clawrise skills install-codex
```

#### Claude Code

Global/shared install:

```bash
clawrise skills install-claude-code
```

Project-local install:

```bash
clawrise skills install-claude-code --skills-dir ./.claude/skills
```

#### OpenClaw

Global/shared install:

```bash
clawrise skills install-openclaw
```

Workspace-local install:

```bash
clawrise skills install-openclaw --skills-dir ./skills
```

#### OpenCode

Global/shared install:

```bash
clawrise skills install-opencode
```

Project-local install:

```bash
clawrise skills install-opencode --skills-dir ./.opencode/skills
```

### Step 4: Verify The Installation

Verify the CLI:

```bash
clawrise doctor
clawrise spec list
```

Also verify that the installed skill directories exist in the target location.

Expected skill names:

- `clawrise-core`
- `clawrise-feishu`
- `clawrise-notion`

### Step 5: Handle Common Issues

If installation fails because the user does not want a global install:

- use `npx @clawrise/clawrise-cli ...`
- prefer project-local skill directories such as `./skills`, `./.claude/skills`, or `./.opencode/skills`

If installation fails because of write permissions:

- switch from a shared install path to a project-local install path

If the user only wants one platform skill:

- install only the required skill names, for example:

```bash
clawrise skills install-claude-code clawrise-core clawrise-feishu
```

### Step 6: Reload The Client

After installing skills, start a new session or restart the client so the skills are reloaded.
