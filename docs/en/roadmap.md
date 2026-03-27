# Clawrise OSS Roadmap

## 1. Purpose

This document tracks forward-looking OSS core priorities only.

It intentionally does not repeat shipped work. Completed capabilities should live in the repository introduction and `README` status instead of staying in the roadmap forever.

## 2. Current OSS Product Boundary

In this repository, Clawrise should stay focused on:

- the core runtime and CLI
- the provider plugin protocol and first-party plugin baseline
- `spec`, completion, and metadata-driven reference surfaces
- local governance, diagnostics, and playbook support

## 3. Near-term Must-have

### 3.1 Official First-party Plugin Release Workflow

Why:

- the plugin-first runtime is already real, but official plugin delivery is still too ad hoc
- users still need a clear, documented path for install, upgrade, and compatibility checks

Deliverables:

- versioned first-party plugin release artifacts
- a documented release manifest shape
- compatibility fields between core and plugin versions
- a documented install and upgrade path for official first-party plugins
- `plugin verify` behavior that can consume published release metadata cleanly

Completion signal:

- a user can install and upgrade an official first-party plugin through a short documented path
- compatibility mismatches are visible before a real execution attempt

### 3.2 Remote-source Trust and Verification Policy

Why:

- `https://` and `npm://` sources already exist, but trust policy is still incomplete
- remote install support without a clearer trust model is not enough for broader adoption

Deliverables:

- a documented trust model for remote plugin sources
- checksum policy and clearer verification semantics
- visible trust and verification results in plugin inspection or verify output
- explicit failure behavior for tampered, incompatible, or incomplete plugin artifacts
- reserved extension points for stronger signature policy later

Completion signal:

- remote plugin install and verify can explain what was checked, what was trusted, and why a plugin was rejected

### 3.3 Onboarding to the First Successful Call

Why:

- the command surface is usable now, but the first practical path is still too manual
- the project needs a shorter path from fresh install to one successful real call

Deliverables:

- a tighter quickstart for core plus first-party plugin setup
- a short documented flow through `config init`, `auth check`, `doctor`, and one real call
- sample inputs that match the current CLI shapes
- clearer links between playbooks and the first runnable operations

Completion signal:

- a new user can get from fresh install to one real call without reading the design documents first

### 3.4 Plugin Authoring and Compatibility DX

Why:

- the open ecosystem cannot grow if third-party plugin authors must reverse-engineer the core
- the plugin protocol exists, but the authoring path still needs better productization

Deliverables:

- a concise plugin author guide
- manifest and compatibility reference docs
- a local validation or compatibility-check flow for plugin authors
- clearer test guidance for plugin handshake, catalog, and execution behavior

Completion signal:

- a third-party author can build and validate a minimal plugin without reading core internals line by line

### 3.5 Metadata-driven Operation Reference

Why:

- `spec export` and completion already exist, but downstream docs can still drift from runtime facts
- the metadata layer should keep becoming the shared source for runtime, docs, and discovery

Deliverables:

- a stable exported metadata contract for downstream consumers
- generated operation reference material from the same metadata layer used by `spec`
- clearer linkage between runtime registry facts, catalog declarations, completion, and generated docs

Completion signal:

- operation reference material is derived from the same metadata layer as `spec export` and completion rather than maintained separately

## 4. Should-have After the Must-have Layer

### 4.1 Broader First-party Provider Surface

Why:

- broader provider coverage is useful, but it should come after release workflow, trust policy, and onboarding are stable

Notes:

- `google` remains a candidate next provider
- it should not become the immediate next milestone before the plugin-first core is better hardened

### 4.2 Expanded Locally Searchable Playbooks

Why:

- the current playbooks are a good baseline, but task coverage should continue to expand around the existing first-party providers

Scope:

- add more high-signal task playbooks for Feishu and Notion
- keep examples close to real CLI input shapes and verifiable paths

## 5. Can Wait

These are still valid ideas, but they should stay clearly behind the Must-have layer above:

- a public plugin marketplace
- untrusted plugin sandboxing
- a REPL-first interactive shell
- a full JSON Schema framework
- a cross-platform workflow engine

## 6. Recommended Order

1. official first-party plugin release workflow
2. remote-source trust and verification policy
3. onboarding to the first successful call
4. plugin authoring and compatibility DX
5. metadata-driven operation reference
6. broader first-party provider surface
7. expanded locally searchable playbooks

## 7. Risks and Cautions

### 7.1 Do Not Reintroduce Hard-coded Providers into the Core

The plugin-first direction should remain the default architecture.

### 7.2 Do Not Fork Metadata Across Runtime, `spec`, Docs, and Completion

Runtime facts, `spec`, generated docs, completion, and playbooks should keep converging on the same metadata layer instead of drifting apart.

### 7.3 Do Not Treat Remote Install Support as a Finished Trust Model

Remote sources already work, but release and trust hardening still need deliberate product work.

### 7.4 Do Not Expand Provider Surface Before Packaging and Onboarding Are Stable

Adding more providers too early risks increasing surface area while the first-run and release path still feel unfinished.

## 8. Completion Signal for the Next Phase

The next OSS-core phase can be considered complete when:

- official first-party plugins have a documented release, install, and upgrade path
- remote install has clear trust and verification behavior
- a new user can reach one real call through a short documented path
- a third-party plugin author can build and validate against a documented compatibility contract
- generated operation reference material reuses the same metadata layer as `spec export` and completion
