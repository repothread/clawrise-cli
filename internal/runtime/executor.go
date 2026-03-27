package runtime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/clawrise/clawrise-cli/internal/adapter"
	"github.com/clawrise/clawrise-cli/internal/apperr"
	"github.com/clawrise/clawrise-cli/internal/config"
)

// Executor runs the normalized operation execution flow.
type Executor struct {
	store    *config.Store
	registry *adapter.Registry
	now      func() time.Time
}

// NewExecutor creates a new executor.
func NewExecutor(store *config.Store, registry *adapter.Registry) *Executor {
	return &Executor{
		store:    store,
		registry: registry,
		now:      time.Now,
	}
}

// Execute runs one operation. At this stage it supports dry-run and the
// normalized validation/error path, but not real adapter execution yet.
func (e *Executor) Execute(ctx context.Context, opts ExecuteOptions) (Envelope, error) {
	startAt := e.now()
	requestID := buildRequestID(startAt)
	governance := newRuntimeGovernance(e.store.Path(), config.RuntimeConfig{}, e.now)
	var input map[string]any

	cfg, err := e.store.Load()
	if err != nil {
		return e.auditEnvelope(governance, e.buildFatalEnvelope(requestID, opts.DryRun, "", "", apperr.New("CONFIG_LOAD_FAILED", err.Error())), input), nil
	}
	governance = newRuntimeGovernance(e.store.Path(), cfg.Runtime, e.now)

	operation, err := ParseOperation(opts.OperationInput, strings.TrimSpace(cfg.Defaults.Platform))
	if err != nil {
		return e.auditEnvelope(governance, e.buildFatalEnvelope(requestID, opts.DryRun, "", opts.OperationInput, apperr.New("INVALID_OPERATION", err.Error())), input), nil
	}

	definition, ok := e.registry.Resolve(operation.Normalized)
	if !ok {
		return e.auditEnvelope(governance, e.finish(startAt, requestID, operation.Normalized, operation.Platform, opts.DryRun, nil, nil, 0, apperr.New("OPERATION_NOT_FOUND", "operation is not registered"), ExecutionProfile{}), input), nil
	}
	canonicalOperation := definition.Operation

	profileName, profile, appErr := resolveProfile(cfg, operation.Platform, opts.ProfileName, opts.SubjectName)
	if appErr != nil {
		return e.auditEnvelope(governance, e.finish(startAt, requestID, canonicalOperation, operation.Platform, opts.DryRun, nil, nil, 0, appErr, ExecutionProfile{}), input), nil
	}

	if err := config.ValidateGrant(profile); err != nil {
		return e.auditEnvelope(governance, e.finish(startAt, requestID, canonicalOperation, operation.Platform, opts.DryRun, nil, nil, 0, apperr.New("INVALID_AUTH_CONFIG", err.Error()), ExecutionProfile{}), input), nil
	}

	stdinReader, _ := opts.Stdin.(io.Reader)
	input, err = ReadInput(opts.InputJSON, opts.InputFile, stdinReader)
	if err != nil {
		return e.auditEnvelope(governance, e.finish(startAt, requestID, canonicalOperation, operation.Platform, opts.DryRun, nil, nil, 0, apperr.New("INVALID_INPUT", err.Error()), ExecutionProfile{}), input), nil
	}

	if !contains(definition.AllowedSubjects, profile.Subject) {
		return e.auditEnvelope(governance, e.finish(startAt, requestID, canonicalOperation, operation.Platform, opts.DryRun, nil, nil, 0, apperr.New("SUBJECT_NOT_ALLOWED", fmt.Sprintf("profile %s with subject %s is not allowed to call %s", profileName, profile.Subject, canonicalOperation)), ExecutionProfile{}), input), nil
	}

	idempotency := buildIdempotency(definition, opts.IdempotencyKey, canonicalOperation, input)
	executionProfile := ExecutionProfile{
		Name:     profileName,
		Platform: profile.Platform,
		Subject:  profile.Subject,
		Grant: map[string]any{
			"type": profile.Grant.Type,
		},
	}

	if opts.DryRun {
		data := map[string]any{
			"dry_run": true,
			"operation": map[string]any{
				"raw":            operation.Raw,
				"normalized":     canonicalOperation,
				"platform":       operation.Platform,
				"resource_path":  operation.ResourcePath,
				"action":         operation.Action,
				"defaulted_from": cfg.Defaults.Platform,
			},
			"profile": executionProfile,
			"input":   input,
		}
		if idempotency != nil {
			idempotency.Status = "dry_run"
		}
		return e.auditEnvelope(governance, e.finish(startAt, requestID, canonicalOperation, operation.Platform, true, data, idempotency, 0, nil, executionProfile), input), nil
	}

	timeout := definition.DefaultTimeout
	if opts.Timeout > 0 {
		timeout = opts.Timeout
	}
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	if definition.Handler == nil {
		return e.auditEnvelope(governance, e.finish(startAt, requestID, canonicalOperation, operation.Platform, false, nil, idempotency, 0, apperr.New("NOT_IMPLEMENTED", "runtime skeleton is ready, but the real adapter is not implemented yet"), executionProfile), input), nil
	}

	var idempotencyRecord *persistedIdempotencyRecord
	if idempotency != nil {
		record, existed, err := governance.startIdempotency(idempotency, canonicalOperation, requestID, input)
		if err != nil {
			return e.auditEnvelope(governance, e.finish(startAt, requestID, canonicalOperation, operation.Platform, false, nil, idempotency, 0, apperr.New("IDEMPOTENCY_STORE_FAILED", err.Error()), executionProfile), input), nil
		}
		idempotencyRecord = record
		if existed {
			if appErr := governance.validateIdempotencyConflict(idempotency, record, canonicalOperation, input); appErr != nil {
				return e.auditEnvelope(governance, e.finish(startAt, requestID, canonicalOperation, operation.Platform, false, nil, idempotency, 0, appErr, executionProfile), input), nil
			}
			if record.Status == "in_progress" {
				idempotency.Status = record.Status
				idempotency.Persisted = true
				idempotency.UpdatedAt = record.UpdatedAt
				return e.auditEnvelope(governance, e.finish(startAt, requestID, canonicalOperation, operation.Platform, false, nil, idempotency, record.RetryCount, apperr.New("IDEMPOTENCY_IN_PROGRESS", "write request with the same idempotency key is already in progress"), executionProfile), input), nil
			}
			return e.auditEnvelope(governance, governance.buildReplayEnvelope(startAt, requestID, executionProfile, idempotency, record), input), nil
		}
	}

	idempotencyKey := ""
	if idempotency != nil {
		idempotencyKey = idempotency.Key
	}

	retryCount := 0
	var data map[string]any
	for {
		data, appErr = definition.Handler(ctx, adapter.Call{
			Profile:        profile,
			Input:          input,
			IdempotencyKey: idempotencyKey,
		})
		if !governance.shouldRetry(definition, appErr, retryCount) {
			break
		}

		retryCount++
		if err := governance.waitBeforeRetry(ctx, retryCount); err != nil {
			appErr = buildRetryAbortError(err)
			break
		}
	}

	envelope := e.finish(startAt, requestID, canonicalOperation, operation.Platform, false, data, idempotency, retryCount, appErr, executionProfile)
	if idempotencyRecord != nil {
		if err := governance.finishIdempotency(idempotency, idempotencyRecord, envelope); err != nil {
			idempotency.Persisted = false
		}
	}
	return e.auditEnvelope(governance, envelope, input), nil
}

