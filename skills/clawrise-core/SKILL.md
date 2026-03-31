---
name: clawrise-core
description: Use when a task needs to use the local Clawrise CLI as an execution tool. This includes discovering platforms and operations, checking config and plugin health with doctor, inspecting auth methods, exporting specs, and executing third-party platform operations through `clawrise`. Pair with clawrise-feishu or clawrise-notion for platform-specific work.
---

# Clawrise Core

Treat Clawrise as the local execution layer, not as a cross-platform business schema layer.

This skill owns:

- setup and environment checks
- capability discovery through `doctor` and `spec`
- auth and account inspection
- generic execution rules

This skill does not own:

- Feishu-specific auth and task guidance
- Notion-specific auth and task guidance
- provider-specific field semantics beyond generic guardrails

## Use This Skill When

- You need to confirm that a local `clawrise` CLI is available
- You need to run `setup <client> [platform...]`
- You need to inspect plugins, config, runtime state, or playbooks
- You need to discover available platforms or operations
- You need to inspect the input fields, examples, or constraints of one operation
- You need to execute a Clawrise operation

## Workflow

1. Prefer `clawrise` from `PATH`.
2. If `clawrise` is not in `PATH`, but `npx` is available, use:

```bash
npx @clawrise/clawrise-cli ...
```

3. If the host still needs initial setup, run:

```bash
clawrise setup <client>
clawrise setup <client> <platform>
```

or:

```bash
npx @clawrise/clawrise-cli setup <client>
npx @clawrise/clawrise-cli setup <client> <platform>
```

4. If `clawrise` is not in `PATH`, but the current workspace is this repository, fall back to:

```bash
GOCACHE=/tmp/clawrise-go-build GOMODCACHE=/tmp/clawrise-gomodcache go run ./cmd/clawrise ...
```

5. Start with:

```bash
clawrise doctor
```

6. Then discover capabilities:

```bash
clawrise spec list
clawrise spec list <path>
clawrise spec get <operation>
```

7. Inspect auth only when the task needs it:

```bash
clawrise auth methods --platform <platform>
clawrise auth inspect <account>
clawrise auth check <account>
```

8. For write operations, always run `--dry-run` first and only perform the real call after the input shape is validated.

9. When building input JSON, keep provider-native field names exactly as defined by the operation. Do not invent a unified cross-platform schema.

10. If the task is Feishu-specific or Notion-specific, use the matching platform skill together with this one.

## Read These References Only When Needed

- `references/workflow.md`
- `references/install-and-layout.md`
