package paths

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const appDirName = "Clawrise"

// ResolveConfigPath 返回主配置文件路径。
func ResolveConfigPath() (string, error) {
	if custom := strings.TrimSpace(os.Getenv("CLAWRISE_CONFIG")); custom != "" {
		return custom, nil
	}

	configDir, err := DefaultConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "config.yaml"), nil
}

// DefaultConfigDir 返回当前操作系统推荐的配置目录。
func DefaultConfigDir() (string, error) {
	if custom := strings.TrimSpace(os.Getenv("CLAWRISE_CONFIG_DIR")); custom != "" {
		return custom, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to resolve user home directory: %w", err)
	}

	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(homeDir, "Library", "Application Support", appDirName), nil
	case "windows":
		if appData := strings.TrimSpace(os.Getenv("APPDATA")); appData != "" {
			return filepath.Join(appData, appDirName), nil
		}
		return filepath.Join(homeDir, "AppData", "Roaming", appDirName), nil
	default:
		if xdgConfig := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); xdgConfig != "" {
			return filepath.Join(xdgConfig, "clawrise"), nil
		}
		return filepath.Join(homeDir, ".config", "clawrise"), nil
	}
}

// ResolveStateDir 返回运行时状态目录。
func ResolveStateDir(configPath string) (string, error) {
	if custom := strings.TrimSpace(os.Getenv("CLAWRISE_STATE_DIR")); custom != "" {
		return custom, nil
	}

	// 当用户显式指定了配置文件路径时，优先把状态目录放在同级目录，方便携带式使用。
	if customConfig := strings.TrimSpace(os.Getenv("CLAWRISE_CONFIG")); customConfig != "" {
		return filepath.Join(filepath.Dir(customConfig), "state"), nil
	}
	if strings.TrimSpace(configPath) != "" {
		defaultConfigPath, err := ResolveConfigPath()
		if err == nil && samePath(defaultConfigPath, configPath) {
			return defaultStateDir()
		}
		return filepath.Join(filepath.Dir(configPath), "state"), nil
	}
	return defaultStateDir()
}

func defaultStateDir() (string, error) {
	if custom := strings.TrimSpace(os.Getenv("CLAWRISE_STATE_HOME")); custom != "" {
		return custom, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to resolve user home directory: %w", err)
	}

	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(homeDir, "Library", "Application Support", appDirName, "state"), nil
	case "windows":
		if localAppData := strings.TrimSpace(os.Getenv("LOCALAPPDATA")); localAppData != "" {
			return filepath.Join(localAppData, appDirName, "State"), nil
		}
		return filepath.Join(homeDir, "AppData", "Local", appDirName, "State"), nil
	default:
		if xdgState := strings.TrimSpace(os.Getenv("XDG_STATE_HOME")); xdgState != "" {
			return filepath.Join(xdgState, "clawrise"), nil
		}
		return filepath.Join(homeDir, ".local", "state", "clawrise"), nil
	}
}

func samePath(left string, right string) bool {
	if left == "" || right == "" {
		return false
	}

	cleanLeft := filepath.Clean(left)
	cleanRight := filepath.Clean(right)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(cleanLeft, cleanRight)
	}
	return cleanLeft == cleanRight
}
