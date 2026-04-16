package metadata

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadPlaybookIndexReturnsDecodeErrorForInvalidYAML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "index.yaml")
	if err := os.WriteFile(path, []byte("playbooks: ["), 0o644); err != nil {
		t.Fatalf("failed to write invalid yaml: %v", err)
	}

	_, err := LoadPlaybookIndex(path)
	if err == nil {
		t.Fatal("expected invalid yaml to return an error")
	}
	if !strings.Contains(err.Error(), "failed to decode playbook index") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidatePlaybooksFileHandlesEmptyPathAndMissingFile(t *testing.T) {
	service := NewServiceWithCatalog(buildMetadataTestRegistry(), nil)

	result, err := ValidatePlaybooksFile("   ", service.Spec())
	if err != nil {
		t.Fatalf("ValidatePlaybooksFile returned error for empty path: %v", err)
	}
	if !result.OK || result.Path != "" || result.Total != 0 {
		t.Fatalf("unexpected empty-path result: %+v", result)
	}

	missingPath := filepath.Join(t.TempDir(), "missing.yaml")
	result, err = ValidatePlaybooksFile(missingPath, service.Spec())
	if err != nil {
		t.Fatalf("ValidatePlaybooksFile returned error for missing file: %v", err)
	}
	if !result.OK || !result.MissingFile || result.Path != missingPath {
		t.Fatalf("unexpected missing-file result: %+v", result)
	}
}

func TestValidatePlaybooksFileReportsMultipleValidationIssues(t *testing.T) {
	rootDir := t.TempDir()
	indexPath := filepath.Join(rootDir, "index.yaml")
	indexYAML := `version: 1
updated_at: "2026-04-10"
playbooks:
  - id: demo-page
    title: Demo 1
    title_en: Demo 1
    platform: other
    operations:
      - demo.page.get
      - other.page.missing
      - "   "
    zh_path: ""
    en_path: en/missing.md
  - id: demo-page
    title: Demo 2
    title_en: Demo 2
    platform: ""
    operations: []
    zh_path: zh/missing.md
    en_path: ""
`
	if err := os.WriteFile(indexPath, []byte(indexYAML), 0o644); err != nil {
		t.Fatalf("failed to write playbook index: %v", err)
	}

	service := NewServiceWithCatalog(buildMetadataTestRegistry(), nil)
	result, err := ValidatePlaybooksFile(indexPath, service.Spec())
	if err != nil {
		t.Fatalf("ValidatePlaybooksFile returned error: %v", err)
	}
	if result.OK {
		t.Fatalf("expected invalid validation result, got %+v", result)
	}
	if result.Total != 2 || result.ValidCount != 0 || result.InvalidCount != 2 {
		t.Fatalf("unexpected validation counters: %+v", result)
	}
	if len(result.PlaybookIDs) != 2 || result.PlaybookIDs[0] != "demo-page" || result.PlaybookIDs[1] != "demo-page" {
		t.Fatalf("unexpected playbook ids: %+v", result.PlaybookIDs)
	}

	issueCodes := make(map[string]int)
	for _, issue := range result.Issues {
		issueCodes[issue.Code]++
	}
	for _, code := range []string{
		"PLAYBOOK_PLATFORM_MISMATCH",
		"PLAYBOOK_OPERATION_NOT_FOUND",
		"PLAYBOOK_OPERATION_EMPTY",
		"PLAYBOOK_DOC_PATH_REQUIRED",
		"PLAYBOOK_DOC_NOT_FOUND",
		"PLAYBOOK_ID_DUPLICATED",
		"PLAYBOOK_PLATFORM_REQUIRED",
		"PLAYBOOK_OPERATIONS_REQUIRED",
	} {
		if issueCodes[code] == 0 {
			t.Fatalf("expected issue code %s in %+v", code, result.Issues)
		}
	}
}

func TestValidatePlaybooksFileAllowsProjectRootRelativeDocs(t *testing.T) {
	rootDir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned error: %v", err)
	}
	if err := os.Chdir(rootDir); err != nil {
		t.Fatalf("Chdir returned error: %v", err)
	}
	defer func() {
		_ = os.Chdir(cwd)
	}()

	if err := os.MkdirAll(filepath.Join(rootDir, "docs", "playbooks", "zh"), 0o755); err != nil {
		t.Fatalf("failed to create zh dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(rootDir, "docs", "playbooks", "en"), 0o755); err != nil {
		t.Fatalf("failed to create en dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rootDir, "docs", "playbooks", "zh", "demo.md"), []byte("# zh\n"), 0o644); err != nil {
		t.Fatalf("failed to write zh doc: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rootDir, "docs", "playbooks", "en", "demo.md"), []byte("# en\n"), 0o644); err != nil {
		t.Fatalf("failed to write en doc: %v", err)
	}

	indexPath := filepath.Join(rootDir, "metadata", "index.yaml")
	if err := os.MkdirAll(filepath.Dir(indexPath), 0o755); err != nil {
		t.Fatalf("failed to create metadata dir: %v", err)
	}
	indexYAML := `version: 1
updated_at: "2026-04-10"
playbooks:
  - id: demo-root-relative
    title: Demo
    title_en: Demo
    platform: demo
    operations:
      - demo.page.get
    zh_path: docs/playbooks/zh/demo.md
    en_path: docs/playbooks/en/demo.md
`
	if err := os.WriteFile(indexPath, []byte(indexYAML), 0o644); err != nil {
		t.Fatalf("failed to write playbook index: %v", err)
	}

	service := NewServiceWithCatalog(buildMetadataTestRegistry(), nil)
	result, err := ValidatePlaybooksFile(indexPath, service.Spec())
	if err != nil {
		t.Fatalf("ValidatePlaybooksFile returned error: %v", err)
	}
	if !result.OK || result.ValidCount != 1 || result.InvalidCount != 0 {
		t.Fatalf("unexpected root-relative validation result: %+v", result)
	}
}

func TestServiceNilHelpersAndValidatePlaybooks(t *testing.T) {
	var nilService *Service
	if nilService.Spec() != nil {
		t.Fatal("expected nil service Spec to return nil")
	}
	if got := nilService.CompletionData(); len(got.Operations) != 0 || len(got.SpecPaths) != 0 {
		t.Fatalf("expected nil service CompletionData to be zero value, got %+v", got)
	}

	emptyService := &Service{}
	result, err := emptyService.ValidatePlaybooks()
	if err != nil {
		t.Fatalf("ValidatePlaybooks returned error for empty service: %v", err)
	}
	if result.OK || result.Path != "" || result.Total != 0 || result.ValidCount != 0 || result.InvalidCount != 0 || result.MissingFile || len(result.Issues) != 0 || len(result.PlaybookIDs) != 0 {
		t.Fatalf("expected empty service ValidatePlaybooks to return zero result, got %+v", result)
	}
	if DefaultPlaybookIndexPath() != filepath.Join("docs", "playbooks", "index.yaml") {
		t.Fatalf("unexpected default playbook index path: %s", DefaultPlaybookIndexPath())
	}
}
