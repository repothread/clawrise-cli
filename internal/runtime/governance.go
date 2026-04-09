package runtime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/clawrise/clawrise-cli/internal/adapter"
	"github.com/clawrise/clawrise-cli/internal/apperr"
	"github.com/clawrise/clawrise-cli/internal/config"
	"github.com/clawrise/clawrise-cli/internal/locator"
	pluginruntime "github.com/clawrise/clawrise-cli/internal/plugin"
)

const (
	defaultRetryBaseDelay = 200 * time.Millisecond
	defaultRetryMaxDelay  = 2 * time.Second
)

// runtimeGovernance manages idempotency, audit, and retry behavior.
type runtimeGovernance struct {
	paths runtimePaths
	store governanceStore
	sinks []auditSink
	// sinkWarnings 保存初始化 audit sink 时发现的配置或发现问题，
	// 这样可以在不阻断主执行的前提下，把问题透传到最终 envelope。
	sinkWarnings []string
	retry        retryPolicy
	now          func() time.Time
}

// runtimePaths describes local directories used by runtime governance data.
type runtimePaths struct {
	rootDir        string
	idempotencyDir string
	auditDir       string
}

// retryPolicy describes automatic retry settings.
type retryPolicy struct {
	maxAttempts int
	baseDelay   time.Duration
	maxDelay    time.Duration
}

type governanceStore interface {
	EnsureIdempotencyDir() error
	LoadIdempotencyRecord(key string) (*persistedIdempotencyRecord, error)
	SaveIdempotencyRecord(record *persistedIdempotencyRecord) error
	AppendAuditRecord(day string, record auditRecord) error
}

type governanceStoreFactory func(paths runtimePaths) governanceStore

var governanceStoreFactories = map[string]governanceStoreFactory{
	"file": func(paths runtimePaths) governanceStore {
		return &fileGovernanceStore{paths: paths}
	},
}

// RegisterGovernanceStoreBackend 注册一个可扩展的 runtime governance backend。
func RegisterGovernanceStoreBackend(name string, factory func(rootDir string) governanceStore) {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" || factory == nil {
		return
	}
	governanceStoreFactories[name] = func(paths runtimePaths) governanceStore {
		return factory(paths.rootDir)
	}
}

// persistedIdempotencyRecord is the on-disk idempotency record.
type persistedIdempotencyRecord struct {
	Key        string     `json:"key"`
	Operation  string     `json:"operation"`
	InputHash  string     `json:"input_hash"`
	Status     string     `json:"status"`
	RequestID  string     `json:"request_id"`
	CreatedAt  string     `json:"created_at"`
	UpdatedAt  string     `json:"updated_at"`
	RetryCount int        `json:"retry_count"`
	Data       any        `json:"data,omitempty"`
	Error      *ErrorBody `json:"error,omitempty"`
	Meta       Meta       `json:"meta"`
}

// auditRecord describes one execution audit summary.
type auditRecord struct {
	Time          string            `json:"time"`
	RequestID     string            `json:"request_id"`
	Operation     string            `json:"operation"`
	Context       *Context          `json:"context,omitempty"`
	OK            bool              `json:"ok"`
	InputSummary  any               `json:"input_summary,omitempty"`
	OutputSummary any               `json:"output_summary,omitempty"`
	Error         *ErrorBody        `json:"error,omitempty"`
	Meta          Meta              `json:"meta"`
	Idempotency   *IdempotencyState `json:"idempotency,omitempty"`
	Warnings      []string          `json:"warnings,omitempty"`
}

type fileGovernanceStore struct {
	paths runtimePaths
}

func newRuntimeGovernance(configPath string, cfg *config.Config, now func() time.Time) *runtimeGovernance {
	if now == nil {
		now = time.Now
	}
	if cfg == nil {
		cfg = config.New()
	}
	paths := resolveRuntimePaths(configPath)
	binding := config.ResolveStorageBinding(cfg, "governance")
	enabledPlugins := config.ResolveEnabledPlugins(cfg)
	sinks, sinkWarnings := openAuditSinks(cfg)
	return &runtimeGovernance{
		paths:        paths,
		store:        openGovernanceStore(paths, binding, enabledPlugins),
		sinks:        sinks,
		sinkWarnings: append([]string(nil), sinkWarnings...),
		retry:        resolveRetryPolicy(cfg.Runtime),
		now:          now,
	}
}

