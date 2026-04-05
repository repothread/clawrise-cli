# Installation And Layout

## Quick Map

- [1. Repository Skill Source](#1-repository-skill-source)
- [2. Client Skill Locations](#2-client-skill-locations)
- [3. Default Clawrise Runtime Paths](#3-default-clawrise-runtime-paths)
- [4. Plugin Install Sources](#4-plugin-install-sources)
- [5. Repository Install Script](#5-repository-install-script)
- [6. No-Clone Distribution Options](#6-no-clone-distribution-options)

## 1. Repository Skill Source

The source directories inside this repository are:

```text
skills/clawrise-core
skills/clawrise-feishu
skills/clawrise-notion
```

## 2. Client Skill Locations

Default shared install targets are:

```text
~/.codex/skills/<skill-name>
~/.claude/skills/<skill-name>
~/.openclaw/skills/<skill-name>
~/.config/opencode/skills/<skill-name>
```

Project-local installs may use client-specific directories such as:

```text
./.claude/skills/<skill-name>
./skills/<skill-name>
./.opencode/skills/<skill-name>
```

The exact target is selected by:

```bash
clawrise setup <client> [platform...]
clawrise setup <platform>
```

or:

```bash
npx @clawrise/clawrise-cli setup <client> [platform...]
npx @clawrise/clawrise-cli setup <platform>
```

## 3. Default Clawrise Runtime Paths

In this project, `clawrise doctor` exposes these default paths:

- config file: `~/.clawrise/config.yaml`
- state directory: `~/.clawrise/state`
- runtime directory: `~/.clawrise/state/runtime`
- repo-local plugin directory: `.clawrise/plugins`
- user plugin directory: `~/.clawrise/plugins`

Always trust the live `clawrise doctor` output over static assumptions.

## 4. Plugin Install Sources

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

## 5. Repository Install Script

This repository provides:

```bash
./scripts/skills/setup-local-codex.sh
```

It copies the repository skills into a local Codex skill directory for repository-level testing.

Restart Codex after installation so the new skills are loaded.

## 6. No-Clone Distribution Options

Recommended non-clone channels are:

- website-hosted versioned skill archives
- npm package bundled with the `skills/` directory

For website distribution, publish:

- a skill manifest such as `index.json`
- versioned archives such as `<version>/<skill-name>.tar.gz`
- one install script that downloads and expands skills into `~/.codex/skills`

For npm distribution, prefer setup commands such as:

```bash
clawrise setup codex
clawrise setup notion
clawrise setup codex feishu
clawrise setup claude-code notion --skills-dir ./.claude/skills
```

Do not use automatic `postinstall` hooks to write into the user's Codex directory.
