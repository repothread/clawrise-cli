'use strict';

const childProcess = require('child_process');
const fs = require('fs');
const os = require('os');
const path = require('path');

const supportedClients = ['codex', 'claude-code', 'openclaw', 'opencode'];
const supportedPlatforms = ['feishu', 'notion'];
const platformSkillMap = {
  feishu: 'clawrise-feishu',
  notion: 'clawrise-notion',
};

function handleSetupCommand(rawArgs) {
  const parsed = parseSetupArgs(rawArgs);
  ensureValidClient(parsed.client);
  ensureValidPlatforms(parsed.platforms);
  ensureCLICommandInstalled(parsed);

  const targetRootDir = resolveClientSkillsDir(parsed);
  const selectedSkills = resolveRequestedSkillNames(parsed.platforms);
  const targetName = displayClientName(parsed.client);

  installBundledSkillsInto(selectedSkills, targetRootDir, targetName);
  return 0;
}

function parseSetupArgs(args) {
  const result = {
    client: '',
    platforms: [],
    codexHome: '',
    claudeHome: '',
    openclawHome: '',
    opencodeConfigHome: '',
    skillsDir: '',
    skipCliInstall: false,
  };

  for (let index = 0; index < args.length; index += 1) {
    const token = String(args[index] || '').trim();
    if (token === '') {
      continue;
    }

    if (token === '--codex-home') {
      result.codexHome = readFlagValue(args, index, '--codex-home');
      index += 1;
      continue;
    }

    if (token === '--claude-home') {
      result.claudeHome = readFlagValue(args, index, '--claude-home');
      index += 1;
      continue;
    }

    if (token === '--openclaw-home') {
      result.openclawHome = readFlagValue(args, index, '--openclaw-home');
      index += 1;
      continue;
    }

    if (token === '--opencode-config-home') {
      result.opencodeConfigHome = readFlagValue(args, index, '--opencode-config-home');
      index += 1;
      continue;
    }

    if (token === '--skills-dir') {
      result.skillsDir = readFlagValue(args, index, '--skills-dir');
      index += 1;
      continue;
    }

    if (token === '--skip-cli-install') {
      result.skipCliInstall = true;
      continue;
    }

    if (token === '--help' || token === '-h' || token === 'help') {
      printSetupHelp();
      process.exit(0);
    }

    if (token.startsWith('-')) {
      throw new Error(`Unsupported argument: ${token}`);
    }

    if (result.client === '') {
      result.client = token;
      continue;
    }

    result.platforms.push(token);
  }

  if (result.client === '') {
    printSetupHelp();
    process.exit(0);
  }

  return result;
}

function readFlagValue(args, index, flagName) {
  const value = String(args[index + 1] || '').trim();
  if (value === '') {
    throw new Error(`${flagName} requires a directory argument.`);
  }
  return value;
}

function ensureValidClient(client) {
  if (supportedClients.includes(client)) {
    return;
  }
  throw new Error(`Unsupported client: ${client}. Supported clients: ${supportedClients.join(', ')}`);
}

function ensureValidPlatforms(platforms) {
  const invalidPlatforms = platforms.filter((platform) => !supportedPlatforms.includes(platform));
  if (invalidPlatforms.length === 0) {
    return;
  }
  throw new Error(`Unsupported platform: ${invalidPlatforms.join(', ')}. Supported platforms: ${supportedPlatforms.join(', ')}`);
}

function ensureCLICommandInstalled(parsed) {
  if (parsed.skipCliInstall) {
    console.log('Skipping global CLI installation because --skip-cli-install was set.');
    return;
  }

  if (!shouldInstallCLI()) {
    console.log('The clawrise command is already available. Skipping global CLI installation.');
    return;
  }

  if (!isCommandAvailable('npm')) {
    throw new Error('npm is required to install the clawrise command globally.');
  }

  const packageName = resolveCurrentPackageName();
  console.log(`Installing the clawrise command globally from npm package: ${packageName}`);

  const result = childProcess.spawnSync('npm', ['install', '-g', packageName], {
    stdio: 'inherit',
  });

  if (result.error) {
    throw result.error;
  }
  if (typeof result.status === 'number' && result.status !== 0) {
    throw new Error(`Global CLI installation failed with exit code ${result.status}.`);
  }
}

function shouldInstallCLI() {
  if (isProbablyNPXInvocation()) {
    return true;
  }
  return !isCommandAvailable('clawrise');
}

function isProbablyNPXInvocation() {
  const executablePath = resolveCurrentExecutablePath();
  return executablePath.includes(`${path.sep}_npx${path.sep}`);
}

