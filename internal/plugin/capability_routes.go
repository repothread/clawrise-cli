package plugin

import "strings"

const (
	selectionModeAuto     = "auto"
	selectionModeManual   = "manual"
	selectionModeDisabled = "disabled"
	auditSinkTypePlugin   = "plugin"
)

// PolicyCapabilitySelector describes one policy capability selector used for diagnostics.
type PolicyCapabilitySelector struct {
	Plugin   string `json:"plugin,omitempty"`
	PolicyID string `json:"policy_id,omitempty"`
}

// AuditSinkSelector describes one audit sink selector used for diagnostics.
type AuditSinkSelector struct {
	Type   string `json:"type,omitempty"`
	Plugin string `json:"plugin,omitempty"`
	SinkID string `json:"sink_id,omitempty"`
}

// CapabilityRouteStatus describes whether one capability participates in the current runtime chain.
type CapabilityRouteStatus struct {
	Type   string `json:"type"`
	ID     string `json:"id,omitempty"`
	Active bool   `json:"active"`
	Reason string `json:"reason,omitempty"`
	Source string `json:"source,omitempty"`
}

func inspectCapabilityRoutes(manifest Manifest, options DiscoveryOptions) []CapabilityRouteStatus {
	routes := make([]CapabilityRouteStatus, 0)
	enabledState := resolvePluginEnabledState(manifest.Name, options.EnabledPlugins)

	for _, capability := range manifest.CapabilitiesByType(CapabilityTypePolicy) {
		routes = append(routes, inspectPolicyCapabilityRoute(manifest, capability, enabledState, options))
	}
	for _, capability := range manifest.CapabilitiesByType(CapabilityTypeAuditSink) {
		routes = append(routes, inspectAuditSinkCapabilityRoute(manifest, capability, enabledState, options))
	}

	if len(routes) == 0 {
		return nil
	}
	return routes
}

func inspectPolicyCapabilityRoute(manifest Manifest, capability CapabilityDescriptor, enabledState pluginEnabledState, options DiscoveryOptions) CapabilityRouteStatus {
	route := CapabilityRouteStatus{
		Type: CapabilityTypePolicy,
		ID:   strings.TrimSpace(capability.ID),
	}
	if !enabledState.Enabled {
		route.Reason = "disabled_by_plugins.enabled"
		return route
	}

	mode := normalizeCapabilitySelectionMode(options.PolicyMode)
	if mode == selectionModeDisabled {
		route.Reason = "runtime.policy.mode=disabled"
		return route
	}

	if len(options.PolicySelectors) == 0 {
		if mode == selectionModeManual {
			route.Reason = "manual_mode_without_selector"
			return route
		}
		route.Active = true
		route.Source = "auto"
		return route
	}

	if policyCapabilityMatchesAnySelector(manifest.Name, capability, options.PolicySelectors) {
		route.Active = true
		route.Source = "configured"
		return route
	}

	route.Reason = "not_selected_by_runtime.policy.plugins"
	return route
}

func inspectAuditSinkCapabilityRoute(manifest Manifest, capability CapabilityDescriptor, enabledState pluginEnabledState, options DiscoveryOptions) CapabilityRouteStatus {
	route := CapabilityRouteStatus{
		Type: CapabilityTypeAuditSink,
		ID:   strings.TrimSpace(capability.ID),
	}
	if !enabledState.Enabled {
		route.Reason = "disabled_by_plugins.enabled"
		return route
	}

	mode := normalizeCapabilitySelectionMode(options.AuditMode)
	if mode == selectionModeDisabled {
		route.Reason = "runtime.audit.mode=disabled"
		return route
	}

	if len(options.AuditSinks) == 0 {
		if mode == selectionModeManual {
			route.Reason = "manual_mode_without_sink_selector"
			return route
		}
		route.Active = true
		route.Source = "auto"
		return route
	}

	if auditSinkCapabilityMatchesAnySelector(manifest.Name, capability, options.AuditSinks) {
		route.Active = true
		route.Source = "configured"
		return route
	}

	route.Reason = "not_selected_by_runtime.audit.sinks"
	return route
}

func policyCapabilityMatchesAnySelector(pluginName string, capability CapabilityDescriptor, selectors []PolicyCapabilitySelector) bool {
	for _, selector := range selectors {
		if !policyCapabilityMatchesSelector(pluginName, capability, selector) {
			continue
		}
		return true
	}
	return false
}

func policyCapabilityMatchesSelector(pluginName string, capability CapabilityDescriptor, selector PolicyCapabilitySelector) bool {
	pluginName = strings.TrimSpace(pluginName)
	selectorPlugin := strings.TrimSpace(selector.Plugin)
	selectorPolicyID := strings.TrimSpace(selector.PolicyID)
	capabilityID := strings.TrimSpace(capability.ID)

	if selectorPlugin != "" && selectorPlugin != pluginName {
		return false
	}
	if selectorPolicyID != "" && selectorPolicyID != capabilityID {
		return false
	}
	return selectorPlugin != "" || selectorPolicyID != ""
}

func auditSinkCapabilityMatchesAnySelector(pluginName string, capability CapabilityDescriptor, selectors []AuditSinkSelector) bool {
	for _, selector := range selectors {
		if !auditSinkCapabilityMatchesSelector(pluginName, capability, selector) {
			continue
		}
		return true
	}
	return false
}

func auditSinkCapabilityMatchesSelector(pluginName string, capability CapabilityDescriptor, selector AuditSinkSelector) bool {
	selectorType := strings.TrimSpace(selector.Type)
	if selectorType != "" && selectorType != auditSinkTypePlugin {
		return false
	}

	pluginName = strings.TrimSpace(pluginName)
	selectorPlugin := strings.TrimSpace(selector.Plugin)
	selectorSinkID := strings.TrimSpace(selector.SinkID)
	capabilityID := strings.TrimSpace(capability.ID)

	if selectorPlugin != "" && selectorPlugin != pluginName {
		return false
	}
	if selectorSinkID != "" && selectorSinkID != capabilityID {
		return false
	}
	return selectorType == auditSinkTypePlugin || selectorPlugin != "" || selectorSinkID != ""
}

func normalizeCapabilitySelectionMode(raw string) string {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "", selectionModeAuto:
		return selectionModeAuto
	case selectionModeManual:
		return selectionModeManual
	case selectionModeDisabled:
		return selectionModeDisabled
	default:
		return selectionModeAuto
	}
}
