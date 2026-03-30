# Clawrise V2 架构与授权重构设计

## 0. 当前实现状态

截至当前仓库状态，这份设计中的内容并未全部开发完成。

已经完成：

- `account` 作为外部主语义已经落地
- `accounts` 配置模型已经落地
- `internal/account` 已经开始承接通用账号选择逻辑
- `internal/metadata` 已经开始承接统一 metadata 聚合与 playbook 校验逻辑
- `auth methods / presets / inspect / login / complete / logout` 已落地
- plugin auth 协议已扩展并接入 Feishu / Notion
- 交互式授权执行器已抽成独立 auth launcher runtime
- 运行时执行前的 `auth.resolve` 已落地
- execute 身份协议已经收敛到 `auth.method + execution_auth` 结构，并保留旧结构兼容解析
- `secret put --from-env` 与显式不安全写入开关已落地
- operation 解析已经改为基于运行时平台集合
- 第一方 plugin 的 catalog 已改为从 registry 动态生成
- core execute 路径已经改为只传递 `account + auth.method + execution_auth`
- provider runtime / process runtime 已不再在 core 中拼接 provider 专有 auth 字段
- Feishu / Notion adapter 已改为在 provider 内部维护各自的 execution profile
- `config init` 已改为基于 plugin metadata 选择 preset / auth method
- doctor 与 `auth inspect` 的账号诊断输出已收敛到稳定扁平字段
- `internal/spec/catalog` 的内置平台清单已删除
- `PathsConfig` 已开始通过 `locator` 模块参与状态路径解析
- playbook 索引校验已接入统一 metadata，并在 `doctor` 中暴露结果
- `auth login` 已开始基于 auth method descriptor 自动按 `device_code -> local_browser -> manual_code -> manual_url` 选择默认交互模式
- `secret/session/authflow/governance` 四类状态存储都已具备 backend 注册点
- `locator` 已补齐路径解析来源输出，`doctor` 会直接暴露 config/state/runtime 的实际生效路径与来源
- Feishu / Notion 当前已按各自公开 OAuth 能力接入交互式授权，第一方 provider 暂不把 `device_code` 作为近期交付项

部分完成：

- 文档和内部遗留命名仍在继续清理
- `subject` 的外部硬编码限制已移除，但内部仍保留少量 config inspection / legacy shim 类型名
- `device_code` 的 core 协议、flow 持久化与 CLI 主流程已落地，但第一方 provider 是否接入该模式将按未来 provider 公开能力与真实需求再决定
- secret/session/authflow/governance 已具备可切换与注册扩展的 backend 接口，但外部可分发 backend/plugin 形态仍可继续扩展
- docs 自动生成已经可以复用统一 metadata 导出 Markdown，但独立的 docs 生成流水线仍可继续收敛
- `profile` / `connection` / `account` 的内部收敛已经不再阻塞 core execute 路径，但 `config` 包里仍保留少量 legacy inspection shim
- `PathsConfig` 已进入定案后的兼容收敛阶段：`config_dir` 视为废弃，`state_dir` 仅作兼容别名保留，推荐统一改用 `locator + env`

尚未完成：

- storage backend 的外部分发 / 安装 / 协议化形态还没有像 provider plugin 一样完全独立
- `PathsConfig` 的兼容收敛还未完全结束，后续仍需决定 `state_dir` 的最终移除窗口

### 0.1 按 Phase 粗略进度

- Phase 1：已完成
- Phase 2：已完成
- Phase 3：已完成；`device_code` 保留为可选通用模式，未来按 provider 能力再接入
- Phase 4：已完成，metadata/playbook/locator/doctor 已收敛到统一事实源与统一路径解析
- Phase 5：已基本完成，Feishu / Notion 已迁移到新协议并按当前公开能力完成边界验证

## 1. 目标

这份设计以“没有存量兼容负担”为前提，直接给出 Clawrise 的目标形态。

目标：

