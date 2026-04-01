package config

import (
	"fmt"
	"strings"
)

// ValidateRuntimeSelectionMode validates one runtime capability selection mode.
func ValidateRuntimeSelectionMode(raw string) (string, error) {
	normalized := normalizeRuntimeSelectionMode(raw)
	trimmed := normalizedRuntimeSelectionModeInput(raw)
	switch trimmed {
	case "", RuntimeSelectionModeAuto, RuntimeSelectionModeManual, RuntimeSelectionModeDisabled:
		return normalized, nil
	}
	return "", fmt.Errorf("selection mode must be one of auto, manual, or disabled")
}

// SetPolicyMode updates runtime.policy.mode with a validated normalized value.
func SetPolicyMode(cfg *Config, mode string) (string, error) {
	if cfg == nil {
		return "", fmt.Errorf("config is required")
	}

	normalized, err := ValidateRuntimeSelectionMode(mode)
	if err != nil {
		return "", err
	}
	cfg.Runtime.Policy.Mode = normalized
	return normalized, nil
}

// AddPolicyPluginBinding appends one normalized policy selector when it is not already present.
func AddPolicyPluginBinding(cfg *Config, binding PolicyPluginBinding) []PolicyPluginBinding {
	if cfg == nil {
		return nil
	}

	normalized := PolicyPluginBinding{
		Plugin:   normalizeString(binding.Plugin),
		PolicyID: normalizeString(binding.PolicyID),
	}
	if normalized.Plugin == "" && normalized.PolicyID == "" {
		return ResolvePolicyPlugins(cfg)
	}

	current := ResolvePolicyPlugins(cfg)
	for _, item := range current {
		if item.Plugin == normalized.Plugin && item.PolicyID == normalized.PolicyID {
			return append([]PolicyPluginBinding(nil), current...)
		}
	}

	current = append(current, normalized)
	cfg.Runtime.Policy.Plugins = current
	return append([]PolicyPluginBinding(nil), current...)
}

// RemovePolicyPluginBinding removes all policy selectors that match the non-empty selector fields.
func RemovePolicyPluginBinding(cfg *Config, selector PolicyPluginBinding) []PolicyPluginBinding {
	if cfg == nil {
		return nil
	}

	selector = PolicyPluginBinding{
		Plugin:   normalizeString(selector.Plugin),
		PolicyID: normalizeString(selector.PolicyID),
	}
	if selector.Plugin == "" && selector.PolicyID == "" {
		return ResolvePolicyPlugins(cfg)
	}

	current := ResolvePolicyPlugins(cfg)
	next := make([]PolicyPluginBinding, 0, len(current))
	for _, item := range current {
		if policyBindingMatchesSelector(item, selector) {
			continue
		}
		next = append(next, item)
	}
	if len(next) == 0 {
		cfg.Runtime.Policy.Plugins = nil
		return nil
	}
	cfg.Runtime.Policy.Plugins = next
	return append([]PolicyPluginBinding(nil), next...)
}

// SetAuditMode updates runtime.audit.mode with a validated normalized value.
func SetAuditMode(cfg *Config, mode string) (string, error) {
	if cfg == nil {
		return "", fmt.Errorf("config is required")
	}

	normalized, err := ValidateRuntimeSelectionMode(mode)
	if err != nil {
		return "", err
	}
	cfg.Runtime.Audit.Mode = normalized
	return normalized, nil
}

// AddAuditSink appends or replaces one normalized audit sink declaration.
func AddAuditSink(cfg *Config, sink AuditSinkConfig) ([]AuditSinkConfig, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	normalized, err := normalizeAuditSinkMutation(sink)
	if err != nil {
		return nil, err
	}

	current := ResolveAuditSinks(cfg)
	replaced := false
	for index, item := range current {
		if auditSinkIdentity(item) != auditSinkIdentity(normalized) {
			continue
		}
		current[index] = normalized
		replaced = true
		break
	}
	if !replaced {
		current = append(current, normalized)
	}

	cfg.Runtime.Audit.Sinks = current
	return append([]AuditSinkConfig(nil), current...), nil
}

