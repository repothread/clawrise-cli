# External Integrations

**Analysis Date:** 2026-04-09

## APIs & External Services

**Notion API:**
- REST API at `https://api.notion.com` (default base URL in `internal/adapter/notion/client.go` line 26)
- API version header `Notion-Version: 2026-03-11` (line 27)
- Auth methods: `notion.internal_token` (static token), `notion.oauth_public` (OAuth2 with refresh)
- Operations: database CRUD, page CRUD, block CRUD, comment CRUD, user queries, search, data sources, file uploads, markdown import/export, page section patching, graph traversal
- SDK/Client: Custom HTTP client at `internal/adapter/notion/client.go` - no third-party Notion SDK
- Plugin binary: `cmd/clawrise-plugin-notion/main.go`
- Operation registration: `internal/adapter/notion/register.go` (40+ operations)
- Auth provider: `internal/adapter/notion/auth_provider.go`

**Feishu (Lark) Open API:**
- REST API at `https://open.feishu.cn` (default base URL in `internal/adapter/feishu/client.go` line 23)
- Auth methods: `feishu.app_credentials` (bot via tenant_access_token), `feishu.oauth_user` (user OAuth2)
- Token endpoint: `/open-apis/auth/v3/tenant_access_token/internal` (bot)
- OAuth endpoints: `/open-apis/authen/v2/oauth/token` (user), `https://accounts.feishu.cn/open-apis/authen/v1/authorize` (authorization URL)
- Operations: calendar CRUD, docs CRUD, wiki operations, contact/department queries, bitable (spreadsheet) record CRUD
- SDK/Client: Custom HTTP client at `internal/adapter/feishu/client.go` - no third-party Feishu SDK
- Plugin binary: `cmd/clawrise-plugin-feishu/main.go`
- Operation registration: `internal/adapter/feishu/register.go` (30+ operations)
- Auth provider: `internal/adapter/feishu/auth_provider.go`

**npm Registry:**
- Used for plugin discovery and installation via `npm` source type in `internal/plugin/install.go`
- Registry base URL: `https://registry.npmjs.org` (line 29)
- Also used for publishing release artifacts via `scripts/release/publish-npm.sh`

## Data Storage

**Databases:**
- No database. All storage is file-based.

**File Storage:**
- Local filesystem only
- Config: YAML file at `~/.clawrise/config.yaml` (loaded/saved via `internal/config/store.go`)
- Sessions: JSON files in runtime state directory (`internal/auth/session.go`)
- Auth flows: JSON files in runtime state directory (`internal/authflow/`)
- Idempotency records: JSON files in `runtime/idempotency/` directory
- Audit records: JSON files in `runtime/audit/` directory
- Plugin manifests: JSON files (`plugin.json`) discovered in plugin directories

**Caching:**
- Auth session caching with 2-minute refresh skew (`internal/auth/session.go` line 21)
- Tenant access tokens cached per account with expiry-based refresh
- OAuth access/refresh token pairs cached with automatic refresh

## Authentication & Identity

**Auth Provider:**
- Custom multi-method auth system
  - Implementation: `internal/auth/`, `internal/secretstore/`, per-provider `auth_provider.go` files
  - Notion: Internal integration token (static Bearer), OAuth2 public flow (authorization code + refresh)
  - Feishu: App credentials (tenant_access_token via app_id/app_secret), OAuth2 user flow (authorization code + refresh)
  - All auth methods defined in `internal/plugin/protocol.go` as `AuthMethodDescriptor` structs

**Secret Storage:**
- `internal/secretstore/store.go` defines `Store` interface
- Backends: macOS Keychain (`keychain`), Linux Secret Service (`secret_service`), encrypted file (`encrypted_file`), plain file (`file`), plugin-provided backends
- Secret reference syntax: `secret:<value>`, `env:<VAR_NAME>` for resolution via `internal/config/config.go`
- Configurable per-account: `auth.secret_store.backend` in config YAML

**Session Storage:**
- `internal/auth/session.go` defines `Store` interface
- Backends: File-based (`file`), plugin-provided backends
- Configurable: `auth.session_store.backend` in config YAML

## Plugin System

**Core-Plugin Protocol:**
- JSON-RPC 2.0 over stdio (`internal/plugin/server.go`, `internal/plugin/process.go`)
- Protocol version: 1 (`internal/plugin/protocol.go` line 16)
- Handshake method: `clawrise.handshake`
- Plugin manifest: `plugin.json` with schema version 2 (`internal/plugin/manifest.go`)
- Plugin kinds: `provider`, `auth_launcher`, `storage_backend`, `multi`, `policy`, `audit_sink`, `workflow`, `registry_source`
- Plugin capabilities defined in `internal/plugin/capability.go`

