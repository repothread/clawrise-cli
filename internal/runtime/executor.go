package runtime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/clawrise/clawrise-cli/internal/adapter"
	"github.com/clawrise/clawrise-cli/internal/apperr"
	authcache "github.com/clawrise/clawrise-cli/internal/auth"
	"github.com/clawrise/clawrise-cli/internal/config"
	pluginruntime "github.com/clawrise/clawrise-cli/internal/plugin"
	"github.com/clawrise/clawrise-cli/internal/secretstore"
)

// Executor runs the normalized operation execution flow.
type Executor struct {
	store    *config.Store
	registry *adapter.Registry
	manager  *pluginruntime.Manager
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

// NewExecutorWithManager creates an executor backed by a provider manager.
func NewExecutorWithManager(store *config.Store, manager *pluginruntime.Manager) *Executor {
	if manager == nil {
		return NewExecutor(store, nil)
	}
	return &Executor{
		store:    store,
		registry: manager.Registry(),
		manager:  manager,
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

	operation, err := ParseOperationWithPlatforms(opts.OperationInput, strings.TrimSpace(cfg.Defaults.Platform), knownPlatforms(e.registry))
	if err != nil {
		return e.auditEnvelope(governance, e.buildFatalEnvelope(requestID, opts.DryRun, "", opts.OperationInput, apperr.New("INVALID_OPERATION", err.Error())), input), nil
	}

	definition, ok := e.registry.Resolve(operation.Normalized)
	if !ok {
		return e.auditEnvelope(governance, e.finish(startAt, requestID, operation.Normalized, operation.Platform, opts.DryRun, nil, nil, 0, apperr.New("OPERATION_NOT_FOUND", "operation is not registered"), ExecutionProfile{}), input), nil
	}
	canonicalOperation := definition.Operation

	connectionName, profile, appErr := resolveConnection(cfg, operation.Platform, opts.AccountName, opts.SubjectName)
	if appErr != nil {
		return e.auditEnvelope(governance, e.finish(startAt, requestID, canonicalOperation, operation.Platform, opts.DryRun, nil, nil, 0, appErr, ExecutionProfile{}), input), nil
	}

	if err := config.ValidateConnectionShape(profile); err != nil {
		return e.auditEnvelope(governance, e.finish(startAt, requestID, canonicalOperation, operation.Platform, opts.DryRun, nil, nil, 0, apperr.New("INVALID_AUTH_CONFIG", err.Error()), ExecutionProfile{}), input), nil
	}
	if !opts.DryRun && e.manager == nil {
		if err := config.ValidateGrant(profile); err != nil {
			return e.auditEnvelope(governance, e.finish(startAt, requestID, canonicalOperation, operation.Platform, opts.DryRun, nil, nil, 0, apperr.New("INVALID_AUTH_CONFIG", err.Error()), ExecutionProfile{}), input), nil
		}
	}

	resolvedProfile := profile
	if !opts.DryRun && e.manager != nil {
		account, ok := cfg.Accounts[connectionName]
		if !ok {
			return e.auditEnvelope(governance, e.finish(startAt, requestID, canonicalOperation, operation.Platform, false, nil, nil, 0, apperr.New("ACCOUNT_NOT_FOUND", fmt.Sprintf("account %s does not exist", connectionName)), ExecutionProfile{}), input), nil
		}
		nextProfile, appErr := e.resolveExecutionProfile(context.Background(), cfg, connectionName, account, profile)
		if appErr != nil {
			return e.auditEnvelope(governance, e.finish(startAt, requestID, canonicalOperation, operation.Platform, false, nil, nil, 0, appErr, ExecutionProfile{}), input), nil
		}
		resolvedProfile = nextProfile
	}

	stdinReader, _ := opts.Stdin.(io.Reader)
	input, err = ReadInput(opts.InputJSON, opts.InputFile, stdinReader)
	if err != nil {
		return e.auditEnvelope(governance, e.finish(startAt, requestID, canonicalOperation, operation.Platform, opts.DryRun, nil, nil, 0, apperr.New("INVALID_INPUT", err.Error()), ExecutionProfile{}), input), nil
	}

	if !contains(definition.AllowedSubjects, profile.Subject) {
		return e.auditEnvelope(governance, e.finish(startAt, requestID, canonicalOperation, operation.Platform, opts.DryRun, nil, nil, 0, apperr.New("SUBJECT_NOT_ALLOWED", fmt.Sprintf("connection %s with subject %s is not allowed to call %s", connectionName, profile.Subject, canonicalOperation)), ExecutionProfile{}), input), nil
	}

	idempotency := buildIdempotency(definition, opts.IdempotencyKey, canonicalOperation, input)
	executionProfile := ExecutionProfile{
		Name:     connectionName,
		Account:  connectionName,
		Platform: resolvedProfile.Platform,
		Subject:  resolvedProfile.Subject,
		Grant: map[string]any{
			"type": resolvedProfile.Grant.Type,
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
			"account": executionProfile,
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
			AccountName:    connectionName,
			Profile:        resolvedProfile,
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
			Account:  profile.Account,
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

func resolveConnection(cfg *config.Config, platform string, explicitAccount string, explicitSubject string) (string, config.Profile, *apperr.AppError) {
	cfg.Ensure()
	desiredSubject := strings.TrimSpace(explicitSubject)

	selectedAccount := strings.TrimSpace(explicitAccount)
	if desiredSubject == "" && selectedAccount == "" {
		desiredSubject = strings.TrimSpace(cfg.Defaults.Subject)
	}

	if selectedAccount != "" {
		account, ok := cfg.Accounts[selectedAccount]
		if !ok {
			return "", config.Profile{}, apperr.New("ACCOUNT_NOT_FOUND", fmt.Sprintf("account %s does not exist", selectedAccount))
		}
		if account.Platform != platform {
			return "", config.Profile{}, apperr.New("ACCOUNT_PLATFORM_MISMATCH", fmt.Sprintf("account %s belongs to platform %s and cannot be used for %s", selectedAccount, account.Platform, platform))
		}
		if desiredSubject != "" && account.Subject != desiredSubject {
			return "", config.Profile{}, apperr.New("ACCOUNT_SUBJECT_MISMATCH", fmt.Sprintf("account %s has subject %s and cannot be used when subject %s is selected", selectedAccount, account.Subject, desiredSubject))
		}
		return selectedAccount, cfg.ResolvedAccount(selectedAccount), nil
	}

	if defaultAccount := strings.TrimSpace(cfg.Defaults.PlatformAccounts[platform]); defaultAccount != "" {
		account, ok := cfg.Accounts[defaultAccount]
		if !ok {
			return "", config.Profile{}, apperr.New("DEFAULT_ACCOUNT_NOT_FOUND", fmt.Sprintf("default account %s does not exist", defaultAccount))
		}
		if account.Platform != platform {
			return "", config.Profile{}, apperr.New("DEFAULT_ACCOUNT_PLATFORM_MISMATCH", fmt.Sprintf("default account %s belongs to platform %s and cannot be used for %s", defaultAccount, account.Platform, platform))
		}
		if desiredSubject == "" || account.Subject == desiredSubject {
			return defaultAccount, cfg.ResolvedAccount(defaultAccount), nil
		}
	}
	if defaultAccount := strings.TrimSpace(cfg.Defaults.Account); defaultAccount != "" {
		account, ok := cfg.Accounts[defaultAccount]
		if !ok {
			return "", config.Profile{}, apperr.New("DEFAULT_ACCOUNT_NOT_FOUND", fmt.Sprintf("default account %s does not exist", defaultAccount))
		}
		if account.Platform != platform {
			return "", config.Profile{}, apperr.New("DEFAULT_ACCOUNT_PLATFORM_MISMATCH", fmt.Sprintf("default account %s belongs to platform %s and cannot be used for %s", defaultAccount, account.Platform, platform))
		}
		if desiredSubject == "" || account.Subject == desiredSubject {
			return defaultAccount, cfg.ResolvedAccount(defaultAccount), nil
		}
	}

	candidateNames := make([]string, 0)
	for name, account := range cfg.Accounts {
		if account.Platform != platform {
			continue
		}
		if desiredSubject != "" && account.Subject != desiredSubject {
			continue
		}
		candidateNames = append(candidateNames, name)
	}
	sort.Strings(candidateNames)
	switch len(candidateNames) {
	case 0:
		if desiredSubject != "" {
			return "", config.Profile{}, apperr.New("ACCOUNT_REQUIRED", fmt.Sprintf("platform %s has no available %s account; run `clawrise account use <name>` or pass --account", platform, desiredSubject))
		}
		return "", config.Profile{}, apperr.New("ACCOUNT_REQUIRED", fmt.Sprintf("platform %s has no available account; run `clawrise account use <name>` or pass --account", platform))
	case 1:
		return candidateNames[0], cfg.ResolvedAccount(candidateNames[0]), nil
	default:
		if desiredSubject != "" {
			return "", config.Profile{}, apperr.New("ACCOUNT_AMBIGUOUS", fmt.Sprintf("platform %s has multiple %s accounts; specify --account explicitly", platform, desiredSubject))
		}
		return "", config.Profile{}, apperr.New("ACCOUNT_AMBIGUOUS", fmt.Sprintf("platform %s has multiple candidate accounts; specify --account explicitly", platform))
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

func (e *Executor) resolveExecutionProfile(ctx context.Context, cfg *config.Config, accountName string, account config.Account, selectedProfile config.Profile) (config.Profile, *apperr.AppError) {
	if e.manager == nil {
		return selectedProfile, nil
	}

	authAccount, err := buildPluginAuthAccount(cfg, e.store.Path(), accountName, account)
	if err != nil {
		return config.Profile{}, apperr.New("AUTH_RESOLVE_FAILED", err.Error())
	}
	result, err := e.manager.ResolveAuth(ctx, account.Platform, pluginruntime.AuthResolveParams{
		Account: authAccount,
	})
	if err != nil {
		return config.Profile{}, apperr.New("AUTH_RESOLVE_FAILED", err.Error())
	}
	if !result.Ready {
		code := "AUTHORIZATION_REQUIRED"
		if result.Status == "invalid_auth_config" {
			code = "INVALID_AUTH_CONFIG"
		}
		message := strings.TrimSpace(result.Message)
		if message == "" {
			message = "authorization is not ready"
		}
		return config.Profile{}, apperr.New(code, message)
	}
	if err := persistAuthPatches(cfg, e.store.Path(), accountName, account, result.SessionPatch, result.SecretPatches); err != nil {
		return config.Profile{}, apperr.New("AUTH_STATE_PERSIST_FAILED", err.Error())
	}
	return buildResolvedProfile(selectedProfile, result.ExecutionAuth), nil
}

func buildPluginAuthAccount(cfg *config.Config, configPath string, accountName string, account config.Account) (pluginruntime.AuthAccount, error) {
	_ = cfg
	secrets := map[string]string{}
	for field, ref := range account.Auth.SecretRefs {
		value, err := config.ResolveSecret(ref)
		if err != nil {
			continue
		}
		secrets[field] = value
	}

	sessionStore, err := authcache.OpenStore(configPath, cfg.Auth.SessionStore.Backend)
	if err != nil {
		return pluginruntime.AuthAccount{}, err
	}
	var sessionPayload *pluginruntime.AuthSessionPayload
	if session, err := sessionStore.Load(accountName); err == nil {
		sessionPayload = pluginruntime.AuthSessionPayloadFromSession(session)
	}

	return pluginruntime.AuthAccount{
		Name:       accountName,
		Platform:   account.Platform,
		Subject:    account.Subject,
		AuthMethod: account.Auth.Method,
		Public:     cloneAnyMap(account.Auth.Public),
		Secrets:    secrets,
		Session:    sessionPayload,
	}, nil
}

func persistAuthPatches(cfg *config.Config, configPath string, accountName string, account config.Account, sessionPatch *pluginruntime.AuthSessionPayload, secretPatches map[string]string) error {
	if sessionPatch != nil {
		sessionStore, err := authcache.OpenStore(configPath, cfg.Auth.SessionStore.Backend)
		if err != nil {
			return err
		}
		session := sessionPatch.ToSession()
		session.AccountName = accountName
		session.Platform = account.Platform
		session.Subject = account.Subject
		session.GrantType = config.LegacyGrantTypeForMethod(account.Auth.Method)
		if err := sessionStore.Save(session); err != nil {
			return err
		}
	}

	if len(secretPatches) > 0 {
		backend := strings.TrimSpace(cfg.Auth.SecretStore.Backend)
		if backend == "" {
			backend = "auto"
		}
		secretStore, err := secretstore.Open(secretstore.Options{
			ConfigPath:      configPath,
			Backend:         backend,
			FallbackBackend: cfg.Auth.SecretStore.FallbackBackend,
		})
		if err != nil {
			return err
		}
		for field, value := range secretPatches {
			value = strings.TrimSpace(value)
			if value == "" {
				continue
			}
			if err := secretStore.Set(accountName, field, value); err != nil {
				return err
			}
		}
	}
	return nil
}

func buildResolvedProfile(selectedProfile config.Profile, executionAuth map[string]any) config.Profile {
	profile := selectedProfile
	profile.Grant.Type = asString(executionAuth["type"])
	profile.Grant.AccessToken = asString(executionAuth["access_token"])
	if value := asString(executionAuth["notion_version"]); value != "" {
		profile.Grant.NotionVer = value
	}
	return profile
}

func cloneAnyMap(values map[string]any) map[string]any {
	if len(values) == 0 {
		return nil
	}
	cloned := make(map[string]any, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func asString(value any) string {
	text, _ := value.(string)
	return strings.TrimSpace(text)
}

func knownPlatforms(registry *adapter.Registry) []string {
	if registry == nil {
		return nil
	}
	items := make([]string, 0)
	seen := map[string]struct{}{}
	for _, definition := range registry.Definitions() {
		platform := strings.TrimSpace(definition.Platform)
		if platform == "" {
			continue
		}
		if _, ok := seen[platform]; ok {
			continue
		}
		seen[platform] = struct{}{}
		items = append(items, platform)
	}
	return items
}
