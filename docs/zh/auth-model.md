# Clawrise 授权与 Account 模型

英文版见 [../en/auth-model.md](../en/auth-model.md)。

## 1. 核心原则

Clawrise 的授权不是“拿到一个 token”这么简单，而是完整的执行身份模型。

运行时必须显式建模：

- 请求发往哪里
- 请求由谁执行
- 凭证如何获得
- 运行时认证 session 如何构建、缓存和刷新

## 2. 核心概念

### Platform

只标识上游平台。

示例：

- `feishu`
- `notion`

### Account

Account 是用户显式选择或默认选中的具体执行身份。

示例：

- `feishu_bot_ops`
- `feishu_user_alice`
- `notion_team_docs`

### Subject

Subject 表示请求在平台侧归属于哪类主体。

当前常见值：

- `bot`
- `user`
- `integration`

未来平台可以扩展新的 subject 字符串，core 不应再把它固定死。

### Auth Method

Auth method 表示这个 account 通过什么方式接入平台认证。

当前第一方方法：

- `feishu.app_credentials`
- `feishu.oauth_user`
- `notion.internal_token`
- `notion.oauth_public`

### Session

Session 是运行时真正用于发请求的认证态。

通常包含：

- access token
- refresh token
- expires_at
- token_type
- provider-native metadata

## 3. 运行时解析流程

每次命令执行时，运行时按以下顺序解析授权：

1. 解析 operation
2. 解析 platform
3. 解析 account
4. 读取 account public fields、secret refs、session
5. 调用 provider plugin 的 `auth.resolve`
6. 持久化 plugin 返回的 session / secret patch
7. 注入 provider-ready auth 并执行 operation

选择规则：

- 显式 `--account` 优先级最高
- 否则使用当前平台的默认 account
- 如果没有匹配 account，直接失败
- 如果有多个可能的 account 但未明确选择，返回歧义错误，而不是猜测

## 4. CLI 约定

平台管理：

```bash
clawrise platform use feishu
clawrise platform current
clawrise platform unset
```

主体管理：

```bash
clawrise subject use bot
clawrise subject current
clawrise subject unset
clawrise subject list
```

Account 管理：

```bash
clawrise account add --platform feishu --preset bot
clawrise account use feishu_bot_ops
clawrise account current
clawrise account list
```

单次调用覆盖：

```bash
clawrise feishu.calendar.event.create --account feishu_bot_ops
clawrise notion.page.create --account notion_team_docs
```

关键规则：

- `platform` 决定“去哪”
- `subject` 决定优先使用哪类主体
- `account` 决定“以谁去”

## 5. 配置结构

当前推荐结构：

```yaml
defaults:
  platform: feishu
  account: feishu_bot_ops
  platform_accounts:
    feishu: feishu_bot_ops
    notion: notion_team_docs

accounts:
  feishu_bot_ops:
    platform: feishu
    subject: bot
    auth:
      method: feishu.app_credentials
      public:
        app_id: cli_xxx
      secret_refs:
        app_secret: secret:feishu_bot_ops:app_secret

  notion_team_docs:
    platform: notion
    subject: integration
    auth:
      method: notion.internal_token
      public:
        notion_version: "2026-03-11"
      secret_refs:
        token: secret:notion_team_docs:token
```

核心点：

- `public` 保存非敏感字段
- `secret_refs` 只保存 secret 引用，不保存明文
- `auth.method` 的语义由 provider plugin 声明

## 6. Feishu 模型

当前架构支持：

- `bot`
- `user`

Bot / app style account：

```yaml
accounts:
  feishu_bot_ops:
    platform: feishu
    subject: bot
    auth:
      method: feishu.app_credentials
      public:
        app_id: cli_xxx
      secret_refs:
        app_secret: secret:feishu_bot_ops:app_secret
```

用户态 account：

```yaml
accounts:
  feishu_user_alice:
    platform: feishu
    subject: user
    auth:
      method: feishu.oauth_user
      public:
        client_id: cli_xxx
        redirect_mode: loopback
        scopes:
          - offline_access
      secret_refs:
        client_secret: secret:feishu_user_alice:client_secret
        refresh_token: secret:feishu_user_alice:refresh_token
```

多个 bot 通过多个 account 表达，而不是在平台层做特殊分支。

## 7. Notion 模型

当前推荐主体类型：

- `integration`

internal integration：

```yaml
accounts:
  notion_team_docs:
    platform: notion
    subject: integration
    auth:
      method: notion.internal_token
      public:
        notion_version: "2026-03-11"
      secret_refs:
        token: secret:notion_team_docs:token
```

public integration：

```yaml
accounts:
  notion_public_workspace_a:
    platform: notion
    subject: integration
    auth:
      method: notion.oauth_public
      public:
        client_id: cli_xxx
        notion_version: "2026-03-11"
        redirect_mode: loopback
      secret_refs:
        client_secret: secret:notion_public_workspace_a:client_secret
        refresh_token: secret:notion_public_workspace_a:refresh_token
```

## 8. Operation 级授权约束

每个 operation 都必须声明授权约束。

推荐结构：

```go
type AuthConstraint struct {
    AllowedSubjects  []string
    PreferredSubject string
    RequiredScopes   []string
}
```

运行时规则：

- 运行时必须显式拒绝不被允许的主体类型
- 运行时不能静默切换 account
- provider plugin 可以实现 provider-native auth 行为，但 account 解析仍应由 core runtime 负责
- 如果选中的 account 主体不在允许列表里，立即失败
- 不允许静默把 `user` 降级为 `bot`

## 9. 身份归因与资源可见性

身份归因与资源可见性是两个独立维度，不能混为一谈。

### 9.1 身份归因

身份归因回答的是：

- 这次改动在平台侧会被记到谁头上

对飞书这类平台，通常由调用时使用的 token 类型决定：

- `tenant_access_token` 更接近 bot / app 归因
- `user_access_token` 更接近 user 归因

### 9.2 资源可见性

资源可见性回答的是：

- 哪些人能看到这份文档、表格或日程

这通常由资源本身的权限、共享关系和归属位置决定，而不是单纯由当前调用身份决定。

### 9.3 推荐策略

如果需求同时包含：

- 目标用户始终可见
- bot 改动与用户改动可区分

那么应把“创建时的身份”和“后续编辑时的身份”分开建模，而不是试图用一个 account 覆盖所有目标。
