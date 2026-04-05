#!/usr/bin/env node

import crypto from 'node:crypto';
import { spawnSync } from 'node:child_process';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

export const releaseTargets = [
  { npmOS: 'darwin', npmCPU: 'arm64', archiveOS: 'darwin', archiveArch: 'arm64' },
  { npmOS: 'darwin', npmCPU: 'x64', archiveOS: 'darwin', archiveArch: 'amd64' },
  { npmOS: 'linux', npmCPU: 'arm64', archiveOS: 'linux', archiveArch: 'arm64' },
  { npmOS: 'linux', npmCPU: 'x64', archiveOS: 'linux', archiveArch: 'amd64' },
  { npmOS: 'win32', npmCPU: 'arm64', archiveOS: 'windows', archiveArch: 'arm64' },
  { npmOS: 'win32', npmCPU: 'x64', archiveOS: 'windows', archiveArch: 'amd64' },
];

const bundledProviderPlugins = ['feishu', 'notion'];
const protocolVersion = 1;
const manifestSchemaVersion = 2;

export function verifyReleaseArtifacts(options = {}) {
  const repoRoot = path.resolve(options.repoRoot || path.join(__dirname, '..', '..'));
  const npmRoot = path.join(repoRoot, 'dist', 'release', 'npm');
  const bundlesRoot = path.join(repoRoot, 'dist', 'release', 'bundles');
  const archivesRoot = path.join(repoRoot, 'dist', 'release', 'archives');
  const metadataPath = path.join(npmRoot, 'release-metadata.json');
  const notesPath = path.join(repoRoot, 'dist', 'release', 'release-notes.md');

  const metadata = readJSON(metadataPath);
  const expectedVersion = String(options.version || metadata.version || '').trim();
  assertCondition(expectedVersion !== '', '缺少 release version，无法验证发布产物。');
  assertCondition(metadata.version === expectedVersion, `release metadata 版本不一致：期望 ${expectedVersion}，实际 ${metadata.version || '<empty>'}`);

  const explicitDistTag = String(options.distTag || '').trim();
  const expectedDistTag = resolveDistTag(expectedVersion, explicitDistTag);
  assertCondition(
    String(metadata.dist_tag || '').trim() === expectedDistTag,
    `release metadata dist-tag 不一致：期望 ${expectedDistTag}，实际 ${metadata.dist_tag || '<empty>'}`,
  );

  verifyRootPackage(npmRoot, metadata, expectedVersion);
  verifyPlatformPackages(npmRoot, metadata, expectedVersion);
  verifyBundles(bundlesRoot, expectedVersion);
  verifyArchives(archivesRoot, expectedVersion);
  verifyReleaseNotes(notesPath, metadata, expectedVersion);
  const skillCount = verifySkillPackages(repoRoot, expectedVersion);

  return {
    version: expectedVersion,
    distTag: expectedDistTag,
    rootPackage: metadata.root_package.package_name,
    platformCount: metadata.platform_packages.length,
    archiveCount: releaseTargets.length,
    skillCount,
  };
}

export function resolveDistTag(version, explicitDistTag = '') {
  const trimmed = String(explicitDistTag || '').trim();
  if (trimmed !== '') {
    return trimmed;
  }
  return String(version || '').includes('-') ? 'next' : 'latest';
}

