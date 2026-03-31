'use strict';

const fs = require('fs');
const os = require('os');
const path = require('path');

function handleSkillsCommand(rawArgs) {
  const args = Array.isArray(rawArgs) ? [...rawArgs] : [];
  if (args.length === 0) {
    printSkillsHelp();
    return 0;
  }

  switch (args[0]) {
    case 'install-codex':
      return installCodexSkills(args.slice(1));
    case 'install-claude-code':
      return installClaudeCodeSkills(args.slice(1));
    case 'install-openclaw':
      return installOpenClawSkills(args.slice(1));
    case 'install-opencode':
      return installOpenCodeSkills(args.slice(1));
    case 'list-codex-packaged':
      return listPackagedSkills();
    case 'help':
    case '--help':
    case '-h':
      printSkillsHelp();
      return 0;
    default:
      return null;
  }
}

function installCodexSkills(rawArgs) {
  const parsed = parseInstallArgs(rawArgs);
  const targetRootDir = resolveCodexSkillsDir(parsed);
  return installBundledSkillsInto(parsed, targetRootDir, 'Codex');
}

function installOpenClawSkills(rawArgs) {
  const parsed = parseInstallArgs(rawArgs);
  const targetRootDir = resolveOpenClawSkillsDir(parsed);
  return installBundledSkillsInto(parsed, targetRootDir, 'OpenClaw');
}

function installClaudeCodeSkills(rawArgs) {
  const parsed = parseInstallArgs(rawArgs);
  const targetRootDir = resolveClaudeCodeSkillsDir(parsed);
  return installBundledSkillsInto(parsed, targetRootDir, 'Claude Code');
}

function installOpenCodeSkills(rawArgs) {
  const parsed = parseInstallArgs(rawArgs);
  const targetRootDir = resolveOpenCodeSkillsDir(parsed);
  return installBundledSkillsInto(parsed, targetRootDir, 'OpenCode');
}

function installBundledSkillsInto(parsed, targetRootDir, targetName) {
  const skillsDir = resolveBundledSkillsDir();
  const packagedSkills = listSkillDirectories(skillsDir);

  if (packagedSkills.length === 0) {
    throw new Error('No bundled skills were found in the current npm package.');
  }

  const selectedSkills = parsed.skillNames.length === 0 ? packagedSkills : parsed.skillNames;
  const invalidSkills = selectedSkills.filter((name) => !packagedSkills.includes(name));
  if (invalidSkills.length > 0) {
    throw new Error(`The following skills are not bundled in the current npm package: ${invalidSkills.join(', ')}`);
  }

  fs.mkdirSync(targetRootDir, { recursive: true });

  for (const skillName of selectedSkills) {
    const sourceDir = path.join(skillsDir, skillName);
    const targetDir = path.join(targetRootDir, skillName);
    fs.rmSync(targetDir, { recursive: true, force: true });
    copyTree(sourceDir, targetDir);
    console.log(`Installed skill: ${skillName} -> ${targetDir}`);
  }

  console.log(`Installation completed. Restart ${targetName} to load the new skills.`);
  return 0;
}

function listPackagedSkills() {
  const skillsDir = resolveBundledSkillsDir();
  const packagedSkills = listSkillDirectories(skillsDir);

  if (packagedSkills.length === 0) {
    console.log('No bundled skills were found in the current npm package.');
    return 0;
  }

  for (const skillName of packagedSkills) {
    console.log(skillName);
  }
  return 0;
}

function parseInstallArgs(args) {
  const result = {
    codexHome: '',
    claudeHome: '',
    openclawHome: '',
    opencodeConfigHome: '',
    skillsDir: '',
    skillNames: [],
  };

  for (let index = 0; index < args.length; index += 1) {
    const token = String(args[index] || '').trim();
    if (token === '') {
      continue;
    }

    if (token === '--codex-home') {
      const value = String(args[index + 1] || '').trim();
      if (value === '') {
        throw new Error('--codex-home requires a directory argument.');
      }
      result.codexHome = value;
      index += 1;
      continue;
    }

    if (token === '--openclaw-home') {
      const value = String(args[index + 1] || '').trim();
      if (value === '') {
        throw new Error('--openclaw-home requires a directory argument.');
      }
      result.openclawHome = value;
      index += 1;
      continue;
    }

    if (token === '--claude-home') {
      const value = String(args[index + 1] || '').trim();
      if (value === '') {
        throw new Error('--claude-home requires a directory argument.');
      }
      result.claudeHome = value;
      index += 1;
      continue;
    }

    if (token === '--opencode-config-home') {
      const value = String(args[index + 1] || '').trim();
      if (value === '') {
        throw new Error('--opencode-config-home requires a directory argument.');
      }
      result.opencodeConfigHome = value;
      index += 1;
      continue;
    }

    if (token === '--skills-dir') {
      const value = String(args[index + 1] || '').trim();
      if (value === '') {
        throw new Error('--skills-dir requires a directory argument.');
      }
      result.skillsDir = value;
      index += 1;
      continue;
    }

    if (token === '--help' || token === '-h') {
      printSkillsHelp();
      process.exit(0);
    }

    if (token.startsWith('-')) {
      throw new Error(`Unsupported argument: ${token}`);
    }

    result.skillNames.push(token);
  }

  return result;
}

