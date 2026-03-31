'use strict';

const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { test } = require('node:test');

const sourceModulePath = path.join(__dirname, 'platform.js');
const currentPlatformSuffix = resolveCurrentPlatformSuffix();
const currentBinaryName = process.platform === 'win32' ? 'clawrise.exe' : 'clawrise';

test('resolves the platform package from a scoped root package.json', { concurrency: false }, () => {
  const fixture = createFixture({
    rootPackageName: '@clawrise/clawrise-cli',
  });
  const envKey = 'CLAWRISE_ROOT_PACKAGE_NAME';
  const previousValue = process.env[envKey];

  try {
    delete process.env[envKey];

    const { resolvePlatformPackage } = require(fixture.modulePath);
    const resolved = resolvePlatformPackage();

    assert.equal(resolved.packageName, fixture.platformPackageName);
    assert.equal(normalizeExistingPath(resolved.dir), normalizeExistingPath(fixture.platformPackageDir));
    assert.equal(
      normalizeExistingPath(resolved.binaryPath),
      normalizeExistingPath(path.join(fixture.platformPackageDir, 'bin', currentBinaryName)),
    );
  } finally {
    restoreEnv(envKey, previousValue);
    cleanupFixture(fixture.rootDir);
  }
});

test('falls back to the environment override when the root package.json is missing', { concurrency: false }, () => {
  const rootPackageName = '@demo/clawrise-cli';
  const fixture = createFixture({
    rootPackageName,
    skipRootPackageJSON: true,
  });
  const envKey = 'CLAWRISE_ROOT_PACKAGE_NAME';
  const previousValue = process.env[envKey];

  try {
    process.env[envKey] = rootPackageName;

    const { resolvePlatformPackage } = require(fixture.modulePath);
    const resolved = resolvePlatformPackage();

    assert.equal(resolved.packageName, fixture.platformPackageName);
    assert.equal(
      normalizeExistingPath(resolved.binaryPath),
      normalizeExistingPath(path.join(fixture.platformPackageDir, 'bin', currentBinaryName)),
    );
  } finally {
    restoreEnv(envKey, previousValue);
    cleanupFixture(fixture.rootDir);
  }
});

test('includes the scoped platform package name in the error message', { concurrency: false }, () => {
  const fixture = createFixture({
    rootPackageName: '@clawrise/clawrise-cli',
    skipPlatformPackage: true,
  });
  const envKey = 'CLAWRISE_ROOT_PACKAGE_NAME';
  const previousValue = process.env[envKey];

  try {
    delete process.env[envKey];

    const { resolvePlatformPackage } = require(fixture.modulePath);
    assert.throws(() => resolvePlatformPackage(), {
      message: new RegExp(escapeRegExp(fixture.platformPackageName)),
    });
  } finally {
    restoreEnv(envKey, previousValue);
    cleanupFixture(fixture.rootDir);
  }
});

function createFixture(options) {
  if (currentPlatformSuffix === '') {
    throw new Error(`The current platform is not in the supported test matrix: ${process.platform}/${process.arch}`);
  }

  const rootDir = fs.mkdtempSync(path.join(os.tmpdir(), 'clawrise-platform-test-'));
  const libDir = path.join(rootDir, 'lib');
  const modulePath = path.join(libDir, 'platform.js');
  const rootPackageName = String(options.rootPackageName || '').trim();
  const platformPackageName = buildPlatformPackageName(rootPackageName, currentPlatformSuffix);
  const platformPackageDir = path.join(rootDir, 'node_modules', ...platformPackageName.split('/'));

  fs.mkdirSync(libDir, { recursive: true });
  fs.copyFileSync(sourceModulePath, modulePath);

  if (!options.skipRootPackageJSON) {
    writeJSON(path.join(rootDir, 'package.json'), {
      name: rootPackageName,
      version: '0.0.0-test',
    });
  }

  if (!options.skipPlatformPackage) {
    fs.mkdirSync(path.join(platformPackageDir, 'bin'), { recursive: true });
    writeJSON(path.join(platformPackageDir, 'package.json'), {
      name: platformPackageName,
      version: '0.0.0-test',
    });
    fs.writeFileSync(path.join(platformPackageDir, 'bin', currentBinaryName), '');
  }

  return {
    rootDir,
    modulePath,
    platformPackageName,
    platformPackageDir,
  };
}

function cleanupFixture(rootDir) {
  fs.rmSync(rootDir, { recursive: true, force: true });
}

function restoreEnv(key, previousValue) {
  if (typeof previousValue === 'string') {
    process.env[key] = previousValue;
    return;
  }
  delete process.env[key];
}

function buildPlatformPackageName(rootPackageName, suffix) {
  if (rootPackageName.startsWith('@')) {
    const separatorIndex = rootPackageName.indexOf('/');
    const scope = rootPackageName.slice(0, separatorIndex + 1);
    const baseName = rootPackageName.slice(separatorIndex + 1);
    return `${scope}${baseName}-${suffix}`;
  }
  return `${rootPackageName}-${suffix}`;
}

function resolveCurrentPlatformSuffix() {
  const platformKey = `${process.platform}:${process.arch}`;
  const mapping = {
    'darwin:arm64': 'darwin-arm64',
    'darwin:x64': 'darwin-x64',
    'linux:arm64': 'linux-arm64',
    'linux:x64': 'linux-x64',
    'win32:arm64': 'win32-arm64',
    'win32:x64': 'win32-x64',
  };
  return mapping[platformKey] || '';
}

function writeJSON(filePath, value) {
  fs.writeFileSync(filePath, `${JSON.stringify(value, null, 2)}\n`);
}

function normalizeExistingPath(filePath) {
  return fs.realpathSync(filePath);
}

function escapeRegExp(value) {
  return String(value).replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}
