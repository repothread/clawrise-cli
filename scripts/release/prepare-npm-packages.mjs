#!/usr/bin/env node

import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const repoRoot = path.resolve(__dirname, '..', '..');
const version = resolveVersion(process.argv[2], process.env.CLAWRISE_RELEASE_VERSION, process.env.GITHUB_REF_NAME);
const bundlesRoot = path.join(repoRoot, 'dist', 'release', 'bundles');
const npmRoot = path.join(repoRoot, 'dist', 'release', 'npm');
const templateRoot = path.join(repoRoot, 'packaging', 'npm', 'root');
const npmScope = normalizeScope(process.env.CLAWRISE_NPM_SCOPE || '');
const packagePrefix = normalizePackagePrefix(process.env.CLAWRISE_NPM_PACKAGE_PREFIX || 'clawrise-cli');
const distTag = resolveDistTag(version, process.env.CLAWRISE_NPM_DIST_TAG || '');

const platforms = [
  { npmOS: 'darwin', npmCPU: 'arm64' },
  { npmOS: 'darwin', npmCPU: 'x64' },
  { npmOS: 'linux', npmCPU: 'arm64' },
  { npmOS: 'linux', npmCPU: 'x64' },
  { npmOS: 'win32', npmCPU: 'arm64' },
  { npmOS: 'win32', npmCPU: 'x64' },
];

const rootPackageBaseName = packagePrefix;
const rootPackageName = toPackageName(rootPackageBaseName);
const platformPackages = platforms.map((platform) => ({
  ...platform,
  packageBaseName: `${packagePrefix}-${platform.npmOS}-${platform.npmCPU}`,
  packageName: toPackageName(`${packagePrefix}-${platform.npmOS}-${platform.npmCPU}`),
}));

prepareRootPackage();
for (const platform of platformPackages) {
  preparePlatformPackage(platform);
}
writeReleaseMetadata();

console.log(`已生成 npm 发布目录: ${npmRoot}`);

function prepareRootPackage() {
  const targetDir = path.join(npmRoot, rootPackageBaseName);
  resetDir(targetDir);

  copyFile(path.join(templateRoot, 'bin', 'clawrise.js'), path.join(targetDir, 'bin', 'clawrise.js'));
  copyFile(path.join(templateRoot, 'lib', 'platform.js'), path.join(targetDir, 'lib', 'platform.js'));
  writeFile(path.join(targetDir, 'README.md'), buildRootReadme());

  const optionalDependencies = {};
  for (const platform of platformPackages) {
    optionalDependencies[platform.packageName] = version;
  }

  const rootPackageJSON = {
    name: rootPackageName,
    version,
    description: 'Clawrise CLI with bundled first-party provider plugins.',
    license: 'MIT',
    repository: {
      type: 'git',
      url: 'git+https://github.com/repothread/clawrise-cli.git',
    },
    bugs: {
      url: 'https://github.com/repothread/clawrise-cli/issues',
    },
    homepage: 'https://github.com/repothread/clawrise-cli#readme',
    bin: {
      clawrise: 'bin/clawrise.js',
    },
    files: [
      'bin',
      'lib',
      'README.md',
    ],
    optionalDependencies,
  };

  writeJSON(path.join(targetDir, 'package.json'), rootPackageJSON);
}

function buildRootReadme() {
  return [
    `# ${rootPackageName}`,
    '',
    'Clawrise CLI root package distributed through npm.',
    '',
    'Clawrise CLI 的 npm 根包。',
    '',
    '```bash',
    `npm install -g ${rootPackageName}`,
    '```',
    '',
    'It automatically resolves the prebuilt binary for the current platform and bundles the first-party `feishu` and `notion` provider plugins.',
    '',
    '## 中文说明',
    '',
    '这是 Clawrise CLI 的 npm 根包。',
    '',
    '安装命令：',
    '',
    '```bash',
    `npm install -g ${rootPackageName}`,
    '```',
    '',
    '安装后会自动选择当前平台对应的预编译二进制，并携带第一方 `feishu` / `notion` provider plugin。',
    '',
  ].join('\n');
}

function preparePlatformPackage(platform) {
  const bundleDir = path.join(bundlesRoot, `${platform.npmOS}-${platform.npmCPU}`);
  if (!fs.existsSync(bundleDir)) {
    throw new Error(`缺少平台 bundle: ${bundleDir}`);
  }

  const targetDir = path.join(npmRoot, platform.packageBaseName);
  resetDir(targetDir);
  copyDir(bundleDir, targetDir);
  writeFile(path.join(targetDir, 'README.md'), buildPlatformReadme(platform));

  const packageJSON = {
    name: platform.packageName,
    version,
    description: `Clawrise CLI prebuilt binary for ${platform.npmOS}-${platform.npmCPU}.`,
    license: 'MIT',
    repository: {
      type: 'git',
      url: 'git+https://github.com/repothread/clawrise-cli.git',
    },
    homepage: 'https://github.com/repothread/clawrise-cli#readme',
    os: [platform.npmOS],
    cpu: [platform.npmCPU],
    files: [
      'bin',
      'plugins',
      'README.md',
    ],
  };

  writeJSON(path.join(targetDir, 'package.json'), packageJSON);
}

