# Technology Stack

**Analysis Date:** 2026-04-09

## Languages

**Primary:**
- Go 1.25.9 - Core CLI runtime, all plugin binaries, adapter layer, config, auth, and execution engine

**Secondary:**
- JavaScript (Node.js) - npm distribution wrapper at `packaging/npm/root/`, skill install scripts, release tooling at `scripts/release/*.mjs`
- Shell (bash) - Build scripts, CI scripts, dev tooling at `scripts/`
- YAML - Config file format (`gopkg.in/yaml.v3`), GitHub Actions workflows at `.github/workflows/`

## Runtime

**Environment:**
- Go 1.25.9 (pinned in `go.mod` line 3)
- Node.js 24 (CI uses `actions/setup-node@v4` with `node-version: '24'`)

**Package Manager:**
- Go modules - primary dependency management (`go.mod`, `go.sum`)
- npm - distribution packaging only (no `package.json` in repo root; generated during release)
- Lockfile: `go.sum` present with 6 lines (minimal dependency footprint)

## Frameworks

**Core:**
- Go standard library only - No web framework; all HTTP calls use `net/http` directly
- `github.com/spf13/pflag` v1.0.5 - CLI flag parsing in `internal/cli/root.go`
- `gopkg.in/yaml.v3` v3.0.1 - Config file serialization in `internal/config/`

**Testing:**
- Go `testing` package - All Go tests use standard `go test`
- Node.js `node --test` - npm wrapper tests at `packaging/npm/root/lib/*.test.js`
- No third-party test frameworks or assertion libraries

**Build/Dev:**
- Go cross-compilation via `GOOS`/`GOARCH` - 6 platform targets in `scripts/release/build-npm-bundles.sh`
- `CGO_ENABLED=0` - Static binaries for all release builds
- `-ldflags` injection for `buildinfo.Version`, `buildinfo.Commit`, `buildinfo.BuildDate`

## Key Dependencies

**Critical:**
- `github.com/spf13/pflag` v1.0.5 - CLI argument parsing; used exclusively in `internal/cli/root.go`
- `gopkg.in/yaml.v3` v3.0.1 - Config file parse/serialize; used in `internal/config/store.go`

**Infrastructure:**
- Go `net/http` - All upstream API calls (Notion, Feishu) use standard library HTTP client
- Go `encoding/json` - JSON-RPC protocol between core and plugins, all API I/O
- Go `crypto/*` - SHA-256 for idempotency keys, SHA-1/SHA-256/SHA-512 for plugin checksum verification, random token generation

## Configuration

**Environment:**
- YAML config file at `~/.clawrise/config.yaml` (resolved via `internal/locator/`)
- Runtime state directory at `~/.clawrise/runtime/` (sessions, auth flows, audit, idempotency)
- Plugin discovery paths: `CLAWRISE_PLUGIN_PATHS` env var, `.clawrise/plugins`, `~/.clawrise/plugins`
- `CLAWRISE_ROOT_PACKAGE_NAME` - Override npm package name during release
- `CLAWRISE_NPM_SCOPE`, `CLAWRISE_NPM_PACKAGE_PREFIX`, `CLAWRISE_NPM_DIST_TAG` - Release publishing config
- `NOTION_INTERNAL_TOKEN`, `FEISHU_APP_ID`, `FEISHU_APP_SECRET` - Setup-time credential env vars

**Build:**
- `go.mod` - Go module definition
- `scripts/release/build-npm-bundles.sh` - Cross-platform binary builds
- `scripts/release/prepare-npm-packages.mjs` - npm package assembly
- `scripts/release/prepare-skill-packages.mjs` - Skill package assembly
- `.github/workflows/ci.yml` - CI pipeline (test, hardening, build, smoke, release-smoke)
- `.github/workflows/release-npm.yml` - Release and npm publish pipeline

## Platform Requirements

**Development:**
- Go 1.25.9+
- Node.js 24+ (for npm wrapper tests and release tooling)
- macOS Keychain or Linux Secret Service (for default secret storage backend)

**Production:**
- Published as npm packages under `@clawrise` scope to npm registry
- Platform-specific binary packages: `darwin-arm64`, `darwin-x64`, `linux-arm64`, `linux-x64`, `win32-arm64`, `win32-x64`
- Supports agent clients: Codex, Claude Code, OpenClaw, OpenCode
- No server-side deployment; CLI is purely client-side

---

*Stack analysis: 2026-04-09*