function resolveCurrentExecutablePath() {
  const currentPath = process.argv[1] || __filename;
  try {
    return fs.realpathSync(currentPath);
  } catch (error) {
    return path.resolve(currentPath);
  }
}

function isCommandAvailable(commandName) {
  const resolverCommand = process.platform === 'win32' ? 'where' : 'which';
  const result = childProcess.spawnSync(resolverCommand, [commandName], {
    stdio: 'ignore',
  });
  return typeof result.status === 'number' && result.status === 0;
}

function resolveCurrentPackageName() {
  const envName = String(process.env.CLAWRISE_ROOT_PACKAGE_NAME || '').trim();
  if (envName !== '') {
    return envName;
  }

  const packageJSONPath = path.join(resolvePackageRootDir(), 'package.json');
  if (fs.existsSync(packageJSONPath)) {
    const packageJSON = JSON.parse(fs.readFileSync(packageJSONPath, 'utf8'));
    if (typeof packageJSON.name === 'string' && packageJSON.name.trim() !== '') {
      return packageJSON.name.trim();
    }
  }

  return '@clawrise/clawrise-cli';
}

function resolvePackageRootDir() {
  return path.resolve(__dirname, '..');
}

function resolveRequestedSkillNames(platforms) {
  const orderedSkills = ['clawrise-core'];
  for (const platform of platforms) {
    const skillName = platformSkillMap[platform];
    if (skillName && !orderedSkills.includes(skillName)) {
      orderedSkills.push(skillName);
    }
  }
  return orderedSkills;
}

function installBundledSkillsInto(selectedSkills, targetRootDir, targetName) {
  const skillsDir = resolveBundledSkillsDir();
  const packagedSkills = listSkillDirectories(skillsDir);

  if (packagedSkills.length === 0) {
    throw new Error('No bundled skills were found in the current npm package.');
  }

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

  console.log(`Setup completed for ${targetName}. Start a new session or restart the client to load the new skills.`);
}

function resolveBundledSkillsDir() {
  return path.join(resolvePackageRootDir(), 'skills');
}

function resolveClientSkillsDir(parsed) {
  if (String(parsed.skillsDir || '').trim() !== '') {
    return path.resolve(parsed.skillsDir);
  }

  switch (parsed.client) {
    case 'codex':
      return path.join(resolveCodexHome(parsed.codexHome), 'skills');
    case 'claude-code':
      return path.join(resolveClaudeCodeHome(parsed.claudeHome), 'skills');
    case 'openclaw':
      return path.join(resolveOpenClawHome(parsed.openclawHome), 'skills');
    case 'opencode':
      return path.join(resolveOpenCodeConfigHome(parsed.opencodeConfigHome), 'skills');
    default:
      throw new Error(`Unsupported client: ${parsed.client}`);
  }
}

function displayClientName(client) {
  switch (client) {
    case 'codex':
      return 'Codex';
    case 'claude-code':
      return 'Claude Code';
    case 'openclaw':
      return 'OpenClaw';
    case 'opencode':
      return 'OpenCode';
    default:
      return client;
  }
}

function resolveCodexHome(explicitValue) {
  const value = String(explicitValue || process.env.CODEX_HOME || '').trim();
  if (value !== '') {
    return path.resolve(value);
  }
  return path.join(os.homedir(), '.codex');
}

function resolveClaudeCodeHome(explicitValue) {
  const value = String(explicitValue || process.env.CLAUDE_HOME || '').trim();
  if (value !== '') {
    return path.resolve(value);
  }
  return path.join(os.homedir(), '.claude');
}

function resolveOpenClawHome(explicitValue) {
  const value = String(explicitValue || process.env.OPENCLAW_HOME || '').trim();
  if (value !== '') {
    return path.resolve(value);
  }
  return path.join(os.homedir(), '.openclaw');
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

function printSetupHelp() {
  console.log('Usage: clawrise setup <client> [platform...]');
  console.log('');
  console.log('Supported clients: codex, claude-code, openclaw, opencode');
  console.log('Supported platforms: feishu, notion');
  console.log('');
  console.log('Examples:');
  console.log('  clawrise setup codex');
  console.log('  clawrise setup codex feishu');
  console.log('  clawrise setup claude-code notion --skills-dir ./.claude/skills');
  console.log('  clawrise setup openclaw feishu --skills-dir ./skills');
  console.log('  clawrise setup opencode notion --skills-dir ./.opencode/skills');
}

module.exports = {
  handleSetupCommand,
};
