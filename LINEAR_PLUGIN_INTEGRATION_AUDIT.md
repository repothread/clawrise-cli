# Linear Plugin Integration Audit

## Scope

This document records the current integration verdict between:

- core: `clawrise-cli`
- external provider plugin: `~/thread/clawrise-plugin-linear`

It is intentionally execution-oriented so future work can proceed without ambiguity.

## Final Verdict

`clawrise-plugin-linear` can be integrated into `clawrise-cli` **today** as an external provider plugin.

No core code change is required for basic integration.
No plugin code change is required for basic integration.

However, the plugin repository still has follow-up work if the goal is a more future-proof and fully aligned implementation of the current public plugin protocol.

## What Was Verified

### Development discovery path

```bash
cd ~/thread/clawrise-plugin-linear
./scripts/build.sh
/Users/liyang/thread/clawrise-cli/scripts/plugin/verify-external-provider.sh "$PWD" linear linear.viewer.get
```

Expected result:

- `clawrise doctor` succeeds with `CLAWRISE_PLUGIN_PATHS` pointing at the plugin repo
- `clawrise auth methods --platform linear` succeeds
- `clawrise spec list linear` succeeds
- `clawrise spec get linear.viewer.get` succeeds

### Production install path

```bash
cd ~/thread/clawrise-plugin-linear
./scripts/package.sh 0.1.0

/Users/liyang/thread/clawrise-cli/scripts/plugin/verify-external-provider-install.sh \
  "file:///Users/liyang/thread/clawrise-plugin-linear/dist/clawrise-plugin-linear-0.1.0-darwin-arm64.tar.gz" \
  linear \
  0.1.0 \
  linear \
  linear.viewer.get
```

Expected result:

- `clawrise plugin install ...` succeeds
- `clawrise plugin info linear 0.1.0` succeeds
- `clawrise plugin verify linear 0.1.0` succeeds
- `clawrise doctor` succeeds after install
- `clawrise auth methods --platform linear` succeeds
- `clawrise spec list linear` succeeds
- `clawrise spec get linear.viewer.get` succeeds

## Why Integration Works Today

The plugin already satisfies the current core contract at the level required for runtime integration:

1. `plugin.json` uses `schema_version: 2`
2. `protocol_version: 1` matches the core runtime
3. the plugin exposes a `provider` capability for platform `linear`
4. the plugin entry is a stdio binary command
5. the plugin implements the required JSON-RPC methods used by the core

## No-Ambiguity Integration Steps

### Option A: Development discovery

1. Build the plugin:

   ```bash
   cd ~/thread/clawrise-plugin-linear
   ./scripts/build.sh
   ```

2. Expose the plugin repository root as a discovery root:

   ```bash
   export CLAWRISE_PLUGIN_PATHS=~/thread/clawrise-plugin-linear
   ```

3. Verify discovery:

   ```bash
   clawrise doctor
   clawrise auth methods --platform linear
   clawrise spec list linear
   clawrise spec get linear.viewer.get
   ```

4. Add an account in `~/.clawrise/config.yaml`:

   ```yaml
   accounts:
     linear_default:
       title: Linear Default
       platform: linear
       subject: integration
       auth:
         method: linear.api_key
         public:
           api_url: https://api.linear.app/graphql
         secret_refs:
           token: secret:linear_default:token
   ```

5. Verify the account and basic operations:

   ```bash
   clawrise auth methods --platform linear
   clawrise linear.viewer.get --json '{}'
   clawrise linear.team.list --json '{}'
   ```

### Option B: Production-style install

1. Package the plugin:

   ```bash
   cd ~/thread/clawrise-plugin-linear
   ./scripts/package.sh 0.1.0
   ```

2. Install the archive with an **absolute** `file://` URL:

   ```bash
   clawrise plugin install \
     file:///Users/liyang/thread/clawrise-plugin-linear/dist/clawrise-plugin-linear-0.1.0-darwin-arm64.tar.gz
   ```

3. Verify the installed plugin:

   ```bash
   clawrise plugin info linear 0.1.0
   clawrise plugin verify linear 0.1.0
   clawrise doctor
   clawrise auth methods --platform linear
   clawrise spec list linear
   clawrise spec get linear.viewer.get
   ```

4. Use the same `accounts` configuration shown above.

## When Provider Binding Is Required

Provider binding is **not** required if only one plugin supports platform `linear` in the active discovery roots.

Provider binding **is** required if multiple plugins support `linear` at the same time.

In that case, set the binding explicitly:

```bash
clawrise config provider use linear linear
```

Or set it in config:

```yaml
plugins:
  bindings:
    providers:
      linear:
        plugin: linear
```

## Plugin Repository Improvements

The following items are not blockers for integration today, but they should be implemented in `clawrise-plugin-linear` if the goal is long-term protocol alignment.

### P1. Align plugin-local protocol structs with the current public core protocol

The plugin currently uses hand-written protocol structs that cover only the fields it needs today.
That is acceptable for current behavior, but it is narrower than the current public protocol contract.

#### Required additions for auth-related structs

Add support for these fields in the plugin-local protocol model:

- `account.session`
- `auth.inspect.result.session_status`
- `auth.resolve.result.session_patch`
- `auth.resolve.result.secret_patches`

Implementation rule:

- the plugin may ignore these fields semantically for now if Linear API key auth does not need them
- the fields must still exist in the plugin-local structs so the plugin remains structurally aligned with the public protocol

Acceptance criteria:

- JSON decoding still succeeds for current requests
- the plugin test suite passes
- the plugin does not regress current `linear.api_key` behavior

### P1. Align execute request/result structs with the current public core protocol

Add support for these request fields:

- `idempotency_key`
- `dry_run`
- `debug_provider_payload`
- `verify_after_write`

Add support for this response field:

- `debug`

Implementation rule:

- the plugin may keep current runtime behavior for now
- the added fields must exist in the plugin-local execute structs
- if `debug_provider_payload` is later implemented, the plugin should return provider payload details only in the `debug` object

Acceptance criteria:

- current operations still pass existing tests
- the plugin can safely ignore unknown or currently unused execution flags
- the plugin response schema is compatible with the core runtime's `debug` handling

### P2. Decide whether to support interactive auth

The current plugin correctly reports that interactive auth is not supported.
That is valid for an API-key-only plugin.

If future scope includes OAuth or browser-based login, implement real behavior for:

- `clawrise.auth.begin`
- `clawrise.auth.complete`

If future scope does **not** include interactive auth, keep the current behavior and make that explicit in:

- `README.md`
- `skills/clawrise-linear/SKILL.md`
- tests for auth capability reporting

### P2. Expand black-box release verification in the plugin repo

The plugin repo should keep these checks as mandatory release gates:

```bash
go test ./...
./scripts/build.sh
./scripts/package.sh <version>
./scripts/verify-with-clawrise.sh /path/to/clawrise-cli
```

Recommended addition:

- add one CI step that packages an archive and runs the install-path verification script from `clawrise-cli`, not only the development discovery script

## Core Repository Changes Needed

None for basic Linear plugin integration.

Core already provides:

- plugin discovery
- plugin process runtime over stdio JSON-RPC
- install / info / verify / upgrade commands
- provider binding selection
- doctor / auth / spec integration over the plugin boundary

## Decision Summary

### Can we integrate now?

Yes.

### Can we ship plugin integration without changing the core?

Yes.

### Does the plugin repo still have worthwhile follow-up work?

Yes.
The main work is protocol-shape alignment and release-hardening, not basic compatibility repair.
