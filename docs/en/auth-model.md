# Clawrise Auth and Account Model

See the Chinese version at [../zh/auth-model.md](../zh/auth-model.md).

## 1. Core Principle

Auth in Clawrise is an execution identity model, not just token retrieval.

The runtime must model:

- where the request goes
- who executes the request
- how credentials are obtained
- how runtime auth sessions are built, cached, and refreshed

## 2. Core Concepts

### Platform

Identifies the upstream provider.

Examples:

- `feishu`
- `notion`

### Account

An account is one concrete executable identity selected explicitly or by defaults.

Examples:

- `feishu_bot_ops`
- `feishu_user_alice`
- `notion_team_docs`

### Subject

Subject describes the identity category used at the provider side.

Current common values:

- `bot`
- `user`
- `integration`

Future platforms may introduce new subject strings. The core should not hard-code the full set.

### Auth Method

Auth method describes how one account authenticates with the provider.

Current first-party methods:

- `feishu.app_credentials`
- `feishu.oauth_user`
- `notion.internal_token`
- `notion.oauth_public`

### Session

Session is the runtime-authenticated state actually used to execute requests.

It typically contains:

- access token
- refresh token
- expires_at
- token_type
- provider-native metadata

## 3. Runtime Resolution

For each command, the runtime resolves auth in the following order:

1. resolve the operation
2. resolve the platform
3. resolve the account
4. load account public fields, secret refs, and session
5. call provider plugin `auth.resolve`
6. persist returned session / secret patches
7. inject provider-ready auth and execute the operation

Selection rules:

- `--account` has highest priority
- otherwise use the default account for the current platform
- if no matching account exists, fail explicitly
- if multiple accounts could apply and none is selected, fail with ambiguity instead of guessing

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

Account management:

```bash
clawrise account add --platform feishu --preset bot
clawrise account use feishu_bot_ops
clawrise account current
clawrise account list
```

Per-call override:

```bash
clawrise feishu.calendar.event.create --account feishu_bot_ops
clawrise notion.page.create --account notion_team_docs
```

Key rule:

- `platform` decides where to call
- `subject` selects the preferred actor category
- `account` decides who calls

## 5. Config Shape

Recommended structure:

```yaml
defaults:
  platform: feishu
  account: feishu_bot_ops
  platform_accounts:
    feishu: feishu_bot_ops
    notion: notion_team_docs

accounts:
  feishu_bot_ops:
    platform: feishu
    subject: bot
    auth:
      method: feishu.app_credentials
      public:
        app_id: cli_xxx
      secret_refs:
        app_secret: secret:feishu_bot_ops:app_secret

  notion_team_docs:
    platform: notion
    subject: integration
    auth:
      method: notion.internal_token
      public:
        notion_version: "2026-03-11"
      secret_refs:
        token: secret:notion_team_docs:token
```

Key points:

- `public` stores non-secret fields
- `secret_refs` stores only secret references, not plaintext values
- the meaning of `auth.method` is declared by the provider plugin

## 6. Feishu Model

Current architecture support:

- `bot`
- `user`

Bot / app style account:

```yaml
accounts:
  feishu_bot_ops:
    platform: feishu
    subject: bot
    auth:
      method: feishu.app_credentials
      public:
        app_id: cli_xxx
      secret_refs:
        app_secret: secret:feishu_bot_ops:app_secret
```

User style account:

```yaml
accounts:
  feishu_user_alice:
    platform: feishu
    subject: user
    auth:
      method: feishu.oauth_user
      public:
        client_id: cli_xxx
        redirect_mode: loopback
        scopes:
          - offline_access
      secret_refs:
        client_secret: secret:feishu_user_alice:client_secret
        refresh_token: secret:feishu_user_alice:refresh_token
```

Multiple bots are represented as multiple accounts, not as a platform special case.

## 7. Notion Model

Current recommended subject type:

- `integration`

Internal integration:

```yaml
accounts:
  notion_team_docs:
    platform: notion
    subject: integration
    auth:
      method: notion.internal_token
      public:
        notion_version: "2026-03-11"
      secret_refs:
        token: secret:notion_team_docs:token
```

Public integration:

```yaml
accounts:
  notion_public_workspace_a:
    platform: notion
    subject: integration
    auth:
      method: notion.oauth_public
      public:
        client_id: cli_xxx
        notion_version: "2026-03-11"
        redirect_mode: loopback
      secret_refs:
        client_secret: secret:notion_public_workspace_a:client_secret
        refresh_token: secret:notion_public_workspace_a:refresh_token
```

## 8. Operation-Level Auth Constraints

Each operation must declare auth constraints.

Recommended shape:

```go
type AuthConstraint struct {
    AllowedSubjects  []string
    PreferredSubject string
    RequiredScopes   []string
}
```

Runtime rules:

- runtime must reject unsupported subject types explicitly
- runtime must not silently switch accounts
- provider plugins may implement provider-native auth behavior, but account resolution remains a core runtime concern
- fail immediately if the selected account subject is not allowed
- do not silently downgrade `user` to `bot`

## 9. Attribution and Visibility

Attribution and resource visibility are separate concerns and should not be merged.

### 9.1 Attribution

Attribution answers:

- who the provider records as the actor behind a change

For platforms like Feishu, this usually depends on the token type used for the call:

- `tenant_access_token` is closer to bot/app attribution
- `user_access_token` is closer to user attribution

### 9.2 Resource Visibility

Resource visibility answers:

- who can see the document, sheet, or calendar resource

This is usually determined by resource permissions, sharing relationships, and ownership location, not only by the calling identity.

### 9.3 Recommended Strategy

If the product requirement includes both:

- the target user should always see the resource
- bot changes should remain distinguishable from user changes

then creation-time identity and follow-up editing identity should be modeled separately instead of trying to make one account cover both goals.
