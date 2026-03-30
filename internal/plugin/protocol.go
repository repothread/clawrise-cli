package plugin

import (
	"time"

	"github.com/clawrise/clawrise-cli/internal/adapter"
	"github.com/clawrise/clawrise-cli/internal/apperr"
	authcache "github.com/clawrise/clawrise-cli/internal/auth"
	speccatalog "github.com/clawrise/clawrise-cli/internal/spec/catalog"
)

const rfc3339Layout = time.RFC3339

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
	AccountName string         `json:"account_name"`
	Auth        map[string]any `json:"auth"`
}

// ExecuteRPCResult describes the plugin execute response payload.
type ExecuteRPCResult struct {
	OK    bool             `json:"ok"`
	Data  map[string]any   `json:"data"`
	Error *apperr.AppError `json:"error,omitempty"`
	Meta  map[string]any   `json:"meta,omitempty"`
}

// AuthFieldDescriptor describes one auth field.
type AuthFieldDescriptor struct {
	Name     string `json:"name"`
	Required bool   `json:"required"`
	Type     string `json:"type,omitempty"`
}

// AuthMethodDescriptor describes one auth method.
type AuthMethodDescriptor struct {
	ID               string                `json:"id"`
	Platform         string                `json:"platform"`
	DisplayName      string                `json:"display_name"`
	Description      string                `json:"description,omitempty"`
	Subjects         []string              `json:"subjects,omitempty"`
	Kind             string                `json:"kind,omitempty"`
	Interactive      bool                  `json:"interactive"`
	InteractiveModes []string              `json:"interactive_modes,omitempty"`
	PublicFields     []AuthFieldDescriptor `json:"public_fields,omitempty"`
	SecretFields     []AuthFieldDescriptor `json:"secret_fields,omitempty"`
}

// AuthMethodsListResult describes the auth method list payload.
type AuthMethodsListResult struct {
	Methods []AuthMethodDescriptor `json:"methods"`
}

// AuthPresetDescriptor describes one user-facing account preset.
type AuthPresetDescriptor struct {
	ID                 string         `json:"id"`
	Platform           string         `json:"platform"`
	DisplayName        string         `json:"display_name"`
	Description        string         `json:"description,omitempty"`
	Subject            string         `json:"subject"`
	AuthMethod         string         `json:"auth_method"`
	DefaultAccountName string         `json:"default_account_name,omitempty"`
	Public             map[string]any `json:"public,omitempty"`
	SecretFields       []string       `json:"secret_fields,omitempty"`
}

// AuthPresetsListResult describes the account preset list payload.
type AuthPresetsListResult struct {
	Presets []AuthPresetDescriptor `json:"presets"`
}

// AuthAction describes one next-step action hint.
type AuthAction struct {
	Type            string `json:"type"`
	Message         string `json:"message,omitempty"`
	URL             string `json:"url,omitempty"`
	DeviceCode      string `json:"device_code,omitempty"`
	UserCode        string `json:"user_code,omitempty"`
	VerificationURL string `json:"verification_url,omitempty"`
	IntervalSec     int    `json:"interval_sec,omitempty"`
	Field           string `json:"field,omitempty"`
}

// AuthSessionPayload is the lightweight session shape used by the protocol.
type AuthSessionPayload struct {
	AccessToken  string            `json:"access_token,omitempty"`
	RefreshToken string            `json:"refresh_token,omitempty"`
	TokenType    string            `json:"token_type,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	ExpiresAt    string            `json:"expires_at,omitempty"`
}

// ToSession converts the protocol session payload into the internal session shape.
func (p AuthSessionPayload) ToSession() authcache.Session {
	session := authcache.Session{
		AccessToken:  p.AccessToken,
		RefreshToken: p.RefreshToken,
		TokenType:    p.TokenType,
		Metadata:     p.Metadata,
	}
	if p.ExpiresAt != "" {
		// Keep the protocol lightweight and ignore invalid expiry values.
		if expiresAt, err := parseRFC3339Time(p.ExpiresAt); err == nil {
			session.ExpiresAt = &expiresAt
		}
	}
	return session
}

// AuthSessionPayloadFromSession converts the internal session shape into the protocol payload.
func AuthSessionPayloadFromSession(session *authcache.Session) *AuthSessionPayload {
	if session == nil {
		return nil
	}

	payload := &AuthSessionPayload{
		AccessToken:  session.AccessToken,
		RefreshToken: session.RefreshToken,
		TokenType:    session.TokenType,
		Metadata:     session.Metadata,
	}
	if session.ExpiresAt != nil {
		payload.ExpiresAt = session.ExpiresAt.UTC().Format(rfc3339Layout)
	}
	return payload
}

func parseRFC3339Time(raw string) (time.Time, error) {
	return time.Parse(time.RFC3339, raw)
}

// AuthAccount describes the account context sent from core to a plugin.
type AuthAccount struct {
	Name       string              `json:"name"`
	Platform   string              `json:"platform"`
	Subject    string              `json:"subject"`
	AuthMethod string              `json:"auth_method"`
	Public     map[string]any      `json:"public,omitempty"`
	Secrets    map[string]string   `json:"secrets,omitempty"`
	Session    *AuthSessionPayload `json:"session,omitempty"`
}

// AuthInspectParams describes an auth inspection request.
type AuthInspectParams struct {
	Account AuthAccount `json:"account"`
}

// AuthInspectResult describes an auth inspection result.
type AuthInspectResult struct {
	Ready               bool         `json:"ready"`
	Status              string       `json:"status"`
	Message             string       `json:"message,omitempty"`
	MissingPublicFields []string     `json:"missing_public_fields,omitempty"`
	MissingSecretFields []string     `json:"missing_secret_fields,omitempty"`
	SessionStatus       string       `json:"session_status,omitempty"`
	HumanRequired       bool         `json:"human_required,omitempty"`
	RecommendedAction   string       `json:"recommended_action,omitempty"`
	NextActions         []AuthAction `json:"next_actions,omitempty"`
}

// AuthFlowPayload describes one in-progress auth flow.
type AuthFlowPayload struct {
	ID               string            `json:"id"`
	Method           string            `json:"method,omitempty"`
	Mode             string            `json:"mode,omitempty"`
	State            string            `json:"state,omitempty"`
	RedirectURI      string            `json:"redirect_uri,omitempty"`
	AuthorizationURL string            `json:"authorization_url,omitempty"`
	DeviceCode       string            `json:"device_code,omitempty"`
	UserCode         string            `json:"user_code,omitempty"`
	VerificationURL  string            `json:"verification_url,omitempty"`
	IntervalSec      int               `json:"interval_sec,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
	ExpiresAt        string            `json:"expires_at,omitempty"`
}

