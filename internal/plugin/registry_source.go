package plugin

import (
	"context"
	"sort"
	"strings"
)

// RegistrySourceRuntime describes one plugin registry metadata source runtime.
type RegistrySourceRuntime interface {
	Name() string
	ID() string
	Priority() int
	Handshake(ctx context.Context) (HandshakeResult, error)
	List(ctx context.Context, params RegistrySourceListParams) (RegistrySourceListResult, error)
	Resolve(ctx context.Context, params RegistrySourceResolveParams) (RegistrySourceResolveResult, error)
	Close() error
}

// ProcessRegistrySource executes JSON-RPC calls against one external registry source plugin.
type ProcessRegistrySource struct {
	runtime    *ProcessRuntime
	capability CapabilityDescriptor
}

// NewProcessRegistrySource creates one process-backed registry source plugin client.
func NewProcessRegistrySource(manifest Manifest, capability CapabilityDescriptor) *ProcessRegistrySource {
	return &ProcessRegistrySource{
		runtime:    NewProcessRuntime(manifest),
		capability: capability,
	}
}

func (r *ProcessRegistrySource) Name() string {
	if r == nil || r.runtime == nil {
		return ""
	}
	return r.runtime.Name()
}

func (r *ProcessRegistrySource) ID() string {
	if r == nil {
		return ""
	}
	return strings.TrimSpace(r.capability.ID)
}

func (r *ProcessRegistrySource) Priority() int {
	if r == nil {
		return 0
	}
	return r.capability.Priority
}

func (r *ProcessRegistrySource) Handshake(ctx context.Context) (HandshakeResult, error) {
	return r.runtime.Handshake(ctx)
}

func (r *ProcessRegistrySource) List(ctx context.Context, params RegistrySourceListParams) (RegistrySourceListResult, error) {
	var result RegistrySourceListResult
	if strings.TrimSpace(params.SourceID) == "" {
		params.SourceID = r.ID()
	}
	if err := r.runtime.call(ctx, "clawrise.registry_source.list", params, &result); err != nil {
		return RegistrySourceListResult{}, err
	}
	return result, nil
}

func (r *ProcessRegistrySource) Resolve(ctx context.Context, params RegistrySourceResolveParams) (RegistrySourceResolveResult, error) {
	var result RegistrySourceResolveResult
	if strings.TrimSpace(params.SourceID) == "" {
		params.SourceID = r.ID()
	}
	if err := r.runtime.call(ctx, "clawrise.registry_source.resolve", params, &result); err != nil {
		return RegistrySourceResolveResult{}, err
	}
	return result, nil
}

func (r *ProcessRegistrySource) Close() error {
	if r == nil || r.runtime == nil {
		return nil
	}
	return r.runtime.Close()
}

// DiscoverRegistrySourceRuntimes discovers all enabled registry source capabilities.
func DiscoverRegistrySourceRuntimes(options DiscoveryOptions) ([]RegistrySourceRuntime, error) {
	roots, err := DefaultDiscoveryRoots()
	if err != nil {
		return nil, err
	}
	manifests, err := DiscoverManifests(roots)
	if err != nil {
		return nil, err
	}
	manifests = filterManifestsByEnabledRules(manifests, options.EnabledPlugins)

	items := make([]RegistrySourceRuntime, 0)
	for _, manifest := range manifests {
		for _, capability := range manifest.CapabilitiesByType(CapabilityTypeRegistrySource) {
			items = append(items, NewProcessRegistrySource(manifest, capability))
		}
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Priority() == items[j].Priority() {
			if items[i].Name() == items[j].Name() {
				return items[i].ID() < items[j].ID()
			}
			return items[i].Name() < items[j].Name()
		}
		return items[i].Priority() > items[j].Priority()
	})
	return items, nil
}
