#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "$0")/../.." && pwd)"
version="$("${repo_root}/scripts/release/resolve-version.sh" "${1:-}")"
metadata_path="${repo_root}/dist/release/npm/release-metadata.json"
template_path="${repo_root}/packaging/release/release-notes.md.tmpl"
output_path="${2:-${CLAWRISE_RELEASE_NOTES_PATH:-${repo_root}/dist/release/release-notes.md}}"
repository="${CLAWRISE_GITHUB_REPOSITORY:-${GITHUB_REPOSITORY:-repothread/clawrise-cli}}"
git_sha="${CLAWRISE_RELEASE_GIT_SHA:-${GITHUB_SHA:-$(git -C "${repo_root}" rev-parse HEAD 2>/dev/null || printf 'HEAD')}}"

if [[ ! -f "${metadata_path}" ]]; then
  echo "缺少 npm 发布元数据文件，请先执行 prepare-npm-packages.mjs: ${metadata_path}" >&2
  exit 1
fi

if [[ ! -f "${template_path}" ]]; then
  echo "缺少 release notes 模板文件: ${template_path}" >&2
  exit 1
fi

install_package="$(
  node -e "const fs=require('fs'); const data=JSON.parse(fs.readFileSync(process.argv[1], 'utf8')); process.stdout.write(data.root_package.package_name);" "${metadata_path}"
)"

archives_dir="${repo_root}/dist/release/archives"
mapfile -t archives < <(find "${archives_dir}" -maxdepth 1 -type f -name '*.tar.gz' -exec basename {} \; | sort)
if [[ "${#archives[@]}" -eq 0 ]]; then
  echo "未找到 release 归档文件，请先执行 build-npm-bundles.sh。" >&2
  exit 1
fi

release_download_base="https://github.com/${repository}/releases/download/v${version}"
asset_lines=()
for archive_name in "${archives[@]}"; do
  asset_lines+=("- [\`${archive_name}\`](${release_download_base}/${archive_name})")
done
asset_lines+=("- [\`SHA256SUMS\`](${release_download_base}/SHA256SUMS)")
asset_list="$(printf '%s\n' "${asset_lines[@]}")"

resolve_previous_tag() {
  git -C "${repo_root}" describe --tags --abbrev=0 "${git_sha}^" 2>/dev/null || true
}

build_contributor_list_from_github() {
  local previous_tag="$1"
  local github_token="$2"

  if [[ -z "${previous_tag}" || -z "${github_token}" ]]; then
    return 1
  fi

  CLAWRISE_RELEASE_REPOSITORY="${repository}" \
    CLAWRISE_RELEASE_BASE_TAG="${previous_tag}" \
    CLAWRISE_RELEASE_HEAD_SHA="${git_sha}" \
    CLAWRISE_RELEASE_GITHUB_TOKEN="${github_token}" \
    node <<'EOF'
const repository = process.env.CLAWRISE_RELEASE_REPOSITORY || '';
const baseTag = process.env.CLAWRISE_RELEASE_BASE_TAG || '';
const headSha = process.env.CLAWRISE_RELEASE_HEAD_SHA || '';
const githubToken = process.env.CLAWRISE_RELEASE_GITHUB_TOKEN || '';

if (!repository || !baseTag || !headSha || !githubToken) {
  process.exit(1);
}

function avatarWithSize(url, size) {
  if (!url) {
    return '';
  }
  return url.includes('?') ? `${url}&s=${size}` : `${url}?s=${size}`;
}

async function main() {
  const compareURL = `https://api.github.com/repos/${repository}/compare/${encodeURIComponent(baseTag)}...${encodeURIComponent(headSha)}`;
  const response = await fetch(compareURL, {
    headers: {
      Accept: 'application/vnd.github+json',
      Authorization: `Bearer ${githubToken}`,
      'X-GitHub-Api-Version': '2022-11-28',
    },
  });

  if (!response.ok) {
    process.exit(1);
  }

  const payload = await response.json();
  const contributorMap = new Map();

  for (const commit of payload.commits || []) {
    const author = commit?.author;
    if (!author?.login) {
      continue;
    }

    const current = contributorMap.get(author.login) || {
      count: 0,
      avatarURL: author.avatar_url || '',
      profileURL: author.html_url || `https://github.com/${author.login}`,
    };

    current.count += 1;
    if (!current.avatarURL && author.avatar_url) {
      current.avatarURL = author.avatar_url;
    }
    if (!current.profileURL && author.html_url) {
      current.profileURL = author.html_url;
    }

    contributorMap.set(author.login, current);
  }

  if (contributorMap.size === 0) {
    process.exit(1);
  }

  const lines = [...contributorMap.entries()]
    .sort((left, right) => {
      const countDiff = right[1].count - left[1].count;
      if (countDiff !== 0) {
        return countDiff;
      }
      return left[0].localeCompare(right[0]);
    })
    .map(([login, info]) => {
      const profileURL = info.profileURL || `https://github.com/${login}`;
      const avatarMarkdown = info.avatarURL
        ? `[![@${login}](${avatarWithSize(info.avatarURL, 40)})](${profileURL})`
        : '';
      const suffix = info.count === 1 ? 'commit' : 'commits';
      return `- ${avatarMarkdown} [@${login}](${profileURL}) (${info.count} ${suffix})`.trim();
    });

  process.stdout.write(lines.join('\n'));
}

try {
  await main();
} catch {
  process.exit(1);
}
EOF
}

build_contributor_list() {
  local previous_tag="$1"
  local revision_range="${git_sha}"
  local contributors

  if [[ -n "${previous_tag}" ]]; then
    revision_range="${previous_tag}..${git_sha}"
  fi

  contributors="$(
    git -C "${repo_root}" shortlog -sn "${revision_range}" 2>/dev/null | awk '
      NF {
        count = $1
        $1 = ""
        sub(/^[ \t]+/, "", $0)
        suffix = (count == 1 ? "commit" : "commits")
        printf("- %s (%s %s)\n", $0, count, suffix)
      }
    '
  )"

  if [[ -n "${contributors}" ]]; then
    printf '%s\n' "${contributors}"
    return 0
  fi

  printf '%s\n' "- Contributors unavailable."
}

previous_tag="$(resolve_previous_tag || true)"
github_token="${CLAWRISE_GITHUB_TOKEN:-${GITHUB_TOKEN:-}}"
contributor_list="$(
  build_contributor_list_from_github "${previous_tag}" "${github_token}" ||
    build_contributor_list "${previous_tag}"
)"

rendered="$(cat "${template_path}")"
rendered="${rendered//\{\{VERSION\}\}/${version}}"
rendered="${rendered//\{\{INSTALL_PACKAGE\}\}/${install_package}}"
rendered="${rendered//\{\{GIT_SHA\}\}/${git_sha}}"
rendered="${rendered//\{\{CONTRIBUTOR_LIST\}\}/${contributor_list}}"
rendered="${rendered//\{\{ASSET_LIST\}\}/${asset_list}}"

mkdir -p "$(dirname "${output_path}")"
printf '%s\n' "${rendered}" > "${output_path}"

echo "已生成 release notes: ${output_path}"
