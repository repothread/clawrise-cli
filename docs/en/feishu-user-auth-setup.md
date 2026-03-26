# Feishu User Auth Setup

See the Chinese version at [../zh/feishu-user-auth-setup.md](../zh/feishu-user-auth-setup.md).

## 1. Purpose

This document explains how to prepare Feishu user-identity credentials for Clawrise.

The goal is to obtain:

- `FEISHU_CLIENT_ID`
- `FEISHU_CLIENT_SECRET`
- `FEISHU_ALICE_ACCESS_TOKEN`
- `FEISHU_ALICE_REFRESH_TOKEN`

These values are used by a profile like:

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

## 2. When to Use This

Only prepare this credential flow when you actually need user-identity execution.

Typical cases:

- creating an empty document under user identity
- accessing user-owned resources
- avoiding the visibility limitations of bot-owned resources

If you also want bot attribution to remain distinct from user attribution, the recommended pattern is:

1. run `clawrise subject use user` and call `feishu.docs.document.create`
2. `grant bot access`
3. run `clawrise subject use bot` and call `feishu.docs.document.edit`

## 3. Official OAuth Flow

Feishu's official user authorization flow has 3 steps:

1. obtain an authorization code
2. exchange the code for `user_access_token`
3. refresh later with `refresh_token`

Official docs:

- Obtain OAuth code: https://open.feishu.cn/document/common-capabilities/sso/api/obtain-oauth-code
- Get `user_access_token`: https://open.feishu.cn/document/uAjLw4CM/ukTMukTMukTM/authentication-management/access-token/get-user-access-token
- Refresh `user_access_token`: https://open.feishu.cn/document/uAjLw4CM/ukTMukTMukTM/authentication-management/access-token/refresh-user-access-token
- Choose token type: https://open.feishu.cn/document/uAjLw4CM/ugTN1YjL4UTN24CO1UjN/trouble-shooting/how-to-choose-which-type-of-token-to-use

## 4. Prerequisites

### 4.1 App Credentials

In the Feishu developer console, prepare:

- App ID
- App Secret

Mapped environment variables:

```bash
export FEISHU_CLIENT_ID=your_app_id
export FEISHU_CLIENT_SECRET=your_app_secret
```

### 4.2 Redirect URI

Configure an OAuth callback URL in the app's security settings.

Example:

```text
http://localhost:3333/callback
```

Feishu requires:

- the `redirect_uri` must be registered first
- the `redirect_uri` used during authorization and token exchange must match

### 4.3 Required Scopes

You must request the user-facing scopes you need in the app console first.

If you want `refresh_token`, also request:

- `offline_access`

## 5. Obtain the Authorization Code

### 5.1 Build the Authorization URL

Authorization endpoint:

```text
https://accounts.feishu.cn/open-apis/authen/v1/authorize
```

Minimal example:

```text
https://accounts.feishu.cn/open-apis/authen/v1/authorize?client_id=YOUR_APP_ID&response_type=code&redirect_uri=http%3A%2F%2Flocalhost%3A3333%2Fcallback&scope=offline_access&state=clawrise_user_alice&prompt=consent
```

Parameter meanings:

- `client_id`: your App ID
- `response_type`: fixed to `code`
- `redirect_uri`: your registered callback URL
- `scope`: space-delimited scopes
- `state`: custom state value returned unchanged
- `prompt=consent`: useful for explicit manual testing

### 5.2 Complete Authorization in Browser

Open the authorization URL in a browser, sign in to Feishu, and approve the request.

After success, the browser is redirected to:

```text
http://localhost:3333/callback?code=xxxxx&state=clawrise_user_alice
```

Extract:

- `code`

Notes:

- the authorization code is valid for only 5 minutes
- the authorization code can be used only once

## 6. Exchange the Code for Tokens

Request:

```text
POST https://open.feishu.cn/open-apis/authen/v2/oauth/token
```

Request body:

```json
{
  "grant_type": "authorization_code",
  "client_id": "your App ID",
  "client_secret": "your App Secret",
  "code": "the authorization code",
  "redirect_uri": "http://localhost:3333/callback"
}
```

If using PKCE, also send:

- `code_verifier`

Success response includes:

- `access_token`
- `expires_in`
- `refresh_token`
- `refresh_token_expires_in`
- `scope`

Where:

- `access_token` should be stored as `FEISHU_ALICE_ACCESS_TOKEN`
- `refresh_token` should be stored as `FEISHU_ALICE_REFRESH_TOKEN`

## 7. Refresh the Token

The refresh endpoint is the same:

```text
POST https://open.feishu.cn/open-apis/authen/v2/oauth/token
```

Request body:

```json
{
  "grant_type": "refresh_token",
  "client_id": "your App ID",
  "client_secret": "your App Secret",
  "refresh_token": "the current refresh token"
}
```

Notes:

- a `refresh_token` can be used only once
- a successful refresh returns a new `access_token`
- with `offline_access`, it also returns a new `refresh_token`
- the old `refresh_token` becomes invalid immediately

## 8. Recommended Local Config

Store the real values in shell environment variables instead of writing secrets directly into config files.

Example for `~/.zshrc`:

```bash
export FEISHU_CLIENT_ID=your_app_id
export FEISHU_CLIENT_SECRET=your_app_secret
export FEISHU_ALICE_ACCESS_TOKEN=your_user_access_token
export FEISHU_ALICE_REFRESH_TOKEN=your_refresh_token
```

Then reload:

```bash
source ~/.zshrc
```

## 9. Current Code Status

In the current repository:

- the `oauth_user` profile shape already exists in the config model
- the naming and design strategy are already documented

But note:

- the currently implemented live Feishu path is bot/app-based calendar creation
- real user-identity execution has not been wired into the runtime yet

So:

- you can prepare the user credentials now and add them to your environment
- but the actual document execution path under `subject=user` still needs to be implemented later
