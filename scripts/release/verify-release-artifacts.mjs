#!/usr/bin/env node

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

  return {
    version: expectedVersion,
    distTag: expectedDistTag,
    rootPackage: metadata.root_package.package_name,
    platformCount: metadata.platform_packages.length,
    archiveCount: releaseTargets.length,
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
      `发布产物校验通过：version=${summary.version} root=${summary.rootPackage} platforms=${summary.platformCount} archives=${summary.archiveCount}`,
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
