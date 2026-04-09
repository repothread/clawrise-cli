package config

import "strings"

const (
	// RuntimeSelectionModeAuto 表示按默认发现顺序自动启用 capability。
	RuntimeSelectionModeAuto = "auto"
	// RuntimeSelectionModeManual 表示仅启用显式配置列表中的 capability。
	RuntimeSelectionModeManual = "manual"
	// RuntimeSelectionModeDisabled 表示关闭该 capability 链。
	RuntimeSelectionModeDisabled = "disabled"
)

const (
	// AuditSinkTypePlugin 表示外部 audit sink plugin。
	AuditSinkTypePlugin = "plugin"
	// AuditSinkTypeStdout 表示内建 stdout sink。
	AuditSinkTypeStdout = "stdout"
	// AuditSinkTypeWebhook 表示内建 webhook sink。
	AuditSinkTypeWebhook = "webhook"
)

// ResolvePolicyMode 返回归一化后的 policy plugin 选择模式。
func ResolvePolicyMode(cfg *Config) string {
	if cfg == nil {
		return RuntimeSelectionModeAuto
	}
	return normalizeRuntimeSelectionMode(cfg.Runtime.Policy.Mode)
}

// ResolvePolicyPlugins 返回归一化后的 policy plugin 选择列表。
func ResolvePolicyPlugins(cfg *Config) []PolicyPluginBinding {
	if cfg == nil || len(cfg.Runtime.Policy.Plugins) == 0 {
		return nil
	}

	items := make([]PolicyPluginBinding, 0, len(cfg.Runtime.Policy.Plugins))
	for _, item := range cfg.Runtime.Policy.Plugins {
		normalized := PolicyPluginBinding{
			Plugin:   strings.TrimSpace(item.Plugin),
			PolicyID: strings.TrimSpace(item.PolicyID),
		}
		if normalized.Plugin == "" && normalized.PolicyID == "" {
			continue
		}
		items = append(items, normalized)
	}
	if len(items) == 0 {
		return nil
	}
	return items
}

// ResolveAuditMode 返回归一化后的 audit sink 选择模式。
func ResolveAuditMode(cfg *Config) string {
	if cfg == nil {
		return RuntimeSelectionModeAuto
	}
	return normalizeRuntimeSelectionMode(cfg.Runtime.Audit.Mode)
}

// ResolveAuditSinks 返回归一化后的 audit sink 配置列表。
func ResolveAuditSinks(cfg *Config) []AuditSinkConfig {
	if cfg == nil || len(cfg.Runtime.Audit.Sinks) == 0 {
		return nil
	}

	items := make([]AuditSinkConfig, 0, len(cfg.Runtime.Audit.Sinks))
	for _, item := range cfg.Runtime.Audit.Sinks {
		normalized := AuditSinkConfig{
			Type:      normalizeAuditSinkType(item),
			Plugin:    strings.TrimSpace(item.Plugin),
			SinkID:    strings.TrimSpace(item.SinkID),
			URL:       strings.TrimSpace(item.URL),
			Headers:   normalizeStringMap(item.Headers),
			TimeoutMS: item.TimeoutMS,
		}
		if normalized.Type == "" && normalized.Plugin == "" && normalized.SinkID == "" && normalized.URL == "" && len(normalized.Headers) == 0 && normalized.TimeoutMS == 0 {
			continue
		}
		items = append(items, normalized)
	}
	if len(items) == 0 {
		return nil
	}
	return items
}

func normalizeRuntimeSelectionMode(raw string) string {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "", RuntimeSelectionModeAuto:
		return RuntimeSelectionModeAuto
	case RuntimeSelectionModeManual:
		return RuntimeSelectionModeManual
	case RuntimeSelectionModeDisabled:
		return RuntimeSelectionModeDisabled
	default:
		return RuntimeSelectionModeAuto
	}
}

func normalizeAuditSinkType(item AuditSinkConfig) string {
	sinkType := strings.TrimSpace(strings.ToLower(item.Type))
	if sinkType != "" {
		return sinkType
	}
	switch {
	case strings.TrimSpace(item.URL) != "":
		return AuditSinkTypeWebhook
	case strings.TrimSpace(item.Plugin) != "" || strings.TrimSpace(item.SinkID) != "":
		return AuditSinkTypePlugin
	default:
		return ""
	}
}

func normalizeStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}

	items := make(map[string]string, len(values))
	for key, value := range values {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		items[key] = value
	}
	if len(items) == 0 {
		return nil
	}
	return items
}
