package plugin

import (
	"context"
	"testing"
	"time"

	"github.com/clawrise/clawrise-cli/internal/adapter"
	"github.com/clawrise/clawrise-cli/internal/apperr"
	"github.com/clawrise/clawrise-cli/internal/config"
	speccatalog "github.com/clawrise/clawrise-cli/internal/spec/catalog"
)

func TestManagerRegistersOperationsThroughRuntimeBoundary(t *testing.T) {
	runtime := newRegistryRuntime("demo", buildDemoRegistry(), []speccatalog.Entry{
		{Operation: "demo.page.get"},
	})

	manager, err := NewManager(context.Background(), []Runtime{runtime})
	if err != nil {
		t.Fatalf("NewManager returned error: %v", err)
	}

	definition, ok := manager.Registry().Resolve("demo.page.get")
	if !ok {
		t.Fatal("expected demo.page.get to be registered")
	}
	if definition.Handler == nil {
		t.Fatal("expected aggregated definition handler to be wired")
	}

	data, appErr := definition.Handler(context.Background(), adapter.Call{
		Profile: config.Profile{
			Platform: "demo",
			Subject:  "integration",
		},
		Input: map[string]any{
			"id": "page_demo",
		},
	})
	if appErr != nil {
		t.Fatalf("expected successful execution, got: %+v", appErr)
	}
	if data["id"] != "page_demo" {
		t.Fatalf("unexpected handler data: %+v", data)
	}
}

func TestManagerAggregatesCatalogEntries(t *testing.T) {
	manager, err := NewManager(context.Background(), []Runtime{
		newRegistryRuntime("demo", buildDemoRegistry(), []speccatalog.Entry{
			{Operation: "demo.page.get"},
		}),
	})
	if err != nil {
		t.Fatalf("NewManager returned error: %v", err)
	}

	entries := manager.CatalogEntries()
	if len(entries) != 1 || entries[0].Operation != "demo.page.get" {
		t.Fatalf("unexpected catalog entries: %+v", entries)
	}
}

func buildDemoRegistry() *adapter.Registry {
	registry := adapter.NewRegistry()
	registry.Register(adapter.Definition{
		Operation:       "demo.page.get",
		Platform:        "demo",
		Mutating:        false,
		DefaultTimeout:  time.Second,
		AllowedSubjects: []string{"integration"},
		Spec: adapter.OperationSpec{
			Summary: "Get one demo page.",
			Input: adapter.InputSpec{
				Sample: map[string]any{
					"id": "page_demo",
				},
			},
		},
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return map[string]any{
				"id": call.Input["id"],
			}, nil
		},
	})
	return registry
}
