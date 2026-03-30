# npm Release Runbook

This document records provenance behavior, common npm release failure cases, and the recommended recovery path.

## Provenance

The current release workflow uses:

- GitHub Actions OIDC permission: `id-token: write`
- npm Trusted Publishing from GitHub Actions

to generate npm provenance for published packages.

Recommended practice:

- perform official releases only from GitHub Actions
- avoid mixing stable releases with manual local `npm publish`
- when auditing a release, compare the GitHub Actions run, the GitHub Release, and npm provenance metadata together

## Recommended Preflight

Run the local preflight script first:

```bash
./scripts/release/check-release-ready.sh 0.1.0
```

It validates:

- version resolution
- that you are on `main`
- clean worktree state
- that the target release tag does not already exist
- `go test ./...`
- multi-platform bundle build
- npm package directory generation
- release notes generation
- `npm pack` on the root package and the current platform package

If you intentionally test in a dirty local workspace:

```bash
CLAWRISE_RELEASE_ALLOW_DIRTY=1 ./scripts/release/check-release-ready.sh 0.1.0-rc.1
```

If you intentionally need to run from a detached HEAD:

```bash
CLAWRISE_RELEASE_ALLOW_DETACHED=1 ./scripts/release/check-release-ready.sh 0.1.0-rc.1
```

If you also want to validate remote auth:

```bash
CLAWRISE_RELEASE_CHECK_REMOTE=1 NODE_AUTH_TOKEN=... ./scripts/release/check-release-ready.sh 0.1.0
```

Without a local npm token, the script skips npm auth verification because Trusted Publishing relies on GitHub Actions OIDC and cannot be fully validated from a local shell.

## Common Failure Cases

### 1. Platform packages published, root package failed

This is usually easy to recover.

Recommended action:

- keep the same version
- rerun the publish script or rerun the workflow
- `scripts/release/publish-npm.sh` skips existing versions and only publishes the missing packages

### 2. The root package was published with the wrong dist-tag

For example, a prerelease landed on `latest`.

Recommended action:

- do not republish the same version
- correct the dist-tag directly

Example:

```bash
npm dist-tag add clawrise-cli@1.2.3 next
npm dist-tag rm clawrise-cli latest
```

If you use a scope, replace the package name accordingly.

### 3. The package contents were wrong for an immutable version

npm versions are immutable. Do not rely on delete-and-republish as the normal recovery path.

Recommended action:

- publish a new version
- use the next patch version for stable fixes, for example `1.2.4`
- continue the prerelease line for prereleases, for example `1.2.4-rc.1`

### 4. GitHub Release assets or notes are wrong

Recommended action:

- rerun the workflow
- or use `gh release upload --clobber` to replace assets
- use `gh release edit` to update release notes

### 5. The workflow failed even though local artifacts look correct

Check:

- whether the target npm packages already trust `release-npm.yml` as a Trusted Publisher
- whether GitHub Actions has `id-token: write`
- whether GitHub or npm registry permissions are restricted
- whether the target version already exists on npm

## Rollback Policy

For stable releases, follow these principles:

- avoid unpublish as the primary rollback path
- remove the wrong version from `latest` instead of deleting the version
- ship a fixed version instead of trying to overwrite an existing one

Recommended order:

1. correct the dist-tag so new installs stop pulling the wrong version
2. publish the fixed version
3. update the GitHub Release notes to call out the affected version

## Manual Decisions Still Required

Automation does not decide:

- whether this release should go to `latest` or `next`
- whether the package name or scope should change
- whether the GitHub Release should be marked as a prerelease
- whether user-facing install docs need to change
