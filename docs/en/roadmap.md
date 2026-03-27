# Clawrise Roadmap

## 1. Purpose

This document tracks Clawrise's current near-term priorities after the plugin-first architecture shift.

It separates:

- foundations that are already in place
- near-term work that should happen next
- work that still belongs later

## 2. Current Direction

Clawrise is now moving toward:

- `clawrise` as the core runtime and CLI
- provider capabilities delivered through external plugins
- first-party Feishu and Notion shipped as first-party provider plugins
- one unified runtime envelope, one unified `spec` surface, and provider-native business schemas

## 3. Completed Foundations

The repository already has:

- a unified runtime and config model
- a substantial set of real Feishu and Notion operations
- `clawrise spec list [path]`
- `clawrise spec get <operation>`
- `clawrise spec status`
- structured operation catalog and registry-to-catalog drift checks
- metadata completeness tests
- provider runtime abstraction in the core
- external-process plugin runtime over `stdio + JSON-RPC`
- first-party Feishu and Notion plugin binaries
- plugin management commands:
  - `clawrise plugin list`
  - `clawrise plugin install <source>`
  - `clawrise plugin info <name> <version>`
  - `clawrise plugin remove <name> <version>`
- current plugin install sources:
  - local directory
  - `file://`
  - `https://`
  - `npm://`

## 4. Near-term Must-have

These are the highest-priority remaining tasks.

### 4.1 Onboarding and First-party Plugin UX

Why:

- the architecture is now plugin-first, so installation and first-run experience matter more than before
- the current implementation is usable, but still too manual for non-developers

Deliverables:

- clearer quickstart for core + plugin installation
- official packaging conventions for first-party plugins
- stronger `doctor`
- `plugin verify` or an equivalent checksum / trust surface
- a minimal `auth` helper surface for inspection and setup checks
- `config init` or an equivalent bootstrap flow

Completion signal:

- a new user can install one official plugin and complete one real call through a short documented path
- common setup failures can be diagnosed without reading implementation details

### 4.2 Local Recipes / Playbooks

Why:

- task-oriented guidance is still missing even though capability discovery now exists
- this is the bridge from operation discovery to real task execution for both humans and agents

Deliverables:

- `docs/recipes` or `docs/playbooks`
- a searchable index such as `index.yaml`
- reusable recipes for:
  - updating a Feishu document
  - updating a Feishu Bitable record
  - creating or updating a Feishu calendar event
  - updating Notion page content
  - querying a Notion data source

Completion signal:

- common tasks can be found through local search
- recipe inputs and command templates are reusable and verifiable

### 4.3 Runtime Governance

Why:

- write paths still need stronger operational guarantees before broad use

Deliverables:

- persisted idempotency state
- basic audit records
- configurable retry behavior
- clearer secret redaction rules

Completion signal:

- write paths can report persisted idempotency state
- audit output does not leak secrets
- retry behavior is visible in normalized metadata

### 4.4 `spec export`, Completion, and Generated Docs

Why:

- `spec` discovery is now in place, but machine-readable export and generated consumers are still missing

Deliverables:

- `clawrise spec export`
- `completion` driven by the same provider metadata used by `spec`
- progressively generated operation docs from registry and catalog metadata

Completion signal:

- machine consumers can fetch a complete export
- completion no longer depends on a separate hand-maintained command tree
- operation docs start converging on structured metadata instead of manual drift

## 5. Should-have

These are valuable, but they should sit after the Must-have layer above.

### 5.1 Official `clawrise-operator` Skill

Why:

- helpful for agents using Clawrise reliably
- should be built on top of `spec`, catalog, recipes, and plugin-aware onboarding

### 5.2 Developer-facing `clawrise-builder` Skill

Why:

- useful for provider extension work
- still lower priority than operator-facing and onboarding-facing material

### 5.3 Plugin Hardening and Distribution Ops

Why:

- plugin architecture now exists, but release operations and trust policy still need stronger productization

Suggested scope:

- plugin release manifests
- checksum policy
- signature policy
- upgrade strategy
- official distribution channels

## 6. Can Wait

These can still be deferred explicitly:

- a public plugin marketplace
- untrusted plugin sandboxing
- a REPL-first interactive shell
- a full JSON Schema framework
- a cross-platform workflow engine

## 7. Recommended Order

1. onboarding and first-party plugin UX
2. local recipes / playbooks
3. runtime governance
4. `spec export`, completion, and generated docs
5. official `clawrise-operator` skill
6. plugin hardening and distribution ops
7. developer-facing `clawrise-builder` skill

## 8. Risks and Cautions

### 8.1 Do Not Reintroduce Hard-coded Providers into the Core

The plugin-first direction should remain the default architecture.

### 8.2 Do Not Fork Discovery Metadata Across Core, Plugins, Docs, and Recipes

`spec`, catalog, docs, completion, and recipes should keep converging on the same metadata layer.

### 8.3 Do Not Ship Remote Installation Without Trust Policy

`https://` and `npm://` support now exist, but release and trust policy still need productization.

### 8.4 Do Not Treat Plugin Install as the End of Onboarding

Installation alone is not enough. Auth setup, profile selection, sample inputs, and diagnostics still determine practical usability.

## 9. Completion Signal

The current near-term roadmap can be considered complete when:

- users can install and verify official first-party plugins easily
- common tasks are covered by local searchable playbooks
- write paths have stronger idempotency and audit guarantees
- `spec export` and completion are driven by the same provider metadata as `spec`
- official operator-facing material reuses structured metadata instead of hand-maintained command lists
