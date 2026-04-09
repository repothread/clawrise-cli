---
phase: 02-context
reviewed: 2026-04-09T00:00:00Z
depth: standard
files_reviewed: 3
files_reviewed_list:
  - internal/cli/root.go
  - internal/plugin/install.go
  - internal/secretstore/store.go
findings:
  critical: 1
  warning: 5
  info: 4
  total: 10
status: issues_found
---

# Phase 2: Code Review Report

**Reviewed:** 2026-04-09T00:00:00Z
**Depth:** standard
**Files Reviewed:** 3
**Status:** issues_found

## Summary

对三个核心源文件进行了标准深度审查：CLI 路由层 (`root.go`)、插件安装系统 (`install.go`)、密钥存储后端 (`store.go`)。整体代码质量较高，错误处理完善，安全防护到位（路径遍历防护、HTTPS 降级检测、信任策略校验等）。发现 1 个严重问题（密钥文件权限创建时的目录权限回退）、5 个警告和 4 个信息级建议。

## Critical Issues

### CR-01: CLAWRISE_MASTER_KEY 环境变量覆盖已有密钥文件时未检查文件内容一致性

**File:** `internal/secretstore/store.go:392-400`
**Issue:** 当设置了 `CLAWRISE_MASTER_KEY` 环境变量时，`resolveEncryptionKey()` 直接从环境变量派生 AES-256 密钥，但仅当密钥文件不存在时才写入。如果密钥文件已存在且内容与环境变量派生的密钥不同，函数将返回环境变量派生的密钥去解密由旧密钥加密的 vault 文件，导致解密静默失败。更严重的是，如果用户临时设置了一个不同的 `CLAWRISE_MASTER_KEY`，已存在的 vault 文件将无法被正确解密，但错误信息 (`failed to decrypt secret payload`) 不会提示密钥来源不一致。

此外，`CLAWRISE_MASTER_KEY` 以明文形式存在于进程环境变量中，通过 `/proc/<pid>/environ`（Linux）或类似机制可被同用户的其他进程读取。项目约束中提到"不能破坏现有插件协议版本"，这里不涉及协议变更，但密钥材料管理属于安全隐患。

**Fix:**
```go
func (s *encryptedFileStore) resolveEncryptionKey() ([]byte, error) {
    if s.backend == "windows_dpapi_file" {
        return nil, nil
    }

    if masterKey := strings.TrimSpace(os.Getenv("CLAWRISE_MASTER_KEY")); masterKey != "" {
        hash := sha256.Sum256([]byte(masterKey))
        key := hash[:]
        if existingData, err := os.ReadFile(s.keyPath()); err == nil {
            // 密钥文件已存在，校验环境变量派生密钥与文件密钥是否一致
            if len(existingData) != 32 {
                return nil, fmt.Errorf("invalid secret store key file: expected 32 bytes, got %d", len(existingData))
            }
            if !bytes.Equal(key, existingData) {
                return nil, fmt.Errorf("CLAWRISE_MASTER_KEY does not match existing key file; vault was created with a different key")
            }
        } else if errors.Is(err, os.ErrNotExist) {
            if err := s.writeKeyFile(key); err != nil {
                return nil, err
            }
        }
        return key, nil
    }

    return s.loadOrCreateLocalKey()
}
```

## Warnings

### WR-01: 多处 resolvePluginManager 错误被静默丢弃

**File:** `internal/cli/root.go:69,81,105,119`
**Issue:** 在 `account`、`doctor`、`auth`、`config` 四个子命令分支中，`resolvePluginManager` 的返回错误被 `_` 丢弃（`manager, _ = resolvePluginManager(ctx, deps, store)`）。当插件管理器初始化失败时，这些子命令会在后续调用 `manager` 方法时触发 nil 指针 panic（如 `runDoctor` 中第 554 行的 `manager.InspectAuth`）。虽然在 `runDoctor` 中有 nil 检查（第 554-556 行），但 `runAccount`、`runAuth`、`runConfig` 中可能没有类似的防御性检查。

