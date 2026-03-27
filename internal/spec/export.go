package spec

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/clawrise/clawrise-cli/internal/adapter"
	speccatalog "github.com/clawrise/clawrise-cli/internal/spec/catalog"
)

// Export 返回当前 spec 的结构化导出结果。
func (s *Service) Export(path string) (ExportResult, error) {
	path, parts, err := normalizePath(path)
	if err != nil {
		return ExportResult{}, err
	}

	catalogIndex, err := speccatalog.Index(s.catalogEntries)
	if err != nil {
		return ExportResult{}, newError("INVALID_SPEC_CATALOG", err.Error())
	}

	result := ExportResult{
		Path:     path,
		NodeType: nodeTypeForExportPath(parts, s.registry, catalogIndex),
	}

	runtimeDefinitions := s.registry.Definitions()
	runtimeIndex := make(map[string]struct{})
	catalogMatches := make(map[string]speccatalog.Entry)
	hasMatch := path == ""

	for _, entry := range s.catalogEntries {
		if !matchesSpecPath(entry.Operation, parts) {
			continue
		}
		hasMatch = true
		catalogMatches[entry.Operation] = entry
	}

	for _, definition := range runtimeDefinitions {
		if !matchesSpecPath(definition.Operation, parts) {
			continue
		}

		hasMatch = true
		runtimeIndex[definition.Operation] = struct{}{}
		result.Summary.Runtime.RegisteredCount++

		catalogStatus := catalogStatusForOperation(definition.Operation, catalogIndex)
		if definition.Handler == nil {
			result.Summary.Runtime.StubbedCount++
			result.RegisteredButStubbed = append(result.RegisteredButStubbed, StatusItem{
				Operation:     definition.Operation,
				RuntimeStatus: runtimeStatusForDefinition(definition),
				CatalogStatus: catalogStatus,
			})
		} else {
			result.Summary.Runtime.ImplementedCount++
		}

		if catalogStatus == "catalog_missing" {
			result.RuntimePresentButCatalogMissing = append(result.RuntimePresentButCatalogMissing, StatusItem{
				Operation:     definition.Operation,
				RuntimeStatus: runtimeStatusForDefinition(definition),
				CatalogStatus: catalogStatus,
			})
		}

		result.Operations = append(result.Operations, OperationExport{
			OperationView: buildOperationView(definition),
			CatalogStatus: catalogStatus,
		})
	}

	for operation := range catalogMatches {
		if _, ok := runtimeIndex[operation]; ok {
			continue
		}
		result.CatalogDeclaredButRuntimeMissing = append(result.CatalogDeclaredButRuntimeMissing, StatusItem{
			Operation:     operation,
			RuntimeStatus: "runtime_missing",
			CatalogStatus: "declared",
		})
	}

	if !hasMatch {
		return ExportResult{}, newError("SPEC_PATH_NOT_FOUND", fmt.Sprintf("spec path %s does not exist", path))
	}

	result.Summary.Catalog.DeclaredCount = len(catalogMatches)
	result.Summary.ExportedOperationCount = len(result.Operations)
	result.Summary.Issues.RegisteredButStubbedCount = len(result.RegisteredButStubbed)
	result.Summary.Issues.CatalogDeclaredButRuntimeMissingCount = len(result.CatalogDeclaredButRuntimeMissing)
	result.Summary.Issues.RuntimePresentButCatalogMissingCount = len(result.RuntimePresentButCatalogMissing)

	sort.Slice(result.Operations, func(i, j int) bool {
		return result.Operations[i].Operation < result.Operations[j].Operation
	})
	sortStatusItems(result.RegisteredButStubbed)
	sortStatusItems(result.CatalogDeclaredButRuntimeMissing)
	sortStatusItems(result.RuntimePresentButCatalogMissing)
	return result, nil
}

