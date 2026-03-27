# Contributing to Clawrise

Thanks for considering a contribution to Clawrise.

This repository is still evolving, so the most helpful contributions are focused, well-explained, and easy to review.

For Chinese contributors, see [CONTRIBUTING.zh.md](CONTRIBUTING.zh.md).

## Ways to Contribute

- Report reproducible bugs.
- Propose new CLI workflows, playbooks, or provider capabilities.
- Improve documentation, examples, and onboarding flows.
- Add tests for runtime behavior, config resolution, and adapter mapping.
- Refine plugin packaging, verification, and discovery behavior.

## Before You Start

- Read [README.md](README.md) for the current project scope and design links.
- Keep the architecture boundary intact: Clawrise unifies runtime execution, not provider resource schemas.
- Prefer small pull requests that address one concern at a time.
- If your change affects CLI behavior, config shape, or operation contracts, update the related docs in the same pull request.

## Development Setup

```bash
go build ./...
go test ./...
go run ./cmd/clawrise version
go run ./cmd/clawrise doctor
```

If local Go caches are restricted, use:

```bash
GOCACHE=/tmp/clawrise-go-build GOMODCACHE=/tmp/clawrise-gomodcache go test ./...
```

## Contribution Workflow

1. Fork the repository and create a focused branch.
2. Make the smallest practical change that solves the problem completely.
3. Run `gofmt` on every Go file you touched.
4. Run `go test ./...` and, when relevant, `go build ./...`.
5. Update documentation, examples, or playbooks when behavior changes.
6. Open a pull request with clear context, tradeoffs, and test evidence.

## Pull Request Expectations

Please include:

- what changed and why
- affected commands, operations, or plugins
- config or compatibility impact
- test evidence such as `go test ./...`
- sample CLI output when it helps reviewers validate behavior

Preferred commit style:

- short, imperative messages
- Conventional Commits when practical, for example `feat: add notion comment append`

## Reporting Bugs and Requesting Features

- Use the GitHub issue templates when possible.
- Include reproduction steps, expected behavior, and actual behavior.
- For API-related issues, include the provider, operation name, and any sanitized payload that helps reproduce the problem.
- Never include real secrets, access tokens, or private tenant data.

## Documentation and Language

- English documentation is the default entry point for global contributors.
- Chinese documentation should stay aligned when the same concept exists in both languages.
- If you only have bandwidth to update one language, note that clearly in the pull request so maintainers can follow up.

## Review Process

Maintainers may ask contributors to:

- narrow the scope of a pull request
- split refactors from behavior changes
- add or adjust tests
- update docs before merge

These requests are meant to keep the repository reviewable and stable as the project grows.