func resolveRuntimePaths(configPath string) runtimePaths {
	rootDir, err := locator.ResolveRuntimeDir(configPath)
	if err != nil {
		rootDir = filepath.Join(filepath.Dir(configPath), "runtime")
	}
	return runtimePaths{
		rootDir:        rootDir,
		idempotencyDir: filepath.Join(rootDir, "idempotency"),
		auditDir:       filepath.Join(rootDir, "audit"),
	}
}

func resolveRetryPolicy(runtimeConfig config.RuntimeConfig) retryPolicy {
	policy := retryPolicy{
		maxAttempts: runtimeConfig.Retry.MaxAttempts,
		baseDelay:   defaultRetryBaseDelay,
		maxDelay:    defaultRetryMaxDelay,
	}
	if runtimeConfig.Retry.BaseDelayMS > 0 {
		policy.baseDelay = time.Duration(runtimeConfig.Retry.BaseDelayMS) * time.Millisecond
	}
	if runtimeConfig.Retry.MaxDelayMS > 0 {
		policy.maxDelay = time.Duration(runtimeConfig.Retry.MaxDelayMS) * time.Millisecond
	}
	if policy.maxDelay < policy.baseDelay {
		policy.maxDelay = policy.baseDelay
	}
	if policy.maxAttempts < 0 {
		policy.maxAttempts = 0
	}
	return policy
}