**Bundled Plugins:**
- `cmd/clawrise-plugin-feishu/main.go` - Feishu provider
- `cmd/clawrise-plugin-notion/main.go` - Notion provider
- `cmd/clawrise-plugin-auth-browser/main.go` - System browser auth launcher
- `cmd/clawrise-plugin-demo/main.go` - Demo provider
- `cmd/clawrise-plugin-sample-audit/main.go` - Sample audit sink
- `cmd/clawrise-plugin-sample-policy/main.go` - Sample policy evaluator

**Plugin Installation Sources:**
- Local path, file archive, HTTPS download, npm package, registry metadata source
- See `internal/plugin/install.go` for source type definitions

## Monitoring & Observability

**Error Tracking:**
- None. Errors are normalized into `apperr.AppError` structs (`internal/apperr/`) with structured error codes.

**Audit:**
- Built-in audit trail in `internal/runtime/` writes to `runtime/audit/` directory
- Audit records include: request ID, operation, context, input/output summaries, error details, idempotency state, warnings
- Configurable audit sinks via plugin system (`audit_sink` capability type)
- Audit mode configurable: `auto`, `manual`, `disabled`

**Policy:**
- Pre-execution policy evaluation via plugin system (`policy` capability type)
- Policy mode configurable: `auto`, `manual`, `disabled`

## CI/CD & Deployment

**Hosting:**
- npm registry (`registry.npmjs.org`) - Published as `@clawrise/clawrise-cli` with platform-specific packages
- GitHub Releases - Archives and SHA256SUMS uploaded per release

**CI Pipeline:**
- GitHub Actions (`.github/workflows/ci.yml`)
  - Jobs: `test`, `hardening` (go vet, race tests, staticcheck, govulncheck), `build`, `script-smoke`, `release-smoke`
- GitHub Actions (`.github/workflows/release-npm.yml`)
  - Triggered by `v*` tags or manual dispatch
  - Builds cross-platform binaries, prepares npm packages, publishes to npm and GitHub Releases
  - Uses `cicd` environment with secrets for npm publishing

**Skill Distribution:**
- Skills are Markdown instruction bundles installed into agent client directories
- Installed to: `~/.codex/skills/`, `~/.claude/skills/`, `~/.openclaw/skills/`, `~/.config/opencode/skills/`
- Skill packages: `skills/clawrise-core/`, `skills/clawrise-feishu/`, `skills/clawrise-notion/`

## Environment Configuration

**Required env vars (for setup flow):**
- `NOTION_INTERNAL_TOKEN` - Notion integration token (or interactive prompt)
- `FEISHU_APP_ID` - Feishu application ID (or interactive prompt)
- `FEISHU_APP_SECRET` - Feishu application secret (or interactive prompt)

**Optional env vars:**
- `CLAWRISE_PLUGIN_PATHS` - Additional plugin discovery paths
- `CLAWRISE_ROOT_PACKAGE_NAME` - Override npm package name
- `CLAWRISE_NPM_SCOPE` - npm scope for publishing
- `CLAWRISE_NPM_PACKAGE_PREFIX` - npm package name prefix
- `CLAWRISE_NPM_DIST_TAG` - npm dist-tag (latest/next/beta)
- `CODEX_HOME`, `CLAUDE_HOME`, `OPENCLAW_HOME`, `OPENCODE_CONFIG_HOME`, `XDG_CONFIG_HOME` - Client directory overrides
- `CLAWRISE_RELEASE_COMMIT`, `CLAWRISE_RELEASE_BUILD_DATE` - Build metadata injection

**Secrets location:**
- macOS Keychain (default on macOS)
- Linux Secret Service (default on Linux)
- Encrypted file fallback
- Plain file fallback (development)
- Configurable via `auth.secret_store.backend` in `~/.clawrise/config.yaml`

## Webhooks & Callbacks

**Incoming:**
- OAuth callback handler during interactive auth flows
  - Listens on `http://localhost:3333/callback` (configurable host/path)
  - Used by Notion OAuth and Feishu OAuth user authorization flows
  - Implemented in `internal/authflow/` and invoked via auth launcher plugins

**Outgoing:**
- None. All API calls are synchronous outbound HTTP requests.

---

*Integration audit: 2026-04-09*