func (e *Executor) buildFatalEnvelope(requestID string, dryRun bool, platform string, operation string, appErr *apperr.AppError) Envelope {
	return Envelope{
		OK:        false,
		Operation: operation,
		RequestID: requestID,
		Data:      nil,
		Error: &ErrorBody{
			Code:         appErr.Code,
			Message:      appErr.Message,
			Retryable:    appErr.Retryable,
			UpstreamCode: appErr.UpstreamCode,
			HTTPStatus:   appErr.HTTPStatus,
		},
		Meta: Meta{
			Platform:   platform,
			DurationMS: 0,
			RetryCount: 0,
			DryRun:     dryRun,
		},
	}
}

func (e *Executor) finish(startAt time.Time, requestID, operation, platform string, dryRun bool, data any, idempotency *IdempotencyState, retryCount int, appErr *apperr.AppError, profile ExecutionProfile) Envelope {
	envelope := Envelope{
		OK:        appErr == nil,
		Operation: operation,
		RequestID: requestID,
		Data:      data,
		Error:     nil,
		Meta: Meta{
			Platform:   platform,
			DurationMS: time.Since(startAt).Milliseconds(),
			RetryCount: retryCount,
			DryRun:     dryRun,
		},
		Idempotency: idempotency,
	}

	if profile.Name != "" || profile.Platform != "" || profile.Subject != "" {
		envelope.Context = &Context{
			Platform: profile.Platform,
			Subject:  profile.Subject,
			Profile:  profile.Name,
		}
	}

	if appErr != nil {
		envelope.Error = &ErrorBody{
			Code:         appErr.Code,
			Message:      appErr.Message,
			Retryable:    appErr.Retryable,
			UpstreamCode: appErr.UpstreamCode,
			HTTPStatus:   appErr.HTTPStatus,
		}
	}
	return envelope
}