func (g *runtimeGovernance) startIdempotency(state *IdempotencyState, operation string, requestID string, input map[string]any) (*persistedIdempotencyRecord, bool, error) {
	if state == nil {
		return nil, false, nil
	}
	if err := g.store.EnsureIdempotencyDir(); err != nil {
		return nil, false, err
	}

	inputHash, err := calculateInputHash(operation, input)
	if err != nil {
		return nil, false, err
	}

	existingRecord, err := g.store.LoadIdempotencyRecord(state.Key)
	if err != nil {
		return nil, false, err
	}
	if existingRecord != nil {
		return existingRecord, true, nil
	}

	now := g.now().UTC().Format(time.RFC3339)
	record := &persistedIdempotencyRecord{
		Key:       state.Key,
		Operation: operation,
		InputHash: inputHash,
		Status:    "in_progress",
		RequestID: requestID,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := g.store.SaveIdempotencyRecord(record); err != nil {
		return nil, false, err
	}

	state.Status = record.Status
	state.Persisted = true
	state.UpdatedAt = record.UpdatedAt
	return record, false, nil
}

func (g *runtimeGovernance) finishIdempotency(state *IdempotencyState, record *persistedIdempotencyRecord, envelope Envelope) error {
	if state == nil || record == nil {
		return nil
	}

	record.Status = "executed"
	if !envelope.OK {
		record.Status = "rejected"
	}
	record.UpdatedAt = g.now().UTC().Format(time.RFC3339)
	record.RetryCount = envelope.Meta.RetryCount
	record.Data = envelope.Data
	record.Error = envelope.Error
	record.Meta = envelope.Meta

	if err := g.store.SaveIdempotencyRecord(record); err != nil {
		return err
	}

	state.Status = record.Status
	state.Persisted = true
	state.UpdatedAt = record.UpdatedAt
	return nil
}

func (g *runtimeGovernance) buildReplayEnvelope(startAt time.Time, requestID string, profile ExecutionProfile, state *IdempotencyState, record *persistedIdempotencyRecord) Envelope {
	envelope := Envelope{
		OK:        record.Error == nil,
		Operation: record.Operation,
		RequestID: requestID,
		Data:      record.Data,
		Error:     record.Error,
		Meta: Meta{
			Platform:   record.Meta.Platform,
			DurationMS: time.Since(startAt).Milliseconds(),
			RetryCount: record.RetryCount,
			DryRun:     false,
		},
		Idempotency: state,
	}

	if state != nil {
		state.Persisted = true
		state.UpdatedAt = record.UpdatedAt
		switch record.Status {
		case "executed":
			state.Status = "replayed"
		default:
			state.Status = record.Status
		}
	}

	if profile.Name != "" || profile.Platform != "" || profile.Subject != "" {
		envelope.Context = &Context{
			Platform: profile.Platform,
			Subject:  profile.Subject,
			Account:  profile.Account,
		}
	}
	return envelope
}

func (g *runtimeGovernance) validateIdempotencyConflict(state *IdempotencyState, record *persistedIdempotencyRecord, operation string, input map[string]any) *apperr.AppError {
	if state == nil || record == nil {
		return nil
	}

	inputHash, err := calculateInputHash(operation, input)
	if err != nil {
		return apperr.New("IDEMPOTENCY_HASH_FAILED", err.Error())
	}

	if record.Operation != operation || record.InputHash != inputHash {
		state.Status = "rejected"
		state.Persisted = true
		state.UpdatedAt = record.UpdatedAt
		return apperr.New("IDEMPOTENCY_KEY_CONFLICT", "the provided idempotency key is already bound to a different write request")
	}
	return nil
}

func (g *runtimeGovernance) writeAudit(envelope Envelope, input map[string]any) []string {
	record := auditRecord{
		Time:          g.now().UTC().Format(time.RFC3339),
		RequestID:     envelope.RequestID,
		Operation:     envelope.Operation,
		Context:       envelope.Context,
		OK:            envelope.OK,
		InputSummary:  redactValue(input),
		OutputSummary: redactValue(envelope.Data),
		Error:         redactError(envelope.Error),
		Meta:          envelope.Meta,
		Idempotency:   cloneIdempotencyState(envelope.Idempotency),
		Warnings:      append([]string(nil), envelope.Warnings...),
	}

	warnings := make([]string, 0)
	warnings = append(warnings, g.sinkWarnings...)
	if err := g.store.AppendAuditRecord(g.now().UTC().Format("2006-01-02"), record); err != nil {
		warnings = append(warnings, "failed to write governance audit record: "+err.Error())
	}
	warnings = append(warnings, g.emitAuditSinks(record)...)
	return warnings
}

// closeSinks 关闭所有需要显式清理的审计 sink（如插件进程）。
// 在 runtimeGovernance 生命周期结束时调用。
func (g *runtimeGovernance) closeSinks() {
	if g == nil {
		return
	}
	closePluginAuditSinks(g.sinks)
}

func (g *runtimeGovernance) shouldRetry(definition adapter.Definition, appErr *apperr.AppError, retryCount int) bool {
	if appErr == nil || !appErr.Retryable {
		return false
	}
	if g.retry.maxAttempts <= 0 || retryCount >= g.retry.maxAttempts {
		return false
	}
	if definition.Mutating && !(definition.Spec.Idempotency.Required || definition.Spec.Idempotency.AutoGenerated) {
		return false
	}
	return true
}

func (g *runtimeGovernance) waitBeforeRetry(ctx context.Context, retryAttempt int) error {
	delay := g.retry.baseDelay
	for i := 1; i < retryAttempt; i++ {
		delay *= 2
		if delay >= g.retry.maxDelay {
			delay = g.retry.maxDelay
			break
		}
	}
	if delay > g.retry.maxDelay {
		delay = g.retry.maxDelay
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func calculateInputHash(operation string, input map[string]any) (string, error) {
	data, err := json.Marshal(input)
	if err != nil {
		return "", fmt.Errorf("failed to encode input summary: %w", err)
	}
	hash := sha256.Sum256([]byte(operation + ":" + string(data)))
	return hex.EncodeToString(hash[:]), nil
}

func safeFilename(value string) string {
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "..", "_")
	return replacer.Replace(value)
}

func atomicWriteJSONFile(path string, value any, perm os.FileMode) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}

	tempFile, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)

	if _, err := tempFile.Write(data); err != nil {
		tempFile.Close()
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tempPath, perm); err != nil {
		return err
	}
	return os.Rename(tempPath, path)
}

func openGovernanceStore(paths runtimePaths, binding config.StoragePluginBinding, enabledPlugins map[string]string) governanceStore {
	backend := strings.TrimSpace(strings.ToLower(binding.Backend))
	pluginName := strings.TrimSpace(binding.Plugin)
	if backend == "" || backend == "auto" {
		backend = "file"
	}

	if pluginName != "" && pluginName != "builtin" {
		store, ok, err := openPluginGovernanceStore(backend, pluginName, enabledPlugins)
		if err != nil {
			return &errorGovernanceStore{err: err}
		}
		if ok {
			return store
		}
		return &errorGovernanceStore{err: fmt.Errorf("unsupported governance backend: %s", backend)}
	}

	if factory, ok := governanceStoreFactories[backend]; ok {
		return factory(paths)
	}

	if store, ok, err := openPluginGovernanceStore(backend, pluginName, enabledPlugins); err == nil && ok {
		return store
	}
	return governanceStoreFactories["file"](paths)
}

