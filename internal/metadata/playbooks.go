package metadata

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/clawrise/clawrise-cli/internal/spec"
	"gopkg.in/yaml.v3"
)

// PlaybookIndex describes the playbook index file format.
type PlaybookIndex struct {
	Version   int             `yaml:"version"`
	UpdatedAt string          `yaml:"updated_at"`
	Playbooks []PlaybookEntry `yaml:"playbooks"`
}

// PlaybookEntry describes one playbook declaration.
type PlaybookEntry struct {
	ID         string   `yaml:"id"`
	Title      string   `yaml:"title"`
	TitleEN    string   `yaml:"title_en"`
	Platform   string   `yaml:"platform"`
	Operations []string `yaml:"operations"`
	ZHPath     string   `yaml:"zh_path"`
	ENPath     string   `yaml:"en_path"`
}

// PlaybookIssue describes one playbook validation issue.
type PlaybookIssue struct {
	PlaybookID string `json:"playbook_id"`
	Code       string `json:"code"`
	Message    string `json:"message"`
}

// PlaybookValidationResult describes the result of playbook validation.
type PlaybookValidationResult struct {
	OK           bool            `json:"ok"`
	Path         string          `json:"path"`
	Total        int             `json:"total"`
	ValidCount   int             `json:"valid_count"`
	InvalidCount int             `json:"invalid_count"`
	MissingFile  bool            `json:"missing_file,omitempty"`
	Issues       []PlaybookIssue `json:"issues,omitempty"`
	PlaybookIDs  []string        `json:"playbook_ids,omitempty"`
}

// LoadPlaybookIndex reads and decodes one playbook index file.
func LoadPlaybookIndex(path string) (PlaybookIndex, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return PlaybookIndex{}, err
	}

	var index PlaybookIndex
	if err := yaml.Unmarshal(data, &index); err != nil {
		return PlaybookIndex{}, fmt.Errorf("failed to decode playbook index: %w", err)
	}
	return index, nil
}

// ValidatePlaybooksFile validates that the playbook index points to real operations and document paths.
func ValidatePlaybooksFile(path string, specService *spec.Service) (PlaybookValidationResult, error) {
	result := PlaybookValidationResult{
		OK:   true,
		Path: strings.TrimSpace(path),
	}
	if strings.TrimSpace(path) == "" {
		return result, nil
	}

	index, err := LoadPlaybookIndex(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			result.MissingFile = true
			return result, nil
		}
		return PlaybookValidationResult{}, err
	}

	rootDir := filepath.Dir(path)
	seenIDs := map[string]struct{}{}
	for _, playbook := range index.Playbooks {
		result.Total++
		result.PlaybookIDs = append(result.PlaybookIDs, strings.TrimSpace(playbook.ID))

		issueCountBefore := len(result.Issues)
		playbookID := strings.TrimSpace(playbook.ID)
		if playbookID == "" {
			result.Issues = append(result.Issues, PlaybookIssue{
				PlaybookID: playbookID,
				Code:       "PLAYBOOK_ID_REQUIRED",
				Message:    "playbook id is required",
			})
		} else if _, exists := seenIDs[playbookID]; exists {
			result.Issues = append(result.Issues, PlaybookIssue{
				PlaybookID: playbookID,
				Code:       "PLAYBOOK_ID_DUPLICATED",
				Message:    fmt.Sprintf("duplicated playbook id: %s", playbookID),
			})
		} else {
			seenIDs[playbookID] = struct{}{}
		}

		if strings.TrimSpace(playbook.Platform) == "" {
			result.Issues = append(result.Issues, PlaybookIssue{
				PlaybookID: playbookID,
				Code:       "PLAYBOOK_PLATFORM_REQUIRED",
				Message:    "playbook platform is required",
			})
		}
		if len(playbook.Operations) == 0 {
			result.Issues = append(result.Issues, PlaybookIssue{
				PlaybookID: playbookID,
				Code:       "PLAYBOOK_OPERATIONS_REQUIRED",
				Message:    "playbook operations must not be empty",
			})
		}

		for _, operation := range playbook.Operations {
			operation = strings.TrimSpace(operation)
			if operation == "" {
				result.Issues = append(result.Issues, PlaybookIssue{
					PlaybookID: playbookID,
					Code:       "PLAYBOOK_OPERATION_EMPTY",
					Message:    "playbook operation must not be empty",
				})
				continue
			}
			if _, err := specService.Get(operation); err != nil {
				result.Issues = append(result.Issues, PlaybookIssue{
					PlaybookID: playbookID,
					Code:       "PLAYBOOK_OPERATION_NOT_FOUND",
					Message:    fmt.Sprintf("playbook operation %s is not registered", operation),
				})
				continue
			}
			if platform := strings.TrimSpace(playbook.Platform); platform != "" && !strings.HasPrefix(operation, platform+".") {
				result.Issues = append(result.Issues, PlaybookIssue{
					PlaybookID: playbookID,
					Code:       "PLAYBOOK_PLATFORM_MISMATCH",
					Message:    fmt.Sprintf("operation %s does not match playbook platform %s", operation, platform),
				})
			}
		}

		for _, documentPath := range []string{playbook.ZHPath, playbook.ENPath} {
			documentPath = strings.TrimSpace(documentPath)
			if documentPath == "" {
				result.Issues = append(result.Issues, PlaybookIssue{
					PlaybookID: playbookID,
					Code:       "PLAYBOOK_DOC_PATH_REQUIRED",
					Message:    "playbook document path is required",
				})
				continue
			}
			resolvedPath := documentPath
			if !filepath.IsAbs(resolvedPath) {
				if _, err := os.Stat(resolvedPath); err == nil {
					// Allow project-root-relative paths to preserve the current docs/playbooks/... layout.
				} else {
					resolvedPath = filepath.Join(rootDir, documentPath)
				}
			}
			if _, err := os.Stat(resolvedPath); err != nil {
				result.Issues = append(result.Issues, PlaybookIssue{
					PlaybookID: playbookID,
					Code:       "PLAYBOOK_DOC_NOT_FOUND",
					Message:    fmt.Sprintf("playbook document %s does not exist", documentPath),
				})
			}
		}

		if len(result.Issues) == issueCountBefore {
			result.ValidCount++
		}
	}

	result.InvalidCount = result.Total - result.ValidCount
	result.OK = result.InvalidCount == 0
	return result, nil
}