- 支持更多平台和更多授权方式时，core 不再持续膨胀
- 授权流程默认安全，同时尽量短，降低接入门槛
- 同时适合 AI 自动执行和人类终端用户使用
- `spec`、completion、doctor、playbooks、runtime 继续收敛到统一事实源
- 平台能力继续通过 plugin 扩展，而不是重新回到 core 内硬编码

## 2. 当前主要问题

当前实现已经具备不错的 provider runtime 抽象，但仍有几类结构性边界还没有完全收好：

- storage backend 已经具备可切换与注册扩展的接口，但外部分发与独立协议仍未完全落地
- `profile` / `connection` / `account` 的内部遗留命名仍在继续清理，但已不再阻塞 core execute 边界
- `subject` 的外部限制已经移除，但 config inspection 里仍保留少量 legacy bridge 结构
- playbook 校验已经接到统一 metadata，但 docs 自动生成流水线仍可继续收敛
- `PathsConfig` 已定为兼容态收敛：`config_dir` 不再作为有效入口，`state_dir` 仅保留兼容语义，但最终下线节奏仍需明确

这些问题不是单点 bug，而是边界没有完全收好。

## 3. 设计原则

### 3.1 core 只负责通用编排

core 只负责：

- 账号选择
- secret/session 存储
- 通用执行编排
- 统一 JSON 输出
- 幂等、重试、超时、审计
- plugin 发现与 RPC 编排

core 不直接理解某个平台的 OAuth 端点、token 刷新字段、scope 语义、provider 专有认证参数。

### 3.2 plugin 负责平台语义

plugin 负责：

- operation 元数据
- provider 请求映射
- provider 认证细节
- provider 错误映射
- provider 级账号模板与授权方法描述

### 3.3 安全默认优先

默认策略：

- 长期 secret 不进配置文件明文
- access token 只进 session store
- refresh token 只进 secret store
- CLI 默认不鼓励把 secret 直接写进命令参数
- 授权流程优先使用最小权限和短生命周期状态

### 3.4 AI 与人类双模式一致

同一套能力同时支持：

- 人类用户的引导式流程
- AI / OpenClaw 之类 agent 的非交互 JSON 流程

差异只体现在交互方式，不体现在底层模型和协议。

### 3.5 单一事实源

operation、账号模板、授权方式、spec、completion、doctor 依赖同一份 plugin 元数据，不再让 core 自己维护一套平台清单。

## 4. 统一命名

V2 建议统一对外使用 `account`，不再同时保留 `profile` 和 `connection`。

原因：

- 对人类更直观
- 对 AI 也更清晰
- “一个账号代表一个可执行身份”比 profile / connection 双概念更稳定

统一后：

- `account` = 一个可执行身份实例
- `subject` = 该身份在平台侧的主体类型
- `auth_method` = 该账号使用的认证接入方式
- `session` = 运行时短期认证态

## 5. 目标模块划分

建议把现有模块收敛为下面几层：

### 5.1 `internal/account`

负责：

- 账号配置模型
- 账号选择
- 默认账号解析
- 账号增删改查

### 5.2 `internal/auth`

负责：

- secret store 访问
- session store 访问
- 通用授权流程编排
- auth 状态检查
- 和 plugin auth RPC 交互

### 5.3 `internal/runtime`

负责：

- operation 解析
- 执行编排
- 幂等、重试、超时、审计
- 统一 envelope

### 5.4 `internal/metadata`

负责：

- 聚合 plugin 暴露的 operation metadata
- 为 `spec`、completion、doctor、playbooks 提供统一读取接口

### 5.5 `internal/plugin`

负责：

- plugin 发现
- manifest
- stdio JSON-RPC
- capability 协商

### 5.6 `plugins/<platform>`

负责：

- 平台 operation
- 平台 auth method
- 平台 account preset
- provider-native 映射

## 6. 新的账号配置模型

V2 中 core 不再定义 Feishu / Notion 专用字段，而是只保存通用壳和 plugin 提供的公开字段 / secret 引用。

建议配置结构：

