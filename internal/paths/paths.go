package paths

import (
	"github.com/clawrise/clawrise-cli/internal/locator"
)

// ResolveConfigPath 返回主配置文件路径。
func ResolveConfigPath() (string, error) {
	return locator.ResolveConfigPath()
}

// DefaultConfigDir 返回当前操作系统推荐的配置目录。
func DefaultConfigDir() (string, error) {
	return locator.DefaultConfigDir()
}

// ResolveStateDir 返回运行时状态目录。
func ResolveStateDir(configPath string) (string, error) {
	return locator.ResolveStateDir(configPath)
}
