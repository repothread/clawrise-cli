# 参与贡献

感谢你考虑为 Clawrise 做贡献。

当前仓库仍在快速演进阶段，因此最有价值的贡献通常具备三个特点：范围聚焦、背景说明清楚、便于评审。

英文版本见 [CONTRIBUTING.md](CONTRIBUTING.md)。

## 你可以如何参与

- 提交可复现的缺陷问题
- 提出新的 CLI 工作流、playbook 或 provider 能力建议
- 完善文档、示例与上手流程
- 为运行时行为、配置解析、适配器映射补充测试
- 改进插件打包、校验和发现流程

## 开始之前

- 先阅读 [README.zh.md](README.zh.md) 以及其中链接的设计文档，了解当前项目边界。
- 请保持架构边界稳定：Clawrise 统一的是运行时执行框架，不是各个 provider 的资源字段模型。
- 尽量将每个 PR 控制在单一主题内，避免把重构、修复和新功能混在一起。
- 如果你的改动影响了 CLI 行为、配置结构或 operation 契约，请在同一个 PR 中同步更新相关文档。

## 本地开发

```bash
go build ./...
go test ./...
go run ./cmd/clawrise version
go run ./cmd/clawrise doctor
go run ./cmd/clawrise docs generate notion.page --out-dir ./docs/generated
```

如果本地 Go 缓存目录受限，可使用：

```bash
GOCACHE=/tmp/clawrise-go-build GOMODCACHE=/tmp/clawrise-gomodcache go test ./...
```

## 建议的提交流程

1. Fork 仓库并创建一个聚焦的分支。
2. 尽量用最小但完整的改动解决问题。
3. 对所有修改过的 Go 文件执行 `gofmt`。
4. 运行 `go test ./...`，必要时补充 `go build ./...`。
5. 如果行为发生变化，同步更新文档、示例或 playbook。
6. 提交 PR 时写清楚背景、取舍和测试依据。

## Pull Request 说明建议

建议在 PR 中明确写出：

- 改了什么，为什么改
- 影响到哪些命令、operation 或 plugin
- 是否存在配置变更或兼容性影响
- 测试依据，例如 `go test ./...`
- 如有必要，附上示例 CLI 输出帮助评审验证

推荐提交信息风格：

- 简短、祈使句
- 条件允许时优先使用 Conventional Commits，例如 `feat: add notion comment append`

## 缺陷与需求反馈

- 优先使用 GitHub Issue 模板提交问题。
- 请尽量提供复现步骤、预期行为和实际行为。
- 如果问题与第三方 API 有关，请写明 provider、operation 名称，以及必要的脱敏请求数据。
- 不要提交真实密钥、访问令牌或租户隐私数据。

## 文档与语言

- 英文文档是面向全球协作者的默认入口。
- 如果同一主题同时存在中英文文档，请尽量保持两者同步。
- 如果你当前只能更新一种语言，请在 PR 描述中明确说明，方便维护者后续补齐。

## 评审原则

维护者可能会要求贡献者：

- 缩小 PR 范围
- 将重构和行为变更拆分
- 增加或调整测试
- 在合并前补全文档

这些要求的目标是让仓库在持续演进中依然保持可评审、可维护。