```yaml
defaults:
  platform_accounts:
    feishu: feishu_bot_main
    notion: notion_workspace_main

accounts:
  feishu_bot_main:
    title: 飞书主 Bot
    platform: feishu
    subject: bot
    auth:
      method: feishu.bot_app
      public:
        app_id: cli_xxx
      secret_refs:
        app_secret: secret:feishu_bot_main:app_secret

  notion_workspace_main:
    title: Notion 团队空间
    platform: notion
    subject: integration
    auth:
      method: notion.internal_token
      public:
        notion_version: "2026-03-11"
      secret_refs:
        token: secret:notion_workspace_main:token
```

核心点：

- `public` 是平台公开字段，core 不解释字段语义
- `secret_refs` 只保存 secret 引用，不保存明文
- `auth.method` 的具体含义由 plugin 声明

## 7. 新的 plugin auth 能力模型

为彻底解耦授权，plugin 需要额外暴露 auth 相关元数据和 RPC。

### 7.1 plugin 需要声明的 auth method descriptor

建议新增：

```json
{
  "id": "feishu.bot_app",
  "platform": "feishu",
  "display_name": "飞书 Bot 应用凭证",
  "subjects": ["bot"],
  "kind": "machine",
  "interactive": false,
  "public_fields": [
    { "name": "app_id", "required": true }
  ],
  "secret_fields": [
    { "name": "app_secret", "required": true }
  ]
}
```

对于交互式 OAuth（以下示例使用当前第一方 provider 已公开支持的模式集合）：

```json
{
  "id": "notion.public_oauth",
  "platform": "notion",
  "display_name": "Notion Public OAuth",
  "subjects": ["integration"],
  "kind": "interactive",
  "interactive": true,
  "interactive_modes": ["local_browser", "manual_code"],
  "public_fields": [
    { "name": "client_id", "required": true },
    { "name": "redirect_mode", "required": false },
    { "name": "scopes", "required": false }
  ],
  "secret_fields": [
    { "name": "client_secret", "required": true }
  ]
}
```

### 7.2 plugin 需要新增的 auth RPC

建议在现有协议上新增：

1. `clawrise.auth.methods.list`
2. `clawrise.auth.presets.list`
3. `clawrise.auth.begin`
4. `clawrise.auth.complete`
5. `clawrise.auth.resolve`
6. `clawrise.auth.inspect`

含义如下：

- `auth.methods.list`
  - 返回当前平台支持的授权方式描述
- `auth.presets.list`
  - 返回面向用户的账号模板，例如“飞书 Bot”“飞书用户态”“Notion internal token”
- `auth.begin`
  - 启动交互式授权，返回授权 URL、device code（如该模式受支持）、下一步动作等
- `auth.complete`
  - 用 callback URL、code，或在 provider 支持时使用 `device_code` 轮询结果完成授权
- `auth.resolve`
  - 在执行前把账号配置、secret、session 解析成“可执行认证上下文”
- `auth.inspect`
  - 检查账号配置是否完整、session 是否可用、下一步该做什么

### 7.3 `auth.resolve` 的关键意义

这是 V2 的核心边界。

流程变成：

1. core 选中账号
2. core 加载 `public`、`secret_refs` 对应的 secret 值、以及 session
3. core 调用 plugin `auth.resolve`
4. plugin 返回：
   - 当前是否可执行
   - 是否需要用户授权
   - 是否需要刷新 session
   - 一个 `execution_auth` 对象
   - 一个可选的 `session_patch`
   - 一个可选的 `secret_patch`
5. core 持久化 patch
6. core 调用 `execute`，把 `execution_auth` 透传给 plugin

这样 core 不需要知道：

- token 叫 access_token 还是 tenant_access_token
- refresh token 是否会轮换
- 哪些字段必须在刷新时参与签名
- 哪个平台支持 device code

## 8. 新的 execute 身份协议

当前 `identity.auth` 还是按固定字段拼接，不够通用。

V2 建议改成：

