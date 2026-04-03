import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { test } from 'node:test';

import { releaseTargets, verifyReleaseArtifacts } from './verify-release-artifacts.mjs';

test('通过完整发布目录校验', () => {
  const fixture = createReleaseFixture('1.2.3');
  try {
    const result = verifyReleaseArtifacts({
      repoRoot: fixture.repoRoot,
      version: '1.2.3',
    });

    assert.equal(result.version, '1.2.3');
    assert.equal(result.distTag, 'latest');
    assert.equal(result.platformCount, releaseTargets.length);
  } finally {
    cleanupFixture(fixture.repoRoot);
  }
});

test('缺少归档校验项时失败', () => {
  const fixture = createReleaseFixture('1.2.3');
  try {
    const checksumPath = path.join(fixture.repoRoot, 'dist', 'release', 'archives', 'SHA256SUMS');
    const lines = fs.readFileSync(checksumPath, 'utf8').trim().split('\n');
    fs.writeFileSync(checksumPath, `${lines.slice(1).join('\n')}\n`);

    assert.throws(
      () => verifyReleaseArtifacts({ repoRoot: fixture.repoRoot, version: '1.2.3' }),
      /SHA256SUMS 缺少归档项/,
    );
  } finally {
    cleanupFixture(fixture.repoRoot);
  }
});

function createReleaseFixture(version) {
  const repoRoot = fs.mkdtempSync(path.join(os.tmpdir(), 'clawrise-release-verify-'));
  const npmRoot = path.join(repoRoot, 'dist', 'release', 'npm');
  const bundlesRoot = path.join(repoRoot, 'dist', 'release', 'bundles');
  const archivesRoot = path.join(repoRoot, 'dist', 'release', 'archives');
  const rootPackage = {
    dir_name: 'clawrise-cli',
    package_name: '@clawrise/clawrise-cli',
  };
  const platformPackages = releaseTargets.map((target) => ({
    dir_name: `clawrise-cli-${target.npmOS}-${target.npmCPU}`,
    package_name: `@clawrise/clawrise-cli-${target.npmOS}-${target.npmCPU}`,
    os: target.npmOS,
    cpu: target.npmCPU,
  }));

  fs.mkdirSync(npmRoot, { recursive: true });
  fs.mkdirSync(bundlesRoot, { recursive: true });
  fs.mkdirSync(archivesRoot, { recursive: true });

  writeJSON(path.join(npmRoot, 'release-metadata.json'), {
    version,
    npm_scope: '@clawrise',
    package_prefix: 'clawrise-cli',
    dist_tag: 'latest',
    root_package: rootPackage,
    platform_packages: platformPackages,
  });

  createRootPackageFixture(npmRoot, rootPackage, platformPackages, version);
  for (const item of platformPackages) {
    createPlatformPackageFixture(npmRoot, bundlesRoot, item, version);
  }
  createArchiveFixture(archivesRoot, version);
  fs.writeFileSync(
    path.join(repoRoot, 'dist', 'release', 'release-notes.md'),
    `### Install\n\n\`\`\`bash\nnpm install -g ${rootPackage.package_name}@${version}\n\`\`\`\n`,
  );

  return { repoRoot };
}

function createRootPackageFixture(npmRoot, rootPackage, platformPackages, version) {
  const rootDir = path.join(npmRoot, rootPackage.dir_name);
  fs.mkdirSync(path.join(rootDir, 'bin'), { recursive: true });
  fs.mkdirSync(path.join(rootDir, 'lib'), { recursive: true });
  fs.mkdirSync(path.join(rootDir, 'skills'), { recursive: true });

  const optionalDependencies = {};
  for (const item of platformPackages) {
    optionalDependencies[item.package_name] = version;
  }

  writeJSON(path.join(rootDir, 'package.json'), {
    name: rootPackage.package_name,
    version,
    optionalDependencies,
  });
  fs.writeFileSync(path.join(rootDir, 'README.md'), `npm install -g ${rootPackage.package_name}\n`);
  fs.writeFileSync(path.join(rootDir, 'bin', 'clawrise.js'), '');
  fs.writeFileSync(path.join(rootDir, 'lib', 'platform.js'), '');
  fs.writeFileSync(path.join(rootDir, 'lib', 'setup.js'), '');
}

function createPlatformPackageFixture(npmRoot, bundlesRoot, item, version) {
  const packageDir = path.join(npmRoot, item.dir_name);
  const bundleDir = path.join(bundlesRoot, `${item.os}-${item.cpu}`);
  fs.mkdirSync(path.join(packageDir, 'bin'), { recursive: true });
  fs.mkdirSync(path.join(bundleDir, 'bin'), { recursive: true });

  const coreBinaryName = item.os === 'win32' ? 'clawrise.exe' : 'clawrise';
  fs.writeFileSync(path.join(packageDir, 'bin', coreBinaryName), '');
  fs.writeFileSync(path.join(bundleDir, 'bin', coreBinaryName), '');

  writeJSON(path.join(packageDir, 'package.json'), {
    name: item.package_name,
    version,
    os: [item.os],
    cpu: [item.cpu],
  });
  fs.writeFileSync(path.join(packageDir, 'README.md'), `package ${item.package_name}\n`);

  for (const pluginName of ['feishu', 'notion']) {
    const pluginBinary = item.os === 'win32' ? `clawrise-plugin-${pluginName}.exe` : `clawrise-plugin-${pluginName}`;
    const packagePluginDir = path.join(packageDir, 'plugins', pluginName, version, 'bin');
    const bundlePluginDir = path.join(bundleDir, 'plugins', pluginName, version, 'bin');
    fs.mkdirSync(packagePluginDir, { recursive: true });
    fs.mkdirSync(bundlePluginDir, { recursive: true });
    fs.writeFileSync(path.join(packagePluginDir, pluginBinary), '');
    fs.writeFileSync(path.join(bundlePluginDir, pluginBinary), '');

    const manifest = {
      schema_version: 1,
      name: pluginName,
      version,
      kind: 'provider',
      protocol_version: 1,
      platforms: [pluginName],
      entry: {
        type: 'binary',
        command: [`./bin/${pluginBinary}`],
      },
    };
    writeJSON(path.join(packageDir, 'plugins', pluginName, version, 'plugin.json'), manifest);
    writeJSON(path.join(bundleDir, 'plugins', pluginName, version, 'plugin.json'), manifest);
  }
}

function createArchiveFixture(archivesRoot, version) {
  const checksumLines = [];
  for (const target of releaseTargets) {
    const archiveName = `clawrise-cli_${version}_${target.archiveOS}-${target.archiveArch}.tar.gz`;
    fs.writeFileSync(path.join(archivesRoot, archiveName), '');
    checksumLines.push(`${'a'.repeat(64)}  ./${archiveName}`);
  }
  fs.writeFileSync(path.join(archivesRoot, 'SHA256SUMS'), `${checksumLines.join('\n')}\n`);
}

function writeJSON(filePath, value) {
  fs.mkdirSync(path.dirname(filePath), { recursive: true });
  fs.writeFileSync(filePath, `${JSON.stringify(value, null, 2)}\n`);
}

function cleanupFixture(rootDir) {
  fs.rmSync(rootDir, { recursive: true, force: true });
}
