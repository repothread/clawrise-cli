# Clawrise Plugin Authoring Spec

See the Chinese version at [../zh/plugin-authoring-spec.md](../zh/plugin-authoring-spec.md).

> This document is the current author-facing source of truth for plugin manifests, protocol surface, distribution guidance, and compatibility boundaries. If it conflicts with [plugin-system-design.md](./plugin-system-design.md), prefer this document.

## 1. Goal

This document defines the public contract for third-party Clawrise plugins.

It exists so that:

- external repositories can build, release, and upgrade independently
- the core and plugins stay protocol-coupled instead of code-coupled
- `doctor`, `spec`, `auth`, runtime, and installation keep one fact source
- Go, Node.js, and other languages can implement the same plugin contract

## 2. Public Boundary

The public standard has five parts:

1. plugin directory layout
2. `plugin.json` manifest v2
3. `stdio + JSON-RPC 2.0` protocol v1
4. installation sources and archive guidance
5. black-box compatibility checks

The following are explicitly outside the public boundary:

- importing `github.com/clawrise/clawrise-cli/internal/...`
- reading the core config file directly from plugins
- re-implementing `env:` parsing, default account selection, or secret store resolution inside plugins
- sharing internal Go structs between core and plugin repositories

## 3. Zero-Coupling Rules

Third-party plugins must follow these rules:

- plugins may depend on the protocol, but not on this repository's `internal/...` packages
- the core resolves config, secrets, sessions, and account selection
- the plugin consumes already-resolved `account`, `execution_auth`, and `input`
- the plugin should not know about config file locations or storage backends

Recommended repository split:

- this repository owns specs, installer behavior, discovery, routing, and validation
- external repositories own provider API mapping, auth details, and execution logic
- optional SDKs should live in separate repositories and must remain optional

## 4. Runtime Layout

The runtime standard is the unpacked plugin directory, not a specific archive format.

Recommended layout:

```text
clawrise-plugin-<name>/
  plugin.json
  bin/
    clawrise-plugin-<name>
  README.md
  LICENSE
```

During development, a discovery root may point either to the plugin directory itself or to one of its parent directories. The core discovers `plugin.json` files recursively.

Default discovery order:

1. `CLAWRISE_PLUGIN_PATHS`
2. `.clawrise/plugins`
3. `~/.clawrise/plugins`

## 5. Manifest V2

Third-party plugins should prefer `schema_version: 2`.

Minimal provider manifest:

```json
{
  "schema_version": 2,
  "name": "linear",
  "version": "0.1.0",
  "protocol_version": 1,
  "min_core_version": "0.1.0",
  "capabilities": [
    {
      "type": "provider",
      "platforms": ["linear"]
    }
  ],
  "entry": {
    "type": "binary",
    "command": ["./bin/clawrise-plugin-linear"]
  }
}
```

Key requirements:

- `schema_version`: use `2` for new authoring
- `name`: stable plugin identity used by provider binding
- `version`: plugin version
- `protocol_version`: currently `1`
- `min_core_version`: strongly recommended for preflight compatibility checks
- `capabilities`: at least one `provider` capability for provider plugins
- `entry.type`: currently must be `binary`
- `entry.command`: resolved relative to the manifest directory unless absolute

For a provider capability:

- `type` must be `provider`
- `platforms` must contain at least one platform name

## 6. Protocol V1

Transport requirements:

- transport: `stdio`
- encoding: one-line `JSON-RPC 2.0`
- `stdout`: protocol messages only
- `stderr`: logs only

### 6.1 Current Required Surface

Under the current implementation, provider plugins should implement the following method matrix.

Required for every plugin:

- `clawrise.handshake`
- `clawrise.capabilities.list`
- `clawrise.health`

Required for every provider:

- `clawrise.operations.list`
- `clawrise.catalog.get`
- `clawrise.execute`

Required for every authenticated provider:

- `clawrise.auth.methods.list`
- `clawrise.auth.inspect`
- `clawrise.auth.resolve`

Recommended for providers that expose account presets:

- `clawrise.auth.presets.list`

Required for providers that support interactive login:

- `clawrise.auth.begin`
- `clawrise.auth.complete`

### 6.2 Semantic Rules

