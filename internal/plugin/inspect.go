package plugin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DiscoveryRootInspection 描述一个 discovery root 的状态。
type DiscoveryRootInspection struct {
	Path        string `json:"path"`
	Exists      bool   `json:"exists"`
	ManifestNum int    `json:"manifest_count"`
	Error       string `json:"error,omitempty"`
}

// DiscoveredPluginInspection 描述一个被发现 plugin 的详细状态。
type DiscoveredPluginInspection struct {
	RootPath        string                  `json:"root_path"`
	ManifestPath    string                  `json:"manifest_path"`
	Name            string                  `json:"name,omitempty"`
	Version         string                  `json:"version,omitempty"`
	Kind            string                  `json:"kind,omitempty"`
	Platforms       []string                `json:"platforms,omitempty"`
	StorageBackend  *StorageBackendManifest `json:"storage_backend,omitempty"`
	Capabilities    []CapabilityDescriptor  `json:"capabilities,omitempty"`
	Path            string                  `json:"path,omitempty"`
	Command         []string                `json:"command,omitempty"`
	CommandPath     string                  `json:"command_path,omitempty"`
	CommandExists   bool                    `json:"command_exists"`
	Install         *InstallMetadata        `json:"install,omitempty"`
	Handshake       *HandshakeResult        `json:"handshake,omitempty"`
	OperationCount  int                     `json:"operation_count"`
	CatalogCount    int                     `json:"catalog_count"`
	Health          *HealthResult           `json:"health,omitempty"`
	Healthy         bool                    `json:"healthy"`
	InspectionError string                  `json:"inspection_error,omitempty"`
}

// DiscoveryInspection 汇总当前环境下可发现 plugin 的状态。
type DiscoveryInspection struct {
	Roots   []DiscoveryRootInspection    `json:"roots"`
	Plugins []DiscoveredPluginInspection `json:"plugins"`
}

// InspectDiscovery 在不依赖 Manager 聚合成功的前提下，逐个检查发现到的 plugin。
func InspectDiscovery(ctx context.Context) (DiscoveryInspection, error) {
	roots, err := DefaultDiscoveryRoots()
	if err != nil {
		return DiscoveryInspection{}, err
	}

	report := DiscoveryInspection{
		Roots:   make([]DiscoveryRootInspection, 0, len(roots)),
		Plugins: []DiscoveredPluginInspection{},
	}

	for _, root := range roots {
		rootItem := DiscoveryRootInspection{Path: root}

		info, statErr := os.Stat(root)
		if statErr != nil {
			if os.IsNotExist(statErr) {
				report.Roots = append(report.Roots, rootItem)
				continue
			}
			rootItem.Error = statErr.Error()
			report.Roots = append(report.Roots, rootItem)
			continue
		}
		if !info.IsDir() {
			rootItem.Error = "plugin root is not a directory"
			report.Roots = append(report.Roots, rootItem)
			continue
		}

		rootItem.Exists = true
		plugins, walkErr := inspectRoot(ctx, root)
		if walkErr != nil {
			rootItem.Error = walkErr.Error()
		}
		rootItem.ManifestNum = len(plugins)
		report.Roots = append(report.Roots, rootItem)
		report.Plugins = append(report.Plugins, plugins...)
	}

	sort.Slice(report.Plugins, func(i, j int) bool {
		if report.Plugins[i].ManifestPath == report.Plugins[j].ManifestPath {
			return report.Plugins[i].Version < report.Plugins[j].Version
		}
		return report.Plugins[i].ManifestPath < report.Plugins[j].ManifestPath
	})
	return report, nil
}

func inspectRoot(ctx context.Context, root string) ([]DiscoveredPluginInspection, error) {
	items := []DiscoveredPluginInspection{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Base(path) != ManifestFileName {
			return nil
		}

		item := inspectManifest(ctx, root, path)
		items = append(items, item)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to inspect plugin root %s: %w", root, err)
	}
	return items, nil
}

