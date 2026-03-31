# Clawrise 插件系统设计

英文版见 [../en/plugin-system-design.md](../en/plugin-system-design.md)。

## 1. 文档目的

这份文档定义 Clawrise 的目标插件系统形态，用于解决以下问题：

- 平台接入不应继续硬编码在 core 二进制中
- 平台能力应支持按需安装、独立发布、独立升级
- `spec`、catalog、runtime、doctor 仍应保持统一事实源与统一外壳
- 插件机制应与分发渠道解耦，`npm` 只能作为可选安装源之一

这份设计按最终形态编写，不以当前内置 Feishu / Notion 注册逻辑为兼容前提。

## 1.1 当前进展

截至当前仓库状态：

- `M1` 已完成：
  - core 中已经存在 provider runtime 抽象
- `M2` 已完成：
  - manifest 解析、插件发现和外部进程 runtime 已落地
- `M3` 已完成：
  - Feishu / Notion 第一方能力已经通过 plugin binary 暴露
- `M4` 已部分完成：
  - 已具备 `plugin list/install/info/remove`
  - 已支持本地目录、`file://`、`https://`、`npm://` 安装
  - release hardening、trust policy、upgrade workflow 仍需继续完善

## 2. 非目标

首版插件系统明确不做：

- 不使用 Go `plugin` `.so` 动态链接模式
- 不把 `npm` 绑定为唯一安装方式
- 不把插件协议设计成语言相关协议
- 不把插件系统扩展成远程 SaaS 托管执行环境
- 不在首版解决完整签名信任链与企业级沙箱隔离

## 3. 设计结论

Clawrise 应采用：

- `core + external provider plugins`
- `stdio + JSON-RPC 2.0`
- 插件进程按需懒启动
- core 负责统一 runtime、config、spec 聚合、审计、幂等、输出
- plugin 负责平台 operation catalog、auth 细节、provider-native 映射与真实执行

分发层与运行层解耦：

- 运行层使用插件目录与 manifest 发现插件
- 分发层可支持 `file://`、`https://`、`npm://`、未来的其他源

## 4. 架构分层

### 4.1 Core 职责

`clawrise-core` 负责：

- CLI 入口与命令解析
- 配置解析与 account 选择
- 认证材料解析与脱敏
- 统一 runtime envelope
- 幂等、重试、超时、审计
- `spec` 聚合
- 插件发现、安装、加载、握手、健康检查

### 4.2 Plugin 职责

`clawrise-plugin-<platform>` 负责：

- 平台 operation 声明
- provider-native 请求构造与响应映射
- 平台认证细节与 token 刷新
- 平台错误到标准错误模型的映射
- 平台 capability 级 `spec` / catalog 输出

### 4.3 关键边界

core 不应直接了解平台细节。

plugin 不应直接读取 core 的主配置文件，也不应自己解释 `env:`、account 选择规则、默认平台规则。

推荐边界：

- core 解析配置与 account
- core 将“已解析好的认证材料”和执行请求发给 plugin
- plugin 只消费执行请求，不关心配置文件来源

## 5. 传输协议

首版协议建议：

- transport: `stdio`
- message format: `JSON-RPC 2.0`
- plugin `stdout`: 只输出协议消息
- plugin `stderr`: 只输出日志

原因：

- 易于跨语言实现
- 不需要端口管理
- 适合本地 CLI 插件场景
- 与 agent / tool 生态兼容度高

## 6. 最小协议面

首版建议只定义 5 个核心方法：

1. `clawrise.handshake`
2. `clawrise.operations.list`
3. `clawrise.catalog.get`
4. `clawrise.execute`
5. `clawrise.health`

后续可扩展：

- `clawrise.auth.probe`
- `clawrise.spec.export`
- `clawrise.install.info`

### 6.1 `clawrise.handshake`

用途：

- 协议版本协商
- 返回插件元信息
- 返回 capability 面

请求：

```json
{
  "jsonrpc": "2.0",
  "id": "1",
  "method": "clawrise.handshake",
  "params": {
    "protocol_version": 1,
    "core": {
      "name": "clawrise",
      "version": "0.2.0"
    }
  }
}
```

响应：

```json
{
  "jsonrpc": "2.0",
  "id": "1",
  "result": {
    "protocol_version": 1,
    "plugin": {
      "name": "feishu",
      "version": "0.1.0"
    },
    "platforms": ["feishu"],
    "capabilities": {
      "operations_list": true,
      "catalog_get": true,
      "execute": true,
      "health": true,
      "auth_probe": false
    }
  }
}
```

