package spec

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/clawrise/clawrise-cli/internal/adapter"
	"github.com/clawrise/clawrise-cli/internal/apperr"
	speccatalog "github.com/clawrise/clawrise-cli/internal/spec/catalog"
)

func TestServiceStatusReportsRuntimeAndCatalogDrift(t *testing.T) {
	registry := adapter.NewRegistry()
	registry.Register(newStatusTestDefinition("demo.page.get", true))
	registry.Register(newStatusTestDefinition("demo.page.update", false))
	registry.Register(newStatusTestDefinition("demo.page.search", true))

	service := newServiceWithCatalog(registry, []speccatalog.Entry{
		{Operation: "demo.page.get"},
		{Operation: "demo.page.update"},
		{Operation: "demo.page.delete"},
	})

	result, err := service.Status()
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	if result.Summary.Runtime.RegisteredCount != 3 {
		t.Fatalf("unexpected runtime registered count: %d", result.Summary.Runtime.RegisteredCount)
	}
	if result.Summary.Runtime.ImplementedCount != 2 {
		t.Fatalf("unexpected runtime implemented count: %d", result.Summary.Runtime.ImplementedCount)
	}
	if result.Summary.Runtime.StubbedCount != 1 {
		t.Fatalf("unexpected runtime stubbed count: %d", result.Summary.Runtime.StubbedCount)
	}
	if result.Summary.Catalog.DeclaredCount != 3 {
		t.Fatalf("unexpected catalog declared count: %d", result.Summary.Catalog.DeclaredCount)
	}
	if len(result.RegisteredButStubbed) != 1 || result.RegisteredButStubbed[0].Operation != "demo.page.update" {
		t.Fatalf("unexpected registered_but_stubbed: %+v", result.RegisteredButStubbed)
	}
	if len(result.CatalogDeclaredButRuntimeMissing) != 1 || result.CatalogDeclaredButRuntimeMissing[0].Operation != "demo.page.delete" {
		t.Fatalf("unexpected catalog_declared_but_runtime_missing: %+v", result.CatalogDeclaredButRuntimeMissing)
	}
	if len(result.RuntimePresentButCatalogMissing) != 1 || result.RuntimePresentButCatalogMissing[0].Operation != "demo.page.search" {
		t.Fatalf("unexpected runtime_present_but_catalog_missing: %+v", result.RuntimePresentButCatalogMissing)
	}
}

func TestExplicitCatalogCoversRegisteredOperations(t *testing.T) {
	registry := newTestRegistry(t)
	service := NewServiceWithCatalog(registry, catalogEntriesFromRegistry(registry))

	result, err := service.Status()
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	if len(result.RuntimePresentButCatalogMissing) != 0 {
		t.Fatalf("expected catalog to cover all registered operations, got: %+v", result.RuntimePresentButCatalogMissing)
	}
}

func TestRegisteredOperationsHaveRequiredMetadata(t *testing.T) {
	registry := newTestRegistry(t)

	for _, definition := range registry.Definitions() {
		if strings.TrimSpace(definition.Operation) == "" {
			t.Fatal("found operation with empty name")
		}
		if strings.TrimSpace(definition.Platform) == "" {
			t.Fatalf("operation %s has empty platform", definition.Operation)
		}
		if !strings.HasPrefix(definition.Operation, definition.Platform+".") {
			t.Fatalf("operation %s does not match platform %s", definition.Operation, definition.Platform)
		}
		if len(definition.AllowedSubjects) == 0 {
			t.Fatalf("operation %s is missing allowed subjects", definition.Operation)
		}
		if definition.DefaultTimeout <= 0 {
			t.Fatalf("operation %s is missing default timeout", definition.Operation)
		}
		if strings.TrimSpace(definition.Spec.Summary) == "" {
			t.Fatalf("operation %s is missing summary metadata", definition.Operation)
		}
		if definition.Spec.Input.Sample == nil {
			t.Fatalf("operation %s is missing input sample metadata", definition.Operation)
		}
		if definition.Mutating && !definition.Spec.Idempotency.Required {
			t.Fatalf("mutating operation %s is missing idempotency metadata", definition.Operation)
		}
	}
}

func newStatusTestDefinition(operation string, implemented bool) adapter.Definition {
	definition := adapter.Definition{
		Operation:       operation,
		Platform:        "demo",
		Mutating:        strings.HasSuffix(operation, ".update"),
		DefaultTimeout:  time.Second,
		AllowedSubjects: []string{"integration"},
		Spec: adapter.OperationSpec{
			Summary: "Demo operation.",
			Input: adapter.InputSpec{
				Sample: map[string]any{
					"id": "demo",
				},
			},
		},
	}

	if implemented {
		definition.Handler = func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return map[string]any{"ok": true}, nil
		}
	}
	return definition
}
