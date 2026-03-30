# npm 发布工作流

本文档面向仓库维护者，说明如何把 Clawrise 的 Go 构建产物打包成 npm 可安装的 CLI。

## 目标

- 从 `main` 上选定发布提交并打版本 tag 后，自动构建各平台预编译二进制
- 发布带 scope 的根包，让用户可以直接执行：

```bash
npm install -g @scope/clawrise-cli
```

- npm 根包自动解析当前平台对应的二进制
- 第一方 `feishu` / `notion` provider plugin 随平台包一起分发，降低首次使用门槛

## 正式发布规范

- 官方发布必须设置 `CLAWRISE_NPM_SCOPE`，最终根包会以带 scope 的形式发布，例如 `@clawrise/clawrise-cli`
- 平台包命名为：`clawrise-cli-<platform>-<arch>`
- 如需 fork 或内部环境改名，可以通过 `CLAWRISE_NPM_PACKAGE_PREFIX` 覆盖默认前缀
- `CLAWRISE_NPM_SCOPE` 应设置为实际发布使用的组织或用户 scope，例如 `@clawrise`
- 默认 `dist-tag` 规则：
  - 稳定版本，如 `1.2.3`，发布到 `latest`
  - 预发布版本，如 `1.2.3-rc.1`、`1.2.3-beta.2`，发布到 `next`
  - 如需自定义 `beta`、`canary` 等 tag，可通过 `CLAWRISE_NPM_DIST_TAG` 覆盖
- GitHub Release tag 统一使用 `v<version>`，例如 `v1.2.3`

## 包结构

发布链路包含两类 npm 包：

- `@scope/clawrise-cli`
  - 根包
  - 暴露 `clawrise` 命令
  - 通过 `optionalDependencies` 依赖各平台包
  - 启动时自动把包内 `plugins/` 注入 `CLAWRISE_PLUGIN_PATHS`
- `@scope/clawrise-cli-<platform>-<arch>`
  - 平台包
  - 例如 `clawrise-cli-linux-x64`
  - 包含当前平台的 `clawrise` 二进制
  - 包含第一方 provider plugin 目录和 `plugin.json`

## 标准发版来源

发布脚本会按以下优先级解析版本：

1. 脚本参数
2. `CLAWRISE_RELEASE_VERSION`
3. `GITHUB_REF_NAME`

支持的输入形式：

- `0.1.0`
- `v0.1.0`

标准推荐流程：

1. 所有功能先合并到 `main`
2. 在 `main` 上完成发版前检查
3. 在 `main` 当前提交上打 tag，例如 `v1.2.3`
4. push 该 tag
5. GitHub Actions 监听 `v*` tag 并自动完成构建、GitHub Release 和 npm 发布

示例：

```bash
git checkout main
git pull origin main
bash ./scripts/release/check-release-ready.sh 1.2.3
git tag -a v1.2.3 -m "Release v1.2.3"
git push origin v1.2.3
```

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

这一步仅用于本地手工补救，先配置发布 token：

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

- push 到 `v*` tag
- `workflow_dispatch`

其中：

- `push tags` 是标准正式发版路径
- `workflow_dispatch` 更适合补发、重试或运维场景

工作流会执行：

1. 解析版本
2. 运行 `go test ./...`
3. 构建各平台 bundle
4. 生成 npm 包目录
5. 上传归档产物与 `SHA256SUMS`
6. 创建或更新 GitHub Release，并上传归档文件
7. 通过 npm Trusted Publishing 自动发布 npm 包

支持的工作流参数与环境变量：

- `npm_scope`
- `npm_package_prefix`
- `npm_dist_tag`
- `cicd` environment 变量 `CLAWRISE_NPM_SCOPE`
- `cicd` environment 变量 `CLAWRISE_NPM_PACKAGE_PREFIX`
- `cicd` environment 变量 `CLAWRISE_NPM_DIST_TAG`

## npm Trusted Publishing

官方 npm 发布链路现在使用 Trusted Publishing，而不是长期有效的 `NPM_TOKEN`。

在 GitHub Actions 发布前，需要先在 npmjs.com 为每个发布包分别配置 Trusted Publisher：

- `@clawrise/clawrise-cli`
- `@clawrise/clawrise-cli-darwin-arm64`
- `@clawrise/clawrise-cli-darwin-x64`
- `@clawrise/clawrise-cli-linux-arm64`
- `@clawrise/clawrise-cli-linux-x64`
- `@clawrise/clawrise-cli-win32-arm64`
- `@clawrise/clawrise-cli-win32-x64`

npm Trusted Publisher 请填写：

- CI/CD provider: `GitHub Actions`
- Organization or user: `repothread`
- Repository: `clawrise-cli`
- Workflow filename: `release-npm.yml`
- Environment name: `cicd`

说明：

- npm Trusted Publishing 当前要求 npm CLI `11.5.1+` 与 Node `22.14.0+`
- 当前 workflow 已切换到 Node `24` 满足该要求
- 对公开仓库里的公开包，Trusted Publishing 会自动生成 provenance，因此 workflow 不再显式传 `--provenance`

## 本地发版前检查

建议在 `main` 上执行：

```bash
bash ./scripts/release/check-release-ready.sh 1.2.3
```

脚本会校验：

- 当前是否位于 `main`
- 工作区是否干净
- 目标 tag `v1.2.3` 是否已存在
- `go test ./...`
- 多平台 bundle 与 npm 包目录生成
- release notes 生成
- 当前平台 npm 包是否能成功 `npm pack`

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
