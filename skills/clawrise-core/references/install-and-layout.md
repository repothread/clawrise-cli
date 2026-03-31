# Installation And Layout

## 1. Codex Skill Location

The source directories inside this repository are:

```text
skills/clawrise-core
skills/clawrise-feishu
skills/clawrise-notion
```

The install target should be:

```text
$CODEX_HOME/skills/<skill-name>
```

If `CODEX_HOME` is not set, the default is usually:

```text
~/.codex/skills/<skill-name>
```

## 2. Default Clawrise Runtime Paths

In this project, `clawrise doctor` exposes these default paths:

- config file: `~/.clawrise/config.yaml`
- state directory: `~/.clawrise/state`
- runtime directory: `~/.clawrise/state/runtime`
- repo-local plugin directory: `.clawrise/plugins`
- user plugin directory: `~/.clawrise/plugins`

Always trust the live `clawrise doctor` output over static assumptions.

## 3. Plugin Install Sources

Clawrise currently supports:

- `file://`
- `https://`
- `npm://`

Useful commands:

```bash
clawrise plugin list
clawrise plugin install <source>
clawrise plugin verify --all
```

## 4. Repository Install Script

This repository provides:

```bash
./scripts/skills/install-codex-skills.sh
```

It copies the repository skills into the local Codex skill directory.

Restart Codex after installation so the new skills are loaded.

## 5. No-Clone Distribution Options

Recommended non-clone channels are:

- website-hosted versioned skill archives
- npm package bundled with the `skills/` directory

For website distribution, publish:

- a skill manifest such as `index.json`
- versioned archives such as `<version>/<skill-name>.tar.gz`
- one install script that downloads and expands skills into `~/.codex/skills`

For npm distribution, prefer an explicit install entry such as:

```bash
clawrise skills install-codex
```

Do not use automatic `postinstall` hooks to write into the user's Codex directory.
