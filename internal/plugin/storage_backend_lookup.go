package plugin

import (
	"fmt"
	"strings"
)

// StorageBackendLookup 描述一次存储后端 capability 查找条件。
type StorageBackendLookup struct {
	Target         string
	Backend        string
	Plugin         string
	EnabledPlugins map[string]string
}

// FindStorageBackendManifest 根据 target/backend/plugin 查找一个匹配的插件 manifest。
func FindStorageBackendManifest(lookup StorageBackendLookup) (Manifest, bool, error) {
	manifests, err := FindStorageBackendManifests(lookup)
	if err != nil {
		return Manifest{}, false, err
	}
	if len(manifests) == 0 {
		return Manifest{}, false, nil
	}
	if len(manifests) > 1 {
		return Manifest{}, false, fmt.Errorf("multiple storage backend plugins match target=%s backend=%s plugin=%s", strings.TrimSpace(lookup.Target), strings.TrimSpace(lookup.Backend), strings.TrimSpace(lookup.Plugin))
	}
	return manifests[0], true, nil
}

// FindStorageBackendManifests 查找所有匹配的存储后端 capability。
func FindStorageBackendManifests(lookup StorageBackendLookup) ([]Manifest, error) {
	roots, err := DefaultDiscoveryRoots()
	if err != nil {
		return nil, err
	}
	allManifests, err := DiscoverManifests(roots)
	if err != nil {
		return nil, err
	}

	target := strings.TrimSpace(lookup.Target)
	backend := strings.TrimSpace(lookup.Backend)
	pluginName := strings.TrimSpace(lookup.Plugin)
	enabledRules := lookup.EnabledPlugins

	// 当用户显式绑定了某个 plugin 时，如果它只是被禁用了，就给出明确报错，
	// 避免上层把它误判成“插件不存在”或“不支持该 backend”。
	if pluginName != "" && pluginName != "builtin" && !resolvePluginEnabledState(pluginName, enabledRules).Enabled {
		for _, manifest := range allManifests {
			if strings.TrimSpace(manifest.Name) != pluginName {
				continue
			}
			if !manifest.SupportsKind(ManifestKindStorageBackend) {
				continue
			}
			if _, ok := findStorageCapability(manifest, target, backend); !ok {
				continue
			}
			return nil, fmt.Errorf("storage backend plugin %s for target %s is disabled by plugins.enabled", pluginName, target)
		}
	}

	manifests := filterManifestsByEnabledRules(allManifests, enabledRules)

	items := make([]Manifest, 0)
	for _, manifest := range manifests {
		if pluginName != "" && pluginName != "builtin" && strings.TrimSpace(manifest.Name) != pluginName {
			continue
		}
		if !manifest.SupportsKind(ManifestKindStorageBackend) {
			continue
		}
		if _, ok := findStorageCapability(manifest, target, backend); !ok {
			continue
		}
		items = append(items, manifest)
	}
	return items, nil
}