**Fix:**
```go
case "account":
    var manager *pluginruntime.Manager
    if deps.PluginManager != nil {
        manager = deps.PluginManager
    } else {
        var err error
        manager, err = resolvePluginManager(ctx, deps, store)
        if err != nil {
            return err
        }
    }
    return runAccount(args[1:], store, deps.Stdout, manager)
```

### WR-02: openNamedStore 中 plugin 为空时的代码重复

**File:** `internal/secretstore/store.go:144-174`
**Issue:** `openNamedStore` 函数中，当 `plugin == "" || plugin == "builtin"` 时查找内置工厂，找不到后落入第 164-173 行再次调用 `discoverSecretStorePlugin`。但这段逻辑与 `else` 分支（第 151-162 行）完全重复。更关键的是，当内置工厂不存在且 plugin 为空时，代码会尝试通过 `discoverSecretStorePlugin("", "", enabledPlugins)` 查找插件，这可能导致意外行为——传入空的 plugin 名称去发现插件。

**Fix:**
```go
func openNamedStore(configPath string, stateDir string, backend string, plugin string, enabledPlugins map[string]string) (Store, error) {
    plugin = strings.TrimSpace(plugin)
    if plugin == "" || plugin == "builtin" {
        factory, ok := storeFactories[backend]
        if ok {
            return factory(configPath, stateDir)
        }
        // 内置后端不可用，返回明确的错误
        return nil, fmt.Errorf("built-in secret store backend %s is not available on this platform", backend)
    }

    manifest, found, err := discoverSecretStorePlugin(backend, plugin, enabledPlugins)
    if err != nil {
        return nil, err
    }
    if !found {
        return nil, fmt.Errorf("unsupported secret store backend: %s", backend)
    }
    return newPluginSecretStore(context.Background(), manifest), nil
}
```

### WR-03: downloadRemoteSource 未限制响应体大小

**File:** `internal/plugin/install.go:916`
**Issue:** `downloadRemoteSource` 通过 `io.Copy(file, response.Body)` 将远程服务器响应直接写入文件，没有设置大小上限。恶意或配置错误的远程源可能返回极大的响应体，导致磁盘空间耗尽。`pluginDownloadHTTPClient` 设置了 60 秒超时（第 28 行），但这仅限制连接/读取超时，不限制总数据量。

**Fix:**
```go
// 定义一个合理的插件包大小上限
const maxPluginArchiveSize = 500 * 1024 * 1024 // 500MB

// 在 downloadRemoteSource 中使用 io.LimitReader
limitedReader := io.LimitReader(response.Body, maxPluginArchiveSize)
if written, err := io.Copy(file, limitedReader); err != nil {
    return remoteDownloadResult{}, fmt.Errorf("failed to write downloaded plugin archive: %w", err)
}
if written >= maxPluginArchiveSize {
    return remoteDownloadResult{}, fmt.Errorf("plugin archive exceeds maximum allowed size of %d bytes", maxPluginArchiveSize)
}
```

### WR-04: commandSecretStore 中 set/delete 操作未对 connectionName 和 field 进行消毒

