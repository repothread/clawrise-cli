# Clawrise Auth and Profile Model

See the Chinese version at [../zh/auth-model.md](../zh/auth-model.md).

## 1. Core Principle

Auth in Clawrise is an execution identity model, not just token retrieval.

The runtime must model:

- where the request goes
- who executes the request
- how credentials are granted
- how runtime auth sessions are built, cached, and refreshed

## 2. Core Concepts

### Platform

Identifies the upstream provider.

Examples:

- `feishu`
- `notion`

### Profile

A concrete executable identity selected by the user or by defaults.

Examples:

- `feishu_bot_ops`
- `feishu_user_alice`
- `notion_team_docs`

### Subject

Normalized identity type at the platform level.

Recommended subject types:

- `bot`
- `user`
- `integration`

### Grant

Describes how credentials are obtained.

Recommended grant types:

- `client_credentials`
- `oauth_user`
- `static_token`
- `oauth_refreshable`

### Session

Runtime-authenticated form used to execute requests.

It typically contains:

- access token
- expiry time
- normalized auth headers
- subject
- profile name

## 3. Runtime Resolution

For every command, runtime should:

1. resolve the operation
2. resolve the platform
3. resolve the profile
4. load credentials
5. build or refresh the auth session
6. verify the subject against operation constraints
7. inject auth headers and execute

Selection rules:

- `--profile` has highest priority
- otherwise use the default profile for the current platform
- if no matching profile exists, fail explicitly
- if multiple profiles could apply and none is selected, fail with ambiguity instead of guessing

## 4. CLI Conventions

Platform management:

```bash
clawrise platform use feishu
clawrise platform current
clawrise platform unset
```

Subject management:

```bash
clawrise subject use bot
clawrise subject current
clawrise subject unset
clawrise subject list
```

Profile management:

```bash
clawrise profile use feishu_bot_ops
clawrise profile current
clawrise profile list
```

Per-call override:

```bash
clawrise feishu.calendar.event.create --profile feishu_bot_ops
clawrise notion.page.create --profile notion_team_docs
```

Key rule:

- `platform` decides where to call
- `subject` selects the preferred actor category
- `profile` decides who calls

## 5. Feishu Model

Architecture support:

- `bot`
- `user`

MVP implementation support:

- `bot`

Bot/app style profile:

```yaml
profiles:
  feishu_bot_ops:
    platform: feishu
    subject: bot
    grant:
      type: client_credentials
      app_id: env:FEISHU_BOT_OPS_APP_ID
      app_secret: env:FEISHU_BOT_OPS_APP_SECRET
```

User style profile reserved for later:

```yaml
profiles:
  feishu_user_alice:
    platform: feishu
    subject: user
    grant:
      type: oauth_user
      client_id: env:FEISHU_CLIENT_ID
      client_secret: env:FEISHU_CLIENT_SECRET
      access_token: env:FEISHU_ALICE_ACCESS_TOKEN
      refresh_token: env:FEISHU_ALICE_REFRESH_TOKEN
```

Multiple bots are represented as multiple profiles, not as a platform special case.

## 6. Notion Model

Recommended subject type:

- `integration`

Internal integration is the MVP path:

```yaml
profiles:
  notion_team_docs:
    platform: notion
    subject: integration
    grant:
      type: static_token
      token: env:NOTION_TEAM_DOCS_TOKEN
      notion_version: "2026-03-11"
```

Public integration is a later extension path:

```yaml
profiles:
  notion_public_workspace_a:
    platform: notion
    subject: integration
    grant:
      type: oauth_refreshable
      client_id: env:NOTION_CLIENT_ID
      client_secret: env:NOTION_CLIENT_SECRET
      access_token: env:NOTION_WS_A_ACCESS_TOKEN
      refresh_token: env:NOTION_WS_A_REFRESH_TOKEN
      notion_version: "2026-03-11"
```

## 7. Operation-Level Auth Constraints

Every operation must declare auth constraints.

Recommended shape:

```go
type AuthConstraint struct {
    AllowedSubjects  []string
    PreferredSubject string
    RequiredScopes   []string
}
```

Runtime rules:

- fail immediately if the selected profile subject is not allowed
- do not silently switch profiles
- do not silently downgrade `user` to `bot`
- explicit failure is better than implicit fallback

## 8. Attribution and Visibility

Attribution and resource visibility are two different dimensions and must not be merged.

### 8.1 Attribution

Attribution answers:

- who the platform records as the actor behind a change

For platforms like Feishu, this is usually tied to the token type used for the call:

- `tenant_access_token` is closer to bot/app attribution
- `user_access_token` is closer to user attribution

### 8.2 Resource Visibility

Resource visibility answers:

- who can see the document, sheet, or calendar resource

This is usually determined by resource permissions, sharing relationships, and ownership location, not only by the calling identity.

### 8.3 Recommended Strategy

If the product requirement includes both:

- the target user should always see the resource
- bot changes should remain distinguishable from user changes

then the recommended flow is:

1. run `clawrise subject use user` and call `feishu.docs.document.create`
2. run `clawrise subject use bot` and call `feishu.docs.document.edit`

That means:

- create the resource under user identity so it is naturally visible to the user
- grant the bot access
- let the bot continue editing under bot identity

### 8.4 Strategy That Should Not Be the Default

Using `clawrise subject use user` as the long-term default for automated edits should not be the default strategy.

It simplifies visibility, but it risks attribution ambiguity:

- bot-generated changes may look like direct user changes in the platform history

## 9. MVP Subject Matrix

Current MVP execution matrix:

- `feishu.calendar.event.create` -> `bot`
- `feishu.calendar.event.list` -> `bot`
- `feishu.docs.document.create` -> `bot`
- `feishu.docs.document.get` -> `bot`
- `feishu.docs.document.list_blocks` -> `bot`
- `feishu.docs.block.get` -> `bot`
- `feishu.docs.block.list_children` -> `bot`
- `feishu.docs.block.update` -> `bot`
- `feishu.docs.block.batch_delete` -> `bot`
- `feishu.contact.user.get` -> `bot`
- `notion.search.query` -> `integration`
- `notion.page.create` -> `integration`
- `notion.page.get` -> `integration`
- `notion.page.markdown.get` -> `integration`
- `notion.page.markdown.update` -> `integration`
- `notion.block.get` -> `integration`
- `notion.block.list_children` -> `integration`
- `notion.block.append` -> `integration`
- `notion.block.update` -> `integration`
- `notion.block.delete` -> `integration`
- `notion.user.get` -> `integration`

## 10. Security Rules

- never store plain-text secrets in audit logs
- keep config and token cache in separate files
- never print raw access tokens in normal CLI output
- redact secrets in logs and debug output

## 11. MVP Scope

MVP must implement:

- multiple profiles
- default platform selection
- default profile selection
- Feishu bot/app credential flow
- Notion internal integration token flow
- operation-level subject validation

MVP may defer:

- Feishu user browser login flow
- Notion public OAuth flow
- system keychain integration
- interactive auth helpers
