#!/usr/bin/env bash

set -euo pipefail

DEFAULT_BASE_URL="__CLAWRISE_SKILLS_BASE_URL__"
DEFAULT_BASE_URL_READY="__CLAWRISE_SKILLS_BASE_URL_READY__"

version=""
codex_home="${CODEX_HOME:-}"
base_url=""
declare -a skill_names=()

usage() {
  cat <<'EOF'
Usage: install.sh [--version <version>] [--codex-home <dir>] [--base-url <url>] [skill...]

Examples:
  install.sh
  install.sh clawrise-core clawrise-feishu
  install.sh --version 0.2.0
  install.sh --codex-home /tmp/codex-home clawrise-notion
EOF
}

while [[ "$#" -gt 0 ]]; do
  case "$1" in
    --version)
      shift
      version="${1:-}"
      ;;
    --codex-home)
      shift
      codex_home="${1:-}"
      ;;
    --base-url)
      shift
      base_url="${1:-}"
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    -*)
      echo "Unsupported argument: $1" >&2
      exit 1
      ;;
    *)
      skill_names+=("$1")
      ;;
  esac
  shift || true
done

if [[ -z "${base_url}" ]]; then
  base_url="${CLAWRISE_SKILLS_BASE_URL:-${DEFAULT_BASE_URL}}"
fi

if [[ -z "${base_url}" ]]; then
  echo "Missing skills base URL. Set CLAWRISE_SKILLS_BASE_URL during release, or pass --base-url at runtime." >&2
  exit 1
fi

if [[ "${DEFAULT_BASE_URL_READY}" != "1" && "${base_url}" == "${DEFAULT_BASE_URL}" ]]; then
  echo "Missing skills base URL. Set CLAWRISE_SKILLS_BASE_URL during release, or pass --base-url at runtime." >&2
  exit 1
fi

if [[ -z "${codex_home}" ]]; then
  codex_home="${HOME}/.codex"
fi

if [[ -z "${version}" ]]; then
  latest_json="$(curl -fsSL "${base_url}/latest.json")"
  version="$(printf '%s' "${latest_json}" | sed -n 's/.*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1)"
fi

if [[ -z "${version}" ]]; then
  echo "Failed to resolve the skills version to install." >&2
  exit 1
fi

if [[ "${#skill_names[@]}" -eq 0 ]]; then
  skill_names=(
    "clawrise-core"
    "clawrise-feishu"
    "clawrise-notion"
  )
fi

target_root="${codex_home}/skills"
mkdir -p "${target_root}"

work_dir="$(mktemp -d)"
cleanup() {
  rm -rf "${work_dir}"
}
trap cleanup EXIT

for skill_name in "${skill_names[@]}"; do
  archive_path="${work_dir}/${skill_name}.tar.gz"
  download_url="${base_url}/${version}/${skill_name}.tar.gz"
  target_dir="${target_root}/${skill_name}"

  echo "Downloading skill: ${download_url}"
  curl -fsSL "${download_url}" -o "${archive_path}"

  rm -rf "${target_dir}"
  mkdir -p "${target_dir}"
  tar -xzf "${archive_path}" -C "${target_dir}"
  echo "Installed skill: ${skill_name} -> ${target_dir}"
done

echo "Installation completed. Restart Codex to load the new skills."
