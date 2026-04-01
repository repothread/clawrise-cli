package plugin

import (
	"time"

	"github.com/clawrise/clawrise-cli/internal/adapter"
	"github.com/clawrise/clawrise-cli/internal/apperr"
	authcache "github.com/clawrise/clawrise-cli/internal/auth"
	"github.com/clawrise/clawrise-cli/internal/authflow"
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

// CapabilityListResult 描述 capability 列表响应。
type CapabilityListResult struct {
	Capabilities []CapabilityDescriptor `json:"capabilities"`
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
	Platform    string      `json:"platform"`
	Subject     string      `json:"subject"`
	AccountName string      `json:"account_name"`
	Auth        ExecuteAuth `json:"auth"`
}

// ExecuteAuth describes the structured execution auth context.
type ExecuteAuth struct {
	Method        string         `json:"method"`
	ExecutionAuth map[string]any `json:"execution_auth,omitempty"`
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

// StorageBackendDescriptor 描述一个外部存储 backend plugin 的能力。
type StorageBackendDescriptor struct {
	Target      string `json:"target"`
	Backend     string `json:"backend"`
	DisplayName string `json:"display_name,omitempty"`
	Description string `json:"description,omitempty"`
}

// StorageBackendDescribeResult 描述存储 backend 元数据响应。
type StorageBackendDescribeResult struct {
	Backend StorageBackendDescriptor `json:"backend"`
}

// StorageStatus 描述一个存储 backend 的可用状态。
type StorageStatus struct {
	Backend   string `json:"backend"`
	Supported bool   `json:"supported"`
	Readable  bool   `json:"readable"`
	Writable  bool   `json:"writable"`
	Secure    bool   `json:"secure"`
	Detail    string `json:"detail,omitempty"`
}

// SecretStoreStatusResult 描述 secret store 状态响应。
type SecretStoreStatusResult struct {
	Status StorageStatus `json:"status"`
}

// SecretStoreGetParams 描述 secret store 读请求。
type SecretStoreGetParams struct {
	AccountName string `json:"account_name"`
	Field       string `json:"field"`
}

// SecretStoreGetResult 描述 secret store 读响应。
type SecretStoreGetResult struct {
	Found bool   `json:"found"`
	Value string `json:"value,omitempty"`
}

// SecretStoreSetParams 描述 secret store 写请求。
type SecretStoreSetParams struct {
	AccountName string `json:"account_name"`
	Field       string `json:"field"`
	Value       string `json:"value"`
}

// SecretStoreDeleteParams 描述 secret store 删除请求。
type SecretStoreDeleteParams struct {
	AccountName string `json:"account_name"`
	Field       string `json:"field"`
}

// SessionStoreStatusResult 描述 session store 状态响应。
type SessionStoreStatusResult struct {
	Status StorageStatus `json:"status"`
}

// SessionStoreLoadParams 描述 session store 读取请求。
type SessionStoreLoadParams struct {
	AccountName string `json:"account_name"`
}

// SessionStoreLoadResult 描述 session store 读取结果。
type SessionStoreLoadResult struct {
	Found   bool               `json:"found"`
	Session *authcache.Session `json:"session,omitempty"`
}

// SessionStoreSaveParams 描述 session store 写入请求。
type SessionStoreSaveParams struct {
	Session authcache.Session `json:"session"`
}

// SessionStoreDeleteParams 描述 session store 删除请求。
type SessionStoreDeleteParams struct {
	AccountName string `json:"account_name"`
}

// AuthFlowStoreStatusResult 描述 authflow store 状态响应。
type AuthFlowStoreStatusResult struct {
	Status StorageStatus `json:"status"`
}

// AuthFlowStoreLoadParams 描述 authflow store 读取请求。
type AuthFlowStoreLoadParams struct {
	FlowID string `json:"flow_id"`
}

// AuthFlowStoreLoadResult 描述 authflow store 读取结果。
type AuthFlowStoreLoadResult struct {
	Found bool           `json:"found"`
	Flow  *authflow.Flow `json:"flow,omitempty"`
}

// AuthFlowStoreSaveParams 描述 authflow store 写入请求。
type AuthFlowStoreSaveParams struct {
	Flow authflow.Flow `json:"flow"`
}

// AuthFlowStoreDeleteParams 描述 authflow store 删除请求。
type AuthFlowStoreDeleteParams struct {
	FlowID string `json:"flow_id"`
}

// GovernanceStoreStatusResult 描述 governance store 状态响应。
type GovernanceStoreStatusResult struct {
	Status StorageStatus `json:"status"`
}

// GovernanceErrorBody 描述治理记录中的错误体。
type GovernanceErrorBody struct {
	Code         string `json:"code"`
	Message      string `json:"message"`
	Retryable    bool   `json:"retryable"`
	UpstreamCode string `json:"upstream_code,omitempty"`
	HTTPStatus   int    `json:"http_status,omitempty"`
}

// GovernanceMeta 描述治理记录中的元信息。
type GovernanceMeta struct {
	Platform   string `json:"platform"`
	DurationMS int64  `json:"duration_ms"`
	RetryCount int    `json:"retry_count"`
	DryRun     bool   `json:"dry_run"`
}

// GovernanceContext 描述治理审计记录中的上下文。
type GovernanceContext struct {
	Platform string `json:"platform,omitempty"`
	Subject  string `json:"subject,omitempty"`
	Account  string `json:"account,omitempty"`
}

// GovernanceIdempotencyState 描述治理审计记录中的幂等状态。
type GovernanceIdempotencyState struct {
	Key       string `json:"key"`
	Status    string `json:"status"`
	Persisted bool   `json:"persisted,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

// GovernanceIdempotencyRecord 描述幂等持久化记录。
type GovernanceIdempotencyRecord struct {
	Key        string               `json:"key"`
	Operation  string               `json:"operation"`
	InputHash  string               `json:"input_hash"`
	Status     string               `json:"status"`
	RequestID  string               `json:"request_id"`
	CreatedAt  string               `json:"created_at"`
	UpdatedAt  string               `json:"updated_at"`
	RetryCount int                  `json:"retry_count"`
	Data       any                  `json:"data,omitempty"`
	Error      *GovernanceErrorBody `json:"error,omitempty"`
	Meta       GovernanceMeta       `json:"meta"`
}

// GovernanceAuditRecord 描述审计持久化记录。
type GovernanceAuditRecord struct {
	Time          string                      `json:"time"`
	RequestID     string                      `json:"request_id"`
	Operation     string                      `json:"operation"`
	Context       *GovernanceContext          `json:"context,omitempty"`
	OK            bool                        `json:"ok"`
	InputSummary  any                         `json:"input_summary,omitempty"`
	OutputSummary any                         `json:"output_summary,omitempty"`
	Error         *GovernanceErrorBody        `json:"error,omitempty"`
	Meta          GovernanceMeta              `json:"meta"`
	Idempotency   *GovernanceIdempotencyState `json:"idempotency,omitempty"`
	Warnings      []string                    `json:"warnings,omitempty"`
}

// GovernanceIdempotencyLoadParams 描述幂等记录读取请求。
type GovernanceIdempotencyLoadParams struct {
	Key string `json:"key"`
}

// GovernanceIdempotencyLoadResult 描述幂等记录读取结果。
type GovernanceIdempotencyLoadResult struct {
	Found  bool                         `json:"found"`
	Record *GovernanceIdempotencyRecord `json:"record,omitempty"`
}

// GovernanceIdempotencySaveParams 描述幂等记录写入请求。
type GovernanceIdempotencySaveParams struct {
	Record GovernanceIdempotencyRecord `json:"record"`
}

// GovernanceAuditAppendParams 描述审计记录追加请求。
type GovernanceAuditAppendParams struct {
	Day    string                `json:"day"`
	Record GovernanceAuditRecord `json:"record"`
}

// PolicyEvaluationContext describes the execution context required for one policy decision.
type PolicyEvaluationContext struct {
	AccountName string `json:"account_name,omitempty"`
	Platform    string `json:"platform,omitempty"`
	Subject     string `json:"subject,omitempty"`
	AuthMethod  string `json:"auth_method,omitempty"`
}

// PolicyEvaluationRequest describes one execution request to be evaluated by policy.
type PolicyEvaluationRequest struct {
	RequestID string                  `json:"request_id"`
	Operation string                  `json:"operation"`
	DryRun    bool                    `json:"dry_run"`
	Mutating  bool                    `json:"mutating"`
	Input     map[string]any          `json:"input,omitempty"`
	Context   PolicyEvaluationContext `json:"context"`
}

// PolicyEvaluateParams describes one policy evaluation request.
type PolicyEvaluateParams struct {
	PolicyID string                  `json:"policy_id,omitempty"`
	Request  PolicyEvaluationRequest `json:"request"`
}

// PolicyEvaluateResult describes one policy evaluation result.
type PolicyEvaluateResult struct {
	Decision    string         `json:"decision"`
	Message     string         `json:"message,omitempty"`
	Annotations map[string]any `json:"annotations,omitempty"`
}

// WorkflowAvailableOperation describes one operation that can be referenced by a workflow plan.
type WorkflowAvailableOperation struct {
	Operation string `json:"operation"`
	Platform  string `json:"platform,omitempty"`
	Summary   string `json:"summary,omitempty"`
	Mutating  bool   `json:"mutating,omitempty"`
}

// WorkflowPlaybookReference describes one optional playbook input that a planner may use.
type WorkflowPlaybookReference struct {
	ID    string `json:"id,omitempty"`
	Path  string `json:"path,omitempty"`
	Title string `json:"title,omitempty"`
}

// WorkflowPlanRequest describes one planning request sent to a workflow plugin.
type WorkflowPlanRequest struct {
	Goal                string                       `json:"goal"`
	Context             map[string]any               `json:"context,omitempty"`
	Constraints         map[string]any               `json:"constraints,omitempty"`
	AvailableOperations []WorkflowAvailableOperation `json:"available_operations,omitempty"`
	Playbooks           []WorkflowPlaybookReference  `json:"playbooks,omitempty"`
}

// WorkflowPlanStep describes one structured step in a workflow plan.
type WorkflowPlanStep struct {
	Type                 string         `json:"type"`
	Title                string         `json:"title,omitempty"`
	Operation            string         `json:"operation,omitempty"`
	Input                map[string]any `json:"input,omitempty"`
	Note                 string         `json:"note,omitempty"`
	RequiresConfirmation bool           `json:"requires_confirmation,omitempty"`
}

// WorkflowMissingInput describes one required input that the planner could not infer.
type WorkflowMissingInput struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	StepIndex   int    `json:"step_index,omitempty"`
}

// WorkflowPlanParams describes one workflow planning request.
type WorkflowPlanParams struct {
	WorkflowID string              `json:"workflow_id,omitempty"`
	Request    WorkflowPlanRequest `json:"request"`
}

// WorkflowPlanResult describes one structured workflow planning result.
type WorkflowPlanResult struct {
	Summary              string                 `json:"summary,omitempty"`
	Steps                []WorkflowPlanStep     `json:"steps,omitempty"`
	Warnings             []string               `json:"warnings,omitempty"`
	MissingInputs        []WorkflowMissingInput `json:"missing_inputs,omitempty"`
	RequiresConfirmation bool                   `json:"requires_confirmation,omitempty"`
}

// RegistryPluginSummary describes one plugin entry exposed by a registry source.
type RegistryPluginSummary struct {
	Name          string `json:"name"`
	LatestVersion string `json:"latest_version,omitempty"`
	Description   string `json:"description,omitempty"`
}

// RegistrySourceListParams describes one registry listing request.
type RegistrySourceListParams struct {
	SourceID string `json:"source_id,omitempty"`
	Query    string `json:"query,omitempty"`
	Limit    int    `json:"limit,omitempty"`
}

// RegistrySourceListResult describes one registry listing response.
type RegistrySourceListResult struct {
	Plugins []RegistryPluginSummary `json:"plugins,omitempty"`
}

// RegistrySourceResolveParams describes one registry artifact resolution request.
type RegistrySourceResolveParams struct {
	SourceID  string `json:"source_id,omitempty"`
	Reference string `json:"reference"`
	Version   string `json:"version,omitempty"`
}

// RegistrySourceResolveResult describes the resolved installable artifact for one logical reference.
type RegistrySourceResolveResult struct {
	Name           string `json:"name,omitempty"`
	Version        string `json:"version,omitempty"`
	ArtifactURL    string `json:"artifact_url,omitempty"`
	ChecksumSHA256 string `json:"checksum_sha256,omitempty"`
	MetadataURL    string `json:"metadata_url,omitempty"`
}

// AuditEmitParams describes one audit event delivery request.
type AuditEmitParams struct {
	SinkID string                `json:"sink_id,omitempty"`
	Record GovernanceAuditRecord `json:"record"`
}