// AuthBeginParams describes an auth-begin request.
type AuthBeginParams struct {
	Account      AuthAccount `json:"account"`
	Mode         string      `json:"mode,omitempty"`
	RedirectURI  string      `json:"redirect_uri,omitempty"`
	CallbackHost string      `json:"callback_host,omitempty"`
	CallbackPath string      `json:"callback_path,omitempty"`
}

// AuthBeginResult describes an auth-begin result.
type AuthBeginResult struct {
	Flow          AuthFlowPayload `json:"flow"`
	HumanRequired bool            `json:"human_required,omitempty"`
	NextActions   []AuthAction    `json:"next_actions,omitempty"`
}

// AuthCompleteParams describes an auth-complete request.
type AuthCompleteParams struct {
	Account     AuthAccount     `json:"account"`
	Flow        AuthFlowPayload `json:"flow"`
	Code        string          `json:"code,omitempty"`
	CallbackURL string          `json:"callback_url,omitempty"`
}

// AuthCompleteResult describes an auth-complete result.
type AuthCompleteResult struct {
	Ready             bool                `json:"ready"`
	Status            string              `json:"status"`
	Message           string              `json:"message,omitempty"`
	ExecutionAuth     map[string]any      `json:"execution_auth,omitempty"`
	SessionPatch      *AuthSessionPayload `json:"session_patch,omitempty"`
	SecretPatches     map[string]string   `json:"secret_patches,omitempty"`
	HumanRequired     bool                `json:"human_required,omitempty"`
	RecommendedAction string              `json:"recommended_action,omitempty"`
	NextActions       []AuthAction        `json:"next_actions,omitempty"`
}

// AuthResolveParams describes a pre-execution auth resolve request.
type AuthResolveParams struct {
	Account AuthAccount `json:"account"`
}

// AuthResolveResult describes a pre-execution auth resolve result.
type AuthResolveResult struct {
	Ready             bool                `json:"ready"`
	Status            string              `json:"status"`
	Message           string              `json:"message,omitempty"`
	ExecutionAuth     map[string]any      `json:"execution_auth,omitempty"`
	SessionPatch      *AuthSessionPayload `json:"session_patch,omitempty"`
	SecretPatches     map[string]string   `json:"secret_patches,omitempty"`
	HumanRequired     bool                `json:"human_required,omitempty"`
	RecommendedAction string              `json:"recommended_action,omitempty"`
	NextActions       []AuthAction        `json:"next_actions,omitempty"`
}

// AuthLaunchContext 描述一次授权动作执行所需的最小上下文。
// 这里刻意不包含 secret / session，避免把敏感信息泄漏给 launcher。
type AuthLaunchContext struct {
	AccountName string `json:"account_name"`
	Platform    string `json:"platform"`
	Subject     string `json:"subject,omitempty"`
	AuthMethod  string `json:"auth_method,omitempty"`
}

// AuthLauncherDescriptor 描述一个授权动作执行器的能力。
type AuthLauncherDescriptor struct {
	ID          string   `json:"id"`
	DisplayName string   `json:"display_name"`
	Description string   `json:"description,omitempty"`
	ActionTypes []string `json:"action_types,omitempty"`
	Platforms   []string `json:"platforms,omitempty"`
	Priority    int      `json:"priority,omitempty"`
}

// AuthLauncherDescribeResult 描述 launcher 元数据响应。
type AuthLauncherDescribeResult struct {
	Launcher AuthLauncherDescriptor `json:"launcher"`
}

// AuthLaunchParams 描述 launcher 执行一次授权动作的请求。
type AuthLaunchParams struct {
	Context AuthLaunchContext `json:"context"`
	Flow    AuthFlowPayload   `json:"flow"`
	Action  AuthAction        `json:"action"`
}

// AuthLaunchResult 描述 launcher 对一次授权动作的执行结果。
type AuthLaunchResult struct {
	Handled    bool           `json:"handled"`
	Status     string         `json:"status"`
	Message    string         `json:"message,omitempty"`
	LauncherID string         `json:"launcher_id,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}