### 6.2 `clawrise.operations.list`

用途：

- 告诉 core 当前 plugin 暴露的 operation 列表
- 返回执行元数据与 discovery 元数据
- 为 `spec list/get`、completion、routing 提供事实源

请求：

```json
{
  "jsonrpc": "2.0",
  "id": "2",
  "method": "clawrise.operations.list",
  "params": {}
}
```

响应：

```json
{
  "jsonrpc": "2.0",
  "id": "2",
  "result": {
    "operations": [
      {
        "operation": "feishu.calendar.event.create",
        "platform": "feishu",
        "mutating": true,
        "default_timeout_ms": 10000,
        "allowed_subjects": ["bot"],
        "spec": {
          "summary": "Create a Feishu calendar event.",
          "dry_run_supported": true,
          "input": {
            "required": ["calendar_id", "summary", "start_at", "end_at"],
            "optional": ["description", "location", "reminders", "timezone"],
            "sample": {
              "calendar_id": "cal_demo"
            }
          },
          "idempotency": {
            "required": true,
            "auto_generated": true
          }
        }
      }
    ]
  }
}
```

### 6.3 `clawrise.catalog.get`

用途：

- 为 `spec status` 提供结构化 catalog
- 做 runtime / catalog 对账

请求：

```json
{
  "jsonrpc": "2.0",
  "id": "3",
  "method": "clawrise.catalog.get",
  "params": {}
}
```

响应：

```json
{
  "jsonrpc": "2.0",
  "id": "3",
  "result": {
    "entries": [
      { "operation": "feishu.calendar.event.create" },
      { "operation": "feishu.calendar.event.list" }
    ]
  }
}
```

### 6.4 `clawrise.execute`

用途：

- 执行指定 operation
- 保持统一输入输出外壳

请求：

```json
{
  "jsonrpc": "2.0",
  "id": "4",
  "method": "clawrise.execute",
  "params": {
    "request": {
      "request_id": "req_123",
      "operation": "feishu.calendar.event.create",
      "input": {
        "calendar_id": "cal_demo",
        "summary": "Weekly sync"
      },
      "timeout_ms": 10000,
      "idempotency_key": "idem_xxx",
      "dry_run": false
    },
    "identity": {
      "platform": "feishu",
      "subject": "bot",
      "profile_name": "feishu_bot",
      "auth": {
        "type": "client_credentials",
        "app_id": "resolved-app-id",
        "app_secret": "resolved-app-secret"
      }
    }
  }
}
```

成功响应：

```json
{
  "jsonrpc": "2.0",
  "id": "4",
  "result": {
    "ok": true,
    "data": {
      "event_id": "evt_123"
    },
    "error": null,
    "meta": {
      "provider_request_id": "",
      "retry_count": 0
    }
  }
}
```

失败响应：

```json
{
  "jsonrpc": "2.0",
  "id": "4",
  "result": {
    "ok": false,
    "data": null,
    "error": {
      "code": "RESOURCE_NOT_FOUND",
      "message": "calendar not found",
      "retryable": false,
      "upstream_code": "191001",
      "http_status": 404
    },
    "meta": {
      "provider_request_id": "",
      "retry_count": 0
    }
  }
}
```

约束：

- JSON-RPC `error` 只用于协议层错误
- operation 业务失败仍通过 `result.ok=false` 返回

### 6.5 `clawrise.health`

用途：

- 插件存活检查
- 基础诊断

请求：

```json
{
  "jsonrpc": "2.0",
  "id": "5",
  "method": "clawrise.health",
  "params": {}
}
```

响应：

```json
{
  "jsonrpc": "2.0",
  "id": "5",
  "result": {
    "ok": true,
    "details": {
      "plugin_name": "feishu",
      "plugin_version": "0.1.0"
    }
  }
}
```

## 7. Manifest 设计

每个插件目录必须包含 `plugin.json`。

建议结构：

```json
{
  "schema_version": 1,
  "name": "feishu",
  "version": "0.1.0",
  "kind": "provider",
  "protocol_version": 1,
  "platforms": ["feishu"],
  "entry": {
    "type": "binary",
    "command": ["./bin/clawrise-plugin-feishu"]
  },
  "catalog_path": "./catalog/operations.json",
  "min_core_version": "0.2.0"
}
```