- `handshake` exposes plugin identity, version, and platforms
- `capabilities.list` exposes the capability routing fact set
- `operations.list` exposes runtime-executable operations
- `catalog.get` exposes the structured catalog; the current minimum entry only needs `operation`
- `auth.inspect` validates account readiness without performing business operations
- `auth.resolve` converts account material into execution-time `execution_auth`
- `execute` should focus on provider execution and should not parse the Clawrise config file

### 6.3 Auth Boundary

The core sends an already-resolved `account` shape to the plugin, including:

- `name`
- `platform`
- `subject`
- `auth_method`
- `public`
- `secrets`
- `session`

The provider should collapse all execution-critical values into `execution_auth` during `auth.resolve`, for example:

- API base URL
- access token or API key
- provider version header
- tenant or workspace context

After that point, `execute` should not depend on the core config file or any storage backend.

## 7. Distribution And Installation

### 7.1 Design Conclusion

The Clawrise runtime standard is "plugin directory + manifest + protocol", not a single archive format.

That means:

- `tar.gz` is not the only standard
- `zip` can also be used as an archive carrier
- `npm`, `https`, and `registry` are installation sources
- the runtime always operates on an installed directory tree

### 7.2 Current Installation Sources

The current core already supports:

- local directories
- `file://`
- `https://`
- direct npm package specs
- `npm://`
- `registry://`

### 7.3 Recommended Priority

Recommended distribution priority for third-party plugins:

1. `registry://`
2. `https://`
3. `npm://`

Why:

- `registry://` is the best long-term abstraction for logical plugin names and platform-specific artifacts
- `https://` is the most general binary distribution path
- `npm://` is convenient for Node.js plugins, but should not become the only supported path

### 7.4 Archive Guidance

For binary plugins, `.tar.gz` is the recommended default artifact:

- language-agnostic
- easy to bundle binaries, README, manifest, and static assets
- easy to checksum, mirror, and distribute in restricted environments

If a target ecosystem needs it, publishing `.zip` in parallel is also acceptable.

## 8. Compatibility Checks

At minimum, a third-party plugin repository should verify:

1. `clawrise doctor`
2. `clawrise auth methods --platform <platform>`
3. `clawrise spec list <platform>`
4. at least one real operation through `--dry-run` or real execution

This repository now ships a local black-box verification script:

```bash
./scripts/plugin/verify-external-provider.sh /abs/path/to/plugin/root linear linear.viewer.get
```

It depends only on discovery, protocol behavior, and CLI integration. It does not care which language implements the plugin.
It is meant for development-time discovery checks, where the current plugin root should be discoverable and able to handshake directly.

For production install validation, this repository also ships an install-path verification script:

```bash
./scripts/plugin/verify-external-provider-install.sh \
  file:///abs/path/to/clawrise-plugin-linear-0.1.0-darwin-arm64.tar.gz \
  linear \
  0.1.0 \
  linear \
  linear.viewer.get
```

This script validates:

1. `plugin install`
2. `plugin info`
3. `plugin verify`
4. `doctor`
5. `auth methods`
6. `spec list/get`

Production recommendation:

- do not treat a source repository directory as the formal production install source
- publish versioned `.tar.gz` artifacts and install them through `file://`, `https://`, or `registry://`
- publish immutable versions together with SHA256 checksums so artifact repositories, mirrors, and audits can verify what was installed

## 9. Language Guidance

The project should maintain both Go and Node.js templates, but with different roles:

- Go: primary template
- Node.js: rapid-development template

Go is recommended for long-lived provider plugins because:

- it ships as a single binary
- it does not require Node.js on the host
- it fits GitHub Release, `https://`, and `registry://` distribution well

Node.js is recommended for:

- fast prototyping
- workflow-heavy plugins
- plugins that rely heavily on npm ecosystem packages

## 10. Reference Implementation

Reference implementations should live in separate repositories instead of inside this spec repository.

The first planned reference implementation repository is:

- `clawrise-plugin-linear`

It should have the following properties:

- it imports none of this repository's Go packages
- it integrates only through `plugin.json + stdio JSON-RPC`
- it uses `Linear API key` auth by default
- it can be released independently through GitHub Releases or other distribution channels
