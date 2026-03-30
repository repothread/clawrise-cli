package locator

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"
)

const appDirName = "Clawrise"

type configPaths struct {
	Paths struct {
		ConfigDir string `yaml:"config_dir"`
		StateDir  string `yaml:"state_dir"`
	} `yaml:"paths"`
}

// ResolveConfigPath returns the main config file path.
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

// DefaultConfigDir returns the OS-recommended config directory.
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

// ResolveStateDir returns the runtime state directory.
func ResolveStateDir(configPath string) (string, error) {
	if custom := strings.TrimSpace(os.Getenv("CLAWRISE_STATE_DIR")); custom != "" {
		return custom, nil
	}

	if configured := readConfiguredStateDir(configPath); configured != "" {
		return configured, nil
	}

	// When a config file path is explicitly selected, prefer a sibling state
	// directory so the setup remains portable.
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

// ResolveRuntimeDir returns the runtime-governance data directory.
func ResolveRuntimeDir(configPath string) (string, error) {
	if custom := strings.TrimSpace(os.Getenv("CLAWRISE_STATE_DIR")); custom != "" {
		return filepath.Join(custom, "runtime"), nil
	}
	if configured := readConfiguredStateDir(configPath); configured != "" {
		return filepath.Join(configured, "runtime"), nil
	}
	if customConfig := strings.TrimSpace(os.Getenv("CLAWRISE_CONFIG")); customConfig != "" {
		return filepath.Join(filepath.Dir(customConfig), "runtime"), nil
	}
	if strings.TrimSpace(configPath) != "" {
		return filepath.Join(filepath.Dir(configPath), "runtime"), nil
	}
	stateDir, err := ResolveStateDir(configPath)
	if err != nil {
		return "", err
	}
	return filepath.Join(stateDir, "runtime"), nil
}

func readConfiguredStateDir(configPath string) string {
	configPath = strings.TrimSpace(configPath)
	if configPath == "" {
		return ""
	}

	data, err := os.ReadFile(configPath)
	if err != nil || len(data) == 0 {
		return ""
	}

	var parsed configPaths
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		return ""
	}
	if configured := strings.TrimSpace(parsed.Paths.StateDir); configured != "" {
		if filepath.IsAbs(configured) {
			return configured
		}
		return filepath.Join(filepath.Dir(configPath), configured)
	}
	return ""
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