function resolveBundledSkillsDir() {
  const packageRootDir = path.resolve(__dirname, '..');
  return path.join(packageRootDir, 'skills');
}

function resolveCodexHome(explicitValue) {
  const value = String(explicitValue || process.env.CODEX_HOME || '').trim();
  if (value !== '') {
    return path.resolve(value);
  }
  return path.join(os.homedir(), '.codex');
}

function resolveOpenClawHome(explicitValue) {
  const value = String(explicitValue || process.env.OPENCLAW_HOME || '').trim();
  if (value !== '') {
    return path.resolve(value);
  }
  return path.join(os.homedir(), '.openclaw');
}

function resolveClaudeCodeHome(explicitValue) {
  const value = String(explicitValue || process.env.CLAUDE_HOME || '').trim();
  if (value !== '') {
    return path.resolve(value);
  }
  return path.join(os.homedir(), '.claude');
}

function resolveOpenCodeConfigHome(explicitValue) {
  const value = String(explicitValue || process.env.OPENCODE_CONFIG_HOME || '').trim();
  if (value !== '') {
    return path.resolve(value);
  }

  const xdgConfigHome = String(process.env.XDG_CONFIG_HOME || '').trim();
  if (xdgConfigHome !== '') {
    return path.join(path.resolve(xdgConfigHome), 'opencode');
  }

  return path.join(os.homedir(), '.config', 'opencode');
}

function resolveCodexSkillsDir(parsed) {
  if (String(parsed.skillsDir || '').trim() !== '') {
    return path.resolve(parsed.skillsDir);
  }
  return path.join(resolveCodexHome(parsed.codexHome), 'skills');
}

function resolveClaudeCodeSkillsDir(parsed) {
  if (String(parsed.skillsDir || '').trim() !== '') {
    return path.resolve(parsed.skillsDir);
  }
  return path.join(resolveClaudeCodeHome(parsed.claudeHome), 'skills');
}

function resolveOpenClawSkillsDir(parsed) {
  if (String(parsed.skillsDir || '').trim() !== '') {
    return path.resolve(parsed.skillsDir);
  }
  return path.join(resolveOpenClawHome(parsed.openclawHome), 'skills');
}

function resolveOpenCodeSkillsDir(parsed) {
  if (String(parsed.skillsDir || '').trim() !== '') {
    return path.resolve(parsed.skillsDir);
  }
  return path.join(resolveOpenCodeConfigHome(parsed.opencodeConfigHome), 'skills');
}

function listSkillDirectories(rootDir) {
  if (!fs.existsSync(rootDir)) {
    return [];
  }

  return fs.readdirSync(rootDir, { withFileTypes: true })
    .filter((entry) => entry.isDirectory())
    .map((entry) => entry.name)
    .sort();
}

function copyTree(sourceDir, targetDir) {
  const sourceStat = fs.statSync(sourceDir);
  if (!sourceStat.isDirectory()) {
    throw new Error(`Skill source directory does not exist or is not a directory: ${sourceDir}`);
  }

  fs.mkdirSync(targetDir, { recursive: true });
  for (const entry of fs.readdirSync(sourceDir, { withFileTypes: true })) {
    const sourcePath = path.join(sourceDir, entry.name);
    const targetPath = path.join(targetDir, entry.name);

    if (entry.isDirectory()) {
      copyTree(sourcePath, targetPath);
      continue;
    }

    if (entry.isSymbolicLink()) {
      const linkTarget = fs.readlinkSync(sourcePath);
      fs.symlinkSync(linkTarget, targetPath);
      continue;
    }

    fs.copyFileSync(sourcePath, targetPath);
  }
}

function printSkillsHelp() {
  console.log('Usage: clawrise skills <install-codex|install-claude-code|install-openclaw|install-opencode|list-codex-packaged>');
  console.log('');
  console.log('Examples:');
  console.log('  clawrise skills list-codex-packaged');
  console.log('  clawrise skills install-codex');
  console.log('  clawrise skills install-codex clawrise-core clawrise-feishu');
  console.log('  clawrise skills install-codex --codex-home /tmp/codex-home');
  console.log('  clawrise skills install-claude-code');
  console.log('  clawrise skills install-claude-code --skills-dir ./.claude/skills');
  console.log('  clawrise skills install-openclaw');
  console.log('  clawrise skills install-openclaw --openclaw-home ~/.openclaw');
  console.log('  clawrise skills install-openclaw --skills-dir ./skills');
  console.log('  clawrise skills install-opencode');
  console.log('  clawrise skills install-opencode --skills-dir ./.opencode/skills');
}

module.exports = {
  handleSkillsCommand,
};
