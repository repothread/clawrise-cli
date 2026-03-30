#!/usr/bin/env bash

set -euo pipefail

# 按优先级解析发布版本：
# 1. 脚本第一个参数
# 2. CLAWRISE_RELEASE_VERSION
# 3. GITHUB_REF_NAME
# 支持：
# - 0.1.0
# - v0.1.0
# - release/0.1.0
# - release/v0.1.0

candidate="${1:-${CLAWRISE_RELEASE_VERSION:-${GITHUB_REF_NAME:-}}}"
candidate="${candidate#refs/heads/}"
candidate="${candidate#refs/tags/}"
candidate="${candidate#release/}"
candidate="${candidate#release-}"
candidate="${candidate#v}"

if [[ -z "${candidate}" ]]; then
  echo "未提供发布版本，请传入参数、设置 CLAWRISE_RELEASE_VERSION，或从 GITHUB_REF_NAME 推导。" >&2
  exit 1
fi

if [[ ! "${candidate}" =~ ^[0-9]+\.[0-9]+\.[0-9]+([-.][0-9A-Za-z.-]+)?$ ]]; then
  echo "无效的发布版本: ${candidate}" >&2
  exit 1
fi

printf '%s\n' "${candidate}"
