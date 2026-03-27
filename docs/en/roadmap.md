# Clawrise Roadmap

## 1. Purpose

This document captures Clawrise's near-term priorities and separates them into:

- what is required soon
- what is worth doing after the foundations are stable
- what can be deliberately deferred

The detailed `spec` subsystem design lives in [spec-design.md](spec-design.md).

## 2. Evaluation Criteria

An item should enter the near-term roadmap only if it answers at least one of these questions well:

- is it a real blocker for practical usage
- is it a dependency for several later capabilities
- does it reduce the risk of maintaining multiple drifting sources of truth

## 3. Current State

The repository already has:

- a unified runtime and config model
- a growing set of real Feishu and Notion operations
- `clawrise spec list [path]`
- `clawrise spec get <operation>`

It is still short of the level needed for broad, stable use by both humans and agents.

Current gaps include:

- discovery is still incomplete because `spec status`, `spec export`, and `completion` are missing
- docs and implementation can still drift because there is no catalog reconciliation yet
- onboarding is still rough because install, config, auth, examples, and troubleshooting are not streamlined enough
- AI usage material is still thin because the repo does not yet ship an official skill or a task-oriented operator guide
- common tasks are not yet captured as searchable local recipes or playbooks
- idempotency, audit, and retry behavior are still only partially realized

## 4. Must-have

These should be in the near-term mainline roadmap.

### 4.1 `spec status + catalog`

Why:

- this is the source-of-truth foundation for docs, recipes, skills, and completion
- without it, the repo will continue to drift across code, docs, and AI-facing material

Deliverables:

- `clawrise spec status`
- structured operation catalog
- registry-to-catalog diff logic
- metadata completeness tests

Completion signal:

- implemented, stubbed, and declared-but-missing operations can be distinguished clearly
- missing metadata or missing catalog coverage is caught by tests

### 4.2 Onboarding Friendliness

Why:

- this determines whether the project moves from "technically usable" to "practically adoptable"

Deliverables:

- `config init` or an equivalent bootstrap flow
- a shorter quickstart
- reusable sample configs and sample inputs
- stronger `doctor`
- a minimal `auth` helper surface for inspection or setup checks

Completion signal:

- a new user can complete one real call through the shortest documented path
- common setup errors can be diagnosed quickly through docs or `doctor`

### 4.3 Local Recipes / Playbooks

Why:

- this is the bridge from capability discovery to task execution
- it helps both humans and agents
- it is lighter and safer to build before an official skill

Suggested shape:

- `docs/recipes` or `docs/playbooks`
- a small searchable index such as `index.yaml`
- one task-focused recipe per common workflow:
  - update a specific Feishu document
  - update a specific Feishu table or record
  - create a Feishu calendar event
  - update Notion page content
  - query a Notion data source

Completion signal:

- common tasks can be found quickly through local search
- command templates and input samples are reusable and verifiable

### 4.4 Basic Runtime Governance

Why:

- without realistic idempotency, audit, and retry behavior, write paths still carry too much operational risk

Deliverables:

- persisted idempotency state
- basic audit records
- configurable retry behavior
- clearer secret redaction rules

Completion signal:

- write paths can report idempotency state
- audit output does not leak secrets
- retry behavior is visible in normalized metadata

## 5. Should-have

These are valuable, but they should sit on top of the Must-have layer.

### 5.1 Official `clawrise-operator` Skill

Why:

- it can help agents learn how to use Clawrise more reliably
- it should be built on top of `spec`, catalog, and recipes rather than replacing them

Suggested scope:

- how to choose `platform / profile / subject`
- how to explore capability through `spec`
- how to find a matching recipe
- how to handle common failures

### 5.2 `completion`, Generated Docs, and `spec export`

Why:

- important, but better treated as force multipliers once the metadata base is stable

Suggested deliverables:

- `completion` driven by `spec`
- progressively generated operation docs
- machine-readable `spec export`

### 5.3 Developer-facing `clawrise-builder` Skill

Why:

- useful for extension and adapter work
- still lower priority than the operator-facing skill

## 6. Can Wait

These can be deferred explicitly:

- a dynamic plugin system
- a REPL-first interactive shell
- a full JSON Schema framework
- a cross-platform workflow engine
- premature external distribution work

## 7. Recommended Order

1. `spec status + catalog`
2. onboarding friendliness
3. local recipes / playbooks
4. basic runtime governance
5. official `clawrise-operator` skill
6. `completion` / generated docs / `spec export`
7. developer-facing `clawrise-builder` skill

## 8. Risks and Cautions

### 8.1 Do Not Turn `spec` into a Documentation Copy

`spec` should primarily serve as structured runtime truth, not as a duplicate Markdown layer.

### 8.2 Do Not Default to Flat Global Listings

As operation count grows, `spec list` must stay hierarchical by default.

### 8.3 Do Not Collapse Catalog and Runtime into One Layer

A single idealized operation list cannot accurately represent what the current binary can actually do.

### 8.4 Do Not Create a Second Knowledge Base in Skills and Recipes

Official skills, local recipes, and onboarding docs should reuse `spec`, catalog, and structured samples instead of maintaining separate command knowledge.

## 9. Completion Signal

This near-term roadmap can be considered complete when:

- runtime capabilities can be discovered hierarchically through `spec`
- implementation drift can be reported explicitly through `spec status`
- new users can get through onboarding with much less friction
- common tasks are covered by local searchable recipes or playbooks
- write paths have more realistic idempotency and audit behavior
- official skills and local playbooks start consuming structured metadata instead of manual command lists
