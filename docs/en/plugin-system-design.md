# Clawrise Plugin System Design

See the Chinese version at [../zh/plugin-system-design.md](../zh/plugin-system-design.md).

> For the current author-facing protocol surface and distribution guidance, use [plugin-authoring-spec.md](./plugin-authoring-spec.md). This document remains the architecture design document.

## 1. Purpose

This document defines the target plugin architecture for Clawrise.

It addresses the following problems:

- provider integrations should not remain hard-coded inside the core binary
- provider capabilities should be installable, releasable, and upgradeable independently
- `spec`, catalog, runtime, and diagnostics should still keep one unified execution shell
- the plugin mechanism must stay decoupled from distribution channels, with `npm` treated only as an optional source

This design is written for the desired end-state. It does not preserve the current built-in Feishu / Notion registration path as a compatibility constraint.

## 1.1 Current Progress

As of the current repository state:

- `M1` is complete:
  - provider runtime abstraction exists in the core
- `M2` is complete:
  - manifest parsing, plugin discovery, and external-process runtime exist
- `M3` is complete:
  - first-party Feishu and Notion are exposed through plugin binaries
- `M4` is largely complete:
  - `plugin list/install/info/remove/verify/upgrade` exist
  - local directory, `file://`, `https://`, direct npm package specs, `npm://`, and `registry://` installation are implemented
  - capability-aware diagnostics for `inspect`, `verify`, `doctor`, and provider binding selection exist
  - storage bindings, auth launcher preferences, and the newer config model are implemented
  - install and upgrade validate `protocol_version` and `min_core_version` before activation
  - install metadata records source, artifact URL, and checksums
  - `plugin verify` emits structured trust results driven by `plugins.install.allowed_sources`, `plugins.install.allowed_hosts`, and `plugins.install.allowed_npm_scopes`
  - `plugin upgrade` and `plugin upgrade --all` distinguish upgraded, unchanged, pinned-source, and downgrade-blocked states
- `M5` first phase is complete:
  - `policy` and `audit_sink` capabilities are part of the execution path
  - structured policy output is written into the execution envelope
  - diagnostics explain active capability routes and selection reasons
- `M6` first phase is complete:
  - `workflow.plan` protocol scaffolding exists as an extension capability
  - `registry_source` protocol scaffolding already participates in `registry://` installation resolution
- the repository still carries the first authoring kit baseline:
  - `examples/config.example.yaml` includes `runtime.policy` and `runtime.audit` examples
  - minimal `policy` and `audit_sink` sample plugins
  - protocol compatibility fixtures and a verification script
- companion plugin authoring and runtime governance docs are maintained in the separate `clawrise` project root markdown files
- the main remaining hardening work is now concentrated in:
  - release hardening
  - trust policy hardening
  - upgrade workflow hardening

## 2. Non-goals

The first plugin system version should explicitly avoid:

- Go `plugin` `.so` dynamic loading
- treating `npm` as the only installation path
- language-specific plugin protocols
- remote hosted execution infrastructure
- a full signing and sandboxing system in the first release

## 3. Design Summary

Clawrise should adopt:

- `core + external provider plugins`
- `stdio + JSON-RPC 2.0`
- lazy plugin process startup
- a core that owns runtime, config, spec aggregation, audit, idempotency, and normalized output
- plugins that own provider catalogs, auth details, provider-native mapping, and real execution

Distribution and runtime must stay decoupled:

- runtime discovers plugins from local directories and manifests
- distribution may support `file://`, `https://`, direct npm package specs, `npm://`, `registry://`, and future sources

## 4. Layering

### 4.1 Core Responsibilities

`clawrise-core` should own:

- CLI entrypoints and command parsing
- config parsing and account resolution
- auth material resolution and redaction
- the normalized runtime envelope
- idempotency, retry, timeout, and audit policies
- `spec` aggregation
- plugin discovery, installation, loading, handshakes, and health checks

### 4.2 Plugin Responsibilities

`clawrise-plugin-<platform>` should own:

- platform operation declarations
- provider-native request and response mapping
- platform auth details and token refresh flows
- provider-specific error mapping
- platform-level `spec` and catalog output

### 4.3 Boundary Rules

The core should not know provider-specific implementation details.

Plugins should not read the main core config file directly, and they should not interpret `env:` references, account defaults, or platform-default resolution rules.

Recommended boundary:

- the core resolves config and account selection
- the core sends already-resolved auth material and execution inputs to the plugin
- the plugin consumes execution requests without knowing where config came from

## 5. Transport

The first protocol version should use:

- transport: `stdio`
- message format: `JSON-RPC 2.0`
- plugin `stdout`: protocol messages only
- plugin `stderr`: logs only

Why:

- easy to implement across languages
- no port management needed
- a natural fit for local CLI plugins
- compatible with agent and tool-driven workflows

## 6. Minimal Protocol Surface

The first version should define only 5 core methods:

1. `clawrise.handshake`
2. `clawrise.operations.list`
3. `clawrise.catalog.get`
4. `clawrise.execute`
5. `clawrise.health`

Future extensions:

- `clawrise.auth.probe`
- `clawrise.spec.export`
- `clawrise.install.info`

### 6.1 `clawrise.handshake`

Purpose:

- protocol version negotiation
- plugin metadata
- capability reporting

Request:

```json
{
  "jsonrpc": "2.0",
  "id": "1",
  "method": "clawrise.handshake",
  "params": {
    "protocol_version": 1,
    "core": {
      "name": "clawrise",
      "version": "0.2.0"
    }
  }
}
```

Response:

```json
{
  "jsonrpc": "2.0",
  "id": "1",
  "result": {
    "protocol_version": 1,
    "plugin": {
      "name": "feishu",
      "version": "0.1.0"
    },
    "platforms": ["feishu"],
    "capabilities": {
      "operations_list": true,
      "catalog_get": true,
      "execute": true,
      "health": true,
      "auth_probe": false
    }
  }
}
```

### 6.2 `clawrise.operations.list`

Purpose:

- expose the operations implemented by the plugin
- return both execution metadata and discovery metadata
- provide the fact-source for `spec list/get`, completion, and routing

Request:

```json
{
  "jsonrpc": "2.0",
  "id": "2",
  "method": "clawrise.operations.list",
  "params": {}
}
```

Response:

```json
{
  "jsonrpc": "2.0",
  "id": "2",
  "result": {
    "operations": [
      {
        "operation": "feishu.calendar.event.create",
        "platform": "feishu",
        "mutating": true,
        "default_timeout_ms": 10000,
        "allowed_subjects": ["bot"],
        "spec": {
          "summary": "Create a Feishu calendar event.",
          "dry_run_supported": true,
          "input": {
            "required": ["calendar_id", "summary", "start_at", "end_at"],
            "optional": ["description", "location", "reminders", "timezone"],
            "sample": {
              "calendar_id": "cal_demo"
            }
          },
          "idempotency": {
            "required": true,
            "auto_generated": true
          }
        }
      }
    ]
  }
}
```

### 6.3 `clawrise.catalog.get`

Purpose:

- provide a structured catalog for `spec status`
- enable runtime-to-catalog reconciliation

Request:

```json
{
  "jsonrpc": "2.0",
  "id": "3",
  "method": "clawrise.catalog.get",
  "params": {}
}
```

Response:

```json
{
  "jsonrpc": "2.0",
  "id": "3",
  "result": {
    "entries": [
      { "operation": "feishu.calendar.event.create" },
      { "operation": "feishu.calendar.event.list" }
    ]
  }
}
```

### 6.4 `clawrise.execute`

Purpose:

- execute a concrete operation
- preserve a unified runtime envelope

Request:

```json
{
  "jsonrpc": "2.0",
  "id": "4",
  "method": "clawrise.execute",
  "params": {
    "request": {
      "request_id": "req_123",
      "operation": "feishu.calendar.event.create",
      "input": {
        "calendar_id": "cal_demo",
        "summary": "Weekly sync"
      },
      "timeout_ms": 10000,
      "idempotency_key": "idem_xxx",
      "dry_run": false
    },
    "identity": {
      "platform": "feishu",
      "subject": "bot",
      "profile_name": "feishu_bot",
      "auth": {
        "type": "client_credentials",
        "app_id": "resolved-app-id",
        "app_secret": "resolved-app-secret"
      }
    }
  }
}
```

Success:

```json
{
  "jsonrpc": "2.0",
  "id": "4",
  "result": {
    "ok": true,
    "data": {
      "event_id": "evt_123"
    },
    "error": null,
    "meta": {
      "provider_request_id": "",
      "retry_count": 0
    }
  }
}
```

Failure:

```json
{
  "jsonrpc": "2.0",
  "id": "4",
  "result": {
    "ok": false,
    "data": null,
    "error": {
      "code": "RESOURCE_NOT_FOUND",
      "message": "calendar not found",
      "retryable": false,
      "upstream_code": "191001",
      "http_status": 404
    },
    "meta": {
      "provider_request_id": "",
      "retry_count": 0
    }
  }
}
```

