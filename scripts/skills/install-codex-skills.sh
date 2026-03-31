#!/usr/bin/env bash

set -euo pipefail

# Install repository-bundled Codex skills into the local skills directory.

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
CODEX_HOME_DIR="${CODEX_HOME:-$HOME/.codex}"
SKILLS_ROOT_DIR="${CODEX_HOME_DIR}/skills"

declare -a SKILL_NAMES=(
  "clawrise-core"
  "clawrise-feishu"
  "clawrise-notion"
)

if [[ "$#" -gt 0 ]]; then
  SKILL_NAMES=("$@")
fi

mkdir -p "${SKILLS_ROOT_DIR}"

for skill_name in "${SKILL_NAMES[@]}"; do
  source_dir="${ROOT_DIR}/skills/${skill_name}"
  target_dir="${SKILLS_ROOT_DIR}/${skill_name}"

  if [[ ! -d "${source_dir}" ]]; then
    echo "Skill directory not found: ${source_dir}" >&2
    exit 1
  fi

  rm -rf "${target_dir}"
  cp -R "${source_dir}" "${target_dir}"
  echo "Installed skill: ${skill_name} -> ${target_dir}"
done

echo "Installation completed. Restart Codex to load the new skills."