// ExportMarkdown 将结构化导出结果渲染成最小可用的 Markdown 文档。
func (s *Service) ExportMarkdown(path string) (string, error) {
	result, err := s.Export(path)
	if err != nil {
		return "", err
	}

	buffer := &bytes.Buffer{}
	_, _ = fmt.Fprintln(buffer, "# Clawrise Spec Export")
	_, _ = fmt.Fprintln(buffer)

	if result.Path == "" {
		_, _ = fmt.Fprintln(buffer, "Scope: `root`")
	} else {
		_, _ = fmt.Fprintf(buffer, "Scope: `%s`\n", result.Path)
	}
	_, _ = fmt.Fprintln(buffer)

	_, _ = fmt.Fprintln(buffer, "## Summary")
	_, _ = fmt.Fprintln(buffer)
	_, _ = fmt.Fprintf(buffer, "- Runtime registered: %d\n", result.Summary.Runtime.RegisteredCount)
	_, _ = fmt.Fprintf(buffer, "- Runtime implemented: %d\n", result.Summary.Runtime.ImplementedCount)
	_, _ = fmt.Fprintf(buffer, "- Runtime stubbed: %d\n", result.Summary.Runtime.StubbedCount)
	_, _ = fmt.Fprintf(buffer, "- Catalog declared: %d\n", result.Summary.Catalog.DeclaredCount)
	_, _ = fmt.Fprintf(buffer, "- Exported operations: %d\n", result.Summary.ExportedOperationCount)
	_, _ = fmt.Fprintln(buffer)

	if len(result.RegisteredButStubbed) > 0 || len(result.RuntimePresentButCatalogMissing) > 0 || len(result.CatalogDeclaredButRuntimeMissing) > 0 {
		_, _ = fmt.Fprintln(buffer, "## Drift")
		_, _ = fmt.Fprintln(buffer)

		if len(result.RegisteredButStubbed) > 0 {
			_, _ = fmt.Fprintln(buffer, "### Registered But Stubbed")
			_, _ = fmt.Fprintln(buffer)
			for _, item := range result.RegisteredButStubbed {
				_, _ = fmt.Fprintf(buffer, "- `%s`\n", item.Operation)
			}
			_, _ = fmt.Fprintln(buffer)
		}

		if len(result.RuntimePresentButCatalogMissing) > 0 {
			_, _ = fmt.Fprintln(buffer, "### Runtime Present But Catalog Missing")
			_, _ = fmt.Fprintln(buffer)
			for _, item := range result.RuntimePresentButCatalogMissing {
				_, _ = fmt.Fprintf(buffer, "- `%s`\n", item.Operation)
			}
			_, _ = fmt.Fprintln(buffer)
		}

		if len(result.CatalogDeclaredButRuntimeMissing) > 0 {
			_, _ = fmt.Fprintln(buffer, "### Catalog Declared But Runtime Missing")
			_, _ = fmt.Fprintln(buffer)
			for _, item := range result.CatalogDeclaredButRuntimeMissing {
				_, _ = fmt.Fprintf(buffer, "- `%s`\n", item.Operation)
			}
			_, _ = fmt.Fprintln(buffer)
		}
	}

	for _, operation := range result.Operations {
		_, _ = fmt.Fprintf(buffer, "## `%s`\n", operation.Operation)
		_, _ = fmt.Fprintln(buffer)

		if strings.TrimSpace(operation.Summary) != "" {
			_, _ = fmt.Fprintln(buffer, operation.Summary)
			_, _ = fmt.Fprintln(buffer)
		}
		if strings.TrimSpace(operation.Description) != "" {
			_, _ = fmt.Fprintln(buffer, operation.Description)
			_, _ = fmt.Fprintln(buffer)
		}

		_, _ = fmt.Fprintln(buffer, "### Metadata")
		_, _ = fmt.Fprintln(buffer)
		_, _ = fmt.Fprintf(buffer, "- Platform: `%s`\n", operation.Platform)
		_, _ = fmt.Fprintf(buffer, "- Resource path: `%s`\n", operation.ResourcePath)
		_, _ = fmt.Fprintf(buffer, "- Action: `%s`\n", operation.Action)
		_, _ = fmt.Fprintf(buffer, "- Mutating: `%t`\n", operation.Mutating)
		_, _ = fmt.Fprintf(buffer, "- Implemented: `%t`\n", operation.Implemented)
		_, _ = fmt.Fprintf(buffer, "- Dry-run supported: `%t`\n", operation.DryRunSupported)
		_, _ = fmt.Fprintf(buffer, "- Default timeout: `%dms`\n", operation.DefaultTimeoutMS)
		_, _ = fmt.Fprintf(buffer, "- Runtime status: `%s`\n", operation.RuntimeStatus)
		_, _ = fmt.Fprintf(buffer, "- Catalog status: `%s`\n", operation.CatalogStatus)
		if len(operation.AllowedSubjects) > 0 {
			_, _ = fmt.Fprintf(buffer, "- Allowed subjects: `%s`\n", strings.Join(operation.AllowedSubjects, "`, `"))
		}
		if operation.Idempotency.Required || operation.Idempotency.AutoGenerated {
			_, _ = fmt.Fprintf(buffer, "- Idempotency: required=`%t`, auto_generated=`%t`\n", operation.Idempotency.Required, operation.Idempotency.AutoGenerated)
		}
		_, _ = fmt.Fprintln(buffer)

		if len(operation.Input.Required) > 0 || len(operation.Input.Optional) > 0 || len(operation.Input.Notes) > 0 {
			_, _ = fmt.Fprintln(buffer, "### Input")
			_, _ = fmt.Fprintln(buffer)
			if len(operation.Input.Required) > 0 {
				_, _ = fmt.Fprintf(buffer, "- Required: `%s`\n", strings.Join(operation.Input.Required, "`, `"))
			}
			if len(operation.Input.Optional) > 0 {
				_, _ = fmt.Fprintf(buffer, "- Optional: `%s`\n", strings.Join(operation.Input.Optional, "`, `"))
			}
			for _, note := range operation.Input.Notes {
				_, _ = fmt.Fprintf(buffer, "- Note: %s\n", note)
			}
			_, _ = fmt.Fprintln(buffer)
		}

		if len(operation.Examples) > 0 {
			_, _ = fmt.Fprintln(buffer, "### Examples")
			_, _ = fmt.Fprintln(buffer)
			for _, example := range operation.Examples {
				if strings.TrimSpace(example.Title) != "" {
					_, _ = fmt.Fprintf(buffer, "- %s\n", example.Title)
				} else {
					_, _ = fmt.Fprintln(buffer, "- Example")
				}
				_, _ = fmt.Fprintln(buffer)
				_, _ = fmt.Fprintln(buffer, "```bash")
				_, _ = fmt.Fprintln(buffer, example.Command)
				_, _ = fmt.Fprintln(buffer, "```")
				_, _ = fmt.Fprintln(buffer)
			}
		}

		if len(operation.Input.Sample) > 0 {
			sample, err := json.MarshalIndent(operation.Input.Sample, "", "  ")
			if err == nil {
				_, _ = fmt.Fprintln(buffer, "### Sample Input")
				_, _ = fmt.Fprintln(buffer)
				_, _ = fmt.Fprintln(buffer, "```json")
				_, _ = buffer.Write(sample)
				_, _ = fmt.Fprintln(buffer)
				_, _ = fmt.Fprintln(buffer, "```")
				_, _ = fmt.Fprintln(buffer)
			}
		}
	}

	return strings.TrimSpace(buffer.String()) + "\n", nil
}

func matchesSpecPath(operation string, parts []string) bool {
	if len(parts) == 0 {
		return true
	}
	return hasPrefix(strings.Split(operation, "."), parts)
}

func nodeTypeForExportPath(parts []string, registry *adapter.Registry, catalogIndex map[string]speccatalog.Entry) string {
	if len(parts) == 0 {
		return "root"
	}
	path := strings.Join(parts, ".")
	if _, ok := registry.Resolve(path); ok {
		return "operation"
	}
	if _, ok := catalogIndex[path]; ok {
		return "operation"
	}
	return nodeTypeForPath(parts)
}
