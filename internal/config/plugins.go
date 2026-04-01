package config

import "strings"

// PluginsConfig describes capability-level plugin enablement, install, and binding settings.
type PluginsConfig struct {
	Enabled      map[string]string         `yaml:"enabled,omitempty"`
	Install      PluginInstallConfig       `yaml:"install,omitempty"`
	Bindings     PluginBindingsConfig      `yaml:"bindings,omitempty"`
	PluginConfig map[string]map[string]any `yaml:"plugin_config,omitempty"`
}

// PluginInstallConfig describes the baseline trust policy for plugin install sources.
type PluginInstallConfig struct {
	AllowedSources   []string `yaml:"allowed_sources,omitempty"`
	AllowedHosts     []string `yaml:"allowed_hosts,omitempty"`
	AllowedNPMScopes []string `yaml:"allowed_npm_scopes,omitempty"`
}

// PluginBindingsConfig describes binding rules for each capability class.
type PluginBindingsConfig struct {
	Providers     map[string]ProviderPluginBinding `yaml:"providers,omitempty"`
	Storage       StorageBindingsConfig            `yaml:"storage,omitempty"`
	AuthLaunchers map[string][]string              `yaml:"auth_launchers,omitempty"`
}

// ProviderPluginBinding describes the explicit plugin binding for one platform provider.
type ProviderPluginBinding struct {
	Plugin string `yaml:"plugin,omitempty"`
}

// StorageBindingsConfig describes bindings for the four storage targets.
type StorageBindingsConfig struct {
	SecretStore   StoragePluginBinding `yaml:"secret_store,omitempty"`
	SessionStore  StoragePluginBinding `yaml:"session_store,omitempty"`
	AuthFlowStore StoragePluginBinding `yaml:"authflow_store,omitempty"`
	Governance    StoragePluginBinding `yaml:"governance,omitempty"`
}

// StoragePluginBinding describes one storage capability binding.
type StoragePluginBinding struct {
	Backend         string `yaml:"backend,omitempty"`
	Plugin          string `yaml:"plugin,omitempty"`
	FallbackBackend string `yaml:"fallback_backend,omitempty"`
}

// ResolveEnabledPlugins returns normalized plugin enablement rules.
func ResolveEnabledPlugins(cfg *Config) map[string]string {
	if cfg == nil || len(cfg.Plugins.Enabled) == 0 {
		return nil
	}

	items := make(map[string]string, len(cfg.Plugins.Enabled))
	for pluginName, rule := range cfg.Plugins.Enabled {
		pluginName = strings.TrimSpace(pluginName)
		if pluginName == "" {
			continue
		}
		items[pluginName] = strings.TrimSpace(rule)
	}
	if len(items) == 0 {
		return nil
	}
	return items
}

// ResolvePluginInstallAllowedSources returns the normalized allowlist for plugin install sources.
// An empty result means the runtime should fall back to its built-in default policy.
func ResolvePluginInstallAllowedSources(cfg *Config) []string {
	if cfg == nil {
		return nil
	}
	return normalizeStringList(cfg.Plugins.Install.AllowedSources)
}

// ResolvePluginInstallAllowedHosts returns the normalized allowlist for remote install hosts.
func ResolvePluginInstallAllowedHosts(cfg *Config) []string {
	if cfg == nil {
		return nil
	}
	return normalizeStringList(cfg.Plugins.Install.AllowedHosts)
}

// ResolvePluginInstallAllowedNPMScopes returns the normalized allowlist for npm package scopes.
func ResolvePluginInstallAllowedNPMScopes(cfg *Config) []string {
	if cfg == nil {
		return nil
	}

	values := normalizeStringList(cfg.Plugins.Install.AllowedNPMScopes)
	if len(values) == 0 {
		return nil
	}

	items := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		switch normalized := normalizeNPMPolicyScope(value); normalized {
		case "":
			continue
		default:
			if _, exists := seen[normalized]; exists {
				continue
			}
			seen[normalized] = struct{}{}
			items = append(items, normalized)
		}
	}
	if len(items) == 0 {
		return nil
	}
	return items
}

// HasValue reports whether the binding carries any explicit configuration.
func (b StoragePluginBinding) HasValue() bool {
	return strings.TrimSpace(b.Backend) != "" ||
		strings.TrimSpace(b.Plugin) != "" ||
		strings.TrimSpace(b.FallbackBackend) != ""
}

// ResolveStorageBinding returns the resolved binding for one storage target.
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
	for key, values := range cfg.Plugins.Bindings.AuthLaunchers {
		if strings.TrimSpace(key) != actionType {
			continue
		}
		return normalizeStringList(values)
	}
	return nil
}

// ResolveAllAuthLauncherPreferences returns the normalized launcher preference map.
func ResolveAllAuthLauncherPreferences(cfg *Config) map[string][]string {
	if cfg == nil || len(cfg.Plugins.Bindings.AuthLaunchers) == 0 {
		return nil
	}

	items := make(map[string][]string, len(cfg.Plugins.Bindings.AuthLaunchers))
	for actionType, values := range cfg.Plugins.Bindings.AuthLaunchers {
		actionType = strings.TrimSpace(actionType)
		if actionType == "" {
			continue
		}
		preferences := normalizeStringList(values)
		if len(preferences) == 0 {
			continue
		}
		items[actionType] = preferences
	}
	if len(items) == 0 {
		return nil
	}
	return items
}

// SetAuthLauncherPreference puts the target launcher at the head of one action-type preference list.
func SetAuthLauncherPreference(cfg *Config, actionType string, launcherID string) []string {
	if cfg == nil {
		return nil
	}

	actionType = strings.TrimSpace(actionType)
	launcherID = strings.TrimSpace(launcherID)
	if actionType == "" || launcherID == "" {
		return nil
	}

	cfg.Ensure()
	current := ResolveAuthLauncherPreferences(cfg, actionType)
	next := make([]string, 0, len(current)+1)
	next = append(next, launcherID)
	for _, item := range current {
		if item == launcherID {
			continue
		}
		next = append(next, item)
	}
	cfg.Plugins.Bindings.AuthLaunchers[actionType] = next
	return append([]string(nil), next...)
}

// UnsetAuthLauncherPreference removes one launcher or one whole action-type preference group.
func UnsetAuthLauncherPreference(cfg *Config, actionType string, launcherID string) []string {
	if cfg == nil {
		return nil
	}

	actionType = strings.TrimSpace(actionType)
	launcherID = strings.TrimSpace(launcherID)
	if actionType == "" {
		return nil
	}

	cfg.Ensure()
	if launcherID == "" {
		delete(cfg.Plugins.Bindings.AuthLaunchers, actionType)
		return nil
	}

	current := ResolveAuthLauncherPreferences(cfg, actionType)
	next := make([]string, 0, len(current))
	for _, item := range current {
		if item == launcherID {
			continue
		}
		next = append(next, item)
	}
	if len(next) == 0 {
		delete(cfg.Plugins.Bindings.AuthLaunchers, actionType)
		return nil
	}
	cfg.Plugins.Bindings.AuthLaunchers[actionType] = next
	return append([]string(nil), next...)
}

func normalizeStringList(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	items := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		items = append(items, value)
	}
	return items
}

func normalizeNPMPolicyScope(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	switch value {
	case "", "*", "unscoped":
		return value
	}
	if !strings.HasPrefix(value, "@") {
		value = "@" + value
	}
	if len(value) == 1 {
		return ""
	}
	return value
}
