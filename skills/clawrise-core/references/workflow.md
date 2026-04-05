# Clawrise General Workflow

## Quick Map

- [1. Choose The Setup Or Runtime Entry Point](#1-choose-the-setup-or-runtime-entry-point)
- [2. Inspect The Environment First](#2-inspect-the-environment-first)
- [3. Discover Before Building Input](#3-discover-before-building-input)
- [4. Inspect Auth And Accounts](#4-inspect-auth-and-accounts)
- [5. Execution Rules](#5-execution-rules)
- [6. Output Shape](#6-output-shape)
- [7. Architectural Boundary](#7-architectural-boundary)

## 1. Choose The Setup Or Runtime Entry Point

Use the published wrapper for setup:

- `clawrise setup ...` when the published package is already installed
- `npx @clawrise/clawrise-cli setup ...` when it is not installed yet

Use the repo-local Go entrypoint for runtime commands while developing in this repository:

```bash
GOCACHE=/tmp/clawrise-go-build GOMODCACHE=/tmp/clawrise-gomodcache go run ./cmd/clawrise ...
```

The repo-local Go entrypoint does not expose the npm-only `setup` wrapper.

## 1.1 Ensure Setup Has Been Run

If the user is on a fresh machine or the current client has not been prepared yet, start with:

```bash
clawrise setup <client>
clawrise setup <client> <platform>
clawrise setup <platform>
```

or:

```bash
npx @clawrise/clawrise-cli setup <client>
npx @clawrise/clawrise-cli setup <client> <platform>
npx @clawrise/clawrise-cli setup <platform>
```

Use platform setup only when the user actually needs that platform.

Examples:

- `setup codex`
- `setup codex feishu`
- `setup notion`
- `setup openclaw notion`

Preferred credential flow:

- `NOTION_INTERNAL_TOKEN=secret_xxx clawrise setup codex notion`
- `FEISHU_APP_ID=cli_xxx FEISHU_APP_SECRET=cli_secret_xxx clawrise setup codex feishu`

When setup completes successfully, the default account names are:

- `notion_bot`
- `feishu_bot`

## 2. Inspect The Environment First

Start with:

```bash
clawrise doctor
```

This command tells you:

- config file path
- state and runtime paths
- plugin discovery roots
- discovered plugins and their health
- playbook validation status

Do not assume any of these before checking `doctor`:

- a platform plugin is installed
- a default account exists
- local config is already complete

## 3. Discover Before Building Input

Preferred sequence:

```bash
clawrise spec list
clawrise spec list feishu
clawrise spec list notion
clawrise spec get <operation>
```

If the task spans many operations, use:

```bash
clawrise spec export <path> --format markdown
clawrise docs generate <path> --out-dir <dir>
```

`spec get` is the most important fact source when you need to build one concrete call.

## 4. Inspect Auth And Accounts

Useful commands:

```bash
clawrise auth methods --platform <platform>
clawrise auth inspect <account>
clawrise auth check <account>
clawrise account list
clawrise account inspect <name>
```

If the user does not explicitly provide an account, do not assume the default account is valid. Check `doctor` or `account current` first.

## 5. Execution Rules

Recommended order:

```bash
clawrise <operation> --dry-run --json '<payload>'
clawrise <operation> --json '<payload>'
```

Common flags include:

- `--account`
- `--subject`
- `--json`
- `--input`
- `--timeout`
- `--dry-run`
- `--idempotency-key`
- `--quiet`

## 6. Output Shape

Execution output is a normalized JSON envelope. Key fields include:

- `ok`
- `operation`
- `context`
- `data`
- `error`
- `meta`
- `idempotency`

If a downstream step only needs the success payload, use `--quiet`.

## 7. Architectural Boundary

Clawrise unifies the execution layer, not the business resource model.

That means:

- Build Feishu inputs with Feishu-native fields
- Build Notion inputs with Notion-native fields
- Do not compress calendars, pages, documents, or records into one invented JSON schema

If the user needs platform-specific auth or task guidance, switch to the matching platform skill after setup:

- `clawrise-feishu` for legacy Feishu-through-Clawrise maintenance only
- `clawrise-notion`
