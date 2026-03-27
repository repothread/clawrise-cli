package spec

import "github.com/clawrise/clawrise-cli/internal/adapter"

// Error is a structured error returned by the `spec` layer.
type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string {
	return e.Message
}

func newError(code, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

// ListItem describes one node returned by `spec list`.
type ListItem struct {
	Name           string `json:"name"`
	FullPath       string `json:"full_path"`
	NodeType       string `json:"node_type"`
	ChildCount     int    `json:"child_count,omitempty"`
	OperationCount int    `json:"operation_count,omitempty"`
	Implemented    *bool  `json:"implemented,omitempty"`
	Mutating       *bool  `json:"mutating,omitempty"`
	Summary        string `json:"summary,omitempty"`
}

// ListResult describes one hierarchical browse result.
type ListResult struct {
	Path     string     `json:"path"`
	NodeType string     `json:"node_type"`
	Depth    int        `json:"depth"`
	Items    []ListItem `json:"items"`
}

// OperationView describes one operation detail view.
type OperationView struct {
	Operation        string                  `json:"operation"`
	Platform         string                  `json:"platform"`
	ResourcePath     string                  `json:"resource_path"`
	Action           string                  `json:"action"`
	Summary          string                  `json:"summary,omitempty"`
	Description      string                  `json:"description,omitempty"`
	AllowedSubjects  []string                `json:"allowed_subjects"`
	Mutating         bool                    `json:"mutating"`
	Implemented      bool                    `json:"implemented"`
	DryRunSupported  bool                    `json:"dry_run_supported"`
	DefaultTimeoutMS int64                   `json:"default_timeout_ms"`
	Idempotency      adapter.IdempotencySpec `json:"idempotency,omitempty"`
	Input            adapter.InputSpec       `json:"input,omitempty"`
	Examples         []adapter.ExampleSpec   `json:"examples,omitempty"`
	RuntimeStatus    string                  `json:"runtime_status"`
}

// OperationExport 描述一个可导出的 operation 详情记录。
type OperationExport struct {
	OperationView
	CatalogStatus string `json:"catalog_status"`
}

// ExportSummary 聚合导出结果中的摘要信息。
type ExportSummary struct {
	Runtime                StatusRuntimeSummary `json:"runtime"`
	Catalog                StatusCatalogSummary `json:"catalog"`
	Issues                 StatusIssueSummary   `json:"issues"`
	ExportedOperationCount int                  `json:"exported_operation_count"`
}

// ExportResult 描述一个完整的结构化导出结果。
type ExportResult struct {
	Path                             string            `json:"path"`
	NodeType                         string            `json:"node_type"`
	Summary                          ExportSummary     `json:"summary"`
	Operations                       []OperationExport `json:"operations,omitempty"`
	RegisteredButStubbed             []StatusItem      `json:"registered_but_stubbed,omitempty"`
	CatalogDeclaredButRuntimeMissing []StatusItem      `json:"catalog_declared_but_runtime_missing,omitempty"`
	RuntimePresentButCatalogMissing  []StatusItem      `json:"runtime_present_but_catalog_missing,omitempty"`
}

// CompletionData 描述 completion 生成所需的最小事实集。
type CompletionData struct {
	Operations []string `json:"operations"`
	SpecPaths  []string `json:"spec_paths"`
}