// RemoveAuditSink removes all audit sinks that match the non-empty selector fields.
func RemoveAuditSink(cfg *Config, selector AuditSinkConfig) []AuditSinkConfig {
	if cfg == nil {
		return nil
	}

	selector = AuditSinkConfig{
		Type:      normalizeAuditSinkType(selector),
		Plugin:    normalizeString(selector.Plugin),
		SinkID:    normalizeString(selector.SinkID),
		URL:       normalizeString(selector.URL),
		Headers:   normalizeStringMap(selector.Headers),
		TimeoutMS: selector.TimeoutMS,
	}
	if selector.Type == "" && selector.Plugin == "" && selector.SinkID == "" && selector.URL == "" {
		return ResolveAuditSinks(cfg)
	}

	current := ResolveAuditSinks(cfg)
	next := make([]AuditSinkConfig, 0, len(current))
	for _, item := range current {
		if auditSinkMatchesSelector(item, selector) {
			continue
		}
		next = append(next, item)
	}
	if len(next) == 0 {
		cfg.Runtime.Audit.Sinks = nil
		return nil
	}
	cfg.Runtime.Audit.Sinks = next
	return append([]AuditSinkConfig(nil), next...)
}

func normalizeAuditSinkMutation(sink AuditSinkConfig) (AuditSinkConfig, error) {
	normalized := AuditSinkConfig{
		Type:      normalizeAuditSinkType(sink),
		Plugin:    normalizeString(sink.Plugin),
		SinkID:    normalizeString(sink.SinkID),
		URL:       normalizeString(sink.URL),
		Headers:   normalizeStringMap(sink.Headers),
		TimeoutMS: sink.TimeoutMS,
	}

	switch normalized.Type {
	case AuditSinkTypeStdout:
		return AuditSinkConfig{Type: AuditSinkTypeStdout}, nil
	case AuditSinkTypeWebhook:
		if normalized.URL == "" {
			return AuditSinkConfig{}, fmt.Errorf("webhook audit sink requires a url")
		}
		return normalized, nil
	case AuditSinkTypePlugin:
		if normalized.Plugin == "" && normalized.SinkID == "" {
			return AuditSinkConfig{}, fmt.Errorf("plugin audit sink requires plugin and/or sink_id")
		}
		return normalized, nil
	default:
		return AuditSinkConfig{}, fmt.Errorf("audit sink type must be one of stdout, webhook, or plugin")
	}
}

func auditSinkIdentity(sink AuditSinkConfig) string {
	switch sink.Type {
	case AuditSinkTypeStdout:
		return AuditSinkTypeStdout
	case AuditSinkTypeWebhook:
		return AuditSinkTypeWebhook + "|" + sink.URL
	case AuditSinkTypePlugin:
		return AuditSinkTypePlugin + "|" + sink.Plugin + "|" + sink.SinkID
	default:
		return ""
	}
}

func policyBindingMatchesSelector(item PolicyPluginBinding, selector PolicyPluginBinding) bool {
	if selector.Plugin != "" && selector.Plugin != item.Plugin {
		return false
	}
	if selector.PolicyID != "" && selector.PolicyID != item.PolicyID {
		return false
	}
	return selector.Plugin != "" || selector.PolicyID != ""
}

func auditSinkMatchesSelector(item AuditSinkConfig, selector AuditSinkConfig) bool {
	if selector.Type != "" && selector.Type != item.Type {
		return false
	}
	if selector.Plugin != "" && selector.Plugin != item.Plugin {
		return false
	}
	if selector.SinkID != "" && selector.SinkID != item.SinkID {
		return false
	}
	if selector.URL != "" && selector.URL != item.URL {
		return false
	}
	return selector.Type != "" || selector.Plugin != "" || selector.SinkID != "" || selector.URL != ""
}

func normalizeString(value string) string {
	return strings.TrimSpace(value)
}

func normalizedRuntimeSelectionModeInput(raw string) string {
	return strings.TrimSpace(strings.ToLower(raw))
}
