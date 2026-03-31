'use strict';

const childProcess = require('child_process');
const fs = require('fs');
const os = require('os');
const path = require('path');
const readlinePromises = require('readline/promises');

const supportedClients = ['codex', 'claude-code', 'openclaw', 'opencode'];
const supportedPlatforms = ['feishu', 'notion'];
const platformSkillMap = {
  feishu: 'clawrise-feishu',
  notion: 'clawrise-notion',
};

async function handleSetupCommand(rawArgs) {
  const parsed = parseSetupArgs(rawArgs);
  ensureValidSetupSelection(parsed);
  ensureCLICommandInstalled(parsed);

  if (parsed.client !== '') {
    const targetRootDir = resolveClientSkillsDir(parsed);
    const selectedSkills = resolveRequestedSkillNames(parsed.platforms);
    const targetName = displayClientName(parsed.client);
    installBundledSkillsInto(selectedSkills, targetRootDir, targetName);
  }

  if (parsed.skipAuth || parsed.platforms.length === 0) {
    return 0;
  }

  for (const platform of parsed.platforms) {
    await setupPlatformAccount(platform, parsed);
  }

  console.log('Setup 已完成，账号和凭证已写入本地配置。');
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
    account: '',
    skipAuth: false,
    token: '',
    appID: '',
    appSecret: '',
    allowInsecureCLISecret: false,
  };

  const positionalArgs = [];
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

    if (token === '--account') {
      result.account = readFlagValue(args, index, '--account');
      index += 1;
      continue;
    }

    if (token === '--token') {
      result.token = readFlagValue(args, index, '--token');
      index += 1;
      continue;
    }

    if (token === '--app-id') {
      result.appID = readFlagValue(args, index, '--app-id');
      index += 1;
      continue;
    }

    if (token === '--app-secret') {
      result.appSecret = readFlagValue(args, index, '--app-secret');
      index += 1;
      continue;
    }

    if (token === '--skip-cli-install') {
      result.skipCliInstall = true;
      continue;
    }

    if (token === '--skip-auth') {
      result.skipAuth = true;
      continue;
    }

    if (token === '--allow-insecure-cli-secret') {
      result.allowInsecureCLISecret = true;
      continue;
    }

    if (token === '--help' || token === '-h' || token === 'help') {
      printSetupHelp();
      process.exit(0);
    }

    if (token.startsWith('-')) {
      throw new Error(`不支持的参数: ${token}`);
    }

    positionalArgs.push(token);
  }

  if (positionalArgs.length === 0) {
    printSetupHelp();
    process.exit(0);
  }

  const firstTarget = positionalArgs[0];
  if (supportedClients.includes(firstTarget)) {
    result.client = firstTarget;
    result.platforms = positionalArgs.slice(1);
    return result;
  }

  result.platforms = positionalArgs;
  return result;
}

function readFlagValue(args, index, flagName) {
  const value = String(args[index + 1] || '').trim();
  if (value === '') {
    throw new Error(`${flagName} 需要一个非空参数。`);
  }
  return value;
}

function ensureValidSetupSelection(parsed) {
  if (parsed.client !== '' && !supportedClients.includes(parsed.client)) {
    throw new Error(`不支持的客户端: ${parsed.client}`);
  }

  if (parsed.platforms.length === 0 && parsed.client === '') {
    throw new Error('setup 至少需要指定一个 client 或 platform。');
  }

  const invalidPlatforms = parsed.platforms.filter((platform) => !supportedPlatforms.includes(platform));
  if (invalidPlatforms.length > 0) {
    throw new Error(`不支持的平台: ${invalidPlatforms.join(', ')}`);
  }

  if (parsed.account !== '' && parsed.platforms.length !== 1) {
    throw new Error('--account 只允许在单平台 setup 中使用。');
  }

  if (parsed.account !== '') {
    const platform = parsed.platforms[0];
    if (!isValidSetupAccountName(platform, parsed.account)) {
      throw new Error(`账号名不合法: ${parsed.account}`);
    }
  }

  if (parsed.token !== '' && !parsed.allowInsecureCLISecret) {
    throw new Error('--token 需要配合 --allow-insecure-cli-secret 使用。');
  }

  if (parsed.appSecret !== '' && !parsed.allowInsecureCLISecret) {
    throw new Error('--app-secret 需要配合 --allow-insecure-cli-secret 使用。');
  }

  if (parsed.token !== '' && !parsed.platforms.includes('notion')) {
    throw new Error('--token 只能用于 notion setup。');
  }

  if ((parsed.appID !== '' || parsed.appSecret !== '') && !parsed.platforms.includes('feishu')) {
    throw new Error('--app-id 和 --app-secret 只能用于 feishu setup。');
  }
}

