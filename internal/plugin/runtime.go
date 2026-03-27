package plugin

import (
	"context"
	"fmt"

	"github.com/clawrise/clawrise-cli/internal/adapter"
	"github.com/clawrise/clawrise-cli/internal/apperr"
	"github.com/clawrise/clawrise-cli/internal/config"
	speccatalog "github.com/clawrise/clawrise-cli/internal/spec/catalog"
)

// Runtime describes one provider runtime backend.
type Runtime interface {
	Name() string
	Handshake(ctx context.Context) (HandshakeResult, error)
	ListOperations(ctx context.Context) ([]adapter.Definition, error)
	GetCatalog(ctx context.Context) ([]speccatalog.Entry, error)
	Execute(ctx context.Context, req ExecuteRequest) (ExecuteResult, error)
	Health(ctx context.Context) (HealthResult, error)
}

// HandshakeResult describes one provider runtime handshake result.
type HandshakeResult struct {
	ProtocolVersion int      `json:"protocol_version"`
	Name            string   `json:"name"`
	Version         string   `json:"version"`
	Platforms       []string `json:"platforms"`
}

// ExecuteRequest describes one normalized provider execution request.
type ExecuteRequest struct {
	Operation      string
	Profile        config.Profile
	Input          map[string]any
	IdempotencyKey string
}

// ExecuteResult describes one provider execution result.
type ExecuteResult struct {
	Data  map[string]any
	Error *apperr.AppError
}

// HealthResult describes one provider runtime health result.
type HealthResult struct {
	OK      bool           `json:"ok"`
	Details map[string]any `json:"details,omitempty"`
}

// Manager aggregates provider runtimes into one execution and discovery view.
type Manager struct {
	registry          *adapter.Registry
	catalogEntries    []speccatalog.Entry
	operationRuntimes map[string]Runtime
}

// NewManager creates one aggregated provider runtime manager.
func NewManager(ctx context.Context, runtimes []Runtime) (*Manager, error) {
	registry := adapter.NewRegistry()
	operationRuntimes := make(map[string]Runtime)
	catalogEntries := make([]speccatalog.Entry, 0)

	for _, runtime := range runtimes {
		handshake, err := runtime.Handshake(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to handshake provider runtime %s: %w", runtime.Name(), err)
		}
		if handshake.Name == "" {
			return nil, fmt.Errorf("provider runtime %s returned an empty handshake name", runtime.Name())
		}

		definitions, err := runtime.ListOperations(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list operations for provider runtime %s: %w", runtime.Name(), err)
		}
		for _, definition := range definitions {
			if _, exists := operationRuntimes[definition.Operation]; exists {
				return nil, fmt.Errorf("duplicate operation registered by provider runtimes: %s", definition.Operation)
			}

			runtimeRef := runtime
			operation := definition.Operation
			definition.Handler = func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
				result, err := runtimeRef.Execute(ctx, ExecuteRequest{
					Operation:      operation,
					Profile:        call.Profile,
					Input:          call.Input,
					IdempotencyKey: call.IdempotencyKey,
				})
				if err != nil {
					return nil, apperr.New("PROVIDER_RUNTIME_FAILED", err.Error())
				}
				return result.Data, result.Error
			}

			registry.Register(definition)
			operationRuntimes[definition.Operation] = runtime
		}

		entries, err := runtime.GetCatalog(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to load catalog for provider runtime %s: %w", runtime.Name(), err)
		}
		catalogEntries = append(catalogEntries, entries...)
	}

	return &Manager{
		registry:          registry,
		catalogEntries:    catalogEntries,
		operationRuntimes: operationRuntimes,
	}, nil
}

// Registry returns the aggregated operation registry view.
func (m *Manager) Registry() *adapter.Registry {
	return m.registry
}

// CatalogEntries returns the aggregated structured catalog view.
func (m *Manager) CatalogEntries() []speccatalog.Entry {
	return append([]speccatalog.Entry(nil), m.catalogEntries...)
}