func inspectManifest(ctx context.Context, root, manifestPath string) DiscoveredPluginInspection {
	item := DiscoveredPluginInspection{
		RootPath:     root,
		ManifestPath: manifestPath,
		Path:         filepath.Dir(manifestPath),
	}

	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		item.InspectionError = err.Error()
		return item
	}

	item.Name = manifest.Name
	item.Version = manifest.Version
	item.Kind = manifest.Kind
	item.Platforms = append([]string(nil), manifest.Platforms...)
	item.StorageBackend = cloneStorageBackendManifest(manifest.StorageBackend)
	item.Capabilities = cloneCapabilityList(manifest.CapabilityList())
	item.Command = append([]string(nil), manifest.Entry.Command...)
	item.CommandPath = manifest.ResolveCommand()[0]
	if _, err := os.Stat(item.CommandPath); err == nil {
		item.CommandExists = true
	} else if !os.IsNotExist(err) {
		item.InspectionError = err.Error()
		return item
	}

	if install, err := loadInstallMetadata(filepath.Dir(manifestPath)); err == nil {
		item.Install = install
	}
	if !item.CommandExists {
		item.InspectionError = "plugin executable does not exist"
		return item
	}

	runtime := NewProcessRuntime(manifest)
	defer func() { _ = runtime.Close() }()

	handshake, err := runtime.Handshake(ctx)
	if err != nil {
		item.InspectionError = err.Error()
		return item
	}
	item.Handshake = &handshake

	if manifest.SupportsKind(ManifestKindProvider) {
		operations, err := runtime.ListOperations(ctx)
		if err != nil {
			item.InspectionError = err.Error()
			return item
		}
		item.OperationCount = len(operations)

		entries, err := runtime.GetCatalog(ctx)
		if err != nil {
			item.InspectionError = err.Error()
			return item
		}
		item.CatalogCount = len(entries)

		health, err := runtime.Health(ctx)
		if err != nil {
			item.InspectionError = err.Error()
			return item
		}
		item.Health = &health
		item.Healthy = health.OK
	}
	if manifest.SupportsKind(ManifestKindAuthLauncher) {
		descriptor, err := runtime.DescribeAuthLauncher(ctx)
		if err != nil {
			item.InspectionError = err.Error()
			return item
		}
		item.Healthy = item.Healthy || strings.TrimSpace(descriptor.ID) != ""
	}
	if manifest.SupportsKind(ManifestKindStorageBackend) {
		for index, capability := range manifest.StorageBackendCapabilities() {
			if index == 0 {
				item.StorageBackend = &StorageBackendManifest{
					Target:      capability.Target,
					Backend:     capability.Backend,
					DisplayName: capability.DisplayName,
					Description: capability.Description,
				}
			}

			healthy, err := inspectStorageCapability(ctx, manifest, capability)
			if err != nil {
				item.InspectionError = err.Error()
				return item
			}
			item.Healthy = item.Healthy || healthy
		}
	}
	return item
}

func inspectStorageCapability(ctx context.Context, manifest Manifest, capability CapabilityDescriptor) (bool, error) {
	switch capability.Target {
	case "secret_store":
		store := NewProcessSecretStore(manifest)
		defer func() { _ = store.Close() }()

		status, err := store.Status(ctx)
		if err != nil {
			return false, err
		}
		return status.Supported, nil
	case "session_store":
		store := NewProcessSessionStore(manifest)
		defer func() { _ = store.Close() }()

		status, err := store.Status(ctx)
		if err != nil {
			return false, err
		}
		return status.Supported, nil
	case "authflow_store":
		store := NewProcessAuthFlowStore(manifest)
		defer func() { _ = store.Close() }()

		status, err := store.Status(ctx)
		if err != nil {
			return false, err
		}
		return status.Supported, nil
	case "governance":
		store := NewProcessGovernanceStore(manifest)
		defer func() { _ = store.Close() }()

		status, err := store.Status(ctx)
		if err != nil {
			return false, err
		}
		return status.Supported, nil
	default:
		return false, fmt.Errorf("unsupported storage backend target: %s", strings.TrimSpace(capability.Target))
	}
}