**File:** `internal/secretstore/store.go:565,579,586,628,643,652`
**Issue:** `newMacOSKeychainStore` 和 `newLinuxSecretServiceStore` 中，`connectionName` 和 `field` 参数直接拼接到命令行参数中，未做任何消毒处理。虽然这些值来源于内部配置而非用户直接输入，但如果 `connectionName` 或 `field` 包含 shell 特殊字符（如 `"`, `'`, `$`, `` ` `` 等），通过 `exec.Command` 传递时会作为独立参数而非 shell 解释，所以不存在 shell 注入。但 `security` 和 `secret-tool` 命令本身可能对这些值有语义限制。建议至少验证输入不为空。

**Fix:**
```go
accountName := func(connectionName string, field string) (string, error) {
    cn := strings.TrimSpace(connectionName)
    f := strings.TrimSpace(field)
    if cn == "" || f == "" {
        return "", fmt.Errorf("connection name and field must not be empty")
    }
    // 防御性检查：拒绝可能干扰命令行解析的字符
    if strings.ContainsAny(cn, "\"'`\\") || strings.ContainsAny(f, "\"'`\\") {
        return "", fmt.Errorf("connection name and field contain invalid characters")
    }
    return "connection/" + cn + "/field/" + f, nil
}
```

### WR-05: checksumTree 跳过 installMetadataFileName 但未在 copyTree 中做相同过滤

**File:** `internal/plugin/install.go:1368-1369` vs `internal/plugin/install.go:1332-1355`
**Issue:** `checksumTree` 在计算校验和时跳过 `install.json` 文件（第 1368-1369 行），但 `copyTree` 在复制文件时没有跳过该文件。这意味着如果插件源目录中存在 `install.json`（例如从之前的安装中残留），它会被复制到安装目录并成为安装的一部分，但不参与完整性校验计算。虽然 `writeInstallMetadata` 会覆盖它，但中间状态不一致可能导致升级时的校验不匹配。

**Fix:** 在 `copyTree` 中增加与 `checksumTree` 相同的跳过逻辑：
```go
if info.Name() == installMetadataFileName {
    return nil
}
```

## Info

### IN-01: ExitError.Error() 返回空字符串

**File:** `internal/cli/root.go:37-39`
**Issue:** `ExitError.Error()` 方法返回空字符串，这违反了 Go 的 `error` 接口惯例。虽然该类型用于携带退出码而非错误消息，但空字符串可能在日志记录和错误链追踪中丢失上下文。当前设计是有意的（避免在 CLI 输出中重复打印错误信息），但值得记录。

**Fix:** 无需修改，当前行为是有意设计。如果需要更好的可观测性，可以考虑返回格式化的描述：
```go
func (e ExitError) Error() string {
    return fmt.Sprintf("exit code %d", e.Code)
}
```

### IN-02: isSupportedSubject 几乎接受任何非空字符串

**File:** `internal/cli/root.go:930-932`
**Issue:** `isSupportedSubject` 仅检查 subject 不为空白字符串，实际上任何非空字符串都会被接受。这使得 `subject use` 命令的验证形同虚设，可能让用户误以为有受控的 subject 列表。

**Fix:** 如果 subject 确实应该是自由文本，可以移除验证函数并添加注释说明。如果需要受控列表，应从配置中获取允许的 subject 类型。

### IN-03: install.go 中 http 源类型在默认允许列表中但不在远程源类型检查中

**File:** `internal/plugin/install.go:40-46` vs `internal/plugin/install.go:1171-1178`
**Issue:** `defaultAllowedInstallSources` 包含 `pluginSourceTypeHTTP`（非加密 HTTP），但 `isRemoteSourceType` 也包含 `pluginSourceTypeHTTP`。这意味着非加密 HTTP 下载在信任策略中被视为远程源并受 host 限制，但默认允许列表本身就允许 HTTP 源。考虑到 HTTPS 到 HTTP 的降级检测仅在 `validateFinalRemoteDownloadTarget` 中存在（初始 URL 为 HTTPS 时），直接使用 HTTP 源时下载内容不受完整性保护。

**Fix:** 考虑从 `defaultAllowedInstallSources` 中移除 `pluginSourceTypeHTTP`，或至少在文档中说明安全风险。

### IN-04: pluginSecretStore 中 context.Background() 的使用

**File:** `internal/secretstore/store.go:161,173`
**Issue:** `pluginSecretStore` 使用 `context.Background()` 构造，因为 `Store` 接口不支持 context 传递。代码注释（第 159-161、171-173 行）已经承认了这一限制并标注了未来改进方向。当前实现中，`pluginSecretStore` 的方法（`Get`、`Set`、`Delete` 等）使用了构造时存储的 `ctx` 引用（从 `newPluginSecretStore` 传入），但由于构造时传入的是 `context.Background()`，实际上所有 JSON-RPC 调用都无法被取消。

**Fix:** 代码注释已充分说明这一限制。长期改进方向是将 `Store` 接口方法签名改为接受 `context.Context` 参数。

---

_Reviewed: 2026-04-09T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
