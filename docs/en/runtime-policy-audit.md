# Runtime Policy And Audit Configuration

This guide explains the official configuration shape for `runtime.policy` and `runtime.audit`.

## 1. Selection Model

Both capability chains use the same runtime selection modes:

- `auto`: use discovered capabilities by default order
- `manual`: only use the capabilities listed in config
- `disabled`: turn the chain off completely

Local policy rules always live in `runtime.policy` and do not require any external plugin.

## 2. Example Configuration

The canonical example lives in [`examples/config.example.yaml`](../../examples/config.example.yaml).

```yaml
runtime:
  policy:
    mode: auto
    require_approval_operations:
      - feishu.calendar.event.delete
    annotate_operations:
      notion.page.update: "Set input.change_reason when updating shared knowledge content."
    # plugins:
    #   - plugin: sample-policy
    #     policy_id: require_reason_for_mutations
  audit:
    mode: manual
    sinks:
      - type: stdout
      # - type: webhook
      #   url: env:CLAWRISE_AUDIT_WEBHOOK_URL
      #   headers:
      #     Authorization: env:CLAWRISE_AUDIT_WEBHOOK_TOKEN
      #   timeout_ms: 3000
      # - type: plugin
      #   plugin: sample-audit
      #   sink_id: file_capture
```

## 3. Policy Fields

- `runtime.policy.mode`: selects `auto`, `manual`, or `disabled`
- `runtime.policy.plugins[]`: optional plugin selectors with `plugin` and/or `policy_id`
- `runtime.policy.deny_operations[]`: local hard-stop rules
- `runtime.policy.require_approval_operations[]`: local approval rules
- `runtime.policy.annotate_operations`: local warning and annotation rules

Use local rules for stable guardrails that belong to the core config. Use plugins when the decision logic should be replaceable or reused across environments.

## 4. Audit Fields

- `runtime.audit.mode`: selects `auto`, `manual`, or `disabled`
- `runtime.audit.sinks[]`: ordered sink declarations
- builtin sink `stdout`: writes newline-delimited JSON audit records to stdout
- builtin sink `webhook`: posts JSON audit records to one HTTP endpoint
- plugin sink: matches a discovered `audit_sink` capability by `plugin` and/or `sink_id`

Webhook fields support secret resolution, so values like `env:CLAWRISE_AUDIT_WEBHOOK_URL` stay outside the config file.

## 5. Diagnostics

Use these commands while changing the runtime chain:

```bash
clawrise doctor
clawrise plugin list
clawrise plugin info <name> <version>
clawrise config policy mode manual
clawrise config policy use sample-policy --policy-id require_reason_for_mutations
clawrise config audit add stdout
clawrise config audit add plugin sample-audit --sink-id file_capture
```

`clawrise doctor` shows:

- local policy summary
- active policy chain
- active audit sinks
- capability route reasons when a discovered plugin is not selected

## 6. Sample Plugins

This repository includes local source fixtures for governance capability development:

- [`examples/plugins/sample-policy/0.1.0/plugin.json`](../../examples/plugins/sample-policy/0.1.0/plugin.json)
- [`examples/plugins/sample-audit/0.1.0/plugin.json`](../../examples/plugins/sample-audit/0.1.0/plugin.json)

Point `CLAWRISE_PLUGIN_PATHS` at `examples/plugins` during development to make them discoverable.
