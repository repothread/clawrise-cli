'use strict';

const assert = require('node:assert/strict');
const { test } = require('node:test');

const {
  defaultSetupAccountName,
  ensureValidSetupSelection,
  isValidSetupAccountName,
  parseSetupArgs,
} = require('./setup');

test('解析 client + platform setup 参数', () => {
  const parsed = parseSetupArgs(['codex', 'notion', '--account', 'notion_bot_a1b2']);

  assert.equal(parsed.client, 'codex');
  assert.deepEqual(parsed.platforms, ['notion']);
  assert.equal(parsed.account, 'notion_bot_a1b2');
});

test('解析纯平台 setup 参数', () => {
  const parsed = parseSetupArgs(['feishu']);

  assert.equal(parsed.client, '');
  assert.deepEqual(parsed.platforms, ['feishu']);
});

test('默认账号名符合平台规则', () => {
  assert.equal(defaultSetupAccountName('notion'), 'notion_bot');
  assert.equal(defaultSetupAccountName('feishu'), 'feishu_bot');
});

test('校验 setup 账号名规则', () => {
  assert.equal(isValidSetupAccountName('notion', 'notion_bot'), true);
  assert.equal(isValidSetupAccountName('notion', 'notion_bot_a1b2'), true);
  assert.equal(isValidSetupAccountName('notion', 'notion_bot_abcd1'), false);
  assert.equal(isValidSetupAccountName('feishu', 'feishu_bot'), true);
  assert.equal(isValidSetupAccountName('feishu', 'feishu_bot_x9k3'), true);
  assert.equal(isValidSetupAccountName('feishu', 'feishu_bot_Ab12'), false);
});

test('多平台 setup 不允许 --account', () => {
  assert.throws(() => ensureValidSetupSelection({
    client: 'codex',
    platforms: ['notion', 'feishu'],
    account: 'notion_bot_a1b2',
    token: '',
    appID: '',
    appSecret: '',
    allowInsecureCLISecret: false,
  }), /--account 只允许在单平台 setup 中使用/);
});

test('secret flag 需要显式允许', () => {
  assert.throws(() => ensureValidSetupSelection({
    client: '',
    platforms: ['notion'],
    account: '',
    token: 'secret_xxx',
    appID: '',
    appSecret: '',
    allowInsecureCLISecret: false,
  }), /--token 需要配合 --allow-insecure-cli-secret 使用/);
});
