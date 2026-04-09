# Clawrise 插件作者规范

英文版见 [../en/plugin-authoring-spec.md](../en/plugin-authoring-spec.md)。

> 当前仓库对外发布给插件作者的 manifest、协议面、分发建议与兼容性边界，以本文为准。若本文与 [plugin-system-design.md](./plugin-system-design.md) 的历史设计表述存在差异，应优先遵循本文。

## 1. 目标

本文定义第三方插件仓库接入 Clawrise 的公开标准，目标是：

- 让第三方仓库可以独立开发、独立发布、独立升级
- 让 core 与 plugin 之间只有协议耦合，没有代码耦合
- 让 `doctor`、`spec`、`auth`、runtime、安装器共享统一事实源
- 让 Go、Node.js 或其他语言都可以实现同一套插件协议

## 2. 规范边界

Clawrise 的公开标准由以下 5 部分组成：

1. 插件目录布局
2. `plugin.json` manifest v2
3. `stdio + JSON-RPC 2.0` protocol v1
4. 插件安装源与归档建议
5. 黑盒兼容性检查路径

以下内容不属于公开标准：

- 插件仓库 import `github.com/clawrise/clawrise-cli/internal/...`
- 插件读取 core 主配置文件
- 插件自己解释 `env:`、默认账号、默认平台、secret store 选择逻辑
- 插件与 core 共享 Go 内部类型

## 3. 零耦合边界

第三方插件仓库必须遵循以下边界：

- plugin 只能依赖协议，不得依赖本仓库 `internal/...` 包
- core 只负责解析配置、解析 secret、选择 account、发起协议调用
- plugin 只消费已经解析好的 `account`、`execution_auth` 与 `input`
- plugin 不需要知道配置文件路径、secret store 实现、默认平台规则

推荐仓库关系：

- 本仓库负责规范、安装器、发现、路由、验证
- 独立插件仓库负责 provider API 映射、鉴权细节、真实执行
- 可选 SDK 应放到独立仓库，且不能成为接入前提

## 4. 运行时目录布局

运行时的标准对象不是压缩包，而是“解压后的插件目录”。

推荐布局：

```text
clawrise-plugin-<name>/
  plugin.json
  bin/
    clawrise-plugin-<name>
  README.md
  LICENSE
```

本地开发时，discovery root 可以直接指向插件目录或其父目录。core 会递归发现 `plugin.json`。

默认 discovery 顺序：

1. `CLAWRISE_PLUGIN_PATHS`
2. `.clawrise/plugins`
3. `~/.clawrise/plugins`

## 5. Manifest V2

第三方插件应优先使用 `schema_version: 2`。

最小 provider manifest 示例：

```json
{
  "schema_version": 2,
  "name": "linear",
  "version": "0.1.0",
  "protocol_version": 1,
  "min_core_version": "0.1.0",
  "capabilities": [
    {
      "type": "provider",
      "platforms": ["linear"]
    }
  ],
  "entry": {
    "type": "binary",
    "command": ["./bin/clawrise-plugin-linear"]
  }
}
```

字段要求：

- `schema_version`：当前公开作者入口应使用 `2`
- `name`：插件名，必须稳定；provider binding 依赖它
- `version`：插件版本
- `protocol_version`：当前为 `1`
- `min_core_version`：建议显式声明，便于安装器前置兼容性检查
- `capabilities`：能力列表；provider 插件至少要声明一个 `provider`
- `entry.type`：当前必须为 `binary`
- `entry.command`：插件启动命令；相对路径相对于 `plugin.json` 所在目录解析

对 provider capability：

- `type` 必须为 `provider`
- `platforms` 必须至少声明一个平台名

## 6. Protocol V1

传输层要求：

- transport：`stdio`
- 编码：单行 `JSON-RPC 2.0`
- `stdout`：只能输出协议消息
- `stderr`：只能输出日志

### 6.1 当前对外协议面

当前实现下，provider 插件的协议面应按下面的能力矩阵实现。

所有插件必需：

- `clawrise.handshake`
- `clawrise.capabilities.list`
- `clawrise.health`

所有 provider 必需：

- `clawrise.operations.list`
- `clawrise.catalog.get`
- `clawrise.execute`

所有带鉴权的 provider 必需：

- `clawrise.auth.methods.list`
- `clawrise.auth.inspect`
- `clawrise.auth.resolve`

带账号预设能力的 provider 推荐实现：

- `clawrise.auth.presets.list`

带交互式登录的 provider 必需：

- `clawrise.auth.begin`
- `clawrise.auth.complete`

### 6.2 协议语义约束

