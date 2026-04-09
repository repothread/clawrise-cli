---
name: clawrise-core
description: Use when a task needs the generic Clawrise CLI workflow rather than provider-specific field semantics. This includes setup through the published `clawrise` command or `npx @clawrise/clawrise-cli`, checking config and plugin health with `doctor`, discovering platforms and operations with `spec`, inspecting auth methods and accounts, exporting specs/docs, and executing validated operations with provider-native payloads. Pair with clawrise-feishu or clawrise-notion when the task becomes platform-specific.
---

# Clawrise Core

Treat Clawrise as the local execution layer, not as a cross-platform business schema layer.

## Boundary

This skill owns:

- setup and environment checks
- capability discovery through `doctor` and `spec`
- auth and account inspection
- generic execution rules

This skill does not own:

- Feishu-specific auth and task guidance
- Notion-specific auth and task guidance
- provider-specific field semantics beyond generic guardrails

## Fast Path

1. For setup, prefer the published wrapper from `PATH`. If it is not installed, use `npx`:

```bash
npx @clawrise/clawrise-cli setup <client>
npx @clawrise/clawrise-cli setup <client> <platform>
npx @clawrise/clawrise-cli setup <platform>
```

```bash
clawrise setup <client>
clawrise setup <client> <platform>
clawrise setup <platform>
```

2. If you are developing inside this repository and need repo-local fallback, use:

```bash
npx @clawrise/clawrise-cli ...
```

```bash
GOCACHE=/tmp/clawrise-go-build GOMODCACHE=/tmp/clawrise-gomodcache go run ./cmd/clawrise ...
```

Use the repo-local `go run ./cmd/clawrise ...` fallback for runtime commands, not for the npm-only `setup` wrapper.

3. Start with:

```bash
clawrise doctor
```

4. Then discover capabilities:

```bash
clawrise spec list
clawrise spec list <path>
clawrise spec get <operation>
```

5. Inspect auth or accounts only when the task needs them:

```bash
clawrise auth methods --platform <platform>
clawrise auth inspect <account>
clawrise auth check <account>
clawrise account list
clawrise account inspect <name>
```

6. For mutating operations:

```bash
clawrise spec get <operation>
clawrise <operation> --dry-run --json '<payload>'
clawrise <operation> --json '<payload>'
```

7. Keep provider-native field names exactly as defined by the operation. Do not invent a unified cross-platform schema.
8. If the task is Notion-specific, add `clawrise-notion` after this one. Add `clawrise-feishu` only for legacy Feishu-through-Clawrise flows that still need maintenance.

## Read These References Only When Needed

- `references/workflow.md` — full command checklist and runtime rules
- `references/install-and-layout.md` — install targets, packaging layout, and distribution notes