function verifyRootPackage(npmRoot, metadata, version) {
  assertCondition(metadata.root_package && typeof metadata.root_package === 'object', 'release metadata 缺少 root_package。');

  const rootDirName = String(metadata.root_package.dir_name || '').trim();
  const rootPackageName = String(metadata.root_package.package_name || '').trim();
  assertCondition(rootDirName !== '', 'release metadata 缺少 root package 目录名。');
  assertCondition(rootPackageName !== '', 'release metadata 缺少 root package 包名。');

  const rootDir = path.join(npmRoot, rootDirName);
  assertDirectory(rootDir, `缺少 root npm 包目录：${rootDir}`);

  const packageJSON = readJSON(path.join(rootDir, 'package.json'));
  assertCondition(packageJSON.name === rootPackageName, `root package 名称不一致：期望 ${rootPackageName}，实际 ${packageJSON.name || '<empty>'}`);
  assertCondition(packageJSON.version === version, `root package 版本不一致：期望 ${version}，实际 ${packageJSON.version || '<empty>'}`);

  const optionalDependencies = packageJSON.optionalDependencies || {};
  const expectedPlatformPackages = new Set(metadata.platform_packages.map((item) => item.package_name));
  assertCondition(
    Object.keys(optionalDependencies).length === expectedPlatformPackages.size,
    'root package optionalDependencies 数量与平台包数量不一致。',
  );
  for (const packageName of expectedPlatformPackages) {
    assertCondition(optionalDependencies[packageName] === version, `root package 缺少或错误声明 optional dependency：${packageName}@${version}`);
  }

  assertFile(path.join(rootDir, 'bin', 'clawrise.js'), 'root package 缺少 bin/clawrise.js。');
  assertFile(path.join(rootDir, 'lib', 'platform.js'), 'root package 缺少 lib/platform.js。');
  assertFile(path.join(rootDir, 'lib', 'setup.js'), 'root package 缺少 lib/setup.js。');
  assertDirectory(path.join(rootDir, 'skills'), 'root package 缺少 skills 目录。');

  const readme = fs.readFileSync(path.join(rootDir, 'README.md'), 'utf8');
  assertCondition(
    readme.includes(`npm install -g ${rootPackageName}`),
    `root package README 缺少安装命令：npm install -g ${rootPackageName}`,
  );
}

function verifyPlatformPackages(npmRoot, metadata, version) {
  assertCondition(Array.isArray(metadata.platform_packages), 'release metadata 缺少 platform_packages。');
  assertCondition(
    metadata.platform_packages.length === releaseTargets.length,
    `release metadata 平台包数量异常：期望 ${releaseTargets.length}，实际 ${metadata.platform_packages.length}`,
  );

  for (const target of releaseTargets) {
    const item = metadata.platform_packages.find((entry) => entry.os === target.npmOS && entry.cpu === target.npmCPU);
    assertCondition(item, `release metadata 缺少平台包：${target.npmOS}/${target.npmCPU}`);

    const packageDir = path.join(npmRoot, item.dir_name);
    assertDirectory(packageDir, `缺少平台 npm 包目录：${packageDir}`);

    const packageJSON = readJSON(path.join(packageDir, 'package.json'));
    assertCondition(packageJSON.name === item.package_name, `平台包名称不一致：期望 ${item.package_name}，实际 ${packageJSON.name || '<empty>'}`);
    assertCondition(packageJSON.version === version, `平台包版本不一致：期望 ${version}，实际 ${packageJSON.version || '<empty>'}`);
    assertCondition(Array.isArray(packageJSON.os) && packageJSON.os.length === 1 && packageJSON.os[0] === target.npmOS, `平台包 os 字段不一致：${item.package_name}`);
    assertCondition(Array.isArray(packageJSON.cpu) && packageJSON.cpu.length === 1 && packageJSON.cpu[0] === target.npmCPU, `平台包 cpu 字段不一致：${item.package_name}`);

    const binaryName = resolveCoreBinaryName(target);
    assertFile(path.join(packageDir, 'bin', binaryName), `平台包缺少二进制：${item.package_name}/bin/${binaryName}`);

    for (const pluginName of bundledProviderPlugins) {
      const pluginDir = path.join(packageDir, 'plugins', pluginName, version);
      assertDirectory(pluginDir, `平台包缺少内建 provider plugin 目录：${item.package_name}/${pluginName}@${version}`);
      verifyBundledProviderManifest(path.join(pluginDir, 'plugin.json'), pluginName, version, target);
    }
  }
}