字段说明：

- `schema_version`: manifest schema 版本
- `name`: 插件名
- `version`: 插件自身版本
- `kind`: 当前固定为 `provider`
- `protocol_version`: 插件协议版本
- `platforms`: 插件负责的平台列表
- `entry.command`: 插件启动命令
- `catalog_path`: 可选的静态 catalog 文件路径
- `min_core_version`: 最低 core 版本

## 8. 插件目录布局

推荐全局目录：

```text
~/.clawrise/plugins/
  feishu/
    0.1.0/
      plugin.json
      bin/clawrise-plugin-feishu
      catalog/operations.json
  notion/
    0.1.0/
      plugin.json
      bin/clawrise-plugin-notion
```

同时支持：

- 项目级目录：`.clawrise/plugins/`
- 环境变量覆盖：`CLAWRISE_PLUGIN_PATHS`

core 的插件发现优先级建议为：

1. 显式环境变量路径
2. 项目级路径
3. 全局路径

## 9. 分发模型

插件机制与分发渠道解耦。

建议支持以下 source scheme：

- `file://`
- `https://`
- `npm://`
- 后续可扩展 `gh://`

示例：

```bash
clawrise plugin install file:///tmp/clawrise-plugin-feishu.tar.gz
clawrise plugin install https://example.com/clawrise-plugin-feishu.tar.gz
clawrise plugin install npm://@clawrise/plugin-feishu
```

如果使用 `npm`：

- `npm` 只作为安装与分发渠道
- 运行时仍建议执行预编译原生二进制
- 不应要求用户在运行插件时依赖 Node runtime

## 10. Core 侧抽象

core 需要新增一个 provider runtime 抽象层。

建议 Go 接口：

```go
type ProviderRuntime interface {
    Handshake(ctx context.Context) (HandshakeResult, error)
    ListOperations(ctx context.Context) ([]adapter.Definition, error)
    GetCatalog(ctx context.Context) ([]catalog.Entry, error)
    Execute(ctx context.Context, req ExecuteRequest) (ExecuteResult, error)
    Health(ctx context.Context) (HealthResult, error)
}
```

### 10.1 Core 的新模块

建议新增：

- `internal/plugin/manifest`
- `internal/plugin/discovery`
- `internal/plugin/runtime`
- `internal/plugin/protocol`
- `internal/plugin/install`

### 10.2 运行时流程

```text
CLI input
  -> resolve operation and flags
  -> load config / resolve profile
  -> resolve installed plugin by platform
  -> lazy start plugin process
  -> handshake
  -> execute via JSON-RPC
  -> normalize envelope
  -> write audit record
```

## 11. 协议结构建议

### 11.1 通用消息结构

```go
type RPCRequest struct {
    JSONRPC string `json:"jsonrpc"`
    ID      string `json:"id"`
    Method  string `json:"method"`
    Params  any    `json:"params,omitempty"`
}

type RPCResponse struct {
    JSONRPC string    `json:"jsonrpc"`
    ID      string    `json:"id"`
    Result  any       `json:"result,omitempty"`
    Error   *RPCError `json:"error,omitempty"`
}

type RPCError struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    Data    any    `json:"data,omitempty"`
}
```

### 11.2 执行请求结构

```go
type ExecuteRequest struct {
    Request  ExecuteEnvelope  `json:"request"`
    Identity ExecuteIdentity  `json:"identity"`
}

type ExecuteEnvelope struct {
    RequestID      string         `json:"request_id"`
    Operation      string         `json:"operation"`
    Input          map[string]any `json:"input"`
    TimeoutMS      int64          `json:"timeout_ms"`
    IdempotencyKey string         `json:"idempotency_key,omitempty"`
    DryRun         bool           `json:"dry_run"`
}

type ExecuteIdentity struct {
    Platform    string         `json:"platform"`
    Subject     string         `json:"subject"`
    ProfileName string         `json:"profile_name"`
    Auth        map[string]any `json:"auth"`
}
```

### 11.3 执行结果结构

