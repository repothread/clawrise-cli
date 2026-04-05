# Release Main Readiness Checklist

This checklist defines the exact path for moving the current `develop` branch to a state that is ready to merge into `main` and release through the official production workflow.

It is intentionally execution-oriented and should be treated as the release gate for the current branch.

## Current Status

### Already true

- Core unit/integration test suite passes locally with Go cache overrides.
- Core builds successfully with `go build ./...`.
- Release preflight technical chain passes when the branch gate and dirty-worktree gate are intentionally bypassed for local validation.
- External plugin integration with `~/thread/clawrise-plugin-linear` passes in both discovery mode and install mode.
- Bundled first-party provider manifests have been upgraded to manifest v2 in the release bundle pipeline.
- CI now includes release-script regression coverage for:
  - `scripts/release/verify-release-artifacts.test.mjs`
  - `scripts/release/publish-npm.test.sh`

### Not yet true

- The current branch is `develop`, but the official release process is `main`-only.
- The current working tree is not clean.
- Some release hardening items are still recommended before calling the project broadly production-ready.

## Hard Release Gates

These items must be true before an official release from `main`.

### Gate 1. Merge the release candidate into `main`

Required because the documented and automated release flow is `main`-only.

Evidence:

- `docs/en/npm-release-workflow.md`
- `.github/workflows/release-npm.yml`
- `scripts/release/check-release-ready.sh`

Pass condition:

- the exact release commit exists on `main`
- `origin/main` contains the release commit

Suggested commands:

```bash
git checkout main
git pull origin main
git merge --no-ff develop
```

Acceptance criteria:

- `git branch --show-current` returns `main`
- `git merge-base --is-ancestor HEAD origin/main` succeeds after push

### Gate 2. Clean the working tree

The standard release preflight stops if there are uncommitted changes.

Pass condition:

- no staged or unstaged tracked-file modifications remain
- only intentional ignored/generated files remain outside git status

Suggested commands:

```bash
git status --short
```

Acceptance criteria:

- `git diff --quiet --ignore-submodules HEAD --` exits successfully

### Gate 3. Choose the release version and confirm the tag is unused

The release preflight checks tag uniqueness.

Pass condition:

- the chosen version is final
- local and remote tags for that version do not already exist

Suggested commands:

```bash
./scripts/release/resolve-version.sh 0.1.1

git rev-parse -q --verify refs/tags/v0.1.1 || true
git ls-remote --tags origin refs/tags/v0.1.1
```

Acceptance criteria:

- no existing tag conflicts for the target version

### Gate 4. Run the official local preflight on `main`

Run the actual release preflight without bypass flags.

Suggested command:

```bash
./scripts/release/check-release-ready.sh <version>
```

What it must validate successfully:

- branch gate
- clean worktree gate
- tag uniqueness
- `go test ./...`
- multi-platform release bundle generation
- npm package directory generation
- release notes generation
- release artifact consistency checks
- `npm pack` validation
- npm runtime smoke validation

Acceptance criteria:

- command exits with code 0
- output ends with `Release readiness checks passed.`

### Gate 5. Push to `main` and tag from `main`

Suggested commands:

```bash
git push origin main
git tag -a v<version> -m "Release v<version>"
git push origin v<version>
```

Acceptance criteria:

- GitHub Actions `release-npm.yml` starts from the pushed tag
- the workflow does not fail the `origin/main` ancestry check

## Strongly Recommended Before Merge

These are not all hard blockers for a controlled first-party release, but they should be reviewed explicitly before merge.

### R1. Confirm release posture for plugin trust

The current plugin system is good enough for controlled first-party release engineering, but it is not yet a fully hardened open plugin ecosystem.

Decision required:

- either explicitly release with a **controlled first-party** posture
- or delay broader ecosystem claims until stronger trust hardening is implemented

Release-note requirement:

- do not imply that arbitrary remote third-party plugins are fully trust-hardened if that is not yet true

### R2. Decide whether to patch the npm setup UX language before release

There is still user-facing Chinese text in the npm wrapper setup flow.

Relevant file:

- `packaging/npm/root/lib/setup.js`

Decision required:

- either normalize all published CLI-facing text to English before release
- or explicitly accept this as release debt and track it immediately after release

### R3. Review whether more live-provider confidence is needed

Current confidence is mostly from mocked integration tests and release smoke checks, with limited live-provider coverage.

Decision required:

- accept current confidence for this release
- or add one more live or semi-live provider verification lane before merge

## Changes Already Landed In This Branch

The following release-hardening improvement is already present and should be preserved when merging to `main`:

- first-party bundled provider plugin manifests in release bundles now use:
  - `schema_version: 2`
  - explicit `capabilities`
  - `min_core_version`

Relevant files:

- `scripts/release/build-npm-bundles.sh`
- `scripts/release/verify-release-artifacts.mjs`
- `scripts/release/verify-release-artifacts.test.mjs`
- `.github/workflows/ci.yml`

## Suggested Merge-To-Release Sequence

Use this exact order.

1. finish branch review on `develop`
2. ensure `git status` is clean
3. merge `develop` into `main`
4. switch to `main`
5. pull latest `origin/main`
6. run:

   ```bash
   ./scripts/release/check-release-ready.sh <version>
   ```

7. if preflight passes, push `main`
8. create tag `v<version>` on `main`
9. push the tag
10. monitor GitHub Actions release workflow
11. verify published artifacts and npm install path after workflow completion

## Final Merge Decision Rule

The branch is ready to merge into `main` for release only when all of the following are true:

- `develop` changes intended for release are complete
- working tree is clean
- the target version is chosen and unclaimed
- `main` contains the exact release candidate commit
- `./scripts/release/check-release-ready.sh <version>` passes on `main` without bypass flags
- release messaging is aligned with the actual plugin trust posture

If any of the above is false, do not cut the release tag.