function verifyBundles(bundlesRoot, version) {
  for (const target of releaseTargets) {
    const bundleDir = path.join(bundlesRoot, `${target.npmOS}-${target.npmCPU}`);
    assertDirectory(bundleDir, `缺少平台 bundle 目录：${bundleDir}`);
    assertFile(path.join(bundleDir, 'bin', resolveCoreBinaryName(target)), `平台 bundle 缺少 core 二进制：${bundleDir}`);

    for (const pluginName of bundledProviderPlugins) {
      const pluginDir = path.join(bundleDir, 'plugins', pluginName, version);
      assertDirectory(pluginDir, `平台 bundle 缺少 provider plugin 目录：${pluginDir}`);
      verifyBundledProviderManifest(path.join(pluginDir, 'plugin.json'), pluginName, version, target);
    }
  }
}

function verifyBundledProviderManifest(manifestPath, pluginName, version, target) {
  const manifest = readJSON(manifestPath);
  assertCondition(manifest.schema_version === manifestSchemaVersion, `内建 provider manifest schema_version 不正确：${manifestPath}`);
  assertCondition(manifest.name === pluginName, `内建 provider manifest 名称不一致：${manifestPath}`);
  assertCondition(manifest.version === version, `内建 provider manifest 版本不一致：${manifestPath}`);
  assertCondition(manifest.protocol_version === protocolVersion, `内建 provider manifest protocol_version 不正确：${manifestPath}`);
  assertCondition(manifest.min_core_version === version, `内建 provider manifest min_core_version 不正确：${manifestPath}`);
  assertCondition(
    Array.isArray(manifest.capabilities) &&
      manifest.capabilities.length === 1 &&
      manifest.capabilities[0] &&
      manifest.capabilities[0].type === 'provider' &&
      Array.isArray(manifest.capabilities[0].platforms) &&
      manifest.capabilities[0].platforms.length === 1 &&
      manifest.capabilities[0].platforms[0] === pluginName,
    `内建 provider manifest capabilities 不正确：${manifestPath}`,
  );

  const expectedBinary = `./bin/${resolvePluginBinaryName(pluginName, target)}`;
  assertCondition(
    manifest.entry && Array.isArray(manifest.entry.command) && manifest.entry.command[0] === expectedBinary,
    `内建 provider manifest entry.command 不正确：${manifestPath}`,
  );
}

function verifyArchives(archivesRoot, version) {
  assertDirectory(archivesRoot, `缺少归档目录：${archivesRoot}`);
  const checksumPath = path.join(archivesRoot, 'SHA256SUMS');
  assertFile(checksumPath, `缺少校验文件：${checksumPath}`);

  const checksumEntries = parseChecksumFile(checksumPath);
  const expectedArchives = new Set();

  for (const target of releaseTargets) {
    const archiveName = `clawrise-cli_${version}_${target.archiveOS}-${target.archiveArch}.tar.gz`;
    expectedArchives.add(archiveName);
    assertFile(path.join(archivesRoot, archiveName), `缺少发布归档：${archiveName}`);
    assertCondition(checksumEntries.has(archiveName), `SHA256SUMS 缺少归档项：${archiveName}`);
  }

  for (const archiveName of checksumEntries.keys()) {
    assertCondition(expectedArchives.has(archiveName), `SHA256SUMS 存在未预期的归档项：${archiveName}`);
  }
}

function verifyReleaseNotes(notesPath, metadata, version) {
  assertFile(notesPath, `缺少 release notes 文件：${notesPath}`);
  const notes = fs.readFileSync(notesPath, 'utf8');
  const installSnippet = `npm install -g ${metadata.root_package.package_name}@${version}`;
  assertCondition(notes.includes(installSnippet), `release notes 缺少安装片段：${installSnippet}`);
}

