import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { spawnSync } from 'node:child_process';
import { test } from 'node:test';

import { prepareSkillPackages } from './prepare-skill-packages.mjs';

test('生成 skills release manifest metadata 和归档', () => {
  const repoRoot = createSkillRepoFixture();
  try {
    const result = prepareSkillPackages({
      repoRoot,
      version: '1.2.3',
      baseURL: 'https://example.com/skills',
      generatedAt: '2026-04-05T00:00:00.000Z',
    });

    assert.equal(result.version, '1.2.3');
    assert.equal(result.skillCount, 2);

    const distRoot = path.join(repoRoot, 'dist', 'release', 'skills');
    const indexManifest = readJSON(path.join(distRoot, 'index.json'));
    const latestManifest = readJSON(path.join(distRoot, 'latest.json'));

    assert.equal(indexManifest.latest_version, '1.2.3');
    assert.equal(latestManifest.version, '1.2.3');
    assert.equal(indexManifest.skills.length, 2);

    const notionEntry = indexManifest.skills.find((entry) => entry.name === 'clawrise-notion');
    assert.ok(notionEntry);
    assert.equal(notionEntry.description, 'Notion workflow skill.');
    assert.deepEqual(notionEntry.platforms, ['notion']);
    assert.deepEqual(notionEntry.requires, ['clawrise-core']);
    assert.deepEqual(notionEntry.reference_files, ['references/common-tasks.md', 'references/operation-map.md']);
    assert.equal(notionEntry.url, 'https://example.com/skills/1.2.3/clawrise-notion.tar.gz');

    const archivedFiles = listArchiveFiles(path.join(distRoot, '1.2.3', 'clawrise-notion.tar.gz'));
    assert.deepEqual(
      archivedFiles,
      ['SKILL.md', 'references/common-tasks.md', 'references/operation-map.md'],
    );

    const installScript = fs.readFileSync(path.join(distRoot, 'install.sh'), 'utf8');
    assert.match(installScript, /https:\/\/example\.com\/skills/);
  } finally {
    fs.rmSync(repoRoot, { recursive: true, force: true });
  }
});

test('frontmatter name 与目录名不一致时失败', () => {
  const repoRoot = createSkillRepoFixture();
  try {
    fs.writeFileSync(
      path.join(repoRoot, 'skills', 'clawrise-core', 'SKILL.md'),
      '---\nname: wrong-name\ndescription: Broken skill.\n---\n',
    );

    assert.throws(
      () => prepareSkillPackages({ repoRoot, version: '1.2.3', generatedAt: '2026-04-05T00:00:00.000Z' }),
      /Skill frontmatter name 与目录名不一致/,
    );
  } finally {
    fs.rmSync(repoRoot, { recursive: true, force: true });
  }
});

function createSkillRepoFixture() {
  const repoRoot = fs.mkdtempSync(path.join(os.tmpdir(), 'clawrise-skill-release-'));

  fs.mkdirSync(path.join(repoRoot, 'packaging', 'skills'), { recursive: true });
  fs.writeFileSync(
    path.join(repoRoot, 'packaging', 'skills', 'install.sh'),
    '#!/usr/bin/env bash\nBASE=__CLAWRISE_SKILLS_BASE_URL__\nREADY=__CLAWRISE_SKILLS_BASE_URL_READY__\n',
  );

  writeSkill(repoRoot, 'clawrise-core', 'Generic workflow skill.', [
    'references/install-and-layout.md',
    'references/workflow.md',
  ]);
  writeSkill(repoRoot, 'clawrise-notion', 'Notion workflow skill.', [
    'references/common-tasks.md',
    'references/operation-map.md',
  ]);

  return repoRoot;
}

function writeSkill(repoRoot, skillName, description, referenceFiles) {
  const skillDir = path.join(repoRoot, 'skills', skillName);
  fs.mkdirSync(skillDir, { recursive: true });
  fs.writeFileSync(
    path.join(skillDir, 'SKILL.md'),
    `---\nname: ${skillName}\ndescription: ${description}\n---\n\n# ${skillName}\n`,
  );

  for (const relativeFile of referenceFiles) {
    const filePath = path.join(skillDir, relativeFile);
    fs.mkdirSync(path.dirname(filePath), { recursive: true });
    fs.writeFileSync(filePath, `${skillName}:${relativeFile}\n`);
  }
}

function listArchiveFiles(archivePath) {
  const result = spawnSync('tar', ['-tzf', archivePath], {
    stdio: 'pipe',
  });
  if (result.status !== 0) {
    const stderr = (result.stderr || Buffer.alloc(0)).toString('utf8').trim();
    throw new Error(`Failed to list archive files: ${stderr}`);
  }

  return String(result.stdout || '')
    .split(/\r?\n/)
    .map((line) => ({
      raw: line.trim(),
      normalized: line.trim().replace(/^\.\/+/, '').replace(/\/+$/, ''),
    }))
    .filter((item) => item.normalized !== '' && !item.raw.endsWith('/'))
    .map((item) => item.normalized)
    .sort();
}

function readJSON(filePath) {
  return JSON.parse(fs.readFileSync(filePath, 'utf8'));
}
