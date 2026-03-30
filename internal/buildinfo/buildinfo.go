package buildinfo

// 这些变量会在发布构建时通过 -ldflags 注入。
// 保留默认值可以兼容本地开发和普通 go build 场景。
var (
	Version   = "0.1.0-dev"
	Commit    = "unknown"
	BuildDate = ""
)