function verifySkillPackages(repoRoot, version) {
  const skillsSourceRoot = path.join(repoRoot, 'skills');
  const skillsDistRoot = path.join(repoRoot, 'dist', 'release', 'skills');
  const indexPath = path.join(skillsDistRoot, 'index.json');
  const latestPath = path.join(skillsDistRoot, 'latest.json');

  assertDirectory(skillsSourceRoot, `缺少 skills 源目录：${skillsSourceRoot}`);
  assertDirectory(skillsDistRoot, `缺少 skills 发布目录：${skillsDistRoot}`);

  const indexManifest = readJSON(indexPath);
  const latestManifest = readJSON(latestPath);

  assertCondition(indexManifest.latest_version === version, `skills index latest_version 不一致：期望 ${version}，实际 ${indexManifest.latest_version || '<empty>'}`);
  assertCondition(latestManifest.version === version, `skills latest version 不一致：期望 ${version}，实际 ${latestManifest.version || '<empty>'}`);

  const sourceSkillNames = fs.readdirSync(skillsSourceRoot, { withFileTypes: true })
    .filter((entry) => entry.isDirectory())
    .map((entry) => entry.name)
    .sort();

  verifySkillManifestEntries(indexManifest.skills, sourceSkillNames, 'skills/index.json');
  verifySkillManifestEntries(latestManifest.skills, sourceSkillNames, 'skills/latest.json');

  for (const skillName of sourceSkillNames) {
    const sourceDir = path.join(skillsSourceRoot, skillName);
    const expectedMetadata = readSkillMetadata(sourceDir, skillName);
    const archiveRelativePath = `${version}/${skillName}.tar.gz`;
    const archivePath = path.join(skillsDistRoot, archiveRelativePath);
    assertFile(archivePath, `缺少 skill 归档：${archiveRelativePath}`);

    const expectedChecksum = sha256File(archivePath);
    const expectedSize = fs.statSync(archivePath).size;
    const archivedFiles = listArchiveFiles(archivePath);

    verifySkillManifestEntry(indexManifest.skills, 'skills/index.json', {
      name: skillName,
      description: expectedMetadata.description,
      platforms: expectedMetadata.platforms,
      requires: expectedMetadata.requires,
      referenceFiles: expectedMetadata.referenceFiles,
      version,
      archiveRelativePath,
      expectedChecksum,
      expectedSize,
    });
    verifySkillManifestEntry(latestManifest.skills, 'skills/latest.json', {
      name: skillName,
      description: expectedMetadata.description,
      platforms: expectedMetadata.platforms,
      requires: expectedMetadata.requires,
      referenceFiles: expectedMetadata.referenceFiles,
      version,
      archiveRelativePath,
      expectedChecksum,
      expectedSize,
    });

    const sourceFiles = listRelativeFiles(sourceDir);
    for (const relativeFile of sourceFiles) {
      assertCondition(
        archivedFiles.has(relativeFile),
        `skill 归档缺少源文件：${skillName}/${relativeFile}`,
      );
    }
  }

  return sourceSkillNames.length;
}

function verifySkillManifestEntries(entries, sourceSkillNames, label) {
  assertCondition(Array.isArray(entries), `${label} 缺少 skills 数组。`);
  const entryNames = entries.map((entry) => String(entry?.name || '').trim()).sort();
  assertCondition(
    JSON.stringify(entryNames) === JSON.stringify(sourceSkillNames),
    `${label} skill 列表与源目录不一致。`,
  );
}

