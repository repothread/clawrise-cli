'use strict';

const fs = require('fs');
const path = require('path');

const platformSuffixMap = {
  'darwin:arm64': 'darwin-arm64',
  'darwin:x64': 'darwin-x64',
  'linux:arm64': 'linux-arm64',
  'linux:x64': 'linux-x64',
  'win32:arm64': 'win32-arm64',
  'win32:x64': 'win32-x64',
};

function resolvePlatformPackage() {
  const platformKey = `${process.platform}:${process.arch}`;
  const packageSuffix = platformSuffixMap[platformKey];

  if (!packageSuffix) {
    throw new Error(`The current platform is not supported: ${process.platform}/${process.arch}`);
  }

  // Keep the platform package under the same scope / prefix as the root package,
  // otherwise scoped releases cannot resolve the installed platform bundle.
  const packageName = appendPackageSuffix(resolveRootPackageName(), packageSuffix);

  let packageJSONPath;
  try {
    packageJSONPath = require.resolve(`${packageName}/package.json`);
  } catch (error) {
    throw new Error(`The platform binary package ${packageName} was not found. Re-run npm install for the root package.`);
  }

  const packageDir = path.dirname(packageJSONPath);
  const binaryName = process.platform === 'win32' ? 'clawrise.exe' : 'clawrise';

  return {
    packageName,
    dir: packageDir,
    binaryPath: path.join(packageDir, 'bin', binaryName),
  };
}

function resolveRootPackageName() {
  const envName = String(process.env.CLAWRISE_ROOT_PACKAGE_NAME || '').trim();
  if (envName !== '') {
    return envName;
  }

  const packageJSONPath = path.join(resolvePackageRootDir(), 'package.json');
  if (!fs.existsSync(packageJSONPath)) {
    throw new Error(`The root package metadata was not found: ${packageJSONPath}`);
  }

  const packageJSON = JSON.parse(fs.readFileSync(packageJSONPath, 'utf8'));
  const packageName = String(packageJSON.name || '').trim();
  if (packageName === '') {
    throw new Error(`The root package name is missing in ${packageJSONPath}`);
  }
  return packageName;
}

function resolvePackageRootDir() {
  return path.resolve(__dirname, '..');
}

function appendPackageSuffix(packageName, suffix) {
  if (packageName.startsWith('@')) {
    const separatorIndex = packageName.indexOf('/');
    if (separatorIndex > 0 && separatorIndex < packageName.length - 1) {
      const scope = packageName.slice(0, separatorIndex + 1);
      const baseName = packageName.slice(separatorIndex + 1);
      return `${scope}${baseName}-${suffix}`;
    }
  }
  return `${packageName}-${suffix}`;
}

module.exports = {
  resolvePlatformPackage,
};