- `handshake` 用于暴露插件名、版本、平台
- `capabilities.list` 用于暴露 capability 路由事实
- `operations.list` 返回 runtime 可执行 operation 元数据
- `catalog.get` 返回结构化 catalog；当前最小条目只需包含 `operation`
- `auth.inspect` 只检查账号配置是否可用，不应执行真实业务请求
- `auth.resolve` 负责把账号材料转成执行期所需的 `execution_auth`
- `execute` 只关心 provider 执行；不要自行解析 Clawrise 主配置

### 6.3 账号与执行边界

core 发送给 plugin 的 `account` 是一个已解析结构，包含：

- `name`
- `platform`
- `subject`
- `auth_method`
- `public`
- `secrets`
- `session`

provider 在 `auth.resolve` 阶段应把执行所需的最终信息收敛到 `execution_auth`，例如：

- API base URL
- access token / API key
- provider version header
- tenant / workspace 上下文

后续 `execute` 不应再依赖 core 配置文件或 secret store。

## 7. 分发与安装

### 7.1 设计结论

Clawrise 运行时标准是“插件目录 + manifest + 协议”，不是某一种单独的压缩格式。

也就是说：

- `tar.gz` 不是唯一标准
- `zip` 也可以作为归档载体
- `npm`、`https`、`registry` 都只是安装源
- 安装完成后，runtime 面对的始终是插件目录树

### 7.2 当前安装源

当前 core 已支持以下安装源：

- 本地目录
- `file://`
- `https://`
- 直接 npm 包名
- `npm://`
- `registry://`

### 7.3 推荐顺序

推荐的第三方分发顺序：

1. `registry://`
2. `https://`
3. `npm://`

原因：

- `registry://` 适合做逻辑插件名到平台产物的解析层
- `https://` 最通用，适合二进制归档发布
- `npm://` 对 Node.js 插件更友好，但不应成为唯一分发路径

### 7.4 归档建议

对于二进制插件，推荐默认发布 `.tar.gz`：

- 与语言无关
- 便于带上二进制、README、manifest、资源文件
- 便于做 checksum、镜像与离线分发

如需兼容更多客户端，也可以额外发布 `.zip`。

## 8. 兼容性检查

第三方插件仓库至少应验证以下链路：

1. `clawrise doctor`
2. `clawrise auth methods --platform <platform>`
3. `clawrise spec list <platform>`
4. 至少一个真实 operation 的 `--dry-run` 或真实执行

当前仓库内已经提供一个本地黑盒验证脚本：

```bash
./scripts/plugin/verify-external-provider.sh /abs/path/to/plugin/root linear linear.viewer.get
```

这个脚本只依赖发现、协议与 CLI 行为，不依赖插件仓库使用什么语言实现。
它面向开发态 discovery 验证，适合检查“当前插件目录能否被 core 直接发现并握手”。

针对生产安装链路，当前仓库还提供了安装态黑盒验证脚本：

```bash
./scripts/plugin/verify-external-provider-install.sh \
  file:///abs/path/to/clawrise-plugin-linear-0.1.0-darwin-arm64.tar.gz \
  linear \
  0.1.0 \
  linear \
  linear.viewer.get
```

这个脚本会验证：

1. `plugin install`
2. `plugin info`
3. `plugin verify`
4. `doctor`
5. `auth methods`
6. `spec list/get`

生产环境建议：

- 不要直接把源码仓库目录作为正式安装源
- 应发布带版本号的 `.tar.gz` 归档，并通过 `file://`、`https://` 或 `registry://` 安装
- 归档发布时应同时提供稳定版本号与 SHA256 checksum，便于制品仓库、镜像与变更审计

## 9. 语言建议

官方建议同时维护 Go 与 Node.js 两种模板，但定位不同：

- Go：默认主模板
- Node.js：快速开发模板

推荐 Go 作为长期维护的 provider 插件主实现，原因：

- 单二进制分发更稳定
- 不要求宿主额外安装 Node.js
- 更适合通过 GitHub Release / `https://` / `registry://` 分发

推荐 Node.js 作为快速验证模板，适合：

- 快速原型
- 工作流类插件
- 强依赖 npm 生态的插件

## 10. 参考实现

参考实现应维护在独立仓库中，而不是当前规范仓库中。

当前计划中的首个参考实现仓库名为：

- `clawrise-plugin-linear`

参考实现应满足以下特点：

- 不 import 本仓库任何 Go 包
- 只通过 `plugin.json + stdio JSON-RPC` 接入
- 默认使用 `Linear API key` 鉴权
- 可以独立发布到 GitHub Releases 或其他分发渠道
