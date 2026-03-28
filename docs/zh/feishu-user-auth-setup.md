# 飞书用户授权凭证获取说明

英文版见 [../en/feishu-user-auth-setup.md](../en/feishu-user-auth-setup.md)。

## 1. 目的

这份文档说明如何为 Clawrise 准备飞书用户身份的授权凭证。

目标是获取：

- `FEISHU_CLIENT_ID`
- `FEISHU_CLIENT_SECRET`
- `FEISHU_ALICE_ACCESS_TOKEN`
- `FEISHU_ALICE_REFRESH_TOKEN`

这些值用于配置类似下面的 profile：

```yaml
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

## 2. 适用场景

只有在你确实需要“以用户身份执行”时，才应该准备这套凭证。

典型场景：

- 以用户身份创建用户可见的空文档
- 以用户身份访问用户自身拥有的资源
- 避免 bot 创建的资源天然不可见

如果你需要保留 bot 改动与用户改动的归因差异，则推荐组合使用：

1. `clawrise subject use user` 后执行 `feishu.docs.document.create`
2. `grant bot access`
3. `clawrise subject use bot` 后执行 `feishu.docs.document.edit`

## 3. 官方 OAuth 链路

飞书官方用户授权链路分成 3 步：

1. 获取授权码
2. 用授权码换 `user_access_token`
3. 后续用 `refresh_token` 刷新

官方文档：

- 获取授权码：https://open.feishu.cn/document/common-capabilities/sso/api/obtain-oauth-code
- 获取 `user_access_token`：https://open.feishu.cn/document/uAjLw4CM/ukTMukTMukTM/authentication-management/access-token/get-user-access-token
- 刷新 `user_access_token`：https://open.feishu.cn/document/uAjLw4CM/ukTMukTMukTM/authentication-management/access-token/refresh-user-access-token
- 如何选择 token 类型：https://open.feishu.cn/document/uAjLw4CM/ugTN1YjL4UTN24CO1UjN/trouble-shooting/how-to-choose-which-type-of-token-to-use

## 4. 前置准备

### 4.1 应用凭证

进入飞书开放平台应用后台，准备：

- App ID
- App Secret

对应到环境变量：

```bash
export FEISHU_CLIENT_ID=你的_App_ID
export FEISHU_CLIENT_SECRET=你的_App_Secret
```

### 4.2 配置重定向 URL

在开发者后台配置一个 OAuth 回调地址。

例如：

```text
http://localhost:3333/callback
```

飞书官方要求：

- `redirect_uri` 必须先在应用后台的安全设置中配置
- 发起授权和换 token 时使用的 `redirect_uri` 必须一致

### 4.3 申请需要的 scope

你必须先在应用后台申请需要用户授权的权限。

如果后续希望拿到 `refresh_token`，还要额外申请：

- `offline_access`

## 5. 获取授权码

### 5.1 构造授权 URL

授权地址固定为：

```text
https://accounts.feishu.cn/open-apis/authen/v1/authorize
```

最小示例：

```text
https://accounts.feishu.cn/open-apis/authen/v1/authorize?client_id=你的AppID&response_type=code&redirect_uri=http%3A%2F%2Flocalhost%3A3333%2Fcallback&scope=offline_access&state=clawrise_user_alice&prompt=consent
```

参数说明：

- `client_id`：你的 App ID
- `response_type`：固定为 `code`
- `redirect_uri`：你的回调地址，需先在后台配置
- `scope`：需要用户授权的权限列表，空格分隔
- `state`：自定义状态值，回调时原样返回
- `prompt=consent`：强制显示授权确认页，便于手动调试

### 5.2 浏览器完成授权

你需要在浏览器中打开授权 URL，登录飞书并同意授权。

授权成功后，浏览器会跳转到：

```text
http://localhost:3333/callback?code=xxxxx&state=clawrise_user_alice
```

你需要从回调 URL 中取出：

- `code`

注意：

- 授权码有效期只有 5 分钟
- 授权码只能使用一次

## 6. 用授权码换 token

请求地址：

```text
POST https://open.feishu.cn/open-apis/authen/v2/oauth/token
```

请求体：

```json
{
  "grant_type": "authorization_code",
  "client_id": "你的 App ID",
  "client_secret": "你的 App Secret",
  "code": "刚拿到的授权码",
  "redirect_uri": "http://localhost:3333/callback"
}
```

如果使用 PKCE，则还需额外传：

- `code_verifier`

成功时会返回：

- `access_token`
- `expires_in`
- `refresh_token`
- `refresh_token_expires_in`
- `scope`

其中：

- `access_token` 就是你要写入 `FEISHU_ALICE_ACCESS_TOKEN` 的值
- `refresh_token` 就是你要写入 `FEISHU_ALICE_REFRESH_TOKEN` 的值

## 7. 刷新 token

刷新地址与换 token 相同：

```text
POST https://open.feishu.cn/open-apis/authen/v2/oauth/token
```

请求体改为：

```json
{
  "grant_type": "refresh_token",
  "client_id": "你的 App ID",
  "client_secret": "你的 App Secret",
  "refresh_token": "当前 refresh_token"
}
```

注意：

- `refresh_token` 只能使用一次
- 刷新成功后会返回新的 `access_token`
- 如果带 `offline_access`，也会返回新的 `refresh_token`
- 旧 `refresh_token` 立即失效

## 8. 本地配置建议

建议把真实凭证写入 shell 环境变量，而不是直接明文写进配置文件。

例如在 `~/.zshrc` 中添加：

```bash
export FEISHU_CLIENT_ID=你的_App_ID
export FEISHU_CLIENT_SECRET=你的_App_Secret
export FEISHU_ALICE_ACCESS_TOKEN=你的_user_access_token
export FEISHU_ALICE_REFRESH_TOKEN=你的_refresh_token
```

然后重新加载：

```bash
source ~/.zshrc
```

## 9. 当前 user 支持范围

当前版本里，`oauth_user` 已经完整接入运行时。

运行时对 `subject=user` 的放开面分成两类：

- 已按官方一手资料核对并放开的接口
- 已在本地实现中切到统一 `user_access_token` / `tenant_access_token` 解析路径并显式放开的接口

其中，官方核对仍优先参考飞书官方 Go SDK `github.com/larksuite/oapi-sdk-go` 的生成代码，版本为 `v1.1.48`。  
这些生成代码会在不少接口旁显式声明允许的 token 类型，例如：

- `request.AccessTokenTypeUser`
- `request.AccessTokenTypeTenant`

对于当前已经在本地实现中显式放开的接口，是否能真正成功调用，仍取决于：

- 飞书应用是否申请了对应权限
- 当前用户是否完成授权并拿到可用的 `user_access_token`
- 当前用户是否对目标知识库空间或节点具备可见性与编辑权限

### 9.1 当前允许 `subject=user` 的 operation

日历：

- `feishu.calendar.calendar.list`
- `feishu.calendar.event.create`
- `feishu.calendar.event.list`
- `feishu.calendar.event.get`
- `feishu.calendar.event.update`
- `feishu.calendar.event.delete`

新版文档 Docx：

- `feishu.docs.document.create`
- `feishu.docs.document.get`
- `feishu.docs.document.list_blocks`
- `feishu.docs.document.append_blocks`
- `feishu.docs.document.edit`
- `feishu.docs.document.get_raw_content`
- `feishu.docs.document.share`
- `feishu.docs.block.get`
- `feishu.docs.block.list_children`
- `feishu.docs.block.update`
- `feishu.docs.block.batch_delete`

知识库 Wiki：

- `feishu.wiki.space.list`
- `feishu.wiki.node.list`
- `feishu.wiki.node.create`

通讯录：

- `feishu.contact.user.get`
- `feishu.contact.user.search`
- `feishu.contact.department.list`
- `feishu.department.user.list`

补充说明：

- `feishu.contact.department.list` 当前实现使用 `GET /open-apis/contact/v3/departments`
- `feishu.department.user.list` 当前实现使用 `GET /open-apis/contact/v3/users` 并带 `department_id`
- 之所以不用旧实现中的其他路径，是因为当前版本只保留已在官方 SDK 中明确暴露并标注 token 类型的接口

多维表格：

- `feishu.bitable.table.list`
- `feishu.bitable.field.list`
- `feishu.bitable.record.list`
- `feishu.bitable.record.get`
- `feishu.bitable.record.create`
- `feishu.bitable.record.batch_create`
- `feishu.bitable.record.update`
- `feishu.bitable.record.batch_update`
- `feishu.bitable.record.delete`
- `feishu.bitable.record.batch_delete`

### 9.2 当前仍保持 `bot` only 的 operation

以下 operation 在当前版本中仍然只允许 `subject=bot`：

- `feishu.docs.block.get_descendants`

保留为 `bot` only 的原因：

- `feishu.docs.block.get_descendants` 在当前实现里依赖 `with_descendants=true` 这一扩展参数，但在当前核对版本的官方 SDK 中没有找到对应声明，因此暂不放开给 `user`

### 9.3 官方核对链接

以下链接是当前版本优先采用的官方核对依据：

- 日历：<https://github.com/larksuite/oapi-sdk-go/blob/v1.1.48/service/calendar/v4/api.go>
- Docx：<https://github.com/larksuite/oapi-sdk-go/blob/v1.1.48/service/docx/v1/api.go>
- 多维表格：<https://github.com/larksuite/oapi-sdk-go/blob/v1.1.48/service/bitable/v1/api.go>
- 通讯录：<https://github.com/larksuite/oapi-sdk-go/blob/v1.1.48/service/contact/v3/api.go>
- 文档权限：<https://github.com/larksuite/oapi-sdk-go/blob/v1.1.48/service/drive/v1/api.go>

## 10. 当前代码状态

当前仓库中：

- `oauth_user` 这种 profile 结构已经接入运行时
- `wiki` 相关 operation 已按当前实现放开给 `user`
- 调用时仍需以 operation 级主体约束为准，运行时不会自动把 `user` 降级为 `bot`

这意味着：

- 你现在可以准备好用户授权凭证并直接用于上述已放开 operation
- 如果某个 operation 仍是 `bot only`，应优先视为当前实现还没有把它安全地放开
