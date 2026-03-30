package plugin

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/clawrise/clawrise-cli/internal/adapter"
	"github.com/clawrise/clawrise-cli/internal/apperr"
	speccatalog "github.com/clawrise/clawrise-cli/internal/spec/catalog"
)

// RegistryRuntimeOptions describes optional capabilities for an in-process runtime.
type RegistryRuntimeOptions struct {
	AuthProvider AuthProvider
}

// NewRegistryRuntime creates one runtime backed by an in-process adapter registry.
func NewRegistryRuntime(name, version string, platforms []string, registry *adapter.Registry, catalogEntries []speccatalog.Entry) Runtime {
	return NewRegistryRuntimeWithOptions(name, version, platforms, registry, catalogEntries, RegistryRuntimeOptions{})
}

// NewRegistryRuntimeWithOptions creates an in-process runtime with optional capabilities.
func NewRegistryRuntimeWithOptions(name, version string, platforms []string, registry *adapter.Registry, catalogEntries []speccatalog.Entry, options RegistryRuntimeOptions) Runtime {
	return &registryRuntime{
		name:           name,
		version:        version,
		platforms:      append([]string(nil), platforms...),
		registry:       registry,
		catalogEntries: append([]speccatalog.Entry(nil), catalogEntries...),
		authProvider:   options.AuthProvider,
	}
}

// FilterCatalogByPrefix filters one catalog slice by operation prefix.
func FilterCatalogByPrefix(entries []speccatalog.Entry, prefix string) []speccatalog.Entry {
	filtered := make([]speccatalog.Entry, 0)
	for _, entry := range entries {
		if strings.HasPrefix(entry.Operation, prefix) {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

// CatalogFromRegistry builds catalog entries dynamically from a runtime registry.
func CatalogFromRegistry(registry *adapter.Registry) []speccatalog.Entry {
	if registry == nil {
		return nil
	}

	definitions := registry.Definitions()
	entries := make([]speccatalog.Entry, 0, len(definitions))
	for _, definition := range definitions {
		entries = append(entries, speccatalog.Entry{
			Operation: definition.Operation,
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Operation < entries[j].Operation
	})
	return entries
}

type registryRuntime struct {
	name           string
	version        string
	platforms      []string
	registry       *adapter.Registry
	catalogEntries []speccatalog.Entry
	authProvider   AuthProvider
}

func (r *registryRuntime) Name() string {
	return r.name
}

func (r *registryRuntime) Handshake(ctx context.Context) (HandshakeResult, error) {
	version := r.version
	if version == "" {
		version = "dev"
	}

	platforms := append([]string(nil), r.platforms...)
	if len(platforms) == 0 && r.name != "" {
		platforms = []string{r.name}
	}

	return HandshakeResult{
		ProtocolVersion: ProtocolVersion,
		Name:            r.name,
		Version:         version,
		Platforms:       platforms,
	}, nil
}

func (r *registryRuntime) ListOperations(ctx context.Context) ([]adapter.Definition, error) {
	definitions := r.registry.Definitions()
	cloned := make([]adapter.Definition, 0, len(definitions))
	for _, definition := range definitions {
		definition.Handler = nil
		cloned = append(cloned, definition)
	}
	return cloned, nil
}

func (r *registryRuntime) GetCatalog(ctx context.Context) ([]speccatalog.Entry, error) {
	return append([]speccatalog.Entry(nil), r.catalogEntries...), nil
}

func (r *registryRuntime) ListAuthMethods(ctx context.Context) ([]AuthMethodDescriptor, error) {
	if r.authProvider == nil {
		return []AuthMethodDescriptor{}, nil
	}
	return r.authProvider.ListMethods(ctx)
}

func (r *registryRuntime) ListAuthPresets(ctx context.Context) ([]AuthPresetDescriptor, error) {
	if r.authProvider == nil {
		return []AuthPresetDescriptor{}, nil
	}
	return r.authProvider.ListPresets(ctx)
}

func (r *registryRuntime) InspectAuth(ctx context.Context, params AuthInspectParams) (AuthInspectResult, error) {
	if r.authProvider == nil {
		return AuthInspectResult{}, fmt.Errorf("provider runtime %s does not expose auth inspection", r.name)
	}
	return r.authProvider.Inspect(ctx, params)
}

func (r *registryRuntime) BeginAuth(ctx context.Context, params AuthBeginParams) (AuthBeginResult, error) {
	if r.authProvider == nil {
		return AuthBeginResult{}, fmt.Errorf("provider runtime %s does not expose auth begin", r.name)
	}
	return r.authProvider.Begin(ctx, params)
}

func (r *registryRuntime) CompleteAuth(ctx context.Context, params AuthCompleteParams) (AuthCompleteResult, error) {
	if r.authProvider == nil {
		return AuthCompleteResult{}, fmt.Errorf("provider runtime %s does not expose auth complete", r.name)
	}
	return r.authProvider.Complete(ctx, params)
}

func (r *registryRuntime) ResolveAuth(ctx context.Context, params AuthResolveParams) (AuthResolveResult, error) {
	if r.authProvider == nil {
		return AuthResolveResult{}, fmt.Errorf("provider runtime %s does not expose auth resolve", r.name)
	}
	return r.authProvider.Resolve(ctx, params)
}

func (r *registryRuntime) Execute(ctx context.Context, req ExecuteRequest) (ExecuteResult, error) {
	definition, ok := r.registry.Resolve(req.Operation)
	if !ok {
		return ExecuteResult{}, fmt.Errorf("operation %s is not registered in provider runtime %s", req.Operation, r.name)
	}
	if definition.Handler == nil {
		return ExecuteResult{
			Error: apperr.New("NOT_IMPLEMENTED", "provider runtime operation is not implemented"),
		}, nil
	}

	ctx = adapter.WithProfileName(ctx, req.AccountName)
	data, appErr := definition.Handler(ctx, adapter.Call{
		ProfileName:    req.AccountName,
		Profile:        req.Profile,
		Input:          req.Input,
		IdempotencyKey: req.IdempotencyKey,
	})
	return ExecuteResult{
		Data:  data,
		Error: appErr,
	}, nil
}

func (r *registryRuntime) Health(ctx context.Context) (HealthResult, error) {
	return HealthResult{
		OK: true,
		Details: map[string]any{
			"name":      r.name,
			"version":   r.version,
			"platforms": append([]string(nil), r.platforms...),
		},
	}, nil
}
