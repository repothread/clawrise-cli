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