```go
type ExecuteResult struct {
    OK    bool              `json:"ok"`
    Data  any               `json:"data"`
    Error *ExecuteErrorBody `json:"error,omitempty"`
    Meta  ExecuteMeta       `json:"meta"`
}

type ExecuteErrorBody struct {
    Code         string `json:"code"`
    Message      string `json:"message"`
    Retryable    bool   `json:"retryable"`
    UpstreamCode string `json:"upstream_code,omitempty"`
    HTTPStatus   int    `json:"http_status,omitempty"`
}

type ExecuteMeta struct {
    ProviderRequestID string `json:"provider_request_id,omitempty"`
    RetryCount        int    `json:"retry_count"`
}
```

## 12. Auth 边界设计

首版推荐：

- core 负责读取配置与解析 secret
- core 将已解析 auth 材料传给 plugin
- plugin 不直接读取主配置文件

原因：

- 避免多语言插件都重复实现 config 解析
- 避免 `env:` 等语法进入插件协议
- 避免插件获得超过执行所需的配置上下文

未来可扩展为：

- plugin 定义 auth schema
- core 做表单化或命令式 auth 输入
- plugin 做 refresh 与 provider-native auth 执行

## 13. 版本兼容规则

必须同时维护以下版本：

- manifest schema version
- plugin protocol version
- core version
- plugin version

core 加载插件时应执行：

1. 校验 `plugin.json`
2. 校验 `schema_version`
3. 校验 `min_core_version`
4. 启动后执行 `handshake`
5. 校验 `protocol_version`

任一步失败都应拒绝加载，并返回明确错误。

## 14. 安全边界

插件本质上是本地代码执行，不应被包装成沙箱能力。

最低限度建议：

- credential 不通过命令行参数传递
- credential 通过协议消息或受控 stdin 传递
- plugin `stdout` 禁止输出非协议内容
- plugin `stderr` 允许日志，但 core 默认不回显敏感内容
- 安装器应记录来源、版本、checksum
- 后续可增加签名校验

## 15. 为什么不使用 Go `plugin`

不建议使用 Go `plugin` 包，原因包括：

- 平台支持面有限，不适合作为跨平台 CLI 的核心扩展机制
- plugin 与主程序需要严格共享 toolchain、依赖、构建条件
- 可调试性与可运维性差
- 运行时隔离边界弱

对 Clawrise 这类面向长期扩展的 CLI，外部进程插件更稳妥。

参考：

- Go `plugin` package: <https://pkg.go.dev/plugin>

## 16. 推荐插件实现语言

推荐默认语言：

- first-party plugins: `Go`
- protocol: language-agnostic
- third-party community plugins: 可开放 `TypeScript` 或其他语言

原因：

- 当前 Feishu / Notion 代码已在 Go 中实现
- first-party plugin 用 Go 迁移成本最低
- 协议层保持无关语言，便于未来开放生态

## 17. 实施路径

建议分四步推进：

### 17.1 M1: 抽象 provider runtime

状态：

- 已完成

- 从 core 中移除平台硬编码构造逻辑
- 引入 `ProviderRuntime`
- 通过 in-process runtime shim 作为过渡层，为第一方 provider 脱离 core 做准备

### 17.2 M2: 实现本地插件协议与插件发现

状态：

- 已完成

- 实现 `stdio + JSON-RPC`
- 实现 manifest 解析
- 实现插件目录发现与懒启动
- 实现 `handshake` / `operations.list` / `execute`

### 17.3 M3: 将 Feishu / Notion 迁移为 first-party plugin

状态：

- 已完成

- 复用当前 adapter 与 registry
- 为每个平台提供单独二进制入口
- core 不再直接 import 平台 adapter

### 17.4 M4: 安装器与远程分发

状态：

- 部分完成

- `clawrise plugin install`
- `clawrise plugin list`
- `clawrise plugin info`
- `clawrise plugin remove`
- 支持 `file://`、`https://`、`npm://`
- 增加 trust、verify 与 upgrade policy

## 18. 对当前仓库的直接影响

当前硬编码入口位于：

- `internal/cli/root.go`

未来需要拆分的重点区域：

- 平台注册逻辑从 core 中移除
- `spec` 改为从 plugin 聚合 operation 与 catalog
- runtime 执行从直接调用本地 handler 改为通过 provider runtime 调用 plugin

## 19. 最小可执行范围

如果只做一个可上线的最小版本，建议范围是：

- 只支持 first-party plugin
- 只支持本地目录安装
- 只支持 `binary + manifest`
- 只实现 5 个核心 RPC 方法
- 先迁移 Feishu / Notion

这个范围已经足够解除 core 与平台的静态耦合，并为后续增加 Google 与其他平台提供稳定底座。
