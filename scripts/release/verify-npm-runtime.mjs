#!/usr/bin/env node

import fs from 'node:fs';
import path from 'node:path';
import { spawnSync } from 'node:child_process';
import { createRequire } from 'node:module';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const repoRoot = path.resolve(__dirname, '..', '..');
const npmRoot = path.join(repoRoot, 'dist', 'release', 'npm');
const metadataPath = path.join(npmRoot, 'release-metadata.json');
const verifyRoot = path.resolve(process.env.CLAWRISE_PACK_VERIFY_ROOT || path.join(repoRoot, 'dist', 'release', 'pack-verify'));
const runtimeVerifyRoot = path.join(verifyRoot, 'runtime');
const npmCacheDir = path.resolve(process.env.CLAWRISE_NPM_CACHE_DIR || path.join(repoRoot, '.cache', 'npm'));
const metadata = readJSON(metadataPath);
const currentPlatform = resolveCurrentPlatformPackage(metadata);

resetDir(runtimeVerifyRoot);
fs.mkdirSync(npmCacheDir, { recursive: true });

// Validate against packed npm artifacts so a directory-level check cannot miss
// problems that only appear after installation.
const rootPackPath = npmPack(path.join(npmRoot, metadata.root_package.dir_name), runtimeVerifyRoot);
const platformPackPath = npmPack(path.join(npmRoot, currentPlatform.dir_name), runtimeVerifyRoot);

const appRoot = path.join(runtimeVerifyRoot, 'app');
const nodeModulesRoot = path.join(appRoot, 'node_modules');

installPackedPackage(rootPackPath, metadata.root_package.package_name, nodeModulesRoot);
installPackedPackage(platformPackPath, currentPlatform.package_name, nodeModulesRoot);

// Call the root package resolver directly to verify the installed require path,
// scope handling, and platform package naming all line up.
const appRequire = createRequire(path.join(appRoot, 'index.js'));
const { resolvePlatformPackage } = appRequire(`${metadata.root_package.package_name}/lib/platform.js`);
const resolved = resolvePlatformPackage();

if (resolved.packageName !== currentPlatform.package_name) {
  throw new Error(`Resolved platform package name mismatch: expected ${currentPlatform.package_name}, got ${resolved.packageName}`);
}

if (!fs.existsSync(resolved.binaryPath)) {
  throw new Error(`Resolved platform binary was not found: ${resolved.binaryPath}`);
}

const pluginDir = path.join(resolved.dir, 'plugins');
if (!fs.existsSync(pluginDir) || !fs.statSync(pluginDir).isDirectory()) {
  throw new Error(`Resolved plugin directory was not found: ${pluginDir}`);
}

console.log(`npm runtime smoke test passed: ${metadata.root_package.package_name} -> ${resolved.packageName}`);

function resolveCurrentPlatformPackage(releaseMetadata) {
  const currentKey = `${process.platform}:${process.arch}`;
  const platform = releaseMetadata.platform_packages.find((item) => `${item.os}:${item.cpu}` === currentKey);
  if (!platform) {
    throw new Error(`No npm platform package is configured for the current platform: ${currentKey}`);
  }
  return platform;
}

function npmPack(packageDir, destinationDir) {
  const args = ['--cache', npmCacheDir, 'pack', '--json', packageDir, '--pack-destination', destinationDir];

  const result = spawnSync('npm', args, {
    cwd: repoRoot,
    stdio: 'pipe',
    encoding: 'utf8',
  });
  if (result.status !== 0) {
    throw new Error(`npm pack failed: ${result.stderr.trim() || result.stdout.trim()}`);
  }

  let payload;
  try {
    payload = JSON.parse(result.stdout);
  } catch (error) {
    throw new Error(`Failed to parse npm pack output: ${result.stdout.trim()}`);
  }

  if (!Array.isArray(payload) || payload.length === 0 || typeof payload[0].filename !== 'string') {
    throw new Error(`npm pack output did not include a filename: ${result.stdout.trim()}`);
  }

  return path.join(destinationDir, payload[0].filename);
}

function installPackedPackage(archivePath, packageName, nodeModulesRoot) {
  const extractRoot = path.join(nodeModulesRoot, '.staging', sanitizePackageName(packageName));
  resetDir(extractRoot);

  const result = spawnSync('tar', ['-xzf', archivePath, '-C', extractRoot], {
    cwd: repoRoot,
    stdio: 'pipe',
    encoding: 'utf8',
  });
  if (result.status !== 0) {
    throw new Error(`Failed to extract npm package ${archivePath}: ${result.stderr.trim() || result.stdout.trim()}`);
  }

  const extractedPackageDir = path.join(extractRoot, 'package');
  if (!fs.existsSync(extractedPackageDir) || !fs.statSync(extractedPackageDir).isDirectory()) {
    throw new Error(`The extracted npm package directory is missing: ${archivePath}`);
  }

  const installDir = packageInstallPath(nodeModulesRoot, packageName);
  fs.mkdirSync(path.dirname(installDir), { recursive: true });
  fs.rmSync(installDir, { recursive: true, force: true });
  fs.cpSync(extractedPackageDir, installDir, { recursive: true });
}

function packageInstallPath(nodeModulesRoot, packageName) {
  const segments = packageName.split('/');
  return path.join(nodeModulesRoot, ...segments);
}

function sanitizePackageName(packageName) {
  return packageName.replaceAll('/', '__');
}

function readJSON(filePath) {
  if (!fs.existsSync(filePath)) {
    throw new Error(`Release metadata file was not found: ${filePath}`);
  }
  return JSON.parse(fs.readFileSync(filePath, 'utf8'));
}

function resetDir(targetDir) {
  fs.rmSync(targetDir, { recursive: true, force: true });
  fs.mkdirSync(targetDir, { recursive: true });
}
