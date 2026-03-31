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
const skillsRoot = path.join(repoRoot, 'skills');
const npmScope = normalizeScope(process.env.CLAWRISE_NPM_SCOPE || '');
const packagePrefix = normalizePackagePrefix(process.env.CLAWRISE_NPM_PACKAGE_PREFIX || 'clawrise-cli');
const distTag = resolveDistTag(version, process.env.CLAWRISE_NPM_DIST_TAG || '');
const aiInstallGuideURL = String(process.env.CLAWRISE_AI_INSTALL_GUIDE_URL || 'https://raw.githubusercontent.com/repothread/clawrise-cli/main/docs/en/ai-install.md').trim();

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

console.log(`Generated npm release directory: ${npmRoot}`);

function prepareRootPackage() {
  const targetDir = path.join(npmRoot, rootPackageBaseName);
  resetDir(targetDir);

  copyFile(path.join(templateRoot, 'bin', 'clawrise.js'), path.join(targetDir, 'bin', 'clawrise.js'));
  copyFile(path.join(templateRoot, 'lib', 'platform.js'), path.join(targetDir, 'lib', 'platform.js'));
  copyFile(path.join(templateRoot, 'lib', 'setup.js'), path.join(targetDir, 'lib', 'setup.js'));
  copyDir(skillsRoot, path.join(targetDir, 'skills'));
  writeFile(path.join(targetDir, 'README.md'), buildRootReadme());

  const optionalDependencies = {};
  for (const platform of platformPackages) {
    optionalDependencies[platform.packageName] = version;
  }

  const rootPackageJSON = {
    name: rootPackageName,
    version,
    description: 'Clawrise CLI with bundled first-party provider plugins and setup flows for AI client skills.',
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
      'skills',
      'README.md',
    ],
    keywords: [
      'clawrise',
      'cli',
      'feishu',
      'notion',
      'codex',
      'skills',
      'setup',
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
    '## Install Globally',
    '',
    '```bash',
    `npm install -g ${rootPackageName}`,
    '```',
    '',
    '## For AI',
    '',
    'Send the following prompt to the AI assistant:',
    '',
    '```text',
    `Access ${aiInstallGuideURL} and follow the steps there to install the \`clawrise\` command and run setup for the current client.`,
    '```',
    '',
  ].join('\n');
}

function preparePlatformPackage(platform) {
  const bundleDir = path.join(bundlesRoot, `${platform.npmOS}-${platform.npmCPU}`);
  if (!fs.existsSync(bundleDir)) {
    throw new Error(`Missing platform bundle: ${bundleDir}`);
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
    `This package only contains the ${displayPlatform} binary bundle. The root package also includes the first-party provider plugins and the bundled Clawrise skills.`,
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
    throw new Error('The npm package prefix must not be empty.');
  }
  if (trimmed.includes('/')) {
    throw new Error(`The npm package prefix must not contain '/': ${trimmed}`);
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
