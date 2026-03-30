# Clawrise Session / Token Cache 设计骨架

## 1. 目标

Session cache 负责承接运行时短生命周期认证态，而不是替代 `account` 配置。

它要解决的问题：

- 让 `account` 继续只描述静态身份与授权方式
- 让运行时可以缓存短期 `access_token`
- 让 OAuth 刷新后返回的新 `refresh_token` 有安全落点
- 避免每次调用都重新换 token
- 在不引入后台进程的前提下，提供稳定的本地复用能力

## 2. 非目标

这一层默认不做：

- 常驻后台进程
- 集中式远端 token 服务
- 跨机器同步
- 替代环境变量或主配置文件

## 3. 存储布局

建议目录：

```text
~/.clawrise/
  config.yaml
  runtime/
    auth/
      feishu_user_alice.json
      notion_public_workspace_a.json
```

规则：

- `config.yaml` 只保留 `account` 与 secret 引用
- `runtime/auth/*.json` 只保留运行时 session
- session 文件权限应为 `0600`
- `runtime/auth/` 目录权限应为 `0700`

## 4. Session 数据模型

当前代码骨架位于：

- [internal/auth/session.go](/Users/liyang/thread/clawrise-cli/internal/auth/session.go)

建议最小字段：

```json
{
  "version": 1,
  "account_name": "feishu_user_alice",
  "platform": "feishu",
  "subject": "user",
  "grant_type": "oauth_user",
  "access_token": "...",
  "refresh_token": "...",
  "token_type": "Bearer",
  "profile_fingerprint": "sha256:...",
  "expires_at": "2026-03-28T10:30:00Z",
  "created_at": "2026-03-28T10:00:00Z",
  "updated_at": "2026-03-28T10:05:00Z",
  "metadata": {
    "scope": "offline_access"
  }
}
```

说明：

- `account_name` 是 session 主键
- `profile_fingerprint` 用于配置变更后的 session 失效判断
- `refresh_token` 只有在 provider 会轮换时才写入
- `metadata` 预留给 provider-native 字段

## 5. 生命周期

每次 CLI 调用按需执行：

1. 根据 `account` 找到对应 session 文件
2. 如果存在且 `access_token` 未过期，直接复用
3. 如果不存在、已过期或即将过期，则执行 refresh / exchange
4. 将最新 session 原子写回本地文件
5. 用最新 session 发起本次请求
6. 进程退出

这里不需要后台进程。

## 6. 刷新策略

统一策略：

- 默认在过期前 2 分钟进入刷新窗口
- provider 返回新的 `refresh_token` 时必须覆盖旧值
- provider 返回新的 `access_token` 但未返回新 `refresh_token` 时，保留旧 `refresh_token`
- refresh 失败时，不回退到已知过期 token

Provider 差异：

- Feishu `oauth_user`
  - refresh 成功后可能返回新的 `refresh_token`
  - 旧 `refresh_token` 可能立即失效
- Notion `oauth_refreshable`
  - refresh 成功后应保存新的 `access_token`
  - 如果上游返回新 `refresh_token`，也应一并替换

## 7. 接入点

建议未来接入流程：

1. `runtime` 解析出最终 `account`
2. 根据 `account` 和主配置路径构造 `auth.FileStore`
3. adapter/provider 的 auth resolver 先尝试读取 session
4. session 不可用时再走 provider-native refresh
5. refresh 成功后写回 session store

职责边界：

- core runtime
  - 负责 session store 生命周期
  - 负责本地路径、文件权限、原子写
- provider adapter
  - 负责 refresh/exchange 细节
  - 负责把上游返回映射成统一 `Session`

## 8. 并发与一致性

第一阶段可接受的最小实现：

- 同一 account 采用单文件原子写
- 通过 `write temp -> rename` 降低半写入风险

第二阶段可以增加：

- 文件锁
- `updated_at` / `profile_fingerprint` 冲突检测
- refresh 去重

## 9. 未来命令

当前版本不再提供单独的 session CLI 子命令。

推荐通过下面的命令观察和驱动 session 生命周期：

- `clawrise auth inspect <account>`
- `clawrise auth check <account>`
- `clawrise auth login <account>`
- `clawrise auth complete <flow_id>`