```json
{
  "platform": "feishu",
  "subject": "bot",
  "account_name": "feishu_bot_main",
  "auth": {
    "method": "feishu.bot_app",
    "execution_auth": {
      "...": "..."
    }
  }
}
```

关键变化：

- core 不再硬编码 `app_id` / `token` / `client_secret`
- `execution_auth` 是 plugin 自己生成、自己消费的结构
- core 只保证安全传输与脱敏，不解释字段语义

## 9. 面向用户的授权流程设计

V2 的授权方案必须同时满足“更安全”和“更简单”。

### 9.1 人类用户推荐流程

推荐命令：

```bash
clawrise account add --platform feishu --preset bot --wizard
clawrise account use feishu_bot_main
clawrise auth inspect feishu_bot_main
```

`--wizard` 行为：

- 根据 plugin 的 preset 自动创建账号骨架
- 提示填写公开字段
- secret 使用无回显输入或 stdin
- 若是交互式授权，自动选择最优模式并给出下一步
- 最后输出结构化 JSON 结果

### 9.2 AI / OpenClaw 推荐流程

推荐命令：

```bash
clawrise account add --platform notion --preset internal_token --output json
printf '%s' \"$NOTION_TOKEN\" | clawrise secret put notion_workspace_main token --stdin
clawrise auth inspect notion_workspace_main
```

对 AI 友好的要求：

- 所有命令都支持纯 JSON 输出
- 不依赖交互式 prompt 才能完成核心流程
- 授权命令始终返回 `next_actions`
- 需要人类协助时，明确返回 `human_required=true`
- 返回 machine-readable 的缺失字段、建议模式、推荐步骤

### 9.3 交互式 OAuth 的模式优先级

对于同时支持多种交互模式的 provider，为了兼顾安全和低门槛，建议模式优先级为：

1. `device_code`
2. `local_browser + loopback`
3. `manual_code`

原因：

- `device_code` 最适合 AI + 人类协作，不需要本机回调和剪贴 URL
- `local_browser` 对纯人类终端最顺手
- `manual_code` 作为兜底

当前第一方 provider（Feishu / Notion）公开能力里并没有 `device_code`，因此现阶段仍以 `local_browser` / `manual_code` 为主；若未来有 provider 公开支持该模式，再按上述优先级接入即可。

### 9.4 secret 输入策略

默认允许的 secret 输入方式：

- `--stdin`
- 无回显交互输入
- `--from-env ENV_NAME`
- `secret://` / `env:` 引用写入配置

默认不推荐：

- `--value xxx`

建议把它改成显式不安全模式，例如：

```bash
clawrise secret put foo token --value xxx --allow-insecure-cli-secret
```

原因：

- 避免 shell history 泄漏
- 降低 AI 直接把 secret 拼进命令行的概率

## 10. doctor / inspect 的目标输出

`doctor` 和 `auth inspect` 需要变成真正的引导层，而不是简单校验器。

输出中必须稳定包含：

- `status`
- `missing_public_fields`
- `missing_secret_fields`
- `session_status`
- `human_required`
- `recommended_action`
- `next_actions`

示例：

```json
{
  "ok": false,
  "data": {
    "account": "notion_workspace_main",
    "status": "authorization_required",
    "human_required": true,
    "recommended_action": "auth.login",
    "next_actions": [
      {
        "type": "device_code",
        "message": "请在浏览器中输入用户码完成授权"
      }
    ]
  }
}
```

这对 AI 和人类都更友好。

## 11. 除授权外，还需要一起解耦的模块

### 11.1 operation 解析

当前 core 还在硬编码已知平台名。

V2 应改成：

- 基于已加载 plugin 的 platform 集合解析
- 或者只按 `<first>.<rest>` 语法解析，再由 registry 决定是否存在

这样新增平台时不需要改 core。

### 11.2 `subject` 枚举

当前 core 固定只有 `bot` / `user` / `integration`。

V2 建议：

- core 只要求 subject 是非空字符串
- subject 推荐值由 plugin preset 和 operation metadata 提供
- operation 仍通过 `allowed_subjects` 做显式约束

