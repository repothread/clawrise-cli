# npm 发布运维说明

本文档记录 npm 发布时的 provenance、常见失败场景以及推荐补救方式。

## Provenance

当前发布 workflow 通过：

- GitHub Actions OIDC 权限：`id-token: write`
- GitHub Actions 上的 npm Trusted Publishing

为 npm 包生成 provenance 证明。

维护建议：

- 仅在 GitHub Actions 中执行正式发布
- 不要把稳定版本改成手工本地发布
- 如果需要追查发布来源，优先检查 GitHub Actions run、GitHub Release 和 npm provenance 元数据是否一致

## 发布前建议

先执行本地检查脚本：

```bash
./scripts/release/check-release-ready.sh 0.1.0
```

脚本会验证：

- 版本解析
- 当前是否位于 `main`
- 工作区是否干净
- 目标版本 tag 是否已经存在
- `go test ./...`
- 多平台 bundle 构建
- npm 发布目录生成
- release notes 生成
- 当前平台 npm 包能否成功 `npm pack`

如果你明确知道当前是在未提交的临时环境中演练，可设置：

```bash
CLAWRISE_RELEASE_ALLOW_DIRTY=1 ./scripts/release/check-release-ready.sh 0.1.0-rc.1
```

如果你确实要在 detached HEAD 上做特殊演练，可设置：

```bash
CLAWRISE_RELEASE_ALLOW_DETACHED=1 ./scripts/release/check-release-ready.sh 0.1.0-rc.1
```

如果还需要检查远端认证：

```bash
CLAWRISE_RELEASE_CHECK_REMOTE=1 NODE_AUTH_TOKEN=... ./scripts/release/check-release-ready.sh 0.1.0
```

如果本地没有 npm token，脚本会跳过 npm 认证检查，因为 Trusted Publishing 依赖 GitHub Actions 的 OIDC，无法在本地 shell 中完整验证。

## 常见失败场景与处理

### 1. 平台包发布成功，根包发布失败

这是最容易补救的一种情况。

处理方式：

- 不要改版本号
- 直接重新执行发布脚本或重新跑 workflow
- `scripts/release/publish-npm.sh` 会自动跳过已存在的包，只发布缺失部分

### 2. 根包已经发布，但 dist-tag 错了

例如预发布版本误上了 `latest`。

处理方式：

- 不要重新发同版本
- 使用 npm dist-tag 修正

示例：

```bash
npm dist-tag add clawrise-cli@1.2.3 next
npm dist-tag rm clawrise-cli latest
```

如果使用了 scope，请替换为实际包名。

### 3. 同版本内容发错了

npm 版本是不可变的，不要依赖删除重发。

建议：

- 直接发布新版本
- 稳定版本走补丁号，例如 `1.2.4`
- 预发布版本继续推进，例如 `1.2.4-rc.1`

### 4. GitHub Release 缺少文件或说明不正确

处理方式：

- 重新跑 workflow
- 或使用 `gh release upload --clobber` 覆盖资产
- release notes 可通过 `gh release edit` 更新

### 5. workflow 失败，但本地产物看起来正常

优先检查：

- 目标 npm 包是否已经把 `release-npm.yml` 配置为 Trusted Publisher
- GitHub Actions 是否拥有 `id-token: write`
- `gh` / npm registry 权限是否受限
- 版本是否已存在于 npm

## 回滚策略

稳定版本建议遵循以下原则：

- 尽量不要执行 unpublish
- 把错误版本从 `latest` 移走，而不是删除版本
- 用新版本覆盖错误行为，而不是尝试复写既有版本

建议顺序：

1. 修正 dist-tag，避免新用户继续安装到错误版本
2. 发布修复版本
3. 更新 GitHub Release 说明，明确哪些版本受影响

## 需要人工确认的事项

自动化不会替你判断以下内容：

- 这次发布是否应该使用 `latest` 还是 `next`
- 包名或 scope 是否需要切换
- GitHub Release 是否应该标记为 prerelease
- 用户文档里的安装命令是否需要同步调整
