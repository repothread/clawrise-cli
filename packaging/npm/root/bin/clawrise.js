#!/usr/bin/env node

'use strict';

const childProcess = require('child_process');
const path = require('path');
const { resolvePlatformPackage } = require('../lib/platform');
const { handleSetupCommand } = require('../lib/setup');

const rawArgs = process.argv.slice(2);

if (rawArgs[0] === 'setup') {
  try {
    const handledExitCode = handleSetupCommand(rawArgs.slice(1));
    if (typeof handledExitCode === 'number') {
      process.exit(handledExitCode);
    }
  } catch (error) {
    console.error(error.message);
    process.exit(1);
  }
}

const platformPackage = resolvePlatformPackage();
const pluginPath = path.join(platformPackage.dir, 'plugins');
const nextEnv = { ...process.env };

nextEnv.CLAWRISE_PLUGIN_PATHS = nextEnv.CLAWRISE_PLUGIN_PATHS
  ? `${pluginPath}${path.delimiter}${nextEnv.CLAWRISE_PLUGIN_PATHS}`
  : pluginPath;

const result = childProcess.spawnSync(platformPackage.binaryPath, rawArgs, {
  stdio: 'inherit',
  env: nextEnv,
});

if (result.error) {
  console.error(result.error.message);
  process.exit(1);
}

if (typeof result.status === 'number') {
  process.exit(result.status);
}

if (result.signal) {
  process.kill(process.pid, result.signal);
}

process.exit(1);
