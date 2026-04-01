package plugin

import "strings"

// pluginEnabledState 描述一个插件在启用规则下的归一化状态。
type pluginEnabledState struct {
	Enabled  bool
	Explicit bool
	Rule     string
}

// pluginSelectionState 描述一个插件在 discovery 视角下是否参与当前运行时。
type pluginSelectionState struct {
	Enabled         bool
	EnableRule      string
	Selected        bool
	SelectionReason string
}

func resolvePluginEnabledState(pluginName string, rules map[string]string) pluginEnabledState {
	pluginName = strings.TrimSpace(pluginName)
	if pluginName == "" {
		return pluginEnabledState{Enabled: true}
	}

	rule, exists := rules[pluginName]
	if !exists {
		return pluginEnabledState{Enabled: true}
	}

	rule = strings.TrimSpace(rule)
	return pluginEnabledState{
		Enabled:  !isDisabledPluginRule(rule),
		Explicit: true,
		Rule:     rule,
	}
}

// 这里先只收敛“显式禁用”语义，其他非空值都保留给未来版本选择能力。
func isDisabledPluginRule(rule string) bool {
	switch strings.ToLower(strings.TrimSpace(rule)) {
	case "0", "false", "off", "disable", "disabled":
		return true
	default:
		return false
	}
}

func filterManifestsByEnabledRules(manifests []Manifest, rules map[string]string) []Manifest {
	if len(manifests) == 0 {
		return nil
	}

	filtered := make([]Manifest, 0, len(manifests))
	for _, manifest := range manifests {
		state := resolvePluginEnabledState(manifest.Name, rules)
		if !state.Enabled {
			continue
		}
		filtered = append(filtered, manifest)
	}
	return filtered
}

func resolveManifestSelectionState(manifest Manifest, options DiscoveryOptions) pluginSelectionState {
	enabledState := resolvePluginEnabledState(manifest.Name, options.EnabledPlugins)
	state := pluginSelectionState{
		Enabled:    enabledState.Enabled,
		EnableRule: enabledState.Rule,
		Selected:   enabledState.Enabled,
	}
	if !enabledState.Enabled {
		state.SelectionReason = "disabled_by_plugins.enabled"
		return state
	}

	// provider 绑定只影响 provider capability；多 capability 插件仍可能因其他能力继续参与运行时。
	if manifest.SupportsKind(ManifestKindProvider) && !shouldKeepProviderManifest(manifest, options.ProviderBindings) {
		providerCapabilities := len(manifest.CapabilitiesByType(CapabilityTypeProvider))
		if providerCapabilities == len(manifest.CapabilityList()) {
			state.Selected = false
			state.SelectionReason = "filtered_by_provider_binding"
			return state
		}
		state.SelectionReason = "partially_selected_by_provider_binding"
		return state
	}

	state.SelectionReason = "active"
	return state
}