Rule:

- JSON-RPC `error` is reserved for protocol-layer failures
- business-level operation failures must still use `result.ok=false`

### 6.5 `clawrise.health`

Purpose:

- liveness checks
- minimal diagnostics

Request:

```json
{
  "jsonrpc": "2.0",
  "id": "5",
  "method": "clawrise.health",
  "params": {}
}
```

Response:

```json
{
  "jsonrpc": "2.0",
  "id": "5",
  "result": {
    "ok": true,
    "details": {
      "plugin_name": "feishu",
      "plugin_version": "0.1.0"
    }
  }
}
```

## 7. Manifest

Each plugin directory must contain `plugin.json`.

Recommended shape:

```json
{
  "schema_version": 1,
  "name": "feishu",
  "version": "0.1.0",
  "kind": "provider",
  "protocol_version": 1,
  "platforms": ["feishu"],
  "entry": {
    "type": "binary",
    "command": ["./bin/clawrise-plugin-feishu"]
  },
  "catalog_path": "./catalog/operations.json",
  "min_core_version": "0.2.0"
}
```

Field notes:

- `schema_version`: manifest schema version
- `name`: plugin name
- `version`: plugin version
- `kind`: currently fixed to `provider`
- `protocol_version`: plugin protocol version
- `platforms`: provider platforms handled by the plugin
- `entry.command`: plugin launch command
- `catalog_path`: optional static catalog location
- `min_core_version`: minimum supported core version

## 8. Directory Layout

Recommended global layout:

```text
~/.clawrise/plugins/
  feishu/
    0.1.0/
      plugin.json
      bin/clawrise-plugin-feishu
      catalog/operations.json
  notion/
    0.1.0/
      plugin.json
      bin/clawrise-plugin-notion
```

Also support:

- project-local plugins: `.clawrise/plugins/`
- environment override: `CLAWRISE_PLUGIN_PATHS`

Suggested discovery priority:

1. explicit environment paths
2. project-local paths
3. global paths

## 9. Distribution Model

The plugin mechanism must stay decoupled from package distribution.

Recommended source schemes:

- local directory
- `file://`
- `https://`
- direct npm package specs
- `npm://`
- `registry://`
- future `gh://`

Examples:

```bash
clawrise plugin install file:///tmp/clawrise-plugin-feishu.tar.gz
clawrise plugin install https://example.com/clawrise-plugin-feishu.tar.gz
clawrise plugin install @clawrise/clawrise-plugin-feishu
clawrise plugin install npm://@clawrise/clawrise-plugin-feishu
clawrise plugin install registry://community/clawrise-plugin-feishu
```

If `npm` is used:

- it should be treated as an installation source only
- runtime should still execute a prebuilt native binary
- users should not be forced to depend on Node at execution time

## 10. Core Abstraction

The core should introduce a provider runtime abstraction layer.

Suggested Go interface:

```go
type ProviderRuntime interface {
    Handshake(ctx context.Context) (HandshakeResult, error)
    ListOperations(ctx context.Context) ([]adapter.Definition, error)
    GetCatalog(ctx context.Context) ([]catalog.Entry, error)
    Execute(ctx context.Context, req ExecuteRequest) (ExecuteResult, error)
    Health(ctx context.Context) (HealthResult, error)
}
```

### 10.1 New Core Modules

Suggested additions:

- `internal/plugin/manifest`
- `internal/plugin/discovery`
- `internal/plugin/runtime`
- `internal/plugin/protocol`
- `internal/plugin/install`

### 10.2 Runtime Flow

```text
CLI input
  -> resolve operation and flags
  -> load config / resolve profile
  -> resolve installed plugin by platform
  -> lazy start plugin process
  -> handshake
  -> execute via JSON-RPC
  -> normalize envelope
  -> write audit record
```

## 11. Suggested Go Message Types

### 11.1 Common RPC Types

```go
type RPCRequest struct {
    JSONRPC string `json:"jsonrpc"`
    ID      string `json:"id"`
    Method  string `json:"method"`
    Params  any    `json:"params,omitempty"`
}

type RPCResponse struct {
    JSONRPC string    `json:"jsonrpc"`
    ID      string    `json:"id"`
    Result  any       `json:"result,omitempty"`
    Error   *RPCError `json:"error,omitempty"`
}

type RPCError struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    Data    any    `json:"data,omitempty"`
}
```

### 11.2 Execution Request

