# Clawrise `spec` Subsystem Design

See the Chinese version at [../zh/spec-design.md](../zh/spec-design.md).

## 1. Purpose

This document describes the `clawrise spec` command surface, data model, status model, and implementation layout.

The point of `spec` is not to duplicate Markdown documentation. The point is to make Clawrise itself the structured source of truth for currently discoverable capabilities.

## 2. Current Implementation Status

Implemented today:

- `clawrise spec list [path]`
- `clawrise spec get <operation>`
- hierarchical browsing over registered operations
- operation detail views backed by registry metadata

Still planned:

- `clawrise spec status`
- `clawrise spec export`
- catalog-backed drift analysis

## 3. Design Goals

The `spec` subsystem should:

- let both humans and agents discover supported operations directly from the CLI
- keep default output bounded as operation count grows
- reuse one structured metadata layer across registry, docs, completion, and tests
- distinguish runtime facts from planned catalog declarations
- preserve Clawrise's boundary of unified runtime semantics with provider-native business schemas

## 4. Command Surface

The current and planned `spec` commands are:

- `clawrise spec list [path]`
- `clawrise spec get <operation>`
- `clawrise spec status`
- `clawrise spec export`

Current behavior:

- `list` is implemented
- `get` is implemented
- `status` is reserved for the next milestone
- `export` is reserved for a later milestone

## 5. Hierarchical Browse Model

`spec list` is intentionally a path browser, not a flat export.

Clawrise operation naming is already hierarchical:

```text
<platform>.<resource-path>.<action>
```

Examples:

- `clawrise spec list`
  - returns platforms such as `feishu` and `notion`
- `clawrise spec list feishu`
  - returns direct groups such as `calendar`, `docs`, `wiki`, and `contact`
- `clawrise spec list feishu.docs.document`
  - returns leaf operations such as `create`, `get`, and `list_blocks`

Current node types:

- `root`
- `platform`
- `group`
- `operation`

## 6. Detail View Model

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

The current detail view is runtime-driven only. Catalog-backed status fields are planned for a later phase.

## 7. Source-of-Truth Model

The target model has two layers:

- `Runtime Registry`
- `Catalog`

Current implementation only uses the runtime registry.

That means:

- `spec list/get` reflect what the current binary knows about
- they can still show stubbed operations if an operation is registered without a handler
- they do not yet report catalog drift

Catalog support is planned for `M2`.

## 8. Metadata Model

The adapter registry now stores both execution metadata and lightweight discovery metadata.

Current registry metadata includes:

- operation identity
- platform
- mutating flag
- default timeout
- allowed subjects
- handler binding
- discovery spec such as summary, input fields, notes, examples, and idempotency behavior

The current `spec` metadata is intentionally lightweight. It does not attempt to be a full JSON Schema system yet.

## 9. Current File Layout

Current implementation files:

- `internal/spec/types.go`
- `internal/spec/service.go`
- `internal/cli/spec.go`
- `internal/adapter/registry.go`

Current metadata placement follows adapter structure instead of a single platform-wide table. Examples:

- `internal/adapter/feishu/calendar_spec.go`
- `internal/adapter/feishu/docx_spec.go`
- `internal/adapter/notion/page_spec.go`
- `internal/adapter/notion/block_spec.go`

This keeps discovery metadata aligned with the existing adapter module boundaries.

## 10. Planned Next Step

The next design step is `M2`:

- add a structured operation catalog
- implement `spec status`
- reconcile runtime registration against catalog declarations
- add metadata completeness checks in tests

## 11. Relationship to Other Subsystems

### 11.1 Completion

`completion` should eventually consume the `spec` tree instead of maintaining a separate command tree.

### 11.2 Documentation

Over time, operation docs should be progressively generated from registry and catalog metadata rather than maintained entirely by hand.

### 11.3 Tests

`spec` should be treated as a contract layer. Tests should cover:

- hierarchical browse behavior
- operation detail behavior
- stubbed operation reporting
- future catalog drift reporting
