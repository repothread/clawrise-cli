# Sample Plugins

This directory contains source fixtures for local plugin development.

- `sample-policy`: a minimal policy plugin that asks for `input.change_reason` on mutating requests
- `sample-audit`: a minimal audit sink plugin that appends newline-delimited JSON to a file

Each fixture uses a local `run.sh` wrapper that executes `go run` against the matching source under `cmd/`.

Formal third-party plugins should live in their own repositories, publish their own release artifacts, and be installed through npm package names such as:

```bash
clawrise plugin install @clawrise/clawrise-plugin-feishu
```

Use this directory as a discovery root during development:

```bash
CLAWRISE_PLUGIN_PATHS=$PWD/examples/plugins clawrise doctor
```

These fixtures are intentionally optimized for authoring and protocol verification inside this repository. They are not release artifacts.
