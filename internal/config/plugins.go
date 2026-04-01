package config

import "strings"

// PluginsConfig 描述 capability 级插件启用与绑定配置。
type PluginsConfig struct {
	Enabled      map[string]string         `yaml:"enabled,omitempty"`
	Bindings     PluginBindingsConfig      `yaml:"bindings,omitempty"`
	PluginConfig map[string]map[string]any `yaml:"plugin_config,omitempty"`
}

// PluginBindingsConfig 描述各类 capability 的绑定规则。
type PluginBindingsConfig struct {
	Providers     map[string]ProviderPluginBinding `yaml:"providers,omitempty"`
	Storage       StorageBindingsConfig            `yaml:"storage,omitempty"`
	AuthLaunchers map[string][]string              `yaml:"auth_launchers,omitempty"`
}

// ProviderPluginBinding 描述一个平台到 provider 插件的绑定。
type ProviderPluginBinding struct {
	Plugin string `yaml:"plugin,omitempty"`
}

// StorageBindingsConfig 描述四类存储位点的绑定。
type StorageBindingsConfig struct {
	SecretStore   StoragePluginBinding `yaml:"secret_store,omitempty"`
	SessionStore  StoragePluginBinding `yaml:"session_store,omitempty"`
	AuthFlowStore StoragePluginBinding `yaml:"authflow_store,omitempty"`
	Governance    StoragePluginBinding `yaml:"governance,omitempty"`
}

// StoragePluginBinding 描述一个 storage capability 的绑定信息。
type StoragePluginBinding struct {
	Backend         string `yaml:"backend,omitempty"`
	Plugin          string `yaml:"plugin,omitempty"`
	FallbackBackend string `yaml:"fallback_backend,omitempty"`
}

// HasValue reports whether the binding carries any explicit configuration.
func (b StoragePluginBinding) HasValue() bool {
	return strings.TrimSpace(b.Backend) != "" ||
		strings.TrimSpace(b.Plugin) != "" ||
		strings.TrimSpace(b.FallbackBackend) != ""
}

// ResolveStorageBinding 返回一个存储位点的最终绑定配置。
func ResolveStorageBinding(cfg *Config, target string) StoragePluginBinding {
	if cfg == nil {
		return StoragePluginBinding{}
	}

	target = strings.TrimSpace(target)
	switch target {
	case "secret_store":
		if cfg.Plugins.Bindings.Storage.SecretStore.HasValue() {
			return normalizeStorageBinding(cfg.Plugins.Bindings.Storage.SecretStore)
		}
		return normalizeStorageBinding(StoragePluginBinding{
			Backend:         cfg.Auth.SecretStore.Backend,
			Plugin:          cfg.Auth.SecretStore.Plugin,
			FallbackBackend: cfg.Auth.SecretStore.FallbackBackend,
		})
	case "session_store":
		if cfg.Plugins.Bindings.Storage.SessionStore.HasValue() {
			return normalizeStorageBinding(cfg.Plugins.Bindings.Storage.SessionStore)
		}
		return normalizeStorageBinding(StoragePluginBinding{
			Backend: cfg.Auth.SessionStore.Backend,
			Plugin:  cfg.Auth.SessionStore.Plugin,
		})
	case "authflow_store":
		if cfg.Plugins.Bindings.Storage.AuthFlowStore.HasValue() {
			return normalizeStorageBinding(cfg.Plugins.Bindings.Storage.AuthFlowStore)
		}
		return normalizeStorageBinding(StoragePluginBinding{
			Backend: cfg.Auth.AuthFlowStore.Backend,
			Plugin:  cfg.Auth.AuthFlowStore.Plugin,
		})
	case "governance":
		if cfg.Plugins.Bindings.Storage.Governance.HasValue() {
			return normalizeStorageBinding(cfg.Plugins.Bindings.Storage.Governance)
		}
		return normalizeStorageBinding(StoragePluginBinding{
			Backend: cfg.Runtime.Governance.Backend,
			Plugin:  cfg.Runtime.Governance.Plugin,
		})
	default:
		return StoragePluginBinding{}
	}
}

func normalizeStorageBinding(binding StoragePluginBinding) StoragePluginBinding {
	return StoragePluginBinding{
		Backend:         strings.TrimSpace(binding.Backend),
		Plugin:          strings.TrimSpace(binding.Plugin),
		FallbackBackend: strings.TrimSpace(binding.FallbackBackend),
	}
}

// ResolveProviderBinding returns the explicitly bound provider plugin for one platform.
func ResolveProviderBinding(cfg *Config, platform string) string {
	if cfg == nil {
		return ""
	}
	platform = strings.TrimSpace(platform)
	if platform == "" || cfg.Plugins.Bindings.Providers == nil {
		return ""
	}
	return strings.TrimSpace(cfg.Plugins.Bindings.Providers[platform].Plugin)
}

// ResolveAuthLauncherPreferences returns the preferred launchers for one auth action type.
func ResolveAuthLauncherPreferences(cfg *Config, actionType string) []string {
	if cfg == nil {
		return nil
	}
	actionType = strings.TrimSpace(actionType)
	if actionType == "" || cfg.Plugins.Bindings.AuthLaunchers == nil {
		return nil
	}
	values := cfg.Plugins.Bindings.AuthLaunchers[actionType]
	items := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		items = append(items, value)
	}
	return items
}