function verifySkillManifestEntry(entries, label, expected) {
  const matches = entries.filter((entry) => String(entry?.name || '').trim() === expected.name);
  assertCondition(matches.length === 1, `${label} 中的 skill 条目数量异常：${expected.name}`);
  const entry = matches[0];

  assertCondition(String(entry.description || '').trim() === expected.description, `${label} skill description 不一致：${expected.name}`);
  assertArrayEqual(entry.platforms, expected.platforms, `${label} skill platforms 不一致：${expected.name}`);
  assertArrayEqual(entry.requires, expected.requires, `${label} skill requires 不一致：${expected.name}`);
  assertArrayEqual(entry.reference_files, expected.referenceFiles, `${label} skill reference_files 不一致：${expected.name}`);
  assertCondition(String(entry.version || '').trim() === expected.version, `${label} skill version 不一致：${expected.name}`);
  assertCondition(String(entry.archive_path || '').trim() === expected.archiveRelativePath, `${label} skill archive_path 不一致：${expected.name}`);
  assertCondition(String(entry.checksum_sha256 || '').trim() === expected.expectedChecksum, `${label} skill checksum 不一致：${expected.name}`);
  assertCondition(Number(entry.size_bytes) === expected.expectedSize, `${label} skill size_bytes 不一致：${expected.name}`);

  const normalizedURL = String(entry.url || '').trim();
  assertCondition(
    normalizedURL === expected.archiveRelativePath || normalizedURL.endsWith(`/${expected.archiveRelativePath}`),
    `${label} skill url 不一致：${expected.name}`,
  );
}

function parseChecksumFile(filePath) {
  const content = fs.readFileSync(filePath, 'utf8');
  const items = new Map();

  for (const rawLine of content.split(/\r?\n/)) {
    const line = rawLine.trim();
    if (line === '') {
      continue;
    }

    const match = rawLine.match(/^[a-fA-F0-9]+\s+\*?(.+?)\s*$/);
    assertCondition(match, `无法解析 SHA256SUMS 行：${rawLine}`);
    const fileName = normalizeChecksumFileName(match[1]);
    assertCondition(fileName !== '', `SHA256SUMS 中存在空文件名行：${rawLine}`);
    assertCondition(!items.has(fileName), `SHA256SUMS 中存在重复文件项：${fileName}`);
    items.set(fileName, true);
  }

  return items;
}

function normalizeChecksumFileName(rawName) {
  return String(rawName || '')
    .trim()
    .replace(/^\.\/+/, '');
}

function resolveCoreBinaryName(target) {
  return target.npmOS === 'win32' ? 'clawrise.exe' : 'clawrise';
}

function resolvePluginBinaryName(pluginName, target) {
  const baseName = `clawrise-plugin-${pluginName}`;
  return target.npmOS === 'win32' ? `${baseName}.exe` : baseName;
}

function assertDirectory(directoryPath, message) {
  assertCondition(fs.existsSync(directoryPath) && fs.statSync(directoryPath).isDirectory(), message);
}

function assertFile(filePath, message) {
  assertCondition(fs.existsSync(filePath) && fs.statSync(filePath).isFile(), message);
}

function readJSON(filePath) {
  assertFile(filePath, `缺少 JSON 文件：${filePath}`);
  return JSON.parse(fs.readFileSync(filePath, 'utf8'));
}

function readSkillMetadata(sourceDir, skillName) {
  const skillDocumentPath = path.join(sourceDir, 'SKILL.md');
  const document = fs.readFileSync(skillDocumentPath, 'utf8');
  const frontmatter = parseFrontmatter(document, skillDocumentPath);
  const declaredName = String(frontmatter.name || '').trim();
  const description = String(frontmatter.description || '').trim();

  assertCondition(declaredName === skillName, `skill frontmatter name 与目录名不一致：${skillDocumentPath}`);
  assertCondition(description !== '', `skill frontmatter 缺少 description：${skillDocumentPath}`);

  return {
    description,
    platforms: inferSkillPlatforms(skillName),
    requires: inferSkillRequirements(skillName),
    referenceFiles: listRelativeFiles(path.join(sourceDir, 'references'), sourceDir),
  };
}