func (s *fileGovernanceStore) EnsureIdempotencyDir() error {
	if err := os.MkdirAll(s.paths.idempotencyDir, 0o755); err != nil {
		return fmt.Errorf("failed to create idempotency directory: %w", err)
	}
	return nil
}

func (s *fileGovernanceStore) LoadIdempotencyRecord(key string) (*persistedIdempotencyRecord, error) {
	filePath := filepath.Join(s.paths.idempotencyDir, safeFilename(key)+".json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read idempotency record: %w", err)
	}

	var record persistedIdempotencyRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, fmt.Errorf("failed to decode idempotency record: %w", err)
	}
	return &record, nil
}

func (s *fileGovernanceStore) SaveIdempotencyRecord(record *persistedIdempotencyRecord) error {
	if err := os.MkdirAll(s.paths.idempotencyDir, 0o755); err != nil {
		return fmt.Errorf("failed to create idempotency directory: %w", err)
	}

	filePath := filepath.Join(s.paths.idempotencyDir, safeFilename(record.Key)+".json")
	if err := atomicWriteJSONFile(filePath, record, 0o600); err != nil {
		return fmt.Errorf("failed to write idempotency record: %w", err)
	}
	return nil
}

func (s *fileGovernanceStore) AppendAuditRecord(day string, record auditRecord) error {
	if err := os.MkdirAll(s.paths.auditDir, 0o755); err != nil {
		return fmt.Errorf("failed to create audit directory: %w", err)
	}

	filePath := filepath.Join(s.paths.auditDir, day+".jsonl")
	encoded, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("failed to encode audit record: %w", err)
	}

	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("failed to open audit log file: %w", err)
	}
	defer file.Close()

	if _, err := file.Write(append(encoded, '\n')); err != nil {
		return fmt.Errorf("failed to write audit log file: %w", err)
	}
	return nil
}

func buildRetryAbortError(err error) *apperr.AppError {
	if err == nil {
		return nil
	}
	return apperr.New("RETRY_ABORTED", err.Error()).WithRetryable(false)
}

func openPluginGovernanceStore(backend string, pluginName string, enabledPlugins map[string]string) (governanceStore, bool, error) {
	manifest, found, err := pluginruntime.FindStorageBackendManifest(pluginruntime.StorageBackendLookup{
		Target:         "governance",
		Backend:        backend,
		Plugin:         pluginName,
		EnabledPlugins: enabledPlugins,
	})
	if err != nil {
		return nil, false, err
	}
	if !found {
		return nil, false, nil
	}
	return &pluginGovernanceStore{
		client: pluginruntime.NewProcessGovernanceStore(manifest),
	}, true, nil
}

type pluginGovernanceStore struct {
	client *pluginruntime.ProcessGovernanceStore
}

type errorGovernanceStore struct {
	err error
}

func (s *errorGovernanceStore) EnsureIdempotencyDir() error {
	return s.err
}

func (s *errorGovernanceStore) LoadIdempotencyRecord(key string) (*persistedIdempotencyRecord, error) {
	return nil, s.err
}

func (s *errorGovernanceStore) SaveIdempotencyRecord(record *persistedIdempotencyRecord) error {
	return s.err
}

func (s *errorGovernanceStore) AppendAuditRecord(day string, record auditRecord) error {
	return s.err
}

func (s *pluginGovernanceStore) EnsureIdempotencyDir() error {
	// 外部治理后端自行处理目录或命名空间准备。
	return nil
}

func (s *pluginGovernanceStore) LoadIdempotencyRecord(key string) (*persistedIdempotencyRecord, error) {
	result, err := s.client.LoadIdempotency(context.Background(), pluginruntime.GovernanceIdempotencyLoadParams{
		Key: key,
	})
	if err != nil {
		return nil, err
	}
	if !result.Found || result.Record == nil {
		return nil, nil
	}
	return convertGovernanceRecordFromPlugin(*result.Record), nil
}

func (s *pluginGovernanceStore) SaveIdempotencyRecord(record *persistedIdempotencyRecord) error {
	if record == nil {
		return nil
	}
	return s.client.SaveIdempotency(context.Background(), pluginruntime.GovernanceIdempotencySaveParams{
		Record: convertGovernanceRecordToPlugin(*record),
	})
}

