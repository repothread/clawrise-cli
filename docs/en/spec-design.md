# Clawrise `spec` Subsystem Design

See the Chinese version at [../zh/spec-design.md](../zh/spec-design.md).

## 1. Purpose

The `spec` subsystem exists to make Clawrise itself the structured source of truth for discoverable capabilities.

It is not meant to duplicate Markdown documentation. It is meant to expose:

- what operations are currently discoverable
- what metadata is attached to those operations
- what the current runtime can really execute
- how runtime and catalog declarations differ

## 2. Current Implementation Status

Implemented today:

- `clawrise spec list [path]`
- `clawrise spec get <operation>`
- `clawrise spec status`
- `clawrise spec export [path] [--format json|markdown]`
- `clawrise completion <bash|zsh|fish>`
- hierarchical discovery over the current runtime registry
- catalog-backed runtime drift analysis
- Markdown doc export from the same metadata layer
- metadata completeness checks in tests

## 3. Current Runtime Model

Clawrise now uses a plugin-first provider architecture.

That means the `spec` subsystem reads from two aggregated layers:

- runtime operation registry aggregated from provider runtimes
- structured catalog aggregated from provider runtimes

In the current repository:

- first-party Feishu and Notion plugins expose operations through the plugin runtime interface
- the core aggregates those provider runtimes into one registry view
- `spec` is built on top of that aggregated view

## 4. Design Goals

The `spec` subsystem should:

- let humans and agents discover operations directly from the CLI
- keep default output bounded as operation count grows
- reuse one metadata layer across runtime, docs, completion, and tests
- distinguish runtime facts from catalog declarations
- preserve Clawrise's boundary of unified runtime semantics with provider-native business schemas

## 5. Command Surface

Current command surface:

- `clawrise spec list [path]`
- `clawrise spec get <operation>`
- `clawrise spec status`
- `clawrise spec export`

Current behavior:

- `list` is implemented
- `get` is implemented
- `status` is implemented
- `export` is implemented

## 6. Hierarchical Browse Model

`spec list` is intentionally a hierarchical browser, not a flat export.

Operation naming remains:

```text
<platform>.<resource-path>.<action>
```

Current node types:

- `root`
- `platform`
- `group`
- `operation`

Default behavior should remain:

- hierarchical
- bounded
- summary-first
- not a machine-export substitute

## 7. Detail View Model

`spec get` returns one operation detail record.

Current fields include:

- `operation`
- `platform`
- `resource_path`
- `action`
- `summary`
- `description`
- `allowed_subjects`
- `mutating`
- `implemented`
- `dry_run_supported`
- `default_timeout_ms`
- `idempotency`
- `input`
- `examples`
- `runtime_status`

The current detail view remains runtime-driven. `spec status` is where runtime/catalog drift is reported explicitly.

## 8. Source-of-Truth Model

The current model has two explicit layers:

- `Runtime Registry`
- `Catalog`

### Runtime Registry

The runtime registry represents operations the current binary can discover through loaded provider runtimes.

It answers:

- what operations are exposed right now
- what metadata is attached right now
- whether an operation is currently implemented

### Catalog

The catalog represents the structured declaration set the project recognizes for the current provider runtimes.

It answers:

- which operations are declared
- which runtime operations are missing catalog coverage
- which catalog entries are missing from runtime

## 9. Status Model

`spec status` is a governance surface, not a browse surface.

It should report:

- total registered operations
- implemented vs stubbed counts
- total catalog declarations
- runtime-present but catalog-missing operations
- catalog-declared but runtime-missing operations

Current status labels:

- runtime:
  - `registered_and_implemented`
  - `registered_but_stubbed`
  - `runtime_missing`
- catalog:
  - `declared`
  - `catalog_missing`

## 10. Metadata Model

The current operation metadata layer is still intentionally lightweight.

It includes:

- execution metadata:
  - operation name
  - platform
  - mutating flag
  - default timeout
  - allowed subjects
- discovery metadata:
  - summary
  - description
  - dry-run support
  - input fields
  - examples
  - idempotency behavior

It still does not try to be a full JSON Schema system.

## 11. Current File Layout

Current implementation files include:

- `internal/spec/types.go`
- `internal/spec/service.go`
- `internal/spec/status.go`
- `internal/spec/catalog/*`
- `internal/cli/spec.go`

Current provider aggregation lives under:

- `internal/plugin/runtime.go`
- `internal/plugin/process.go`
- `internal/plugin/registry_runtime.go`

The important architectural point is:

- `spec` no longer assumes providers are hard-coded directly into the core
- it consumes the aggregated runtime and catalog view built by the provider runtime layer

## 12. Relationship to Other Subsystems

### 12.1 Completion

`completion` should consume the same metadata layer as `spec` rather than maintaining a separate command tree.

### 12.2 Documentation

Operation docs should progressively be generated from provider metadata and catalog entries.

### 12.3 Tests

`spec` should be treated as a contract layer.

Tests should cover:

- hierarchical browse behavior
- operation detail behavior
- runtime/catalog drift reporting
- metadata completeness

## 13. Next Step

The next major step after the current implementation is:

- broaden the downstream uses of `spec export`
- let more docs consume the exported metadata directly
- keep completion and generated docs converging on the same metadata layer

## 14. Completion Signal

The near-term `spec` work can be considered complete when:

- runtime capabilities are discoverable through `list/get`
- runtime/catalog drift is visible through `status`
- `export` exists for machine consumers
- completion and docs stop depending on separate hand-maintained command knowledge
