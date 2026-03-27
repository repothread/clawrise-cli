package plugin

import (
	"github.com/clawrise/clawrise-cli/internal/adapter"
	"github.com/clawrise/clawrise-cli/internal/apperr"
	speccatalog "github.com/clawrise/clawrise-cli/internal/spec/catalog"
)

// ProtocolVersion is the current core-to-plugin protocol version.
const ProtocolVersion = 1

// RPCRequest describes one JSON-RPC request envelope.
type RPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      string `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

// RPCResponse describes one JSON-RPC response envelope.
type RPCResponse struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      string    `json:"id"`
	Result  any       `json:"result,omitempty"`
	Error   *RPCError `json:"error,omitempty"`
}

// RPCError describes one JSON-RPC protocol error.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// HandshakeParams describes the core handshake request payload.
type HandshakeParams struct {
	ProtocolVersion int         `json:"protocol_version"`
	Core            CoreVersion `json:"core"`
}

// CoreVersion describes core version identity during handshake.
type CoreVersion struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// OperationsListResult describes the operation list response payload.
type OperationsListResult struct {
	Operations []OperationDescriptor `json:"operations"`
}

// OperationDescriptor describes one operation exposed by a plugin.
type OperationDescriptor struct {
	Operation        string                `json:"operation"`
	Platform         string                `json:"platform"`
	Mutating         bool                  `json:"mutating"`
	DefaultTimeoutMS int64                 `json:"default_timeout_ms"`
	AllowedSubjects  []string              `json:"allowed_subjects"`
	Spec             adapter.OperationSpec `json:"spec"`
}

// CatalogResult describes the catalog response payload.
type CatalogResult struct {
	Entries []speccatalog.Entry `json:"entries"`
}

// ExecuteParams describes the plugin execute request payload.
type ExecuteParams struct {
	Request  ExecuteEnvelope `json:"request"`
	Identity ExecuteIdentity `json:"identity"`
}

// ExecuteEnvelope describes the normalized provider request body.
type ExecuteEnvelope struct {
	RequestID      string         `json:"request_id"`
	Operation      string         `json:"operation"`
	Input          map[string]any `json:"input"`
	TimeoutMS      int64          `json:"timeout_ms"`
	IdempotencyKey string         `json:"idempotency_key,omitempty"`
	DryRun         bool           `json:"dry_run"`
}

// ExecuteIdentity describes the resolved execution identity.
type ExecuteIdentity struct {
	Platform    string         `json:"platform"`
	Subject     string         `json:"subject"`
	ProfileName string         `json:"profile_name"`
	Auth        map[string]any `json:"auth"`
}

// ExecuteRPCResult describes the plugin execute response payload.
type ExecuteRPCResult struct {
	OK    bool             `json:"ok"`
	Data  map[string]any   `json:"data"`
	Error *apperr.AppError `json:"error,omitempty"`
	Meta  map[string]any   `json:"meta,omitempty"`
}