function buildPlatformReadme(platform) {
  const displayPlatform = buildPlatformDisplayName(platform);

  return [
    `# ${platform.packageName}`,
    '',
    `Prebuilt Clawrise CLI binary package for ${displayPlatform}.`,
    '',
    'You usually do not need to install this package directly. Install the root package instead:',
    '',
    '```bash',
    `npm install -g ${rootPackageName}`,
    '```',
    '',
    '## 中文说明',
    '',
    '这是 Clawrise CLI 的平台二进制分发包。',
    `目标平台：${displayPlatform}`,
    '',
    '通常不需要直接安装这个包，请安装根包：',
    '',
    '```bash',
    `npm install -g ${rootPackageName}`,
    '```',
    '',
  ].join('\n');
}

function buildPlatformDisplayName(platform) {
  const osNameMap = {
    darwin: 'macOS',
    linux: 'Linux',
    win32: 'Windows',
  };

  const osName = osNameMap[platform.npmOS] || platform.npmOS;
  return `${osName} ${platform.npmCPU}`;
}

function writeReleaseMetadata() {
  const metadata = {
    version,
    npm_scope: npmScope,
    package_prefix: packagePrefix,
    dist_tag: distTag,
    root_package: {
      dir_name: rootPackageBaseName,
      package_name: rootPackageName,
    },
    platform_packages: platformPackages.map((platform) => ({
      dir_name: platform.packageBaseName,
      package_name: platform.packageName,
      os: platform.npmOS,
      cpu: platform.npmCPU,
    })),
  };

  writeJSON(path.join(npmRoot, 'release-metadata.json'), metadata);
}

function resolveVersion(cliArg, envVersion, refName) {
  const raw = firstNonEmpty(cliArg, envVersion, refName);
  if (!raw) {
    throw new Error('未提供发布版本，请传入参数、设置 CLAWRISE_RELEASE_VERSION，或提供 GITHUB_REF_NAME。');
  }

  const normalized = raw
    .replace(/^refs\/heads\//, '')
    .replace(/^refs\/tags\//, '')
    .replace(/^release\//, '')
    .replace(/^release-/, '')
    .replace(/^v/, '');

  if (!/^[0-9]+\.[0-9]+\.[0-9]+([-.][0-9A-Za-z.-]+)?$/.test(normalized)) {
    throw new Error(`无效的发布版本: ${normalized}`);
  }
  return normalized;
}

function resolveDistTag(version, explicitTag) {
  if (typeof explicitTag === 'string' && explicitTag.trim() !== '') {
    return explicitTag.trim();
  }
  return version.includes('-') ? 'next' : 'latest';
}

function normalizeScope(scope) {
  if (typeof scope !== 'string' || scope.trim() === '') {
    return '';
  }

  const trimmed = scope.trim().replace(/\/+$/, '');
  if (!trimmed.startsWith('@')) {
    return `@${trimmed}`;
  }
  return trimmed;
}

function normalizePackagePrefix(prefix) {
  const trimmed = String(prefix || '').trim();
  if (trimmed === '') {
    throw new Error('npm 包名前缀不能为空。');
  }
  if (trimmed.includes('/')) {
    throw new Error(`npm 包名前缀不能包含 /: ${trimmed}`);
  }
  return trimmed;
}

function toPackageName(baseName) {
  return npmScope ? `${npmScope}/${baseName}` : baseName;
}

function firstNonEmpty(...values) {
  for (const value of values) {
    if (typeof value === 'string' && value.trim() !== '') {
      return value.trim();
    }
  }
  return '';
}

function resetDir(targetDir) {
  fs.rmSync(targetDir, { recursive: true, force: true });
  fs.mkdirSync(targetDir, { recursive: true });
}

function copyDir(sourceDir, targetDir) {
  fs.mkdirSync(targetDir, { recursive: true });
  for (const entry of fs.readdirSync(sourceDir, { withFileTypes: true })) {
    const sourcePath = path.join(sourceDir, entry.name);
    const targetPath = path.join(targetDir, entry.name);
    if (entry.isDirectory()) {
      copyDir(sourcePath, targetPath);
      continue;
    }
    copyFile(sourcePath, targetPath);
  }
}

function copyFile(sourcePath, targetPath) {
  fs.mkdirSync(path.dirname(targetPath), { recursive: true });
  fs.copyFileSync(sourcePath, targetPath);
  const mode = fs.statSync(sourcePath).mode;
  fs.chmodSync(targetPath, mode);
}

function writeJSON(targetPath, value) {
  writeFile(targetPath, JSON.stringify(value, null, 2) + '\n');
}

function writeFile(targetPath, content) {
  fs.mkdirSync(path.dirname(targetPath), { recursive: true });
  fs.writeFileSync(targetPath, content);
}