function parseFrontmatter(document, filePath) {
  const match = String(document || '').match(/^---\r?\n([\s\S]*?)\r?\n---(?:\r?\n|$)/);
  assertCondition(Boolean(match), `skill frontmatter 格式无效：${filePath}`);

  const result = {};
  for (const rawLine of match[1].split(/\r?\n/)) {
    const line = rawLine.trim();
    if (line === '' || line.startsWith('#')) {
      continue;
    }

    const separatorIndex = line.indexOf(':');
    if (separatorIndex <= 0) {
      continue;
    }

    const key = line.slice(0, separatorIndex).trim();
    const value = line.slice(separatorIndex + 1).trim();
    result[key] = stripWrappedQuotes(value);
  }
  return result;
}

function stripWrappedQuotes(value) {
  const trimmed = String(value || '').trim();
  if (
    (trimmed.startsWith('"') && trimmed.endsWith('"')) ||
    (trimmed.startsWith('\'') && trimmed.endsWith('\''))
  ) {
    return trimmed.slice(1, -1);
  }
  return trimmed;
}

function inferSkillPlatforms(skillName) {
  if (skillName === 'clawrise-core') {
    return [];
  }
  if (skillName.startsWith('clawrise-')) {
    return [skillName.slice('clawrise-'.length)];
  }
  return [];
}

function inferSkillRequirements(skillName) {
  if (skillName === 'clawrise-core') {
    return [];
  }
  return ['clawrise-core'];
}

function listRelativeFiles(directoryPath, rootDir = directoryPath) {
  if (!fs.existsSync(directoryPath) || !fs.statSync(directoryPath).isDirectory()) {
    return [];
  }

  return walkFiles(directoryPath)
    .map((filePath) => path.relative(rootDir, filePath).split(path.sep).join('/'))
    .sort();
}

function walkFiles(directoryPath) {
  const files = [];
  for (const entry of fs.readdirSync(directoryPath, { withFileTypes: true })) {
    const entryPath = path.join(directoryPath, entry.name);
    if (entry.isDirectory()) {
      files.push(...walkFiles(entryPath));
      continue;
    }
    if (entry.isFile()) {
      files.push(entryPath);
    }
  }
  return files;
}

function listArchiveFiles(archivePath) {
  const result = spawnSync('tar', ['-tzf', archivePath], {
    stdio: 'pipe',
  });
  if (result.status !== 0) {
    const stderr = (result.stderr || Buffer.alloc(0)).toString('utf8').trim();
    throw new Error(`无法读取 skill 归档：${archivePath}${stderr ? ` (${stderr})` : ''}`);
  }

  const files = new Set();
  for (const rawLine of String(result.stdout || '').split(/\r?\n/)) {
    const normalized = normalizeArchivePath(rawLine);
    if (normalized === '' || rawLine.trim().endsWith('/')) {
      continue;
    }
    files.add(normalized);
  }
  return files;
}

function normalizeArchivePath(rawPath) {
  return String(rawPath || '')
    .trim()
    .replace(/^\.\/+/, '')
    .replace(/\/+$/, '');
}

function sha256File(filePath) {
  const hash = crypto.createHash('sha256');
  hash.update(fs.readFileSync(filePath));
  return hash.digest('hex');
}

function assertArrayEqual(actual, expected, message) {
  const normalizedActual = Array.isArray(actual) ? [...actual] : [];
  const normalizedExpected = Array.isArray(expected) ? [...expected] : [];
  assertCondition(
    JSON.stringify(normalizedActual) === JSON.stringify(normalizedExpected),
    message,
  );
}

function assertCondition(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}

function main() {
  try {
    const summary = verifyReleaseArtifacts({
      version: process.argv[2] || process.env.CLAWRISE_RELEASE_VERSION || '',
      distTag: process.env.CLAWRISE_NPM_DIST_TAG || '',
    });
    console.log(
      `发布产物校验通过：version=${summary.version} root=${summary.rootPackage} platforms=${summary.platformCount} archives=${summary.archiveCount} skills=${summary.skillCount}`,
    );
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    console.error(`发布产物校验失败：${message}`);
    process.exit(1);
  }
}

if (path.resolve(process.argv[1] || '') === __filename) {
  main();
}
