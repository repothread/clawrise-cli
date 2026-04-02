package runtime

import "time"

// ExecuteOptions describes one operation execution request.
type ExecuteOptions struct {
	OperationInput string
	AccountName    string
	SubjectName    string
	InputJSON      string
	InputFile      string
	Timeout        time.Duration
	DryRun         bool
	IdempotencyKey string
	Output         string
	Quiet          bool
	Stdin          any
}

// Envelope is the normalized output envelope for execution commands.
type Envelope struct {
	OK          bool              `json:"ok"`
	Operation   string            `json:"operation"`
	RequestID   string            `json:"request_id"`
	Context     *Context          `json:"context,omitempty"`
	Data        any               `json:"data"`
	Error       *ErrorBody        `json:"error"`
	Meta        Meta              `json:"meta"`
	Idempotency *IdempotencyState `json:"idempotency,omitempty"`
	Policy      *PolicyResult     `json:"policy,omitempty"`
	Warnings    []string          `json:"warnings,omitempty"`
}

// Context describes the resolved execution context for the current command.
type Context struct {
	Platform string `json:"platform,omitempty"`
	Subject  string `json:"subject,omitempty"`
	Account  string `json:"account,omitempty"`
}

// ErrorBody is the normalized error payload.
type ErrorBody struct {
	Code         string `json:"code"`
	Message      string `json:"message"`
	Retryable    bool   `json:"retryable"`
	UpstreamCode string `json:"upstream_code,omitempty"`
	HTTPStatus   int    `json:"http_status,omitempty"`
}

// Meta stores execution metadata.
type Meta struct {
	Platform   string `json:"platform"`
	DurationMS int64  `json:"duration_ms"`
	RetryCount int    `json:"retry_count"`
	DryRun     bool   `json:"dry_run"`
}

// IdempotencyState describes the idempotency state of the current request.
type IdempotencyState struct {
	Key       string `json:"key"`
	Status    string `json:"status"`
	Persisted bool   `json:"persisted,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

// PolicyResult stores the structured policy evaluation summary for one request.
type PolicyResult struct {
	FinalDecision string      `json:"final_decision"`
	Hits          []PolicyHit `json:"hits,omitempty"`
}

// PolicyHit records one matched local rule or plugin policy decision.
type PolicyHit struct {
	SourceType  string         `json:"source_type"`
	SourceName  string         `json:"source_name"`
	Decision    string         `json:"decision"`
	Message     string         `json:"message,omitempty"`
	MatchedRule string         `json:"matched_rule,omitempty"`
	Annotations map[string]any `json:"annotations,omitempty"`
}

// ExecutionProfile is the resolved execution identity at runtime.
type ExecutionProfile struct {
	Name       string `json:"name"`
	Account    string `json:"account"`
	Platform   string `json:"platform"`
	Subject    string `json:"subject"`
	AuthMethod string `json:"auth_method,omitempty"`
}