func (e *Executor) auditEnvelope(governance *runtimeGovernance, envelope Envelope, input map[string]any) Envelope {
	if governance == nil || envelope.Meta.DryRun {
		return envelope
	}
	_ = governance.writeAudit(envelope, input)
	return envelope
}

func resolveProfile(cfg *config.Config, platform string, explicitProfile string, explicitSubject string) (string, config.Profile, *apperr.AppError) {
	cfg.Ensure()
	desiredSubject := strings.TrimSpace(explicitSubject)
	if desiredSubject == "" {
		desiredSubject = strings.TrimSpace(cfg.Defaults.Subject)
	}

	if explicitProfile != "" {
		profile, ok := cfg.Profiles[explicitProfile]
		if !ok {
			return "", config.Profile{}, apperr.New("PROFILE_NOT_FOUND", fmt.Sprintf("profile %s does not exist", explicitProfile))
		}
		if profile.Platform != platform {
			return "", config.Profile{}, apperr.New("PROFILE_PLATFORM_MISMATCH", fmt.Sprintf("profile %s belongs to platform %s and cannot be used for %s", explicitProfile, profile.Platform, platform))
		}
		if desiredSubject != "" && profile.Subject != desiredSubject {
			return "", config.Profile{}, apperr.New("PROFILE_SUBJECT_MISMATCH", fmt.Sprintf("profile %s has subject %s and cannot be used when subject %s is selected", explicitProfile, profile.Subject, desiredSubject))
		}
		return explicitProfile, profile, nil
	}

	if cfg.Defaults.Profile != "" {
		profile, ok := cfg.Profiles[cfg.Defaults.Profile]
		if !ok {
			return "", config.Profile{}, apperr.New("DEFAULT_PROFILE_NOT_FOUND", fmt.Sprintf("default profile %s does not exist", cfg.Defaults.Profile))
		}
		if profile.Platform != platform {
			return "", config.Profile{}, apperr.New("DEFAULT_PROFILE_PLATFORM_MISMATCH", fmt.Sprintf("default profile %s belongs to platform %s and cannot be used for %s", cfg.Defaults.Profile, profile.Platform, platform))
		}
		if desiredSubject != "" && profile.Subject != desiredSubject {
			return "", config.Profile{}, apperr.New("DEFAULT_PROFILE_SUBJECT_MISMATCH", fmt.Sprintf("default profile %s has subject %s and cannot be used when subject %s is selected", cfg.Defaults.Profile, profile.Subject, desiredSubject))
		}
		return cfg.Defaults.Profile, profile, nil
	}

	candidates := cfg.CandidateProfilesBySubject(platform, desiredSubject)
	switch len(candidates) {
	case 0:
		if desiredSubject != "" {
			return "", config.Profile{}, apperr.New("PROFILE_REQUIRED", fmt.Sprintf("platform %s has no available %s profile; run `clawrise profile use <name>` or pass --profile", platform, desiredSubject))
		}
		return "", config.Profile{}, apperr.New("PROFILE_REQUIRED", fmt.Sprintf("platform %s has no available profile; run `clawrise profile use <name>` or pass --profile", platform))
	case 1:
		return candidates[0].Name, candidates[0].Profile, nil
	default:
		if desiredSubject != "" {
			return "", config.Profile{}, apperr.New("PROFILE_AMBIGUOUS", fmt.Sprintf("platform %s has multiple %s profiles; specify --profile explicitly", platform, desiredSubject))
		}
		return "", config.Profile{}, apperr.New("PROFILE_AMBIGUOUS", fmt.Sprintf("platform %s has multiple candidate profiles; specify --profile explicitly", platform))
	}
}

func buildIdempotency(definition adapter.Definition, explicitKey string, operation string, input map[string]any) *IdempotencyState {
	if !definition.Mutating {
		return nil
	}

	key := strings.TrimSpace(explicitKey)
	if key == "" {
		encoded, err := json.Marshal(input)
		if err != nil {
			encoded = []byte(fmt.Sprintf("%s:%v", operation, input))
		}
		hash := sha256.Sum256([]byte(operation + ":" + string(encoded)))
		key = "idem_" + hex.EncodeToString(hash[:])
	}

	return &IdempotencyState{
		Key:    key,
		Status: "prepared",
	}
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func buildRequestID(now time.Time) string {
	hash := sha256.Sum256([]byte(now.UTC().Format(time.RFC3339Nano)))
	return "req_" + hex.EncodeToString(hash[:6])
}

// ExecuteContext is kept as a small seam for future adapter integration.
func (e *Executor) ExecuteContext(ctx context.Context, opts ExecuteOptions) (Envelope, error) {
	return e.Execute(ctx, opts)
}
