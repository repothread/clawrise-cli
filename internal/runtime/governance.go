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
)

const (
	defaultRetryBaseDelay = 200 * time.Millisecond
	defaultRetryMaxDelay  = 2 * time.Second
)

// runtimeGovernance 负责幂等、审计和重试能力。
type runtimeGovernance struct {
	paths runtimePaths
	retry retryPolicy
	now   func() time.Time
}

// runtimePaths 描述运行时治理数据的本地目录。
type runtimePaths struct {
	rootDir        string
	idempotencyDir string
	auditDir       string
}

// retryPolicy 描述自动重试配置。
type retryPolicy struct {
	maxAttempts int
	baseDelay   time.Duration
	maxDelay    time.Duration
}

// persistedIdempotencyRecord 是本地持久化的幂等记录。
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

// auditRecord 描述一次执行的审计摘要。
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
}

func newRuntimeGovernance(configPath string, runtimeConfig config.RuntimeConfig, now func() time.Time) *runtimeGovernance {
	if now == nil {
		now = time.Now
	}
	return &runtimeGovernance{
		paths: resolveRuntimePaths(configPath),
		retry: resolveRetryPolicy(runtimeConfig),
		now:   now,
	}
}

func resolveRuntimePaths(configPath string) runtimePaths {
	rootDir := filepath.Join(filepath.Dir(configPath), "runtime")
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
	if err := os.MkdirAll(g.paths.idempotencyDir, 0o755); err != nil {
		return nil, false, fmt.Errorf("failed to create idempotency directory: %w", err)
	}

	inputHash, err := calculateInputHash(operation, input)
	if err != nil {
		return nil, false, err
	}

	existingRecord, err := g.loadIdempotencyRecord(state.Key)
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
	if err := g.saveIdempotencyRecord(record); err != nil {
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

	if err := g.saveIdempotencyRecord(record); err != nil {
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
			Profile:  profile.Name,
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

func (g *runtimeGovernance) writeAudit(envelope Envelope, input map[string]any) error {
	if err := os.MkdirAll(g.paths.auditDir, 0o755); err != nil {
		return fmt.Errorf("failed to create audit directory: %w", err)
	}

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
	}

	filePath := filepath.Join(g.paths.auditDir, g.now().UTC().Format("2006-01-02")+".jsonl")
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

func (g *runtimeGovernance) loadIdempotencyRecord(key string) (*persistedIdempotencyRecord, error) {
	filePath := filepath.Join(g.paths.idempotencyDir, safeFilename(key)+".json")
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

func (g *runtimeGovernance) saveIdempotencyRecord(record *persistedIdempotencyRecord) error {
	if err := os.MkdirAll(g.paths.idempotencyDir, 0o755); err != nil {
		return fmt.Errorf("failed to create idempotency directory: %w", err)
	}

	filePath := filepath.Join(g.paths.idempotencyDir, safeFilename(record.Key)+".json")
	if err := atomicWriteJSONFile(filePath, record, 0o600); err != nil {
		return fmt.Errorf("failed to write idempotency record: %w", err)
	}
	return nil
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

func buildRetryAbortError(err error) *apperr.AppError {
	if err == nil {
		return nil
	}
	return apperr.New("RETRY_ABORTED", err.Error()).WithRetryable(false)
}
