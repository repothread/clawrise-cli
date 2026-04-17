package metadata

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/clawrise/clawrise-cli/internal/adapter"
	feishuadapter "github.com/clawrise/clawrise-cli/internal/adapter/feishu"
	notionadapter "github.com/clawrise/clawrise-cli/internal/adapter/notion"
)

func TestValidatePlaybooksFilePassesForRegisteredOperations(t *testing.T) {
	rootDir := t.TempDir()
	playbookDir := filepath.Join(rootDir, "docs", "playbooks")
	if err := os.MkdirAll(filepath.Join(playbookDir, "zh"), 0o755); err != nil {
		t.Fatalf("failed to create zh playbook dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(playbookDir, "en"), 0o755); err != nil {
		t.Fatalf("failed to create en playbook dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(playbookDir, "zh", "demo-page.md"), []byte("# demo zh\n"), 0o644); err != nil {
		t.Fatalf("failed to write zh playbook doc: %v", err)
	}
	if err := os.WriteFile(filepath.Join(playbookDir, "en", "demo-page.md"), []byte("# demo en\n"), 0o644); err != nil {
		t.Fatalf("failed to write en playbook doc: %v", err)
	}

	indexPath := filepath.Join(playbookDir, "index.yaml")
	indexYAML := `version: 1
updated_at: "2026-03-30"
playbooks:
  - id: demo-page
    title: Demo Page
    title_en: Demo Page
    platform: demo
    operations:
      - demo.page.get
    zh_path: zh/demo-page.md
    en_path: en/demo-page.md
`
	if err := os.WriteFile(indexPath, []byte(indexYAML), 0o644); err != nil {
		t.Fatalf("failed to write playbook index: %v", err)
	}

	service := NewServiceWithCatalog(buildMetadataTestRegistry(), nil)
	result, err := ValidatePlaybooksFile(indexPath, service.Spec())
	if err != nil {
		t.Fatalf("ValidatePlaybooksFile returned error: %v", err)
	}
	if !result.OK || result.InvalidCount != 0 || result.ValidCount != 1 {
		t.Fatalf("unexpected validation result: %+v", result)
	}
}

func TestValidatePlaybooksFileDetectsUnknownOperations(t *testing.T) {
	rootDir := t.TempDir()
	indexPath := filepath.Join(rootDir, "index.yaml")
	indexYAML := `version: 1
updated_at: "2026-03-30"
playbooks:
  - id: broken-playbook
    title: Broken
    title_en: Broken
    platform: demo
    operations:
      - demo.page.missing
    zh_path: zh/missing.md
    en_path: en/missing.md
`
	if err := os.WriteFile(indexPath, []byte(indexYAML), 0o644); err != nil {
		t.Fatalf("failed to write playbook index: %v", err)
	}

	service := NewServiceWithCatalog(buildMetadataTestRegistry(), nil)
	result, err := ValidatePlaybooksFile(indexPath, service.Spec())
	if err != nil {
		t.Fatalf("ValidatePlaybooksFile returned error: %v", err)
	}
	if result.OK || result.InvalidCount != 1 {
		t.Fatalf("unexpected validation result: %+v", result)
	}
	if len(result.Issues) == 0 || !strings.Contains(result.Issues[0].Code, "PLAYBOOK_OPERATION_NOT_FOUND") {
		t.Fatalf("expected missing operation issue, got: %+v", result.Issues)
	}
}

func TestValidateProjectPlaybooksIndex(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve current test file path")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned error: %v", err)
	}
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatalf("Chdir returned error: %v", err)
	}
	defer func() {
		_ = os.Chdir(cwd)
	}()

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

	service := NewServiceWithCatalog(registry, nil)
	result, err := service.ValidatePlaybooks()
	if err != nil {
		t.Fatalf("ValidatePlaybooks returned error: %v", err)
	}
	if !result.OK || result.InvalidCount != 0 {
		t.Fatalf("expected project playbooks to validate cleanly, got %+v", result)
	}
}

func buildMetadataTestRegistry() *adapter.Registry {
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
	})
	return registry
}
