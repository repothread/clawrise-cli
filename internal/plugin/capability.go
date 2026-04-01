package plugin

import (
	"fmt"
	"sort"
	"strings"
)

const (
	// ManifestSchemaVersionV1 表示旧版 manifest 结构。
	ManifestSchemaVersionV1 = 1
	// ManifestSchemaVersionV2 表示 capability 化后的 manifest 结构。
	ManifestSchemaVersionV2 = 2

	// ManifestKindMulti 表示同一个插件包暴露了多种能力。
	ManifestKindMulti = "multi"
)

const (
	// CapabilityTypeProvider 表示 provider 能力。
	CapabilityTypeProvider = "provider"
	// CapabilityTypeAuthLauncher 表示 auth launcher 能力。
	CapabilityTypeAuthLauncher = "auth_launcher"
	// CapabilityTypeStorageBackend 表示 storage backend 能力。
	CapabilityTypeStorageBackend = "storage_backend"
	// CapabilityTypePolicy marks one pre-execution policy capability.
	CapabilityTypePolicy = "policy"
	// CapabilityTypeAuditSink marks one audit event sink capability.
	CapabilityTypeAuditSink = "audit_sink"
	// CapabilityTypeWorkflow marks one workflow planning capability.
	CapabilityTypeWorkflow = "workflow"
	// CapabilityTypeRegistrySource marks one plugin registry metadata source capability.
	CapabilityTypeRegistrySource = "registry_source"
)

// CapabilityDescriptor 描述一个插件暴露的单个 capability。
type CapabilityDescriptor struct {
	Type        string   `json:"type"`
	Platforms   []string `json:"platforms,omitempty"`
	Target      string   `json:"target,omitempty"`
	Backend     string   `json:"backend,omitempty"`
	ID          string   `json:"id,omitempty"`
	ActionTypes []string `json:"action_types,omitempty"`
	DisplayName string   `json:"display_name,omitempty"`
	Description string   `json:"description,omitempty"`
	Priority    int      `json:"priority,omitempty"`
}

// Validate 校验一个 capability 描述是否合法。
func (c CapabilityDescriptor) Validate() error {
	capabilityType := strings.TrimSpace(c.Type)
	switch capabilityType {
	case CapabilityTypeProvider:
		if len(trimmedStrings(c.Platforms)) == 0 {
			return fmt.Errorf("provider capability platforms must not be empty")
		}
	case CapabilityTypeAuthLauncher:
		// launcher 可以保持平台无关；如果后续需要 action_types，可由插件自行补充。
	case CapabilityTypeStorageBackend:
		descriptor := StorageBackendManifest{
			Target:      c.Target,
			Backend:     c.Backend,
			DisplayName: c.DisplayName,
			Description: c.Description,
		}
		if err := descriptor.Validate(); err != nil {
			return err
		}
	case CapabilityTypePolicy:
		// Policy capabilities may optionally filter by platform.
	case CapabilityTypeAuditSink:
		// Audit sinks currently only require the capability marker itself.
	case CapabilityTypeWorkflow:
		// Workflow planners currently only require the capability marker itself.
	case CapabilityTypeRegistrySource:
		// Registry sources currently only require the capability marker itself.
	default:
		return fmt.Errorf("unsupported capability type: %s", capabilityType)
	}
	return nil
}

func normalizeCapabilityList(items []CapabilityDescriptor) []CapabilityDescriptor {
	if len(items) == 0 {
		return nil
	}

	normalized := make([]CapabilityDescriptor, 0, len(items))
	for _, item := range items {
		item.Type = strings.TrimSpace(item.Type)
		item.Target = strings.TrimSpace(item.Target)
		item.Backend = strings.TrimSpace(item.Backend)
		item.ID = strings.TrimSpace(item.ID)
		item.DisplayName = strings.TrimSpace(item.DisplayName)
		item.Description = strings.TrimSpace(item.Description)
		item.Platforms = trimmedStrings(item.Platforms)
		item.ActionTypes = trimmedStrings(item.ActionTypes)
		normalized = append(normalized, item)
	}

	sort.SliceStable(normalized, func(i, j int) bool {
		left := capabilitySortKey(normalized[i])
		right := capabilitySortKey(normalized[j])
		return left < right
	})
	return normalized
}

func cloneCapabilityList(items []CapabilityDescriptor) []CapabilityDescriptor {
	if len(items) == 0 {
		return nil
	}
	cloned := make([]CapabilityDescriptor, 0, len(items))
	for _, item := range items {
		cloned = append(cloned, CapabilityDescriptor{
			Type:        item.Type,
			Platforms:   append([]string(nil), item.Platforms...),
			Target:      item.Target,
			Backend:     item.Backend,
			ID:          item.ID,
			ActionTypes: append([]string(nil), item.ActionTypes...),
			DisplayName: item.DisplayName,
			Description: item.Description,
			Priority:    item.Priority,
		})
	}
	return cloned
}

func findStorageCapability(manifest Manifest, target, backend string) (CapabilityDescriptor, bool) {
	target = strings.TrimSpace(target)
	backend = strings.TrimSpace(backend)
	for _, capability := range manifest.StorageBackendCapabilities() {
		if target != "" && capability.Target != target {
			continue
		}
		if backend != "" && capability.Backend != backend {
			continue
		}
		return capability, true
	}
	return CapabilityDescriptor{}, false
}

func capabilitySortKey(item CapabilityDescriptor) string {
	return strings.Join([]string{
		strings.TrimSpace(item.Type),
		strings.TrimSpace(item.Target),
		strings.TrimSpace(item.Backend),
		strings.TrimSpace(item.ID),
		strings.Join(trimmedStrings(item.Platforms), ","),
	}, "|")
}

func trimmedStrings(values []string) []string {
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

func deriveCapabilitiesFromLegacyManifest(manifest Manifest) []CapabilityDescriptor {
	switch strings.TrimSpace(manifest.Kind) {
	case ManifestKindProvider:
		return []CapabilityDescriptor{{
			Type:      CapabilityTypeProvider,
			Platforms: trimmedStrings(manifest.Platforms),
		}}
	case ManifestKindAuthLauncher:
		return []CapabilityDescriptor{{
			Type: CapabilityTypeAuthLauncher,
		}}
	case ManifestKindStorageBackend:
		if manifest.StorageBackend == nil {
			return nil
		}
		return []CapabilityDescriptor{{
			Type:        CapabilityTypeStorageBackend,
			Target:      strings.TrimSpace(manifest.StorageBackend.Target),
			Backend:     strings.TrimSpace(manifest.StorageBackend.Backend),
			DisplayName: strings.TrimSpace(manifest.StorageBackend.DisplayName),
			Description: strings.TrimSpace(manifest.StorageBackend.Description),
		}}
	default:
		return nil
	}
}
