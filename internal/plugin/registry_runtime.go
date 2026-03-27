package plugin

import (
	"context"
	"fmt"
	"strings"

	"github.com/clawrise/clawrise-cli/internal/adapter"
	"github.com/clawrise/clawrise-cli/internal/apperr"
	speccatalog "github.com/clawrise/clawrise-cli/internal/spec/catalog"
)

// NewRegistryRuntime creates one runtime backed by an in-process adapter registry.
func NewRegistryRuntime(name, version string, platforms []string, registry *adapter.Registry, catalogEntries []speccatalog.Entry) Runtime {
	return &registryRuntime{
		name:           name,
		version:        version,
		platforms:      append([]string(nil), platforms...),
		registry:       registry,
		catalogEntries: append([]speccatalog.Entry(nil), catalogEntries...),
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

type registryRuntime struct {
	name           string
	version        string
	platforms      []string
	registry       *adapter.Registry
	catalogEntries []speccatalog.Entry
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

	ctx = adapter.WithProfileName(ctx, req.ProfileName)
	data, appErr := definition.Handler(ctx, adapter.Call{
		ProfileName:    req.ProfileName,
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
