# Plugin Author Guide

This guide is the practical entry point for building Clawrise plugins in the current repository.

## 1. Start With One Capability

Declare capabilities in `plugin.json` through `capabilities[]`.

```json
{
  "schema_version": 2,
  "name": "sample-policy",
  "version": "0.1.0",
  "protocol_version": 1,
  "capabilities": [
    {
      "type": "policy",
      "id": "require_reason_for_mutations",
      "priority": 80
    }
  ],
  "entry": {
    "type": "binary",
    "command": ["./run.sh"]
  }
}
```

Rules to keep stable:

- prefer `capabilities[]` over legacy `kind`
- keep provider contracts provider-native
- let the core own config loading, account resolution, retries, audit, idempotency, and the final execution envelope
- let the plugin own capability logic and upstream integration details

## 2. Minimal Protocol Surface

All plugins must keep:

- `stdout`: JSON-RPC protocol messages only
- `stderr`: logs only

Each capability should support:

- `clawrise.handshake`
- `clawrise.capabilities.list`
- the minimal capability-specific method surface

Examples:

- `policy`: `clawrise.policy.evaluate`
- `audit_sink`: `clawrise.audit.emit`
- `workflow`: `clawrise.workflow.plan`
- `registry_source`: `clawrise.registry_source.list` and `clawrise.registry_source.resolve`

## 3. Repository Samples

This repository now includes the first local authoring fixtures:

- policy sample source: [`cmd/clawrise-plugin-sample-policy/main.go`](../../cmd/clawrise-plugin-sample-policy/main.go)
- audit sink sample source: [`cmd/clawrise-plugin-sample-audit/main.go`](../../cmd/clawrise-plugin-sample-audit/main.go)
- sample manifests: [`examples/plugins`](../../examples/plugins)

These samples intentionally optimize for readability over completeness. They are the recommended starting point for third-party governance capability development.

They are not the intended long-term distribution shape for external plugins. A real third-party plugin should usually:

- live in its own repository
- publish its own npm package
- be installed through `clawrise plugin install @scope/clawrise-plugin-<name>`

## 4. Local Validation Workflow

Use the sample discovery root:

```bash
CLAWRISE_PLUGIN_PATHS=$PWD/examples/plugins clawrise doctor
```

Run the sample compatibility fixture:

```bash
./scripts/plugin/verify-sample-plugins.sh
```

The fixture validates:

- handshake
- capabilities list
- minimal policy evaluation behavior
- minimal audit emission behavior
- discovery and capability route inspection

For packaged plugins that were actually installed into a Clawrise plugin root, also run:

```bash
clawrise plugin verify <name> <version>
```

## 5. Common Mistakes

- printing logs to `stdout`, which breaks the protocol stream
- declaring one capability in the manifest but exposing a different capability set at runtime
- reading the main Clawrise config file directly from the plugin process
- returning free-form text where a structured capability response already exists
- hiding provider-native resource fields behind a cross-platform schema too early

## 6. Recommended Next Steps

When creating a new capability plugin:

1. copy the closest sample plugin
2. update `plugin.json`
3. implement the minimal JSON-RPC surface
4. point `CLAWRISE_PLUGIN_PATHS` at the local plugin root
5. run `clawrise doctor`
6. run the compatibility fixture or targeted `go test` coverage
