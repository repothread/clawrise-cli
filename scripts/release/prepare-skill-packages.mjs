#!/usr/bin/env node

import crypto from 'node:crypto';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { spawnSync } from 'node:child_process';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const repoRoot = path.resolve(__dirname, '..', '..');
const version = resolveVersion(process.argv[2], process.env.CLAWRISE_RELEASE_VERSION, process.env.GITHUB_REF_NAME);
const skillsSourceRoot = path.join(repoRoot, 'skills');
const distRoot = path.join(repoRoot, 'dist', 'release', 'skills');
const versionRoot = path.join(distRoot, version);
const templateInstallScriptPath = path.join(repoRoot, 'packaging', 'skills', 'install.sh');
const baseURL = normalizeBaseURL(process.env.CLAWRISE_SKILLS_BASE_URL || '');
const generatedAt = new Date().toISOString();

const skillNames = fs.readdirSync(skillsSourceRoot, { withFileTypes: true })
  .filter((entry) => entry.isDirectory())
  .map((entry) => entry.name)
  .sort();

if (skillNames.length === 0) {
  throw new Error(`No skill directories were found: ${skillsSourceRoot}`);
}

resetDir(distRoot);
fs.mkdirSync(versionRoot, { recursive: true });

const latestSkills = [];
for (const skillName of skillNames) {
  const sourceDir = path.join(skillsSourceRoot, skillName);
  const archivePath = path.join(versionRoot, `${skillName}.tar.gz`);

  archiveSkill(sourceDir, archivePath);
  const checksum = sha256File(archivePath);
  const relativeArchivePath = `${version}/${skillName}.tar.gz`;

  latestSkills.push({
    name: skillName,
    version,
    archive_path: relativeArchivePath,
    url: joinURL(baseURL, relativeArchivePath),
    checksum_sha256: checksum,
    size_bytes: fs.statSync(archivePath).size,
  });
}

writeJSON(path.join(distRoot, 'latest.json'), {
  version,
  generated_at: generatedAt,
  skills: latestSkills,
});

writeJSON(path.join(distRoot, 'index.json'), {
  generated_at: generatedAt,
  latest_version: version,
  skills: latestSkills,
});

const installScript = fs.readFileSync(templateInstallScriptPath, 'utf8')
  .replaceAll('__CLAWRISE_SKILLS_BASE_URL__', baseURL || '__CLAWRISE_SKILLS_BASE_URL__')
  .replaceAll('__CLAWRISE_SKILLS_BASE_URL_READY__', baseURL ? '1' : '0');
fs.writeFileSync(path.join(distRoot, 'install.sh'), installScript);
fs.chmodSync(path.join(distRoot, 'install.sh'), 0o755);

console.log(`Generated skills release directory: ${distRoot}`);

function archiveSkill(sourceDir, archivePath) {
  const result = spawnSync('tar', ['-C', sourceDir, '-czf', archivePath, '.'], {
    stdio: 'pipe',
  });
  if (result.status !== 0) {
    const stderr = (result.stderr || Buffer.alloc(0)).toString('utf8').trim();
    throw new Error(`Failed to archive skill ${sourceDir}: ${stderr}`);
  }
}

function sha256File(filePath) {
  const hash = crypto.createHash('sha256');
  hash.update(fs.readFileSync(filePath));
  return hash.digest('hex');
}

function writeJSON(filePath, value) {
  fs.writeFileSync(filePath, `${JSON.stringify(value, null, 2)}\n`);
}

function resetDir(targetDir) {
  fs.rmSync(targetDir, { recursive: true, force: true });
  fs.mkdirSync(targetDir, { recursive: true });
}

function normalizeBaseURL(rawValue) {
  const trimmed = String(rawValue || '').trim();
  if (trimmed === '') {
    return '';
  }
  return trimmed.replace(/\/+$/, '');
}

function joinURL(base, relativePath) {
  const cleanedRelativePath = String(relativePath || '').replace(/^\/+/, '');
  if (base === '') {
    return cleanedRelativePath;
  }
  return `${base}/${cleanedRelativePath}`;
}

function resolveVersion(cliArg, envVersion, refName) {
  const raw = firstNonEmpty(cliArg, envVersion, refName);
  if (!raw) {
    throw new Error('Missing release version. Pass it as an argument, set CLAWRISE_RELEASE_VERSION, or provide GITHUB_REF_NAME.');
  }

  const normalized = raw
    .replace(/^refs\/heads\//, '')
    .replace(/^refs\/tags\//, '')
    .replace(/^release\//, '')
    .replace(/^release-/, '')
    .replace(/^v/, '');

  if (!/^[0-9]+\.[0-9]+\.[0-9]+([-.][0-9A-Za-z.-]+)?$/.test(normalized)) {
    throw new Error(`Invalid release version: ${normalized}`);
  }
  return normalized;
}

function firstNonEmpty(...values) {
  for (const value of values) {
    if (typeof value === 'string' && value.trim() !== '') {
      return value.trim();
    }
  }
  return '';
}
