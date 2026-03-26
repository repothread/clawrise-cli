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

## 9. 当前代码状态

当前仓库中：

- `oauth_user` 这种 profile 结构已经在配置模型中预留
- 相关文档与命名策略已经确定

但请注意：

- 当前代码已经打通的是 bot/app 身份的 Feishu 日历创建
- 用户身份的真实执行链路还没有接入运行时代码

也就是说：

- 你现在可以先把用户授权凭证准备好并写入环境变量
- 但真正基于 `subject=user` 的文档执行链路还需要后续实现
