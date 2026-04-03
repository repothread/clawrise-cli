package spec

import (
	"sort"
	"testing"

	"github.com/clawrise/clawrise-cli/internal/adapter"
	feishuadapter "github.com/clawrise/clawrise-cli/internal/adapter/feishu"
	notionadapter "github.com/clawrise/clawrise-cli/internal/adapter/notion"
	speccatalog "github.com/clawrise/clawrise-cli/internal/spec/catalog"
)

func TestServiceListRoot(t *testing.T) {
	registry := newTestRegistry(t)
	service := NewServiceWithCatalog(registry, catalogEntriesFromRegistry(registry))

	result, err := service.List("")
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if result.NodeType != "root" {
		t.Fatalf("unexpected node type: %s", result.NodeType)
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 root items, got %d", len(result.Items))
	}
	if result.Items[0].FullPath != "feishu" {
		t.Fatalf("unexpected first root path: %s", result.Items[0].FullPath)
	}
	if result.Items[1].FullPath != "notion" {
		t.Fatalf("unexpected second root path: %s", result.Items[1].FullPath)
	}
}

func TestServiceListGroup(t *testing.T) {
	registry := newTestRegistry(t)
	service := NewServiceWithCatalog(registry, catalogEntriesFromRegistry(registry))

	result, err := service.List("feishu.docs")
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if result.NodeType != "group" {
		t.Fatalf("unexpected node type: %s", result.NodeType)
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 child groups, got %d", len(result.Items))
	}
	if result.Items[0].FullPath != "feishu.docs.block" {
		t.Fatalf("unexpected first child path: %s", result.Items[0].FullPath)
	}
	if result.Items[1].FullPath != "feishu.docs.document" {
		t.Fatalf("unexpected second child path: %s", result.Items[1].FullPath)
	}
}

func TestServiceGetOperation(t *testing.T) {
	registry := newTestRegistry(t)
	service := NewServiceWithCatalog(registry, catalogEntriesFromRegistry(registry))

	result, err := service.Get("notion.page.create")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if result.Operation != "notion.page.create" {
		t.Fatalf("unexpected operation: %s", result.Operation)
	}
	if result.ResourcePath != "page" {
		t.Fatalf("unexpected resource path: %s", result.ResourcePath)
	}
	if result.Action != "create" {
		t.Fatalf("unexpected action: %s", result.Action)
	}
	if !result.Implemented {
		t.Fatal("expected notion.page.create to be implemented")
	}
	if len(result.Input.Required) != 1 {
		t.Fatalf("unexpected required fields: %+v", result.Input.Required)
	}
}

func TestServiceGetStubbedOperation(t *testing.T) {
	registry := adapter.NewRegistry()
	registry.Register(adapter.Definition{
		Operation:       "demo.page.stubbed",
		Platform:        "demo",
		Mutating:        false,
		AllowedSubjects: []string{"integration"},
		Spec: adapter.OperationSpec{
			Summary: "Stubbed demo operation.",
			Input: adapter.InputSpec{
				Sample: map[string]any{
					"id": "demo",
				},
			},
		},
	})
	service := NewServiceWithCatalog(registry, catalogEntriesFromRegistry(registry))

	result, err := service.Get("demo.page.stubbed")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if result.Implemented {
		t.Fatal("expected feishu.calendar.event.list to be marked as stubbed")
	}
	if result.RuntimeStatus != "registered_but_stubbed" {
		t.Fatalf("unexpected runtime status: %s", result.RuntimeStatus)
	}
}

func TestServiceListRejectsOperationPath(t *testing.T) {
	registry := newTestRegistry(t)
	service := NewServiceWithCatalog(registry, catalogEntriesFromRegistry(registry))

	_, err := service.List("notion.page.create")
	if err == nil {
		t.Fatal("expected List to reject a full operation path")
	}

	specErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("expected spec error, got: %T", err)
	}
	if specErr.Code != "SPEC_PATH_IS_OPERATION" {
		t.Fatalf("unexpected error code: %s", specErr.Code)
	}
}

func newTestRegistry(t *testing.T) *adapter.Registry {
	t.Helper()

	registry := adapter.NewRegistry()

	feishuClient, err := feishuadapter.NewClient(feishuadapter.Options{})
	if err != nil {
		t.Fatalf("failed to construct feishu client: %v", err)
	}
	notionClient, err := notionadapter.NewClient(notionadapter.Options{})
	if err != nil {
		t.Fatalf("failed to construct notion client: %v", err)
	}

	feishuadapter.RegisterOperations(registry, feishuClient)
	notionadapter.RegisterOperations(registry, notionClient)
	return registry
}

func catalogEntriesFromRegistry(registry *adapter.Registry) []speccatalog.Entry {
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
