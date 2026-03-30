# npm 发布工作流

本文档面向仓库维护者，说明如何把 Clawrise 的 Go 构建产物打包成 npm 可安装的 CLI。

## 目标

- 合并到指定版本分支后，自动构建各平台预编译二进制
- 发布 `clawrise-cli` 根包，让用户可以直接执行：

```bash
npm install -g clawrise-cli
```

- npm 根包自动解析当前平台对应的二进制
- 第一方 `feishu` / `notion` provider plugin 随平台包一起分发，降低首次使用门槛

## 正式发布规范

- 默认对外发布包名使用无 scope 的最短形式：`clawrise-cli`
- 平台包命名为：`clawrise-cli-<platform>-<arch>`
- 如需 fork 或内部环境改名，可以通过 `CLAWRISE_NPM_PACKAGE_PREFIX` 覆盖默认前缀
- 如果需要企业内网镜像、灰度空间或组织账号名义发布，可以通过 `CLAWRISE_NPM_SCOPE` 注入 scope，例如 `@clawrise`
- 默认 `dist-tag` 规则：
  - 稳定版本，如 `1.2.3`，发布到 `latest`
  - 预发布版本，如 `1.2.3-rc.1`、`1.2.3-beta.2`，发布到 `next`
  - 如需自定义 `beta`、`canary` 等 tag，可通过 `CLAWRISE_NPM_DIST_TAG` 覆盖
- GitHub Release tag 统一使用 `v<version>`，例如 `v1.2.3`

## 包结构

发布链路包含两类 npm 包：

- `clawrise-cli`
  - 根包
  - 暴露 `clawrise` 命令
  - 通过 `optionalDependencies` 依赖各平台包
  - 启动时自动把包内 `plugins/` 注入 `CLAWRISE_PLUGIN_PATHS`
- `clawrise-cli-<platform>-<arch>`
  - 平台包
  - 例如 `clawrise-cli-linux-x64`
  - 包含当前平台的 `clawrise` 二进制
  - 包含第一方 provider plugin 目录和 `plugin.json`

如果启用了 scope，则最终包名会变成：

- `@scope/clawrise-cli`
- `@scope/clawrise-cli-linux-x64`

## 版本分支约定

建议版本分支使用：

```text
release/v0.1.0
```

发布脚本会按以下优先级解析版本：

1. 脚本参数
2. `CLAWRISE_RELEASE_VERSION`
3. `GITHUB_REF_NAME`

支持的输入形式：

- `0.1.0`
- `v0.1.0`
- `release/0.1.0`
- `release/v0.1.0`

## 本地构建

### 1. 生成各平台 bundle

```bash
./scripts/release/build-npm-bundles.sh 0.1.0
```

输出目录：

- `dist/release/bundles/`
- `dist/release/archives/`

每个平台 bundle 包含：

- `bin/clawrise`
- `plugins/feishu/<version>/...`
- `plugins/notion/<version>/...`

### 2. 生成 npm 发布目录

```bash
node ./scripts/release/prepare-npm-packages.mjs 0.1.0
```

输出目录：

- `dist/release/npm/clawrise-cli`
- `dist/release/npm/clawrise-cli-darwin-arm64`
- `dist/release/npm/clawrise-cli-darwin-x64`
- `dist/release/npm/clawrise-cli-linux-arm64`
- `dist/release/npm/clawrise-cli-linux-x64`
- `dist/release/npm/clawrise-cli-win32-arm64`
- `dist/release/npm/clawrise-cli-win32-x64`
- `dist/release/npm/release-metadata.json`

### 3. 发布到 npm

先配置：

```bash
export NODE_AUTH_TOKEN=your_npm_token
```

然后执行：

```bash
./scripts/release/publish-npm.sh 0.1.0
```

脚本会先发布各平台包，再发布根包；如果目标版本已经存在，会自动跳过。
默认会根据版本自动选择 `dist-tag`，也可以通过环境变量覆盖：

```bash
export CLAWRISE_NPM_DIST_TAG=beta
./scripts/release/publish-npm.sh 0.1.0-beta.1
```

## GitHub Actions

工作流文件：

- `.github/workflows/release-npm.yml`

触发方式：

- push 到 `release/**`
- `workflow_dispatch`

工作流会执行：

1. 解析版本
2. 运行 `go test ./...`
3. 构建各平台 bundle
4. 生成 npm 包目录
5. 上传归档产物与 `SHA256SUMS`
6. 创建或更新 GitHub Release，并上传归档文件
7. 如果配置了 `NPM_TOKEN`，自动发布 npm 包

支持的工作流参数与仓库变量：

- `npm_scope`
- `npm_package_prefix`
- `npm_dist_tag`
- `vars.CLAWRISE_NPM_SCOPE`
- `vars.CLAWRISE_NPM_PACKAGE_PREFIX`
- `vars.CLAWRISE_NPM_DIST_TAG`

## Release Notes

release notes 通过模板文件生成：

- `packaging/release/release-notes.md.tmpl`
- `scripts/release/generate-release-notes.sh`

生成方式：

```bash
./scripts/release/generate-release-notes.sh 0.1.0
```

默认输出：

- `dist/release/release-notes.md`

## 版本注入

Go 二进制版本信息通过 `-ldflags` 注入：

- `internal/buildinfo.Version`
- `internal/buildinfo.Commit`
- `internal/buildinfo.BuildDate`

这样 `clawrise` 主程序和第一方 plugin 会共享同一发布版本。

## 相关文档

- [npm 发布运维说明](./npm-release-runbook.md)
