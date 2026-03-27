package plugin

import (
	"context"
	"fmt"
	"strings"

	"github.com/clawrise/clawrise-cli/internal/adapter"
	feishuadapter "github.com/clawrise/clawrise-cli/internal/adapter/feishu"
	notionadapter "github.com/clawrise/clawrise-cli/internal/adapter/notion"
	"github.com/clawrise/clawrise-cli/internal/apperr"
	speccatalog "github.com/clawrise/clawrise-cli/internal/spec/catalog"
)

// NewBuiltinRuntimes creates the current first-party provider runtimes.
func NewBuiltinRuntimes() ([]Runtime, error) {
	feishuClient, err := feishuadapter.NewClient(feishuadapter.Options{})
	if err != nil {
		return nil, err
	}
	notionClient, err := notionadapter.NewClient(notionadapter.Options{})
	if err != nil {
		return nil, err
	}

	feishuRegistry := adapter.NewRegistry()
	feishuadapter.RegisterOperations(feishuRegistry, feishuClient)

	notionRegistry := adapter.NewRegistry()
	notionadapter.RegisterOperations(notionRegistry, notionClient)

	return []Runtime{
		newRegistryRuntime("feishu", feishuRegistry, filterCatalogByPrefix(speccatalog.All(), "feishu.")),
		newRegistryRuntime("notion", notionRegistry, filterCatalogByPrefix(speccatalog.All(), "notion.")),
	}, nil
}

// NewBuiltinManager creates the current aggregated first-party provider manager.
func NewBuiltinManager(ctx context.Context) (*Manager, error) {
	runtimes, err := NewBuiltinRuntimes()
	if err != nil {
		return nil, err
	}
	return NewManager(ctx, runtimes)
}

type registryRuntime struct {
	name           string
	registry       *adapter.Registry
	catalogEntries []speccatalog.Entry
}

func newRegistryRuntime(name string, registry *adapter.Registry, catalogEntries []speccatalog.Entry) Runtime {
	return &registryRuntime{
		name:           name,
		registry:       registry,
		catalogEntries: append([]speccatalog.Entry(nil), catalogEntries...),
	}
}

func (r *registryRuntime) Name() string {
	return r.name
}

func (r *registryRuntime) Handshake(ctx context.Context) (HandshakeResult, error) {
	return HandshakeResult{
		ProtocolVersion: 1,
		Name:            r.name,
		Version:         "builtin",
		Platforms:       []string{r.name},
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

	data, appErr := definition.Handler(ctx, adapter.Call{
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
			"name": r.name,
		},
	}, nil
}

func filterCatalogByPrefix(entries []speccatalog.Entry, prefix string) []speccatalog.Entry {
	filtered := make([]speccatalog.Entry, 0)
	for _, entry := range entries {
		if strings.HasPrefix(entry.Operation, prefix) {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}
