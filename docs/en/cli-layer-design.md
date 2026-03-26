# Clawrise CLI Layer Design

See the Chinese version at [../zh/cli-layer-design.md](../zh/cli-layer-design.md).

## 1. Product Definition

Clawrise is an agent-native CLI execution layer, not a generic SDK wrapper.

Its job is to turn third-party SaaS APIs into stable command-style operations that AI agents can call directly.

Core positioning:

- `CLI = system-call interface for AI`
- `Clawrise = agent-native CLI layer`
- `MCP` is better treated as a governance layer, while `CLI` is the execution layer

## 2. Design Goals

Clawrise should provide:

- low token overhead
- stable invocation contracts
- adapter-based extensibility
- controllable auth, retry, timeout, rate limit, idempotency, and audit behavior
- normalized machine-friendly output

It should not start as:

- a GUI product
- a workflow engine
- a massive hand-written command tree
- an MCP-first execution path

Modeling boundary:

- Clawrise should unify runtime semantics, not provider resource schemas.
- Business resource fields should remain provider-native.
- Feishu docs, Notion pages, calendars, sheets, and future APIs must not be forced into one shared global field model.
- If a cross-platform abstraction becomes useful later, it should be added as an optional higher-level workflow layer rather than baked into the core operation contracts.

## 3. Command Model

Primary execution entry:

```bash
clawrise <operation> [flags]
```

Examples:

```bash
clawrise feishu.calendar.event.create --input @event.json
clawrise notion.page.create --json '{"title":"Project Notes"}'

clawrise platform use feishu
clawrise calendar.event.create --input @event.json
```

Operation path format:

```text
<platform>.<resource-path>.<action>
```

Examples:

- `feishu.calendar.event.create`
- `feishu.docs.document.create`
- `notion.page.create`

If a default platform is configured, runtime may also accept:

```text
<resource-path>.<action>
```

Examples:

- `calendar.event.create`
- `docs.document.create`

Reserved management commands:

- `clawrise platform ...`
- `clawrise subject ...`
- `clawrise profile ...`
- `clawrise auth ...`
- `clawrise config ...`
- `clawrise batch ...`
- `clawrise spec ...`
- `clawrise doctor`
- `clawrise version`
- `clawrise completion`

## 4. Input and Output

Preferred input forms:

- `--json '<json>'`
- `--input @file.json`
- `stdin`

Common runtime flags:

- `--profile`
- `--json`
- `--input`
- `--timeout`
- `--dry-run`
- `--idempotency-key`
- `--output`
- `--quiet`

All commands should return normalized JSON.

Success envelope:

```json
{
  "ok": true,
  "operation": "notion.page.create",
  "request_id": "req_01HYTEST",
  "context": {
    "platform": "notion",
    "subject": "integration",
    "profile": "notion_team_docs"
  },
  "data": {},
  "error": null,
  "meta": {
    "platform": "notion",
    "duration_ms": 184,
    "retry_count": 0,
    "dry_run": false
  },
  "idempotency": {
    "key": "idem_xxx",
    "status": "executed"
  }
}
```

Failure envelope:

```json
{
  "ok": false,
  "operation": "feishu.calendar.event.create",
  "request_id": "req_01HXXX",
  "data": null,
  "error": {
    "code": "RATE_LIMITED",
    "message": "rate limited by upstream",
    "retryable": true,
    "upstream_code": "99991400",
    "http_status": 429
  },
  "meta": {
    "platform": "feishu",
    "duration_ms": 512,
    "retry_count": 2,
    "dry_run": false
  },
  "idempotency": {
    "key": "idem_xxx",
    "status": "rejected"
  }
}
```

## 5. Runtime Modules

Recommended runtime responsibilities:

1. command parser
2. config loader
3. auth manager
4. adapter registry
5. operation metadata resolver
6. input validator
7. idempotency controller
8. executor
9. output encoder
10. audit logger

Execution flow:

```text
CLI input
  -> resolve operation and flags
  -> load config / profile
  -> resolve adapter and operation metadata
  -> read JSON input
  -> validate and normalize
  -> resolve idempotency
  -> apply timeout / retry / rate limit policy
  -> execute adapter
  -> normalize output
  -> write audit record
```

