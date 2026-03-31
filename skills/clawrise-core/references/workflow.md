# Clawrise General Workflow

## 1. Inspect The Environment First

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

## 2. Discover Before Building Input

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

## 3. Inspect Auth And Accounts

Useful commands:

```bash
clawrise auth methods --platform <platform>
clawrise auth inspect <account>
clawrise auth check <account>
clawrise account list
clawrise account inspect <name>
```

If the user does not explicitly provide an account, do not assume the default account is valid. Check `doctor` or `account current` first.

## 4. Execution Rules

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

## 5. Output Shape

Execution output is a normalized JSON envelope. Key fields include:

- `ok`
- `operation`
- `context`
- `data`
- `error`
- `meta`
- `idempotency`

If a downstream step only needs the success payload, use `--quiet`.

## 6. Architectural Boundary

Clawrise unifies the execution layer, not the business resource model.

That means:

- Build Feishu inputs with Feishu-native fields
- Build Notion inputs with Notion-native fields
- Do not compress calendars, pages, documents, or records into one invented JSON schema
