# npm Release Workflow

This document is for repository maintainers. It explains how Clawrise Go build outputs are packaged into an npm-installable CLI.

## Goal

- build precompiled binaries for each target platform after a release tag is created from `main`
- publish a scoped root package so users can run:

```bash
npm install -g @scope/clawrise-cli
```

- let the npm root package resolve the correct platform binary automatically
- bundle the first-party `feishu` and `notion` provider plugins with the platform package to reduce first-run friction

## Release Policy

- official releases must set `CLAWRISE_NPM_SCOPE` so the published root package is scoped, for example `@clawrise/clawrise-cli`
- platform packages follow: `clawrise-cli-<platform>-<arch>`
- if a fork or internal environment needs a different package family name, override it with `CLAWRISE_NPM_PACKAGE_PREFIX`
- set `CLAWRISE_NPM_SCOPE` to the publishing organization or user scope, for example `@clawrise`
- default `dist-tag` policy:
  - stable releases such as `1.2.3` go to `latest`
  - prereleases such as `1.2.3-rc.1` or `1.2.3-beta.2` go to `next`
  - override with `CLAWRISE_NPM_DIST_TAG` for tags such as `beta` or `canary`
- GitHub Release tags use `v<version>`, for example `v1.2.3`

## Package Layout

The release flow publishes two package types:

- `@scope/clawrise-cli`
  - the root package
  - exposes the `clawrise` command
  - depends on platform packages through `optionalDependencies`
  - injects the bundled `plugins/` directory into `CLAWRISE_PLUGIN_PATHS`
- `@scope/clawrise-cli-<platform>-<arch>`
  - platform package
  - for example `clawrise-cli-linux-x64`
  - contains the prebuilt `clawrise` binary for that platform
  - contains the bundled first-party provider plugin directories and manifests

## Standard Release Source

Release scripts resolve the version in this order:

1. explicit script argument
2. `CLAWRISE_RELEASE_VERSION`
3. `GITHUB_REF_NAME`

Accepted formats:

- `0.1.0`
- `v0.1.0`

Recommended standard flow:

1. merge release-ready changes into `main`
2. run the local preflight on `main`
3. create a tag on the current `main` commit, for example `v1.2.3`
4. push the tag
5. GitHub Actions listens to `v*` tags and completes the build, GitHub Release, and npm publish steps

Example:

```bash
git checkout main
git pull origin main
bash ./scripts/release/check-release-ready.sh 1.2.3
git tag -a v1.2.3 -m "Release v1.2.3"
git push origin v1.2.3
```

## Local Build

### 1. Build all platform bundles

```bash
./scripts/release/build-npm-bundles.sh 0.1.0
```

Output:

- `dist/release/bundles/`
- `dist/release/archives/`

Each platform bundle includes:

- `bin/clawrise`
- `plugins/feishu/<version>/...`
- `plugins/notion/<version>/...`

### 2. Generate npm package directories

```bash
node ./scripts/release/prepare-npm-packages.mjs 0.1.0
```

Output:

- `dist/release/npm/clawrise-cli`
- `dist/release/npm/clawrise-cli-darwin-arm64`
- `dist/release/npm/clawrise-cli-darwin-x64`
- `dist/release/npm/clawrise-cli-linux-arm64`
- `dist/release/npm/clawrise-cli-linux-x64`
- `dist/release/npm/clawrise-cli-win32-arm64`
- `dist/release/npm/clawrise-cli-win32-x64`
- `dist/release/npm/release-metadata.json`

### 3. Publish to npm

Configure:

```bash
export NODE_AUTH_TOKEN=your_npm_token
```

Then run:

```bash
./scripts/release/publish-npm.sh 0.1.0
```

The script publishes platform packages first and the root package last. Existing versions are skipped automatically.
By default it selects the `dist-tag` from the version, and you can override it:

```bash
export CLAWRISE_NPM_DIST_TAG=beta
./scripts/release/publish-npm.sh 0.1.0-beta.1
```

## GitHub Actions

Workflow file:

- `.github/workflows/release-npm.yml`

Triggers:

- push to `v*` tags
- `workflow_dispatch`

Where:

- tag push is the standard release path
- `workflow_dispatch` is better for reruns, backfills, or operational recovery

The workflow does:

1. resolve the release version
2. run `go test ./...`
3. build all platform bundles
4. generate npm package directories
5. upload archive artifacts and `SHA256SUMS`
6. create or update the GitHub Release and upload archives
7. publish npm packages when `NPM_TOKEN` is configured

Supported workflow inputs and environment variables:

- `npm_scope`
- `npm_package_prefix`
- `npm_dist_tag`
- `cicd` environment variable `CLAWRISE_NPM_SCOPE`
- `cicd` environment variable `CLAWRISE_NPM_PACKAGE_PREFIX`
- `cicd` environment variable `CLAWRISE_NPM_DIST_TAG`

## Local Preflight

Run this on `main` before creating the tag:

```bash
bash ./scripts/release/check-release-ready.sh 1.2.3
```

It checks:

- that you are on `main`
- that the worktree is clean
- that the target tag `v1.2.3` does not already exist
- `go test ./...`
- multi-platform bundles and npm package directory generation
- release notes generation
- `npm pack` for the root package and the current platform package

## Release Notes

Release notes are generated from:

- `packaging/release/release-notes.md.tmpl`
- `scripts/release/generate-release-notes.sh`

Generate them with:

```bash
./scripts/release/generate-release-notes.sh 0.1.0
```

Default output:

- `dist/release/release-notes.md`

## Version Injection

Version metadata is injected into Go binaries through `-ldflags`:

- `internal/buildinfo.Version`
- `internal/buildinfo.Commit`
- `internal/buildinfo.BuildDate`

This keeps the `clawrise` core binary and first-party plugins aligned on the same release version.

## Related Docs

- [npm Release Runbook](./npm-release-runbook.md)
