# Clawrise Roadmap

## 1. Purpose

This document captures Clawrise's near-term delivery priorities and clarifies:

- which gaps matter most right now
- what each milestone is meant to complete
- what is intentionally out of scope for the current phase

The detailed `spec` subsystem design lives in [spec-design.md](spec-design.md).

## 2. Current State

The repository already includes a unified runtime, config model, adapter registration, and a growing set of real Feishu and Notion operations. It is still short of the level of self-description and runtime governance needed for broad agent use.

Current highlights:

- `clawrise spec list [path]` and `clawrise spec get <operation>` are implemented
- operation metadata is now attached to the adapter registry
- `spec status` and `spec export` are still planned, not implemented

Remaining problem areas:

- discovery is only partially complete because `status`, `export`, and `completion` are still missing
- documentation and implementation can still drift because there is no catalog diff yet
- idempotency, retry, audit, and rate-limit behavior are still only partially realized at runtime
- extension remains code-driven rather than catalog-driven

## 3. Near-Term Goals

The near-term goal is not to turn Clawrise into a REPL, workflow engine, or MCP replacement. The goal is to complete it as a practical agent-native CLI execution layer with stronger self-description and governance.

We want to reach a state where:

- current runtime capabilities are discoverable from the CLI itself
- operation metadata is the primary source of truth
- documentation, runtime registration, and tests can be reconciled automatically
- mutating operations have more realistic idempotency and audit support
- completion and generated docs can later reuse the same structured metadata

## 4. Milestones

### 4.1 M1: Minimal `spec` Loop

Status:

- completed

Delivered:

- `clawrise spec list [path]`
- `clawrise spec get <operation>`
- extended adapter registry metadata
- per-operation summaries, input fields, examples, and implementation state

Exit criteria:

- root-level platforms can be listed
- operation paths can be browsed hierarchically
- a single operation can be inspected without credentials

### 4.2 M2: Status and Drift Control

Goal:

- distinguish implemented, stubbed, and planned operations clearly
- prevent docs, runtime code, and tests from drifting further apart

Deliverables:

- `clawrise spec status`
- structured operation catalog
- registry-to-catalog diff logic
- metadata completeness tests

Exit criteria:

- stubbed operations can be identified
- catalog-declared but runtime-missing operations are surfaced
- adding a new operation without matching metadata or catalog coverage is caught by tests

### 4.3 M3: Runtime Governance

Goal:

- move idempotency, audit, and retry from "present in shape" to "actually usable"

Deliverables:

- persisted idempotency state
- basic audit records
- configurable retry policy and error classification
- stronger secret redaction rules

### 4.4 M4: Completion and Generated Docs

Goal:

- turn `spec` into a shared base layer for multiple consumers

Deliverables:

- `completion` driven by the `spec` tree
- progressively generated Chinese and English operation docs
- machine-oriented `spec export`

## 5. Out of Scope

The current phase explicitly does not target:

- a REPL-first interactive product
- a complex cross-platform workflow engine
- forced provider schema unification
- a heavy plugin system up front
- a full JSON Schema generation and validation framework up front

## 6. Recommended Order

1. `spec status` plus catalog
2. idempotency, audit, and retry behavior
3. completion and generated docs

## 7. Completion Signal

This roadmap phase can be considered complete when:

- runtime capabilities can be discovered hierarchically through `spec`
- implementation drift can be reported explicitly through `spec status`
- mutating operations have stronger idempotency and audit behavior
- completion and docs start consuming structured metadata instead of manual lists
