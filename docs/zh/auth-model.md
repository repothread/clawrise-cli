# Clawrise 授权与 Profile 模型

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

### Profile

Profile 是用户显式选择或默认选中的具体执行身份。

示例：

- `feishu_bot_ops`
- `feishu_user_alice`
- `notion_team_docs`

### Subject

Subject 表示请求在平台侧归属于哪类主体。

推荐统一类型：

- `bot`
- `user`
- `integration`

### Grant

Grant 表示凭证是如何获得的。

推荐统一类型：

- `client_credentials`
- `oauth_user`
- `static_token`
- `oauth_refreshable`

### Session

Session 是运行时真正用于发请求的认证态。

通常包含：

- access token
- expires_at
- headers
- subject
- profile

## 3. 运行时解析流程

每次命令执行时，运行时应按以下顺序解析授权：

1. 解析 operation
2. 解析 platform
3. 解析 profile
4. 加载凭证
5. 构建或刷新 auth session
6. 校验该 session 的主体是否被 operation 允许
7. 注入认证头并执行请求

选择规则：

- 显式 `--profile` 优先级最高
- 否则使用当前平台的默认 profile
- 如果没有匹配 profile，直接失败
- 如果有多个可能的 profile 但未明确选择，返回歧义错误，而不是猜测

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

Profile 管理：

```bash
clawrise profile use feishu_bot_ops
clawrise profile current
clawrise profile list
```

单次调用覆盖：

```bash
clawrise feishu.calendar.event.create --profile feishu_bot_ops
clawrise notion.page.create --profile notion_team_docs
```

关键规则：

- `platform` 决定“去哪”
- `subject` 决定优先使用哪类主体
- `profile` 决定“以谁去”

## 5. Feishu 模型

架构层支持：

- `bot`
- `user`

MVP 实现优先支持：

- `bot`

Bot / 应用态 profile：

```yaml
profiles:
  feishu_bot_ops:
    platform: feishu
    subject: bot
    grant:
      type: client_credentials
      app_id: env:FEISHU_BOT_OPS_APP_ID
      app_secret: env:FEISHU_BOT_OPS_APP_SECRET
```

用户态 profile 预留结构：

```yaml
profiles:
  feishu_user_alice:
    platform: feishu
    subject: user
    grant:
      type: oauth_user
      client_id: env:FEISHU_CLIENT_ID
      client_secret: env:FEISHU_CLIENT_SECRET
      access_token: env:FEISHU_ALICE_ACCESS_TOKEN
      refresh_token: env:FEISHU_ALICE_REFRESH_TOKEN
```

多个 bot 通过多个 profile 表达，而不是在平台层做特殊分支。

## 6. Notion 模型

推荐主体类型：

- `integration`

internal integration 是 MVP 路径：

```yaml
profiles:
  notion_team_docs:
    platform: notion
    subject: integration
    grant:
      type: static_token
      token: env:NOTION_TEAM_DOCS_TOKEN
      notion_version: "2026-03-11"
```

public integration 是后续扩展路径：

```yaml
profiles:
  notion_public_workspace_a:
    platform: notion
    subject: integration
    grant:
      type: oauth_refreshable
      client_id: env:NOTION_CLIENT_ID
      client_secret: env:NOTION_CLIENT_SECRET
      access_token: env:NOTION_WS_A_ACCESS_TOKEN
      refresh_token: env:NOTION_WS_A_REFRESH_TOKEN
      notion_version: "2026-03-11"
```

## 7. Operation 级授权约束

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

- 如果选中的 profile 主体不在允许列表里，立即失败
- 不允许静默切换 profile
- 不允许静默把 `user` 降级为 `bot`
- 显式失败优先于隐式兜底

## 8. 身份归因与资源可见性

身份归因与资源可见性是两个独立维度，不能混为一谈。

### 8.1 身份归因

身份归因回答的是：

- 这次改动在平台侧会被记到谁头上

对飞书这类平台，通常由调用时使用的 token 类型决定：

- `tenant_access_token` 更接近 bot / app 归因
- `user_access_token` 更接近 user 归因

### 8.2 资源可见性

资源可见性回答的是：

- 哪些人能看到这份文档、表格或日程

这通常由资源本身的权限、共享关系和归属位置决定，而不是单纯由当前调用身份决定。

### 8.3 推荐策略

如果需求同时包含：

- 目标用户始终可见
- bot 改动与用户改动可区分

则推荐采用两阶段模式：

1. `clawrise subject use user` 后执行 `feishu.docs.document.create`
2. `clawrise subject use bot` 后执行 `feishu.docs.document.edit`

也就是：

- 先用用户身份创建资源，确保资源天然对用户可见
- 再给 bot 授权
- 后续 bot 用自己的身份持续编辑

### 8.4 不推荐的默认策略

不建议默认使用：

- `clawrise subject use user` 后长期把自动化编辑也放在 `feishu.docs.document.edit`

因为这样虽然可以简化资源可见性问题，但会带来归因混淆：

- bot 发起的改动可能会被平台记成用户本人改动

## 9. MVP 主体矩阵

当前 MVP 执行矩阵：

- `feishu.calendar.event.create` -> `bot`
- `feishu.calendar.event.list` -> `bot`
- `feishu.docs.document.create` -> `bot`
- `feishu.contact.user.get` -> `bot`
- `notion.page.create` -> `integration`
- `notion.page.get` -> `integration`
- `notion.block.append` -> `integration`
- `notion.user.get` -> `integration`

## 10. 安全规则

- 审计日志里绝不能记录明文密钥
- 主配置与 token 缓存要分文件存放
- 正常 CLI 输出中不能打印原始 access token
- 日志与 debug 输出中的密钥字段必须脱敏

## 11. MVP 边界

MVP 必须实现：

- 多 profile 支持
- 默认平台选择
- 默认 profile 选择
- Feishu bot/app 凭证流
- Notion internal integration token 流
- operation 级主体校验

MVP 可后置：

- Feishu 用户态浏览器登录流
- Notion public OAuth 流
- 系统密钥链集成
- 交互式授权辅助命令
