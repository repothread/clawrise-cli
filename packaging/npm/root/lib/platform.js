'use strict';

const path = require('path');

const packageMap = {
  'darwin:arm64': 'clawrise-cli-darwin-arm64',
  'darwin:x64': 'clawrise-cli-darwin-x64',
  'linux:arm64': 'clawrise-cli-linux-arm64',
  'linux:x64': 'clawrise-cli-linux-x64',
  'win32:arm64': 'clawrise-cli-win32-arm64',
  'win32:x64': 'clawrise-cli-win32-x64',
};

function resolvePlatformPackage() {
  const platformKey = `${process.platform}:${process.arch}`;
  const packageName = packageMap[platformKey];

  if (!packageName) {
    throw new Error(`The current platform is not supported: ${process.platform}/${process.arch}`);
  }

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

module.exports = {
  resolvePlatformPackage,
};
