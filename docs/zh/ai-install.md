# Clawrise AI 安装引导

当 AI 助手需要为当前客户端安装已发布的 `clawrise` npm 包，并执行打包提供的 `setup` 包装层时，使用这份文档。

## 给人类用户

把下面这段 prompt 发给 AI 助手：

```text
访问 https://raw.githubusercontent.com/repothread/clawrise-cli/main/docs/zh/ai-install.md，并按文档步骤为当前客户端安装 `clawrise` 命令并执行 setup。
```

## 给 AI 助手

请直接读取 raw 文件：

```bash
curl -fsSL https://raw.githubusercontent.com/repothread/clawrise-cli/main/docs/zh/ai-install.md
```

执行安装流程时，不要依赖 GitHub 渲染页。

### 第 0 步：确认目标环境

在安装前先和用户确认：

1. 当前要接入哪个客户端？
   - Codex
   - Claude Code
   - OpenClaw
   - OpenCode
2. 当前需要哪些平台？
   - `feishu`
   - `notion`
3. skills 是安装到客户端默认共享目录，还是项目内自定义目录？

### 第 1 步：执行预检

先确认所需命令存在：

```bash
command -v npm
command -v npx
```

如果缺少 `npm`，先让用户安装 Node.js 和 npm。

### 第 2 步：选择 setup 入口

已发布的 npm 根包会暴露 `clawrise setup ...`。

直接执行 `go run ./cmd/clawrise` 或使用开发态原生二进制时，只会暴露 runtime 命令，不包含这个包装层。

如果当前环境已经安装了已发布的 `clawrise` 命令，就直接使用：

```bash
clawrise setup ...
```

否则使用：

```bash
npx @clawrise/clawrise-cli setup ...
```

`setup` 负责：

- 在未使用 `--skip-cli-install` 时，确保宿主机存在已发布的 `clawrise` 命令
- 安装 `clawrise-core`
- 安装请求的平台 skills
- 在凭证可用时初始化默认平台账号

setup 生成的默认账号名来自所选 auth preset 的元数据，不同平台和鉴权方式可能不同。

除非你在 setup 时显式传了 `--account`，否则不要在自动化里硬编码账号名。

如果没有指定平台，`setup` 只安装 `clawrise-core`，不会初始化平台账号。

### 第 3 步：执行 setup

如果 `clawrise` 已经安装可用：

```bash
clawrise setup codex
clawrise setup codex feishu
```

如果 `clawrise` 还没有安装：

```bash
npx @clawrise/clawrise-cli setup codex
npx @clawrise/clawrise-cli setup codex feishu
```

平台凭证优先使用环境变量：

```bash
NOTION_INTERNAL_TOKEN=secret_xxx clawrise setup codex notion
FEISHU_APP_ID=cli_xxx FEISHU_APP_SECRET=cli_secret_xxx clawrise setup codex feishu
```

如果缺少环境变量且当前 shell 可交互，`setup` 会直接提示输入所需凭证。

环境变量在这里只是导入来源。

setup 导入凭证后，Clawrise 会把它们持久化到配置好的 secret store 中供后续正常执行使用。长期自动化不应依赖反复从 `~/.bashrc` 之类的 shell rc 文件注入。

按客户端区分的示例：

```bash
clawrise setup codex
clawrise setup claude-code notion
clawrise setup openclaw feishu
clawrise setup opencode notion
```

只初始化平台账号的示例：

```bash
clawrise setup notion
clawrise setup feishu
```

安装到项目本地 skills 目录的示例：

```bash
clawrise setup claude-code notion --skills-dir ./.claude/skills
clawrise setup openclaw feishu --skills-dir ./skills
clawrise setup opencode notion --skills-dir ./.opencode/skills
```

### 第 4 步：验证安装结果

验证 CLI：

```bash
clawrise version
clawrise doctor
clawrise spec list
```

如果 setup 里包含平台初始化，先验证当前默认选择：

```bash
clawrise auth check
```

如果你确实需要拿到 setup 生成的显式账号名，再先查看后续要引用的名字：

```bash
clawrise account list
clawrise auth presets --platform notion
clawrise auth presets --platform feishu
```

同时确认目标目录中已出现这些 skill：

- `clawrise-core`
- `clawrise-feishu`
- `clawrise-notion`

### 第 5 步：处理常见问题

如果安装失败是因为写权限不足：

- 改为安装到项目内 skills 目录

如果你在 `setup` 之外单独导入 secret：

- 把 `clawrise auth secret set ...` 理解成一次性导入到 secret store
- 优先使用 `--stdin`、真实进程环境里的 `--from-env`，或直接写入所配置的 secret backend
- 避免依赖非交互 shell 的启动文件来反复注入 secret

如果用户只需要一个平台：

- 只传必需的平台名重新执行 setup，例如：

```bash
clawrise setup claude-code feishu
```

如果用户直接在仓库里执行 `go run ./cmd/clawrise`，并以为应该存在 `setup`：

- 改用已发布的 npm 包入口，或者执行 `npx @clawrise/clawrise-cli setup ...`

### 第 6 步：重启客户端

setup 完成后，重新打开会话或重启客户端，让新安装的 skills 生效。
