package spec

import (
	"sort"

	speccatalog "github.com/clawrise/clawrise-cli/internal/spec/catalog"
)

// StatusResult describes the reconciliation result between runtime and catalog.
type StatusResult struct {
	Summary                          StatusSummary `json:"summary"`
	RegisteredButStubbed             []StatusItem  `json:"registered_but_stubbed,omitempty"`
	CatalogDeclaredButRuntimeMissing []StatusItem  `json:"catalog_declared_but_runtime_missing,omitempty"`
	RuntimePresentButCatalogMissing  []StatusItem  `json:"runtime_present_but_catalog_missing,omitempty"`
}

// StatusSummary aggregates runtime, catalog, and issue counts.
type StatusSummary struct {
	Runtime StatusRuntimeSummary `json:"runtime"`
	Catalog StatusCatalogSummary `json:"catalog"`
	Issues  StatusIssueSummary   `json:"issues"`
}

// StatusRuntimeSummary aggregates runtime registration counts.
type StatusRuntimeSummary struct {
	RegisteredCount  int `json:"registered_count"`
	ImplementedCount int `json:"implemented_count"`
	StubbedCount     int `json:"stubbed_count"`
}

// StatusCatalogSummary aggregates catalog declaration counts.
type StatusCatalogSummary struct {
	DeclaredCount int `json:"declared_count"`
}

// StatusIssueSummary aggregates drift and stub-related issue counts.
type StatusIssueSummary struct {
	RegisteredButStubbedCount             int `json:"registered_but_stubbed_count"`
	CatalogDeclaredButRuntimeMissingCount int `json:"catalog_declared_but_runtime_missing_count"`
	RuntimePresentButCatalogMissingCount  int `json:"runtime_present_but_catalog_missing_count"`
}

// StatusItem describes one operation that needs attention.
type StatusItem struct {
	Operation     string `json:"operation"`
	RuntimeStatus string `json:"runtime_status"`
	CatalogStatus string `json:"catalog_status"`
}

// Status returns the structured diff between runtime registry and catalog.
func (s *Service) Status() (StatusResult, error) {
	catalogIndex, err := speccatalog.Index(s.catalogEntries)
	if err != nil {
		return StatusResult{}, newError("INVALID_SPEC_CATALOG", err.Error())
	}

	runtimeDefinitions := s.registry.Definitions()
	runtimeIndex := make(map[string]struct{}, len(runtimeDefinitions))

	result := StatusResult{}
	result.Summary.Runtime.RegisteredCount = len(runtimeDefinitions)
	result.Summary.Catalog.DeclaredCount = len(catalogIndex)

	for _, definition := range runtimeDefinitions {
		runtimeIndex[definition.Operation] = struct{}{}

		if definition.Handler == nil {
			result.RegisteredButStubbed = append(result.RegisteredButStubbed, StatusItem{
				Operation:     definition.Operation,
				RuntimeStatus: runtimeStatusForDefinition(definition),
				CatalogStatus: catalogStatusForOperation(definition.Operation, catalogIndex),
			})
			result.Summary.Runtime.StubbedCount++
		} else {
			result.Summary.Runtime.ImplementedCount++
		}

		if _, ok := catalogIndex[definition.Operation]; !ok {
			result.RuntimePresentButCatalogMissing = append(result.RuntimePresentButCatalogMissing, StatusItem{
				Operation:     definition.Operation,
				RuntimeStatus: runtimeStatusForDefinition(definition),
				CatalogStatus: "catalog_missing",
			})
		}
	}

	for operation := range catalogIndex {
		if _, ok := runtimeIndex[operation]; ok {
			continue
		}
		result.CatalogDeclaredButRuntimeMissing = append(result.CatalogDeclaredButRuntimeMissing, StatusItem{
			Operation:     operation,
			RuntimeStatus: "runtime_missing",
			CatalogStatus: "declared",
		})
	}

	sortStatusItems(result.RegisteredButStubbed)
	sortStatusItems(result.CatalogDeclaredButRuntimeMissing)
	sortStatusItems(result.RuntimePresentButCatalogMissing)

	result.Summary.Issues.RegisteredButStubbedCount = len(result.RegisteredButStubbed)
	result.Summary.Issues.CatalogDeclaredButRuntimeMissingCount = len(result.CatalogDeclaredButRuntimeMissing)
	result.Summary.Issues.RuntimePresentButCatalogMissingCount = len(result.RuntimePresentButCatalogMissing)
	return result, nil
}

func catalogStatusForOperation(operation string, catalogIndex map[string]speccatalog.Entry) string {
	if _, ok := catalogIndex[operation]; ok {
		return "declared"
	}
	return "catalog_missing"
}

func sortStatusItems(items []StatusItem) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].Operation < items[j].Operation
	})
}
