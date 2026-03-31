#!/usr/bin/env node

'use strict';

const childProcess = require('child_process');
const path = require('path');
const { resolvePlatformPackage } = require('../lib/platform');
const { handleSetupCommand } = require('../lib/setup');

async function main() {
  const rawArgs = process.argv.slice(2);

  if (rawArgs[0] === 'setup') {
    const handledExitCode = await handleSetupCommand(rawArgs.slice(1));
    if (typeof handledExitCode === 'number') {
      process.exit(handledExitCode);
    }
    return;
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
    throw result.error;
  }

  if (typeof result.status === 'number') {
    process.exit(result.status);
  }

  if (result.signal) {
    process.kill(process.pid, result.signal);
  }

  process.exit(1);
}

main().catch((error) => {
  console.error(error.message);
  process.exit(1);
});
