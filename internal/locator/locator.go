package locator

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const appDirName = "Clawrise"

// PathResolution 描述一个路径的最终值和解析来源。
type PathResolution struct {
	Path   string `json:"path"`
	Source string `json:"source"`
}

// ResolveConfigPath returns the main config file path.
func ResolveConfigPath() (string, error) {
	resolution, err := ResolveConfigPathResolution()
	if err != nil {
		return "", err
	}
	return resolution.Path, nil
}

// ResolveConfigPathResolution returns the main config file path together with its source.
func ResolveConfigPathResolution() (PathResolution, error) {
	if custom := strings.TrimSpace(os.Getenv("CLAWRISE_CONFIG")); custom != "" {
		return PathResolution{
			Path:   custom,
			Source: "env.CLAWRISE_CONFIG",
		}, nil
	}

	configDir, source, err := defaultConfigDirWithSource()
	if err != nil {
		return PathResolution{}, err
	}
	return PathResolution{
		Path:   filepath.Join(configDir, "config.yaml"),
		Source: source,
	}, nil
}

// DefaultConfigDir returns the OS-recommended config directory.
func DefaultConfigDir() (string, error) {
	configDir, _, err := defaultConfigDirWithSource()
	return configDir, err
}

func defaultConfigDirWithSource() (string, string, error) {
	if custom := strings.TrimSpace(os.Getenv("CLAWRISE_CONFIG_DIR")); custom != "" {
		return custom, "env.CLAWRISE_CONFIG_DIR", nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", "", fmt.Errorf("failed to resolve user home directory: %w", err)
	}

	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(homeDir, "Library", "Application Support", appDirName), "default", nil
	case "windows":
		if appData := strings.TrimSpace(os.Getenv("APPDATA")); appData != "" {
			return filepath.Join(appData, appDirName), "env.APPDATA", nil
		}
		return filepath.Join(homeDir, "AppData", "Roaming", appDirName), "default", nil
	default:
		if xdgConfig := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); xdgConfig != "" {
			return filepath.Join(xdgConfig, "clawrise"), "env.XDG_CONFIG_HOME", nil
		}
		return filepath.Join(homeDir, ".config", "clawrise"), "default", nil
	}
}

// ResolveStateDir returns the runtime state directory.
func ResolveStateDir(configPath string) (string, error) {
	resolution, err := ResolveStateDirResolution(configPath)
	if err != nil {
		return "", err
	}
	return resolution.Path, nil
}

// ResolveStateDirResolution returns the runtime state directory together with its source.
func ResolveStateDirResolution(configPath string) (PathResolution, error) {
	if custom := strings.TrimSpace(os.Getenv("CLAWRISE_STATE_DIR")); custom != "" {
		return PathResolution{
			Path:   custom,
			Source: "env.CLAWRISE_STATE_DIR",
		}, nil
	}

	// When a config file path is explicitly selected, prefer a sibling state
	// directory so the setup remains portable.
	if customConfig := strings.TrimSpace(os.Getenv("CLAWRISE_CONFIG")); customConfig != "" {
		return PathResolution{
			Path:   filepath.Join(filepath.Dir(customConfig), "state"),
			Source: "env.CLAWRISE_CONFIG_sibling",
		}, nil
	}
	if strings.TrimSpace(configPath) != "" {
		defaultConfigPath, err := ResolveConfigPath()
		if err == nil && samePath(defaultConfigPath, configPath) {
			return defaultStateDirResolution()
		}
		return PathResolution{
			Path:   filepath.Join(filepath.Dir(configPath), "state"),
			Source: "config_path_sibling",
		}, nil
	}
	return defaultStateDirResolution()
}

// ResolveRuntimeDir returns the runtime-governance data directory.
func ResolveRuntimeDir(configPath string) (string, error) {
	resolution, err := ResolveRuntimeDirResolution(configPath)
	if err != nil {
		return "", err
	}
	return resolution.Path, nil
}

// ResolveRuntimeDirResolution returns the runtime-governance data directory together with its source.
func ResolveRuntimeDirResolution(configPath string) (PathResolution, error) {
	if custom := strings.TrimSpace(os.Getenv("CLAWRISE_STATE_DIR")); custom != "" {
		return PathResolution{
			Path:   filepath.Join(custom, "runtime"),
			Source: "env.CLAWRISE_STATE_DIR",
		}, nil
	}
	if customConfig := strings.TrimSpace(os.Getenv("CLAWRISE_CONFIG")); customConfig != "" {
		return PathResolution{
			Path:   filepath.Join(filepath.Dir(customConfig), "runtime"),
			Source: "env.CLAWRISE_CONFIG_sibling",
		}, nil
	}
	if strings.TrimSpace(configPath) != "" {
		defaultConfigPath, err := ResolveConfigPath()
		if err == nil && samePath(defaultConfigPath, configPath) {
			stateResolution, err := defaultStateDirResolution()
			if err == nil {
				return PathResolution{
					Path:   filepath.Join(stateResolution.Path, "runtime"),
					Source: stateResolution.Source,
				}, nil
			}
		}
		return PathResolution{
			Path:   filepath.Join(filepath.Dir(configPath), "runtime"),
			Source: "config_path_sibling",
		}, nil
	}
	stateResolution, err := ResolveStateDirResolution(configPath)
	if err != nil {
		return PathResolution{}, err
	}
	return PathResolution{
		Path:   filepath.Join(stateResolution.Path, "runtime"),
		Source: stateResolution.Source,
	}, nil
}

func defaultStateDir() (string, error) {
	resolution, err := defaultStateDirResolution()
	if err != nil {
		return "", err
	}
	return resolution.Path, nil
}

func defaultStateDirResolution() (PathResolution, error) {
	if custom := strings.TrimSpace(os.Getenv("CLAWRISE_STATE_HOME")); custom != "" {
		return PathResolution{
			Path:   custom,
			Source: "env.CLAWRISE_STATE_HOME",
		}, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return PathResolution{}, fmt.Errorf("failed to resolve user home directory: %w", err)
	}

	switch runtime.GOOS {
	case "darwin":
		return PathResolution{
			Path:   filepath.Join(homeDir, "Library", "Application Support", appDirName, "state"),
			Source: "default",
		}, nil
	case "windows":
		if localAppData := strings.TrimSpace(os.Getenv("LOCALAPPDATA")); localAppData != "" {
			return PathResolution{
				Path:   filepath.Join(localAppData, appDirName, "State"),
				Source: "env.LOCALAPPDATA",
			}, nil
		}
		return PathResolution{
			Path:   filepath.Join(homeDir, "AppData", "Local", appDirName, "State"),
			Source: "default",
		}, nil
	default:
		if xdgState := strings.TrimSpace(os.Getenv("XDG_STATE_HOME")); xdgState != "" {
			return PathResolution{
				Path:   filepath.Join(xdgState, "clawrise"),
				Source: "env.XDG_STATE_HOME",
			}, nil
		}
		return PathResolution{
			Path:   filepath.Join(homeDir, ".local", "state", "clawrise"),
			Source: "default",
		}, nil
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
