import assert from 'node:assert/strict';
import crypto from 'node:crypto';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { spawnSync } from 'node:child_process';
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

test('skill 归档缺少源文件时失败', () => {
  const fixture = createReleaseFixture('1.2.3', {
    omitArchivedFiles: {
      'clawrise-notion': ['references/operation-map.md'],
    },
  });
  try {
    assert.throws(
      () => verifyReleaseArtifacts({ repoRoot: fixture.repoRoot, version: '1.2.3' }),
      /skill 归档缺少源文件：clawrise-notion\/references\/operation-map\.md/,
    );
  } finally {
    cleanupFixture(fixture.repoRoot);
  }
});

function createReleaseFixture(version, options = {}) {
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
  createSkillsFixture(repoRoot, version, options);
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
      schema_version: 2,
      name: pluginName,
      version,
      protocol_version: 1,
      min_core_version: version,
      capabilities: [
        {
          type: 'provider',
          platforms: [pluginName],
        },
      ],
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

function createSkillsFixture(repoRoot, version, options = {}) {
  const skillsSourceRoot = path.join(repoRoot, 'skills');
  const skillsDistRoot = path.join(repoRoot, 'dist', 'release', 'skills');
  const versionRoot = path.join(skillsDistRoot, version);
  const generatedAt = '2026-04-05T00:00:00.000Z';
  const skillDefinitions = [
    {
      name: 'clawrise-core',
      description: 'Generic Clawrise CLI workflow and setup.',
      referenceFiles: ['references/install-and-layout.md', 'references/workflow.md'],
    },
    {
      name: 'clawrise-feishu',
      description: 'Legacy Feishu operations exposed through Clawrise.',
      referenceFiles: ['references/common-tasks.md'],
    },
    {
      name: 'clawrise-notion',
      description: 'Notion operations exposed through Clawrise.',
      referenceFiles: ['references/common-tasks.md', 'references/operation-map.md'],
    },
  ];

  fs.mkdirSync(versionRoot, { recursive: true });

  const manifestSkills = [];
  for (const skill of skillDefinitions) {
    const sourceDir = path.join(skillsSourceRoot, skill.name);
    fs.mkdirSync(sourceDir, { recursive: true });
    fs.writeFileSync(
      path.join(sourceDir, 'SKILL.md'),
      `---\nname: ${skill.name}\ndescription: ${skill.description}\n---\n\n# ${skill.name}\n`,
    );
    for (const relativeFile of skill.referenceFiles) {
      const filePath = path.join(sourceDir, relativeFile);
      fs.mkdirSync(path.dirname(filePath), { recursive: true });
      fs.writeFileSync(filePath, `${skill.name}:${relativeFile}\n`);
    }

    const archivePath = path.join(versionRoot, `${skill.name}.tar.gz`);
    const stageDir = createArchiveStage(sourceDir, options.omitArchivedFiles?.[skill.name] || []);
    archiveDirectory(stageDir, archivePath);

    const relativeArchivePath = `${version}/${skill.name}.tar.gz`;
    manifestSkills.push({
      name: skill.name,
      description: skill.description,
      platforms: inferSkillPlatforms(skill.name),
      requires: inferSkillRequirements(skill.name),
      reference_files: [...skill.referenceFiles].sort(),
      version,
      archive_path: relativeArchivePath,
      url: relativeArchivePath,
      checksum_sha256: sha256File(archivePath),
      size_bytes: fs.statSync(archivePath).size,
    });
  }

  writeJSON(path.join(skillsDistRoot, 'index.json'), {
    generated_at: generatedAt,
    latest_version: version,
    skills: manifestSkills,
  });
  writeJSON(path.join(skillsDistRoot, 'latest.json'), {
    generated_at: generatedAt,
    version,
    skills: manifestSkills,
  });
}

function createArchiveStage(sourceDir, omittedFiles) {
  if (!Array.isArray(omittedFiles) || omittedFiles.length === 0) {
    return sourceDir;
  }

  const repoRoot = path.dirname(path.dirname(sourceDir));
  const stageRoot = path.join(repoRoot, '.skill-archive-stage');
  fs.mkdirSync(stageRoot, { recursive: true });
  const stageDir = fs.mkdtempSync(path.join(stageRoot, `${path.basename(sourceDir)}-stage-`));
  copyTree(sourceDir, stageDir);
  for (const relativeFile of omittedFiles) {
    fs.rmSync(path.join(stageDir, relativeFile), { recursive: true, force: true });
  }
  return stageDir;
}

function archiveDirectory(sourceDir, archivePath) {
  const result = spawnSync('tar', ['-C', sourceDir, '-czf', archivePath, '.'], {
    stdio: 'pipe',
  });
  if (result.status !== 0) {
    const stderr = (result.stderr || Buffer.alloc(0)).toString('utf8').trim();
    throw new Error(`Failed to archive fixture directory: ${stderr}`);
  }
}

function copyTree(sourceDir, targetDir) {
  fs.mkdirSync(targetDir, { recursive: true });
  for (const entry of fs.readdirSync(sourceDir, { withFileTypes: true })) {
    const sourcePath = path.join(sourceDir, entry.name);
    const targetPath = path.join(targetDir, entry.name);
    if (entry.isDirectory()) {
      copyTree(sourcePath, targetPath);
      continue;
    }
    if (entry.isFile()) {
      fs.mkdirSync(path.dirname(targetPath), { recursive: true });
      fs.copyFileSync(sourcePath, targetPath);
    }
  }
}

function inferSkillPlatforms(skillName) {
  if (skillName === 'clawrise-core') {
    return [];
  }
  return [skillName.slice('clawrise-'.length)];
}

function inferSkillRequirements(skillName) {
  if (skillName === 'clawrise-core') {
    return [];
  }
  return ['clawrise-core'];
}

function sha256File(filePath) {
  const hash = crypto.createHash('sha256');
  hash.update(fs.readFileSync(filePath));
  return hash.digest('hex');
}

function writeJSON(filePath, value) {
  fs.mkdirSync(path.dirname(filePath), { recursive: true });
  fs.writeFileSync(filePath, `${JSON.stringify(value, null, 2)}\n`);
}

function cleanupFixture(rootDir) {
  fs.rmSync(rootDir, { recursive: true, force: true });
}