这样将来接入 GitHub App、Google service account、Slack installation 等模型时不需要先改 core 枚举。

### 11.3 metadata 单一事实源

当前 `internal/spec/catalog` 仍保留 core 内静态 operation 列表。

V2 建议：

- catalog 从 plugin metadata 聚合而来
- `spec list/get/status/export`
- completion
- docs 生成
- playbooks 索引校验

都基于同一份 metadata 层

core 不再手写 Feishu / Notion operation 清单。

### 11.4 配置路径与状态路径

当前这部分已经定案：

- 路径解析统一收敛到 `locator` 模块，并以环境变量和系统默认目录作为主入口
- `config.paths.config_dir` 视为废弃字段，不再参与配置文件发现
- `config.paths.state_dir` 仅作为兼容别名保留，推荐改用 `CLAWRISE_STATE_DIR` / `CLAWRISE_STATE_HOME`

这样可以避免继续保留“模型里有，运行时不一致”或“发现 config 还要先读取 config 自己的位置配置”这类自相矛盾的状态。

### 11.5 `profile` / `connection` / `account` 术语收敛

V2 建议：

- CLI 对外只保留 `account`
- 内部结构也同步统一
- 删除 `--profile` 和 `--connection` 双入口
- 仅保留 `--account`

### 11.6 plugin 协议中的 auth 透传方式

当前 process runtime 在 core 中做了 provider 字段拼接。

V2 应改成：

- core 只传 `method + execution_auth`
- plugin 自己定义 `execution_auth` 的结构

这一步和 auth 解耦必须一起改。

## 12. 推荐的新 CLI 面

建议统一成下面这组命令：

```bash
clawrise account list
clawrise account add
clawrise account inspect <account>
clawrise account use <account>
clawrise account remove <account>

clawrise secret put <account> <field> --stdin
clawrise secret delete <account> <field>

clawrise auth methods [--platform <name>]
clawrise auth presets [--platform <name>]
clawrise auth inspect <account>
clawrise auth login <account>
clawrise auth complete <flow_id>
clawrise auth logout <account>

clawrise doctor
clawrise spec list
clawrise spec get <operation>
```

其中：

- `account add` 负责创建账号骨架
- `secret put` 负责长期 secret
- `auth login` 负责交互式授权
- `auth inspect` 负责诊断和给下一步建议

## 13. 推荐的迁移步骤

因为没有存量兼容负担，可以直接按最终边界推进：

### Phase 1

- 统一命名为 `account`
- 移除 core 中 `profile/connection` 双概念
- 删除 `subject` / `platform` 硬编码枚举
- 删除 `internal/spec/catalog` 的内置平台清单

### Phase 2

- 重构配置模型为 `accounts + auth.public + auth.secret_refs`
- 删除 core 中 Feishu / Notion 专有授权字段
- 补齐新的 account store 和 selector

### Phase 3

- 扩展 plugin 协议，加入 auth methods / presets / resolve / begin / complete / inspect
- 把 `cli/auth_*` 里的平台 switch 下沉到 plugin

### Phase 4

- 重构 `doctor`、completion、spec，使其只依赖统一 metadata
- 接入 playbook 校验，确保 playbook 指向真实 operation

### Phase 5

- 为 Feishu / Notion plugin 迁移到新协议
- 用它们验证多平台 + 多 auth method 的边界是否稳定

## 14. 最终结论

V2 的核心结论是：

- operation 扩展继续走 plugin registry
- auth 扩展也必须走 plugin capability，而不是继续留在 core switch
- core 不再保存平台专有 auth 字段模型
- 用户侧统一成 `account` 语义
- 人类和 AI 共用同一套底层流程，只是入口模式不同
- 授权默认走 secret store + session store，拒绝把不安全方式做成默认主路径

如果按这个方向重构，Clawrise 后续扩平台时，core 主要增加的是通用能力，而不是继续给每个平台补 if/switch。