```go
type ExecuteRequest struct {
    Request  ExecuteEnvelope `json:"request"`
    Identity ExecuteIdentity `json:"identity"`
}

type ExecuteEnvelope struct {
    RequestID      string         `json:"request_id"`
    Operation      string         `json:"operation"`
    Input          map[string]any `json:"input"`
    TimeoutMS      int64          `json:"timeout_ms"`
    IdempotencyKey string         `json:"idempotency_key,omitempty"`
    DryRun         bool           `json:"dry_run"`
}

type ExecuteIdentity struct {
    Platform    string         `json:"platform"`
    Subject     string         `json:"subject"`
    ProfileName string         `json:"profile_name"`
    Auth        map[string]any `json:"auth"`
}
```

### 11.3 Execution Result

```go
type ExecuteResult struct {
    OK    bool              `json:"ok"`
    Data  any               `json:"data"`
    Error *ExecuteErrorBody `json:"error,omitempty"`
    Meta  ExecuteMeta       `json:"meta"`
}

type ExecuteErrorBody struct {
    Code         string `json:"code"`
    Message      string `json:"message"`
    Retryable    bool   `json:"retryable"`
    UpstreamCode string `json:"upstream_code,omitempty"`
    HTTPStatus   int    `json:"http_status,omitempty"`
}

type ExecuteMeta struct {
    ProviderRequestID string `json:"provider_request_id,omitempty"`
    RetryCount        int    `json:"retry_count"`
}
```

## 12. Auth Boundary

Recommended first version:

- the core resolves config and secrets
- the core passes resolved auth material to the plugin
- plugins do not read the main config file directly

Why:

- avoids re-implementing config parsing in each plugin language
- keeps `env:` and similar syntax out of the wire protocol
- keeps configuration semantics centralized in the core

Future extensions may allow:

- plugin-defined auth schemas
- core-side auth prompting and storage
- plugin-owned refresh and provider-native auth operations

## 13. Version Compatibility

The system must track:

- manifest schema version
- plugin protocol version
- core version
- plugin version

When loading a plugin, the core should:

1. validate `plugin.json`
2. check `schema_version`
3. check `min_core_version`
4. start the process and run `handshake`
5. verify `protocol_version`

Any failure should reject loading with an explicit error.

## 14. Security Boundary

Plugins are local code execution. They should not be presented as a sandbox.

Minimum recommendations:

- do not pass credentials via process arguments
- pass credentials via protocol payloads or controlled stdin
- forbid non-protocol output on plugin `stdout`
- allow logs on `stderr`, but redact sensitive content in the core
- record source, version, and checksum on installation
- add signature verification later

## 15. Why Not Go `plugin`

The Go `plugin` package is not recommended here because:

- platform support is limited and unsuitable for a cross-platform CLI
- plugins and the main binary must share an extremely strict build environment
- debugging and operations are harder
- isolation is weaker than an external process model

For a long-lived extensible CLI like Clawrise, external process plugins are the safer default.

Reference:

- Go `plugin` package: <https://pkg.go.dev/plugin>

## 16. Recommended Plugin Languages

Recommended defaults:

- first-party plugins: `Go`
- protocol: language-agnostic
- third-party community plugins: may later use `TypeScript` or other languages

Why:

- current Feishu / Notion code already exists in Go
- first-party migration cost is lowest in Go
- a language-neutral protocol keeps the ecosystem open

## 17. Phase Status Summary

- `M1` through `M3` are complete and define the current plugin-first baseline.
- `M4` first operational phase is complete. Installation, verification, and upgrade are already part of the public CLI surface, while the remaining work is hardening around release and trust behavior.
- `M5` first operational phase is complete. `policy` and `audit_sink` capabilities already participate in the execution path and in diagnostics.
- `M6` first extension phase is complete. `workflow.plan` and `registry_source` now exist as protocol-level extension points, while higher-level orchestration remains optional rather than forced into the core runtime.

## 18. Direct Impact on the Current Repository

The current repository already reflects this architecture in several concrete places:

- `internal/cli/root.go` still owns the shared CLI bootstrap and cross-cutting runtime wiring
- `spec` aggregates operations and catalog metadata from plugins instead of relying on static provider knowledge
- first-party Feishu and Notion execution flows are routed through provider runtimes rather than direct in-core provider handlers

## 19. Current Operational Baseline

The current operational baseline in this repository is:

- first-party Feishu and Notion ship as external plugin binaries
- local and remote installation paths coexist, including direct npm package specs and `registry://`
- the core/plugin contract is centered on `handshake`, `operations.list`, `catalog.get`, `execute`, and `health`
- trust verification and upgrade behavior are part of the normal CLI surface
- higher-level workflow planning remains optional and capability-driven instead of being forced into the core runtime
