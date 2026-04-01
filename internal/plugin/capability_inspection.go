package plugin

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

// runtimeCapabilityInspection describes one manifest/runtime capability comparison result.
type runtimeCapabilityInspection struct {
	RuntimeCapabilities []CapabilityDescriptor
	Warnings            []string
}

// inspectRuntimeCapabilities attempts to load runtime capabilities and compare them with the manifest.
func inspectRuntimeCapabilities(ctx context.Context, manifest Manifest) runtimeCapabilityInspection {
	command := manifest.ResolveCommand()
	if len(command) == 0 {
		return runtimeCapabilityInspection{
			Warnings: []string{"plugin manifest entry.command is empty; runtime capabilities are unavailable"},
		}
	}

	if _, err := os.Stat(command[0]); err != nil {
		if os.IsNotExist(err) {
			return runtimeCapabilityInspection{
				Warnings: []string{"plugin executable does not exist; runtime capabilities are unavailable"},
			}
		}
		return runtimeCapabilityInspection{
			Warnings: []string{fmt.Sprintf("failed to stat plugin executable during runtime capability inspection: %v", err)},
		}
	}

	runtime := NewProcessRuntime(manifest)
	defer func() { _ = runtime.Close() }()

	runtimeCapabilities, err := runtime.ListCapabilities(ctx)
	if err != nil {
		return runtimeCapabilityInspection{
			Warnings: []string{fmt.Sprintf("failed to load runtime capabilities: %v", err)},
		}
	}

	return runtimeCapabilityInspection{
		RuntimeCapabilities: runtimeCapabilities,
		Warnings:            capabilityMismatchWarnings(manifest.CapabilityList(), runtimeCapabilities),
	}
}

// matchedProviderBindingPlatforms returns the provider-binding platforms matched by the current plugin.
func matchedProviderBindingPlatforms(manifest Manifest, bindings map[string]string) []string {
	if len(bindings) == 0 {
		return nil
	}

	items := make([]string, 0)
	seen := make(map[string]struct{})
	for _, capability := range manifest.CapabilitiesByType(CapabilityTypeProvider) {
		for _, platform := range capability.Platforms {
			platform = strings.TrimSpace(platform)
			if platform == "" {
				continue
			}
			if strings.TrimSpace(bindings[platform]) != strings.TrimSpace(manifest.Name) {
				continue
			}
			if _, exists := seen[platform]; exists {
				continue
			}
			seen[platform] = struct{}{}
			items = append(items, platform)
		}
	}
	sort.Strings(items)
	return items
}

func capabilityMismatchWarnings(manifestCapabilities []CapabilityDescriptor, runtimeCapabilities []CapabilityDescriptor) []string {
	manifestSet := capabilityDescriptorSet(manifestCapabilities)
	runtimeSet := capabilityDescriptorSet(runtimeCapabilities)

	missing := capabilitySetDifference(manifestSet, runtimeSet)
	extra := capabilitySetDifference(runtimeSet, manifestSet)

	warnings := make([]string, 0, 2)
	if len(missing) > 0 {
		warnings = append(warnings, "runtime capabilities are missing manifest declarations: "+strings.Join(missing, ", "))
	}
	if len(extra) > 0 {
		warnings = append(warnings, "runtime capabilities expose undeclared items: "+strings.Join(extra, ", "))
	}
	return warnings
}

func capabilityDescriptorSet(items []CapabilityDescriptor) map[string]CapabilityDescriptor {
	normalized := normalizeCapabilityList(items)
	descriptors := make(map[string]CapabilityDescriptor, len(normalized))
	for _, item := range normalized {
		descriptors[capabilityComparisonKey(item)] = item
	}
	return descriptors
}

func capabilitySetDifference(left map[string]CapabilityDescriptor, right map[string]CapabilityDescriptor) []string {
	items := make([]CapabilityDescriptor, 0)
	for key, descriptor := range left {
		if _, exists := right[key]; exists {
			continue
		}
		items = append(items, descriptor)
	}

	sort.SliceStable(items, func(i, j int) bool {
		return capabilitySortKey(items[i]) < capabilitySortKey(items[j])
	})

	labels := make([]string, 0, len(items))
	for _, item := range items {
		labels = append(labels, formatCapabilityDescriptor(item))
	}
	return labels
}

func capabilityComparisonKey(item CapabilityDescriptor) string {
	return strings.Join([]string{
		strings.TrimSpace(item.Type),
		strings.Join(trimmedStrings(item.Platforms), ","),
		strings.TrimSpace(item.Target),
		strings.TrimSpace(item.Backend),
		strings.TrimSpace(item.ID),
		strings.Join(trimmedStrings(item.ActionTypes), ","),
		strings.TrimSpace(item.DisplayName),
		strings.TrimSpace(item.Description),
		strconv.Itoa(item.Priority),
	}, "|")
}

func formatCapabilityDescriptor(item CapabilityDescriptor) string {
	parts := make([]string, 0, 4)
	switch strings.TrimSpace(item.Type) {
	case CapabilityTypeProvider:
		parts = append(parts, "provider")
		if len(item.Platforms) > 0 {
			parts = append(parts, "platforms="+strings.Join(trimmedStrings(item.Platforms), "+"))
		}
	case CapabilityTypeAuthLauncher:
		parts = append(parts, "auth_launcher")
		if strings.TrimSpace(item.ID) != "" {
			parts = append(parts, "id="+strings.TrimSpace(item.ID))
		}
		if len(item.ActionTypes) > 0 {
			parts = append(parts, "actions="+strings.Join(trimmedStrings(item.ActionTypes), "+"))
		}
	case CapabilityTypeStorageBackend:
		parts = append(parts, "storage_backend")
		if strings.TrimSpace(item.Target) != "" {
			parts = append(parts, "target="+strings.TrimSpace(item.Target))
		}
		if strings.TrimSpace(item.Backend) != "" {
			parts = append(parts, "backend="+strings.TrimSpace(item.Backend))
		}
	case CapabilityTypePolicy:
		parts = append(parts, "policy")
		if strings.TrimSpace(item.ID) != "" {
			parts = append(parts, "id="+strings.TrimSpace(item.ID))
		}
		if len(item.Platforms) > 0 {
			parts = append(parts, "platforms="+strings.Join(trimmedStrings(item.Platforms), "+"))
		}
	case CapabilityTypeAuditSink:
		parts = append(parts, "audit_sink")
		if strings.TrimSpace(item.ID) != "" {
			parts = append(parts, "id="+strings.TrimSpace(item.ID))
		}
	case CapabilityTypeWorkflow:
		parts = append(parts, "workflow")
		if strings.TrimSpace(item.ID) != "" {
			parts = append(parts, "id="+strings.TrimSpace(item.ID))
		}
	case CapabilityTypeRegistrySource:
		parts = append(parts, "registry_source")
		if strings.TrimSpace(item.ID) != "" {
			parts = append(parts, "id="+strings.TrimSpace(item.ID))
		}
	default:
		if strings.TrimSpace(item.Type) != "" {
			parts = append(parts, strings.TrimSpace(item.Type))
		}
	}

	if len(parts) == 0 {
		return "unknown"
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return parts[0] + "[" + strings.Join(parts[1:], ",") + "]"
}