function isValidSetupAccountName(platform, accountName) {
  const normalizedName = String(accountName || '').trim();
  if (normalizedName === '') {
    return false;
  }

  switch (platform) {
    case 'notion':
      return /^notion_bot(?:_[a-z0-9]{4})?$/.test(normalizedName);
    case 'feishu':
      return /^feishu_bot(?:_[a-z0-9]{4})?$/.test(normalizedName);
    default:
      return false;
  }
}

function defaultSetupAccountName(platform) {
  switch (platform) {
    case 'notion':
      return 'notion_bot';
    case 'feishu':
      return 'feishu_bot';
    default:
      throw new Error(`不支持的平台: ${platform}`);
  }
}

function ensureCLICommandInstalled(parsed) {
  if (parsed.skipCliInstall) {
    console.log('已跳过全局 CLI 安装。');
    return;
  }

  if (!shouldInstallCLI()) {
    console.log('检测到 clawrise 命令已可用，跳过全局安装。');
    return;
  }

  if (!isCommandAvailable('npm')) {
    throw new Error('缺少 npm，无法安装 clawrise 命令。');
  }

  const packageName = resolveCurrentPackageName();
  console.log(`正在全局安装 ${packageName} ...`);

  const result = childProcess.spawnSync('npm', ['install', '-g', packageName], {
    stdio: 'inherit',
  });

  if (result.error) {
    throw result.error;
  }
  if (typeof result.status === 'number' && result.status !== 0) {
    throw new Error(`全局安装 clawrise 失败，退出码: ${result.status}`);
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
    throw new Error('当前 npm 包中没有可安装的 skills。');
  }

  const invalidSkills = selectedSkills.filter((name) => !packagedSkills.includes(name));
  if (invalidSkills.length > 0) {
    throw new Error(`当前 npm 包中缺少以下 skills: ${invalidSkills.join(', ')}`);
  }

  fs.mkdirSync(targetRootDir, { recursive: true });

  for (const skillName of selectedSkills) {
    const sourceDir = path.join(skillsDir, skillName);
    const targetDir = path.join(targetRootDir, skillName);
    fs.rmSync(targetDir, { recursive: true, force: true });
    copyTree(sourceDir, targetDir);
    console.log(`已安装 skill: ${skillName} -> ${targetDir}`);
  }

  console.log(`${targetName} 的 skills 安装完成，请重新打开会话或重启客户端。`);
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
      throw new Error(`不支持的客户端: ${parsed.client}`);
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
    throw new Error(`skill 源目录不存在或不是目录: ${sourceDir}`);
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

async function setupPlatformAccount(platform, parsed) {
  const accountName = parsed.account || defaultSetupAccountName(platform);

  switch (platform) {
    case 'notion':
      await setupNotionAccount(accountName, parsed);
      return;
    case 'feishu':
      await setupFeishuAccount(accountName, parsed);
      return;
    default:
      throw new Error(`不支持的平台: ${platform}`);
  }
}

async function setupNotionAccount(accountName, parsed) {
  const token = await resolveSecretInput({
    envName: 'NOTION_INTERNAL_TOKEN',
    explicitValue: parsed.token,
    promptText: '请输入 Notion Internal Integration Token: ',
    missingMessage: '缺少 Notion token，请设置 NOTION_INTERNAL_TOKEN，或使用 --token，或在交互终端中输入。',
  });

  runCurrentClawriseCommand([
    'account', 'ensure', accountName,
    '--platform', 'notion',
    '--preset', 'internal_token',
    '--use',
  ]);
  runCurrentClawriseCommand([
    'auth', 'secret', 'set', accountName, 'token', '--stdin',
  ], { input: `${token}\n` });
  runCurrentClawriseCommand(['auth', 'check', accountName]);
  console.log(`Notion 接入完成，账号名: ${accountName}`);
}

async function setupFeishuAccount(accountName, parsed) {
  const appID = await resolveTextInput({
    envName: 'FEISHU_APP_ID',
    explicitValue: parsed.appID,
    promptText: '请输入 Feishu App ID: ',
    missingMessage: '缺少 Feishu App ID，请设置 FEISHU_APP_ID，或使用 --app-id，或在交互终端中输入。',
  });
  const appSecret = await resolveSecretInput({
    envName: 'FEISHU_APP_SECRET',
    explicitValue: parsed.appSecret,
    promptText: '请输入 Feishu App Secret: ',
    missingMessage: '缺少 Feishu App Secret，请设置 FEISHU_APP_SECRET，或使用 --app-secret，或在交互终端中输入。',
  });

  runCurrentClawriseCommand([
    'account', 'ensure', accountName,
    '--platform', 'feishu',
    '--preset', 'bot',
    '--use',
    '--public', `app_id=${appID}`,
  ]);
  runCurrentClawriseCommand([
    'auth', 'secret', 'set', accountName, 'app_secret', '--stdin',
  ], { input: `${appSecret}\n` });
  runCurrentClawriseCommand(['auth', 'check', accountName]);
  console.log(`Feishu 接入完成，账号名: ${accountName}`);
}

async function resolveTextInput(options) {
  const explicitValue = String(options.explicitValue || '').trim();
  if (explicitValue !== '') {
    return explicitValue;
  }

  const envValue = String(process.env[options.envName] || '').trim();
  if (envValue !== '') {
    return envValue;
  }

  if (!canPromptInteractively()) {
    throw new Error(options.missingMessage);
  }

  const promptedValue = String(await promptVisible(options.promptText)).trim();
  if (promptedValue === '') {
    throw new Error(options.missingMessage);
  }
  return promptedValue;
}

async function resolveSecretInput(options) {
  const explicitValue = String(options.explicitValue || '').trim();
  if (explicitValue !== '') {
    return explicitValue;
  }

  const envValue = String(process.env[options.envName] || '').trim();
  if (envValue !== '') {
    return envValue;
  }

  if (!canPromptInteractively()) {
    throw new Error(options.missingMessage);
  }

  const promptedValue = String(await promptSecret(options.promptText)).trim();
  if (promptedValue === '') {
    throw new Error(options.missingMessage);
  }
  return promptedValue;
}

function canPromptInteractively() {
  return Boolean(process.stdin && process.stdin.isTTY && process.stdout && process.stdout.isTTY);
}

async function promptVisible(message) {
  const rl = readlinePromises.createInterface({
    input: process.stdin,
    output: process.stdout,
  });
  try {
    return await rl.question(message);
  } finally {
    rl.close();
  }
}

function promptSecret(message) {
  if (!canPromptInteractively()) {
    return Promise.resolve('');
  }

  return new Promise((resolve, reject) => {
    const stdin = process.stdin;
    const stdout = process.stdout;
    let value = '';

    stdout.write(message);
    stdin.setEncoding('utf8');
    if (typeof stdin.setRawMode === 'function') {
      stdin.setRawMode(true);
    }
    stdin.resume();

    const cleanup = () => {
      stdin.removeListener('data', onData);
      if (typeof stdin.setRawMode === 'function') {
        stdin.setRawMode(false);
      }
      stdin.pause();
    };

    const onData = (chunk) => {
      for (const char of String(chunk)) {
        if (char === '\u0003') {
          cleanup();
          reject(new Error('用户取消了 setup。'));
          return;
        }
        if (char === '\r' || char === '\n') {
          stdout.write(os.EOL);
          cleanup();
          resolve(value);
          return;
        }
        if (char === '\u007f' || char === '\b') {
          value = value.slice(0, -1);
          continue;
        }
        value += char;
      }
    };

    stdin.on('data', onData);
  });
}

function runCurrentClawriseCommand(args, options) {
  const config = options || {};
  const commandArgs = [resolveCurrentCLIBinPath(), ...args];
  const result = childProcess.spawnSync(process.execPath, commandArgs, {
    cwd: process.cwd(),
    env: { ...process.env, ...(config.env || {}) },
    input: config.input,
    encoding: 'utf8',
    stdio: ['pipe', 'pipe', 'pipe'],
  });

  if (result.error) {
    throw result.error;
  }
  if (typeof result.status === 'number' && result.status !== 0) {
    const message = [result.stdout, result.stderr]
      .map((value) => String(value || '').trim())
      .filter((value) => value !== '')
      .join('\n');
    throw new Error(message || `clawrise ${args.join(' ')} 执行失败，退出码: ${result.status}`);
  }
  return String(result.stdout || '');
}

function resolveCurrentCLIBinPath() {
  return path.join(resolvePackageRootDir(), 'bin', 'clawrise.js');
}

function printSetupHelp() {
  console.log('用法: clawrise setup <client> [platform...]');
  console.log('      clawrise setup <platform>');
  console.log('');
  console.log('支持的客户端: codex, claude-code, openclaw, opencode');
  console.log('支持的平台: feishu, notion');
  console.log('');
  console.log('常用示例:');
  console.log('  clawrise setup codex');
  console.log('  clawrise setup codex notion');
  console.log('  clawrise setup codex feishu');
  console.log('  clawrise setup notion');
  console.log('  clawrise setup feishu');
}

module.exports = {
  ensureValidSetupSelection,
  defaultSetupAccountName,
  handleSetupCommand,
  isValidSetupAccountName,
  parseSetupArgs,
};