func (s *pluginGovernanceStore) AppendAuditRecord(day string, record auditRecord) error {
	return s.client.AppendAudit(context.Background(), pluginruntime.GovernanceAuditAppendParams{
		Day:    day,
		Record: convertAuditRecordToPlugin(record),
	})
}

func convertGovernanceRecordFromPlugin(record pluginruntime.GovernanceIdempotencyRecord) *persistedIdempotencyRecord {
	return &persistedIdempotencyRecord{
		Key:        record.Key,
		Operation:  record.Operation,
		InputHash:  record.InputHash,
		Status:     record.Status,
		RequestID:  record.RequestID,
		CreatedAt:  record.CreatedAt,
		UpdatedAt:  record.UpdatedAt,
		RetryCount: record.RetryCount,
		Data:       record.Data,
		Error:      convertGovernanceErrorFromPlugin(record.Error),
		Meta:       convertGovernanceMetaFromPlugin(record.Meta),
	}
}

func convertGovernanceRecordToPlugin(record persistedIdempotencyRecord) pluginruntime.GovernanceIdempotencyRecord {
	return pluginruntime.GovernanceIdempotencyRecord{
		Key:        record.Key,
		Operation:  record.Operation,
		InputHash:  record.InputHash,
		Status:     record.Status,
		RequestID:  record.RequestID,
		CreatedAt:  record.CreatedAt,
		UpdatedAt:  record.UpdatedAt,
		RetryCount: record.RetryCount,
		Data:       record.Data,
		Error:      convertGovernanceErrorToPlugin(record.Error),
		Meta:       convertGovernanceMetaToPlugin(record.Meta),
	}
}

func convertAuditRecordToPlugin(record auditRecord) pluginruntime.GovernanceAuditRecord {
	return pluginruntime.GovernanceAuditRecord{
		Time:          record.Time,
		RequestID:     record.RequestID,
		Operation:     record.Operation,
		Context:       convertGovernanceContextToPlugin(record.Context),
		OK:            record.OK,
		InputSummary:  record.InputSummary,
		OutputSummary: record.OutputSummary,
		Error:         convertGovernanceErrorToPlugin(record.Error),
		Meta:          convertGovernanceMetaToPlugin(record.Meta),
		Idempotency:   convertGovernanceIdempotencyStateToPlugin(record.Idempotency),
		Warnings:      append([]string(nil), record.Warnings...),
	}
}

func convertGovernanceErrorFromPlugin(value *pluginruntime.GovernanceErrorBody) *ErrorBody {
	if value == nil {
		return nil
	}
	return &ErrorBody{
		Code:         value.Code,
		Message:      value.Message,
		Retryable:    value.Retryable,
		UpstreamCode: value.UpstreamCode,
		HTTPStatus:   value.HTTPStatus,
	}
}

func convertGovernanceErrorToPlugin(value *ErrorBody) *pluginruntime.GovernanceErrorBody {
	if value == nil {
		return nil
	}
	return &pluginruntime.GovernanceErrorBody{
		Code:         value.Code,
		Message:      value.Message,
		Retryable:    value.Retryable,
		UpstreamCode: value.UpstreamCode,
		HTTPStatus:   value.HTTPStatus,
	}
}

func convertGovernanceMetaFromPlugin(value pluginruntime.GovernanceMeta) Meta {
	return Meta{
		Platform:   value.Platform,
		DurationMS: value.DurationMS,
		RetryCount: value.RetryCount,
		DryRun:     value.DryRun,
	}
}

func convertGovernanceMetaToPlugin(value Meta) pluginruntime.GovernanceMeta {
	return pluginruntime.GovernanceMeta{
		Platform:   value.Platform,
		DurationMS: value.DurationMS,
		RetryCount: value.RetryCount,
		DryRun:     value.DryRun,
	}
}

func convertGovernanceContextToPlugin(value *Context) *pluginruntime.GovernanceContext {
	if value == nil {
		return nil
	}
	return &pluginruntime.GovernanceContext{
		Platform: value.Platform,
		Subject:  value.Subject,
		Account:  value.Account,
	}
}

func convertGovernanceIdempotencyStateToPlugin(value *IdempotencyState) *pluginruntime.GovernanceIdempotencyState {
	if value == nil {
		return nil
	}
	return &pluginruntime.GovernanceIdempotencyState{
		Key:       value.Key,
		Status:    value.Status,
		Persisted: value.Persisted,
		UpdatedAt: value.UpdatedAt,
	}
}
