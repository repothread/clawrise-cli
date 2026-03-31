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

resolve_previous_tag() {
  git -C "${repo_root}" describe --tags --abbrev=0 "${git_sha}^" 2>/dev/null || true
}

build_changelog_sections() {
  local previous_tag="$1"
  local revision_range="${git_sha}"

  if [[ -n "${previous_tag}" ]]; then
    revision_range="${previous_tag}..${git_sha}"
  fi

  CLAWRISE_RELEASE_REPO_ROOT="${repo_root}" \
    CLAWRISE_RELEASE_REVISION_RANGE="${revision_range}" \
    node <<'EOF'
const { execFileSync } = require('node:child_process');

const repoRoot = process.env.CLAWRISE_RELEASE_REPO_ROOT || '';
const revisionRange = process.env.CLAWRISE_RELEASE_REVISION_RANGE || '';

if (!repoRoot || !revisionRange) {
  process.exit(1);
}

function parseConventionalSubject(subject) {
  const trimmed = String(subject || '').trim();
  const match = trimmed.match(/^([a-z0-9_-]+)(?:\(([^)]+)\))?(!)?:\s*(.+)$/i);
  if (!match) {
    return {
      type: '',
      scope: '',
      breaking: false,
      description: trimmed,
    };
  }
  return {
    type: String(match[1] || '').toLowerCase(),
    scope: String(match[2] || '').trim(),
    breaking: Boolean(match[3]),
    description: String(match[4] || '').trim(),
  };
}

function normalizeBulletText(record) {
  const description = record.description || record.subject;
  let text = record.scope ? `${record.scope}: ${description}` : description;
  text = text.trim();
  if (text === '') {
    text = record.subject.trim();
  }
  if (text !== '' && !/[.!?]$/.test(text)) {
    text += '.';
  }
  return text;
}

function typePriority(type) {
  switch (type) {
    case 'feat':
      return 0;
    case 'fix':
      return 1;
    case 'refactor':
      return 2;
    case 'perf':
      return 3;
    case 'docs':
      return 4;
    case 'test':
      return 5;
    case 'build':
    case 'ci':
    case 'chore':
      return 6;
    default:
      return 7;
  }
}

// 按提交信息生成接近 openclaw 风格的分节说明，避免在 release body 中渲染头像。
function collectSections() {
  const rawLog = execFileSync(
    'git',
    ['-C', repoRoot, 'log', '--format=%H%x00%s%x00%b%x1e', revisionRange],
    { encoding: 'utf8' }
  );

  const commits = rawLog
    .split('\x1e')
    .map((entry) => entry.trim())
    .filter(Boolean)
    .map((entry, index) => {
      const fields = entry.split('\x00');
      const subject = String(fields[1] || '').trim();
      const body = String(fields.slice(2).join('\x00') || '').trim();
      const parsed = parseConventionalSubject(subject);
      return {
        index,
        subject,
        body,
        type: parsed.type,
        scope: parsed.scope,
        description: parsed.description || subject,
        breaking: parsed.breaking || /\bBREAKING CHANGE\b/i.test(body),
      };
    })
    .filter((entry) => entry.subject !== '');

  if (commits.length === 0) {
    return [
      '### Changes',
      '',
      '- No user-facing changes were detected in this release.',
    ].join('\n');
  }

  const breaking = [];
  const changes = [];

  for (const commit of commits) {
    const item = {
      index: commit.index,
      priority: typePriority(commit.type),
      text: normalizeBulletText(commit),
    };
    if (commit.breaking) {
      breaking.push(item);
      continue;
    }
    changes.push(item);
  }

  changes.sort((left, right) => {
    if (left.priority !== right.priority) {
      return left.priority - right.priority;
    }
    return left.index - right.index;
  });

  const sections = [];
  if (breaking.length > 0) {
    sections.push(
      [
        '### Breaking',
        '',
        ...breaking.map((item) => `- ${item.text}`),
      ].join('\n')
    );
  }

  sections.push(
    [
      '### Changes',
      '',
      ...(
        changes.length > 0
          ? changes.map((item) => `- ${item.text}`)
          : ['- No non-breaking user-facing changes were detected in this release.']
      ),
    ].join('\n')
  );

  return sections.join('\n\n');
}

try {
  process.stdout.write(collectSections());
} catch {
  process.exit(1);
}
EOF
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
      profileURL: author.html_url || `https://github.com/${author.login}`,
    };

    current.count += 1;
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
      const suffix = info.count === 1 ? 'commit' : 'commits';
      return `- [@${login}](${profileURL}) (${info.count} ${suffix})`;
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
changelog_sections="$(
  build_changelog_sections "${previous_tag}"
)"
contributor_list="$(
  build_contributor_list_from_github "${previous_tag}" "${github_token}" ||
    build_contributor_list "${previous_tag}"
)"

rendered="$(cat "${template_path}")"
rendered="${rendered//\{\{VERSION\}\}/${version}}"
rendered="${rendered//\{\{INSTALL_PACKAGE\}\}/${install_package}}"
rendered="${rendered//\{\{CHANGELOG_SECTIONS\}\}/${changelog_sections}}"
rendered="${rendered//\{\{CONTRIBUTOR_LIST\}\}/${contributor_list}}"

mkdir -p "$(dirname "${output_path}")"
printf '%s\n' "${rendered}" > "${output_path}"

echo "已生成 release notes: ${output_path}"