## 6. Adapter Model

The runtime should not know platform-specific details directly.

Recommended Go interfaces:

```go
type Adapter interface {
    Name() string
    Resolve(operation string) (OperationHandler, error)
}

type OperationHandler interface {
    Meta() OperationMeta
    Validate(ctx context.Context, input map[string]any) error
    Execute(ctx context.Context, req ExecuteRequest) (*ExecuteResult, error)
}
```

Recommended operation metadata should cover:

- operation name
- platform
- mutating or read-only
- idempotent or not
- dry-run support
- default timeout
- retry policy
- rate limit key
- auth constraint

## 7. Auth Boundary

See the full auth model at [auth-model.md](auth-model.md).

The key rules are:

- `platform` decides where the request goes
- `profile` decides which identity executes the request
- one platform must support multiple profiles
- operations must declare allowed subject types
- runtime must not silently switch profiles or downgrade subjects

Recommended config shape:

```yaml
defaults:
  platform: feishu
  subject: bot
  profile: feishu_bot_ops

profiles:
  feishu_bot_ops:
    platform: feishu
    subject: bot
    grant:
      type: client_credentials
      app_id: env:FEISHU_APP_ID
      app_secret: env:FEISHU_APP_SECRET

  notion_team_docs:
    platform: notion
    subject: integration
    grant:
      type: static_token
      token: env:NOTION_ACCESS_TOKEN
      notion_version: "2026-03-11"
```

## 8. OpenAPI and Generation

Generation should not try to produce the final product automatically.

Recommended outputs:

- manifest files
- generated request/response structures
- base mapping code

Manual overlay is still required for:

- auth quirks
- idempotency rules
- error normalization
- undocumented behavior
- field semantics

Recommended principle:

`generated + overlay`

## 9. Resource Visibility and Sharing Model

"Successfully executed" does not mean "visible to the target user."

This must be modeled explicitly in Clawrise, especially for office-style platforms such as Feishu docs, sheets, and calendars.

### 9.1 Core Principle

Resource visibility is determined by resource permissions, not by request success alone.

That means:

- a bot may create a resource successfully
- but the target user may still not see it automatically
- a bot may edit a resource successfully
- but only if the bot itself already has access to that resource

### 9.2 Typical Cases

Case A: editing an existing shared resource

- if the bot already has edit access
- and the user already has read access
- then the bot's changes are visible to the user

Case B: creating a new resource as a bot

- if the resource is created in bot-owned space or an app-owned folder
- it will not automatically appear in the user's personal resource list
- follow-up sharing, authorization, or creation in a user-visible location is still required

### 9.3 Implications for Feishu Docs and Sheets

With application identity, Feishu docs and sheets typically allow the app to operate on:

- resources created by the app itself
- or resources explicitly granted to the app

Therefore:

- `create` operations must not imply immediate user visibility
- `update` operations should distinguish between existing shared resources and app-owned resources
- sharing or permission-granting operations will be required later

### 9.4 Implications for Clawrise Command Design

Operation semantics should distinguish clearly between:

- `create`
- `update`
- `share`
- `grant`

Creating a resource and making it visible to a target user should not be treated as the same default action.

### 9.5 MVP Constraint

In MVP:

- `create` only guarantees that the resource was created
- it does not guarantee that the intended end user can see it
- if visibility matters to the business result, the contract should say so explicitly

## 10. Idempotency, Retry, and Audit

Write operations should default to idempotent handling.

Recommended idempotency states:

- `executed`
- `replayed`
- `in_progress`
- `rejected`

Recommended storage progression:

- MVP: local SQLite
- later service mode: Redis or PostgreSQL

Retry should only apply when:

- the operation is read-only
- the write operation is explicitly idempotent
- the upstream error is classified as temporary

Audit records should include:

- request id
- operation
- profile
- input summary
- output summary
- duration
- retry count
- final status

## 11. MVP Scope

MVP platforms:

- `feishu`
- `notion`

Recommended MVP sequence:

1. build the runtime core
2. implement the Feishu adapter MVP
3. implement the Notion adapter MVP
4. build the generation pipeline
5. extend to Google later

Detailed operation contracts are documented in [mvp-operation-spec.md](mvp-operation-spec.md).
