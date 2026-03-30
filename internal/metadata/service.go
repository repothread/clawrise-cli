package metadata

import (
	"path/filepath"

	"github.com/clawrise/clawrise-cli/internal/adapter"
	"github.com/clawrise/clawrise-cli/internal/spec"
	speccatalog "github.com/clawrise/clawrise-cli/internal/spec/catalog"
)

// Service aggregates runtime registry, catalog, and playbook metadata.
type Service struct {
	specService       *spec.Service
	playbookIndexPath string
}

// NewServiceWithCatalog creates a metadata service from an explicit catalog source.
func NewServiceWithCatalog(registry *adapter.Registry, catalogEntries []speccatalog.Entry) *Service {
	return &Service{
		specService:       spec.NewServiceWithCatalog(registry, catalogEntries),
		playbookIndexPath: DefaultPlaybookIndexPath(),
	}
}

// Spec returns the underlying spec service for existing CLI consumers.
func (s *Service) Spec() *spec.Service {
	if s == nil {
		return nil
	}
	return s.specService
}

// CompletionData returns the unified fact set used by shell completion.
func (s *Service) CompletionData() spec.CompletionData {
	if s == nil || s.specService == nil {
		return spec.CompletionData{}
	}
	return s.specService.CompletionData()
}

// ExportMarkdown exports Markdown from the unified metadata layer.
func (s *Service) ExportMarkdown(path string) (string, error) {
	return s.specService.ExportMarkdown(path)
}

// ValidatePlaybooks validates that the playbook index points to real operations and docs.
func (s *Service) ValidatePlaybooks() (PlaybookValidationResult, error) {
	if s == nil || s.specService == nil {
		return PlaybookValidationResult{}, nil
	}
	return ValidatePlaybooksFile(s.playbookIndexPath, s.specService)
}

// DefaultPlaybookIndexPath returns the default playbook index path.
func DefaultPlaybookIndexPath() string {
	return filepath.Join("docs", "playbooks", "index.yaml")
}
