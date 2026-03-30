package spec

import (
	"strings"
	"testing"

	"github.com/clawrise/clawrise-cli/internal/adapter"
	speccatalog "github.com/clawrise/clawrise-cli/internal/spec/catalog"
)

func TestServiceExportReturnsStructuredResult(t *testing.T) {
	registry := adapter.NewRegistry()
	registry.Register(newStatusTestDefinition("demo.page.get", true))
	registry.Register(newStatusTestDefinition("demo.page.update", false))
	registry.Register(newStatusTestDefinition("demo.page.search", true))

	service := newServiceWithCatalog(registry, []speccatalog.Entry{
		{Operation: "demo.page.get"},
		{Operation: "demo.page.update"},
		{Operation: "demo.page.delete"},
	})

	result, err := service.Export("demo.page")
	if err != nil {
		t.Fatalf("Export returned error: %v", err)
	}

	if result.Path != "demo.page" {
		t.Fatalf("unexpected export path: %s", result.Path)
	}
	if result.Summary.Runtime.RegisteredCount != 3 {
		t.Fatalf("unexpected runtime registered count: %d", result.Summary.Runtime.RegisteredCount)
	}
	if result.Summary.Catalog.DeclaredCount != 3 {
		t.Fatalf("unexpected catalog declared count: %d", result.Summary.Catalog.DeclaredCount)
	}
	if result.Summary.ExportedOperationCount != 3 {
		t.Fatalf("unexpected exported operation count: %d", result.Summary.ExportedOperationCount)
	}
	if len(result.RegisteredButStubbed) != 1 || result.RegisteredButStubbed[0].Operation != "demo.page.update" {
		t.Fatalf("unexpected registered but stubbed items: %+v", result.RegisteredButStubbed)
	}
	if len(result.RuntimePresentButCatalogMissing) != 1 || result.RuntimePresentButCatalogMissing[0].Operation != "demo.page.search" {
		t.Fatalf("unexpected runtime present but catalog missing items: %+v", result.RuntimePresentButCatalogMissing)
	}
	if len(result.CatalogDeclaredButRuntimeMissing) != 1 || result.CatalogDeclaredButRuntimeMissing[0].Operation != "demo.page.delete" {
		t.Fatalf("unexpected catalog declared but runtime missing items: %+v", result.CatalogDeclaredButRuntimeMissing)
	}
	if result.Operations[0].Operation != "demo.page.get" {
		t.Fatalf("unexpected first exported operation: %+v", result.Operations[0])
	}
}

func TestServiceExportSupportsCatalogOnlyOperationPath(t *testing.T) {
	registry := adapter.NewRegistry()
	registry.Register(newStatusTestDefinition("demo.page.get", true))

	service := newServiceWithCatalog(registry, []speccatalog.Entry{
		{Operation: "demo.page.get"},
		{Operation: "demo.page.delete"},
	})

	result, err := service.Export("demo.page.delete")
	if err != nil {
		t.Fatalf("Export returned error: %v", err)
	}

	if result.NodeType != "operation" {
		t.Fatalf("unexpected node type: %s", result.NodeType)
	}
	if len(result.Operations) != 0 {
		t.Fatalf("expected no runtime operations, got: %+v", result.Operations)
	}
	if len(result.CatalogDeclaredButRuntimeMissing) != 1 || result.CatalogDeclaredButRuntimeMissing[0].Operation != "demo.page.delete" {
		t.Fatalf("unexpected catalog-only result: %+v", result.CatalogDeclaredButRuntimeMissing)
	}
}

func TestServiceExportMarkdownUsesSameMetadata(t *testing.T) {
	registry := newTestRegistry(t)
	service := NewServiceWithCatalog(registry, catalogEntriesFromRegistry(registry))

	document, err := service.ExportMarkdown("notion.page.create")
	if err != nil {
		t.Fatalf("ExportMarkdown returned error: %v", err)
	}

	if !strings.Contains(document, "# Clawrise Spec Export") {
		t.Fatalf("expected export title in markdown, got: %s", document)
	}
	if !strings.Contains(document, "## `notion.page.create`") {
		t.Fatalf("expected operation heading in markdown, got: %s", document)
	}
	if !strings.Contains(document, "### Sample Input") {
		t.Fatalf("expected sample input section in markdown, got: %s", document)
	}
}

func TestServiceCompletionDataIncludesOperationsAndPaths(t *testing.T) {
	registry := adapter.NewRegistry()
	registry.Register(newStatusTestDefinition("demo.page.get", true))

	service := newServiceWithCatalog(registry, []speccatalog.Entry{
		{Operation: "demo.page.get"},
		{Operation: "demo.page.delete"},
	})

	data := service.CompletionData()
	if len(data.Operations) != 1 || data.Operations[0] != "demo.page.get" {
		t.Fatalf("unexpected operations completion data: %+v", data.Operations)
	}

	expectedPaths := map[string]bool{
		"demo":             false,
		"demo.page":        false,
		"demo.page.get":    false,
		"demo.page.delete": false,
	}
	for _, path := range data.SpecPaths {
		if _, ok := expectedPaths[path]; ok {
			expectedPaths[path] = true
		}
	}
	for path, found := range expectedPaths {
		if !found {
			t.Fatalf("expected spec path %s in completion data, got: %+v", path, data.SpecPaths)
		}
	}
}
