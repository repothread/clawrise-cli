package plugin

import (
	"fmt"
	"sort"
	"strings"
)

// ProviderCandidate 描述一个平台可用的 provider 插件候选。
type ProviderCandidate struct {
	Platform string `json:"platform"`
	Plugin   string `json:"plugin"`
	Version  string `json:"version"`
}

// DiscoverProviderCandidates 扫描当前 discovery roots 下的 provider 候选。
func DiscoverProviderCandidates() ([]ProviderCandidate, error) {
	roots, err := DefaultDiscoveryRoots()
	if err != nil {
		return nil, err
	}
	manifests, err := DiscoverManifests(roots)
	if err != nil {
		return nil, err
	}
	return providerCandidatesFromManifests(manifests), nil
}

func providerCandidatesFromManifests(manifests []Manifest) []ProviderCandidate {
	items := make([]ProviderCandidate, 0)
	for _, manifest := range manifests {
		for _, capability := range manifest.CapabilitiesByType(CapabilityTypeProvider) {
			for _, platform := range capability.Platforms {
				items = append(items, ProviderCandidate{
					Platform: strings.TrimSpace(platform),
					Plugin:   strings.TrimSpace(manifest.Name),
					Version:  strings.TrimSpace(manifest.Version),
				})
			}
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Platform == items[j].Platform {
			if items[i].Plugin == items[j].Plugin {
				return items[i].Version < items[j].Version
			}
			return items[i].Plugin < items[j].Plugin
		}
		return items[i].Platform < items[j].Platform
	})
	return items
}

// ValidateProviderBindings 校验 provider 绑定是否与已发现插件一致。
func ValidateProviderBindings(manifests []Manifest, bindings map[string]string) error {
	return ValidateProviderBindingsFromCandidates(providerCandidatesFromManifests(manifests), bindings)
}

// ValidateProviderBindingsFromCandidates 基于候选列表校验 provider 绑定。
func ValidateProviderBindingsFromCandidates(candidates []ProviderCandidate, bindings map[string]string) error {
	candidatesByPlatform := groupProviderCandidatesByPlatform(candidates)

	for platform, pluginName := range bindings {
		platform = strings.TrimSpace(platform)
		pluginName = strings.TrimSpace(pluginName)
		if platform == "" || pluginName == "" {
			continue
		}
		candidates := candidatesByPlatform[platform]
		if len(candidates) == 0 {
			return fmt.Errorf("provider binding for platform %s points to %s, but no provider plugin supports this platform", platform, pluginName)
		}
		found := false
		for _, candidate := range candidates {
			if candidate.Plugin == pluginName {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("provider binding for platform %s points to %s, but available plugins are: %s", platform, pluginName, joinProviderPluginNames(candidates))
		}
	}

	for platform, candidates := range candidatesByPlatform {
		if len(candidates) <= 1 {
			continue
		}
		if strings.TrimSpace(bindings[platform]) != "" {
			continue
		}
		return fmt.Errorf("multiple provider plugins support platform %s: %s; configure plugins.bindings.providers.%s.plugin to select one", platform, joinProviderPluginNames(candidates), platform)
	}
	return nil
}

func groupProviderCandidatesByPlatform(candidates []ProviderCandidate) map[string][]ProviderCandidate {
	grouped := make(map[string][]ProviderCandidate)
	for _, candidate := range candidates {
		grouped[candidate.Platform] = append(grouped[candidate.Platform], candidate)
	}
	return grouped
}

func joinProviderPluginNames(candidates []ProviderCandidate) string {
	seen := map[string]struct{}{}
	names := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if _, exists := seen[candidate.Plugin]; exists {
			continue
		}
		seen[candidate.Plugin] = struct{}{}
		names = append(names, candidate.Plugin)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}
