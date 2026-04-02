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
	governance := newRuntimeGovernance(e.store.Path(), config.New(), e.now)
	var input map[string]any
	var warnings []string

	cfg, err := e.store.Load()
	if err != nil {
		return e.auditEnvelope(governance, e.buildFatalEnvelope(requestID, opts.DryRun, "", "", apperr.New("CONFIG_LOAD_FAILED", err.Error())), input), nil
	}
	governance = newRuntimeGovernance(e.store.Path(), cfg, e.now)

	operation, err := ParseOperationWithPlatforms(opts.OperationInput, strings.TrimSpace(cfg.Defaults.Platform), knownPlatforms(e.registry))
	if err != nil {
		return e.auditEnvelope(governance, e.buildFatalEnvelope(requestID, opts.DryRun, "", opts.OperationInput, apperr.New("INVALID_OPERATION", err.Error())), input), nil
	}

	definition, ok := e.registry.Resolve(operation.Normalized)
	if !ok {
		return e.auditEnvelope(governance, e.finish(startAt, requestID, operation.Normalized, operation.Platform, opts.DryRun, nil, nil, 0, apperr.New("OPERATION_NOT_FOUND", "operation is not registered"), ExecutionProfile{}), input), nil
	}
	canonicalOperation := definition.Operation

	accountName, account, appErr := resolveAccountSelection(cfg, operation.Platform, opts.AccountName, opts.SubjectName)
	if appErr != nil {
		return e.auditEnvelope(governance, e.finish(startAt, requestID, canonicalOperation, operation.Platform, opts.DryRun, nil, nil, 0, appErr, ExecutionProfile{}), input), nil
	}

	if err := config.ValidateAccountShape(accountName, account); err != nil {
		return e.auditEnvelope(governance, e.finish(startAt, requestID, canonicalOperation, operation.Platform, opts.DryRun, nil, nil, 0, apperr.New("INVALID_AUTH_CONFIG", err.Error()), ExecutionProfile{}), input), nil
	}
	if !opts.DryRun && e.manager == nil {
		if err := config.ValidateAccount(accountName, account); err != nil {
			return e.auditEnvelope(governance, e.finish(startAt, requestID, canonicalOperation, operation.Platform, opts.DryRun, nil, nil, 0, apperr.New("INVALID_AUTH_CONFIG", err.Error()), ExecutionProfile{}), input), nil
		}
	}

	identity := adapter.Identity{
		AccountName: accountName,
		Platform:    account.Platform,
		Subject:     account.Subject,
		AuthMethod:  strings.TrimSpace(account.Auth.Method),
	}
	if !opts.DryRun && e.manager != nil {
		nextIdentity, appErr := e.resolveExecutionIdentity(context.Background(), cfg, accountName, account)
		if appErr != nil {
			return e.auditEnvelope(governance, e.finish(startAt, requestID, canonicalOperation, operation.Platform, false, nil, nil, 0, appErr, ExecutionProfile{}), input), nil
		}
		identity = nextIdentity
	} else if !opts.DryRun {
		nextIdentity, appErr := e.buildFallbackExecutionIdentity(cfg, accountName, account)
		if appErr != nil {
			return e.auditEnvelope(governance, e.finish(startAt, requestID, canonicalOperation, operation.Platform, false, nil, nil, 0, appErr, ExecutionProfile{}), input), nil
		}
		identity = nextIdentity
	}

	stdinReader, _ := opts.Stdin.(io.Reader)
	input, err = ReadInput(opts.InputJSON, opts.InputFile, stdinReader)
	if err != nil {
		return e.auditEnvelope(governance, e.finish(startAt, requestID, canonicalOperation, operation.Platform, opts.DryRun, nil, nil, 0, apperr.New("INVALID_INPUT", err.Error()), ExecutionProfile{}), input), nil
	}

	if !contains(definition.AllowedSubjects, account.Subject) {
		return e.auditEnvelope(governance, e.finish(startAt, requestID, canonicalOperation, operation.Platform, opts.DryRun, nil, nil, 0, apperr.New("SUBJECT_NOT_ALLOWED", fmt.Sprintf("account %s with subject %s is not allowed to call %s", accountName, account.Subject, canonicalOperation)), ExecutionProfile{}), input), nil
	}

	warnings = append(warnings, writeEnhancementWarnings(canonicalOperation, opts)...)

	executionProfile := ExecutionProfile{
		Name:       accountName,
		Account:    accountName,
		Platform:   account.Platform,
		Subject:    account.Subject,
		AuthMethod: strings.TrimSpace(account.Auth.Method),
	}
	policyResult, policyWarnings, policyErr := e.evaluatePolicies(ctx, cfg, definition, requestID, canonicalOperation, input, executionProfile, opts.DryRun)
	warnings = append(warnings, policyWarnings...)
	if policyErr != nil {
		return e.auditEnvelope(governance, withExecutionMetadata(e.finish(startAt, requestID, canonicalOperation, operation.Platform, opts.DryRun, nil, nil, 0, policyErr, executionProfile), warnings, &policyResult), input), nil
	}

	idempotency := buildIdempotency(definition, opts.IdempotencyKey, canonicalOperation, input)

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
		return e.auditEnvelope(governance, withExecutionMetadata(e.finish(startAt, requestID, canonicalOperation, operation.Platform, true, data, idempotency, 0, nil, executionProfile), warnings, &policyResult), input), nil
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
		return e.auditEnvelope(governance, withExecutionMetadata(e.finish(startAt, requestID, canonicalOperation, operation.Platform, false, nil, idempotency, 0, apperr.New("NOT_IMPLEMENTED", "runtime skeleton is ready, but the real adapter is not implemented yet"), executionProfile), warnings, &policyResult), input), nil
	}

	var idempotencyRecord *persistedIdempotencyRecord
	if idempotency != nil {
		record, existed, err := governance.startIdempotency(idempotency, canonicalOperation, requestID, input)
		if err != nil {
			return e.auditEnvelope(governance, withExecutionMetadata(e.finish(startAt, requestID, canonicalOperation, operation.Platform, false, nil, idempotency, 0, apperr.New("IDEMPOTENCY_STORE_FAILED", err.Error()), executionProfile), warnings, &policyResult), input), nil
		}
		idempotencyRecord = record
		if existed {
			if appErr := governance.validateIdempotencyConflict(idempotency, record, canonicalOperation, input); appErr != nil {
				return e.auditEnvelope(governance, withExecutionMetadata(e.finish(startAt, requestID, canonicalOperation, operation.Platform, false, nil, idempotency, 0, appErr, executionProfile), warnings, &policyResult), input), nil
			}
			if record.Status == "in_progress" {
				idempotency.Status = record.Status
				idempotency.Persisted = true
				idempotency.UpdatedAt = record.UpdatedAt
				return e.auditEnvelope(governance, withExecutionMetadata(e.finish(startAt, requestID, canonicalOperation, operation.Platform, false, nil, idempotency, record.RetryCount, apperr.New("IDEMPOTENCY_IN_PROGRESS", "write request with the same idempotency key is already in progress"), executionProfile), warnings, &policyResult), input), nil
			}
			return e.auditEnvelope(governance, withExecutionMetadata(governance.buildReplayEnvelope(startAt, requestID, executionProfile, idempotency, record), warnings, &policyResult), input), nil
		}
	}

	idempotencyKey := ""
	if idempotency != nil {
		idempotencyKey = idempotency.Key
	}

	ctx = adapter.WithRuntimeOptions(ctx, adapter.RuntimeOptions{
		DebugProviderPayload: opts.DebugProviderPayload && !opts.DryRun && supportsProviderPayloadDebug(canonicalOperation),
		VerifyAfterWrite:     opts.VerifyAfterWrite && !opts.DryRun && supportsWriteVerification(canonicalOperation),
	})
	if opts.DebugProviderPayload && !opts.DryRun && supportsProviderPayloadDebug(canonicalOperation) {
		ctx, _ = adapter.WithProviderDebugCapture(ctx)
	}

	retryCount := 0
	var data map[string]any
	for {
		data, appErr = definition.Handler(ctx, adapter.Call{
			AccountName:    accountName,
			Identity:       identity,
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

	envelope := withExecutionMetadata(e.finish(startAt, requestID, canonicalOperation, operation.Platform, false, data, idempotency, retryCount, appErr, executionProfile), warnings, &policyResult)
	if idempotencyRecord != nil {
		if err := governance.finishIdempotency(idempotency, idempotencyRecord, envelope); err != nil {
			idempotency.Persisted = false
		}
	}
	envelope.Debug = adapter.ProviderDebugFromContext(ctx)
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
	envelope.Warnings = append(envelope.Warnings, governance.writeAudit(envelope, input)...)
	return envelope
}

func withExecutionMetadata(envelope Envelope, warnings []string, policy *PolicyResult) Envelope {
	if len(warnings) > 0 {
		envelope.Warnings = appendUniqueStrings(envelope.Warnings, warnings...)
	}
	if policy != nil {
		envelope.Policy = clonePolicyResult(policy)
	}
	return envelope
}

func clonePolicyResult(result *PolicyResult) *PolicyResult {
	if result == nil {
		return nil
	}

	cloned := &PolicyResult{
		FinalDecision: strings.TrimSpace(result.FinalDecision),
	}
	if cloned.FinalDecision == "" {
		cloned.FinalDecision = policyDecisionAllow
	}
	if len(result.Hits) > 0 {
		cloned.Hits = make([]PolicyHit, 0, len(result.Hits))
		for _, hit := range result.Hits {
			cloned.Hits = append(cloned.Hits, PolicyHit{
				SourceType:  strings.TrimSpace(hit.SourceType),
				SourceName:  strings.TrimSpace(hit.SourceName),
				Decision:    strings.TrimSpace(hit.Decision),
				Message:     strings.TrimSpace(hit.Message),
				MatchedRule: strings.TrimSpace(hit.MatchedRule),
				Annotations: cloneAnyMap(hit.Annotations),
			})
		}
	}
	return cloned
}

func appendUniqueStrings(existing []string, values ...string) []string {
	if len(values) == 0 {
		return existing
	}

	seen := make(map[string]struct{}, len(existing)+len(values))
	items := make([]string, 0, len(existing)+len(values))
	for _, value := range existing {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		items = append(items, value)
	}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		items = append(items, value)
	}
	if len(items) == 0 {
		return nil
	}
	return items
}

func resolveAccountSelection(cfg *config.Config, platform string, explicitAccount string, explicitSubject string) (string, config.Account, *apperr.AppError) {
	cfg.Ensure()
	desiredSubject := strings.TrimSpace(explicitSubject)

	selectedAccount := strings.TrimSpace(explicitAccount)
	if desiredSubject == "" && selectedAccount == "" {
		desiredSubject = strings.TrimSpace(cfg.Defaults.Subject)
	}

	if selectedAccount != "" {
		account, ok := cfg.Accounts[selectedAccount]
		if !ok {
			return "", config.Account{}, apperr.New("ACCOUNT_NOT_FOUND", fmt.Sprintf("account %s does not exist", selectedAccount))
		}
		if account.Platform != platform {
			return "", config.Account{}, apperr.New("ACCOUNT_PLATFORM_MISMATCH", fmt.Sprintf("account %s belongs to platform %s and cannot be used for %s", selectedAccount, account.Platform, platform))
		}
		if desiredSubject != "" && account.Subject != desiredSubject {
			return "", config.Account{}, apperr.New("ACCOUNT_SUBJECT_MISMATCH", fmt.Sprintf("account %s has subject %s and cannot be used when subject %s is selected", selectedAccount, account.Subject, desiredSubject))
		}
		return selectedAccount, account, nil
	}

	if defaultAccount := strings.TrimSpace(cfg.Defaults.PlatformAccounts[platform]); defaultAccount != "" {
		account, ok := cfg.Accounts[defaultAccount]
		if !ok {
			return "", config.Account{}, apperr.New("DEFAULT_ACCOUNT_NOT_FOUND", fmt.Sprintf("default account %s does not exist", defaultAccount))
		}
		if account.Platform != platform {
			return "", config.Account{}, apperr.New("DEFAULT_ACCOUNT_PLATFORM_MISMATCH", fmt.Sprintf("default account %s belongs to platform %s and cannot be used for %s", defaultAccount, account.Platform, platform))
		}
		if desiredSubject == "" || account.Subject == desiredSubject {
			return defaultAccount, account, nil
		}
	}
	if defaultAccount := strings.TrimSpace(cfg.Defaults.Account); defaultAccount != "" {
		account, ok := cfg.Accounts[defaultAccount]
		if !ok {
			return "", config.Account{}, apperr.New("DEFAULT_ACCOUNT_NOT_FOUND", fmt.Sprintf("default account %s does not exist", defaultAccount))
		}
		if account.Platform != platform {
			return "", config.Account{}, apperr.New("DEFAULT_ACCOUNT_PLATFORM_MISMATCH", fmt.Sprintf("default account %s belongs to platform %s and cannot be used for %s", defaultAccount, account.Platform, platform))
		}
		if desiredSubject == "" || account.Subject == desiredSubject {
			return defaultAccount, account, nil
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
			return "", config.Account{}, apperr.New("ACCOUNT_REQUIRED", fmt.Sprintf("platform %s has no available %s account; run `clawrise account use <name>` or pass --account", platform, desiredSubject))
		}
		return "", config.Account{}, apperr.New("ACCOUNT_REQUIRED", fmt.Sprintf("platform %s has no available account; run `clawrise account use <name>` or pass --account", platform))
	case 1:
		return candidateNames[0], cfg.Accounts[candidateNames[0]], nil
	default:
		if desiredSubject != "" {
			return "", config.Account{}, apperr.New("ACCOUNT_AMBIGUOUS", fmt.Sprintf("platform %s has multiple %s accounts; specify --account explicitly", platform, desiredSubject))
		}
		return "", config.Account{}, apperr.New("ACCOUNT_AMBIGUOUS", fmt.Sprintf("platform %s has multiple candidate accounts; specify --account explicitly", platform))
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

func writeEnhancementWarnings(operation string, opts ExecuteOptions) []string {
	warnings := make([]string, 0)
	if opts.DebugProviderPayload {
		switch {
		case opts.DryRun:
			warnings = append(warnings, "skipped --debug-provider-payload because --dry-run does not send upstream provider requests")
		case !supportsProviderPayloadDebug(operation):
			warnings = append(warnings, "operation "+operation+" currently ignores --debug-provider-payload; this flag is supported only for notion.page.create, notion.block.append, and notion.block.update")
		}
	}
	if opts.VerifyAfterWrite {
		switch {
		case opts.DryRun:
			warnings = append(warnings, "skipped --verify because --dry-run does not execute mutating operations")
		case !supportsWriteVerification(operation):
			warnings = append(warnings, "operation "+operation+" currently ignores --verify; this flag is supported only for notion.page.create, notion.block.append, and notion.block.update")
		}
	}
	return warnings
}

func supportsProviderPayloadDebug(operation string) bool {
	switch strings.TrimSpace(operation) {
	case "notion.page.create", "notion.block.append", "notion.block.update":
		return true
	default:
		return false
	}
}

func supportsWriteVerification(operation string) bool {
	switch strings.TrimSpace(operation) {
	case "notion.page.create", "notion.block.append", "notion.block.update":
		return true
	default:
		return false
	}
}

// ExecuteContext is kept as a small seam for future adapter integration.
func (e *Executor) ExecuteContext(ctx context.Context, opts ExecuteOptions) (Envelope, error) {
	return e.Execute(ctx, opts)
}

func (e *Executor) resolveExecutionIdentity(ctx context.Context, cfg *config.Config, accountName string, account config.Account) (adapter.Identity, *apperr.AppError) {
	authAccount, err := buildPluginAuthAccount(cfg, e.store.Path(), accountName, account)
	if err != nil {
		return adapter.Identity{}, apperr.New("AUTH_RESOLVE_FAILED", err.Error())
	}
	result, err := e.manager.ResolveAuth(ctx, account.Platform, pluginruntime.AuthResolveParams{
		Account: authAccount,
	})
	if err != nil {
		return adapter.Identity{}, apperr.New("AUTH_RESOLVE_FAILED", err.Error())
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
		return adapter.Identity{}, apperr.New(code, message)
	}
	if err := persistAuthPatches(cfg, e.store.Path(), accountName, account, result.SessionPatch, result.SecretPatches); err != nil {
		return adapter.Identity{}, apperr.New("AUTH_STATE_PERSIST_FAILED", err.Error())
	}
	return adapter.Identity{
		AccountName:   accountName,
		Platform:      account.Platform,
		Subject:       account.Subject,
		AuthMethod:    strings.TrimSpace(account.Auth.Method),
		ExecutionAuth: cloneAnyMap(result.ExecutionAuth),
	}, nil
}

func (e *Executor) buildFallbackExecutionIdentity(cfg *config.Config, accountName string, account config.Account) (adapter.Identity, *apperr.AppError) {
	authAccount, err := buildPluginAuthAccount(cfg, e.store.Path(), accountName, account)
	if err != nil {
		return adapter.Identity{}, apperr.New("AUTH_CONTEXT_BUILD_FAILED", err.Error())
	}

	identity := adapter.Identity{
		AccountName: accountName,
		Platform:    account.Platform,
		Subject:     account.Subject,
		AuthMethod:  strings.TrimSpace(account.Auth.Method),
		Public:      cloneAnyMap(authAccount.Public),
		Secrets:     cloneStringMap(authAccount.Secrets),
	}
	if authAccount.Session != nil {
		session := authAccount.Session.ToSession()
		session.AccountName = accountName
		session.Platform = account.Platform
		session.Subject = account.Subject
		identity.Session = &session
	}
	return identity, nil
}

func buildPluginAuthAccount(cfg *config.Config, configPath string, accountName string, account config.Account) (pluginruntime.AuthAccount, error) {
	enabledPlugins := config.ResolveEnabledPlugins(cfg)
	secrets := map[string]string{}
	for field, ref := range account.Auth.SecretRefs {
		value, err := config.ResolveSecret(ref)
		if err != nil {
			continue
		}
		secrets[field] = value
	}

	sessionBinding := config.ResolveStorageBinding(cfg, "session_store")
	sessionStore, err := authcache.OpenStoreWithOptions(authcache.StoreOptions{
		ConfigPath:     configPath,
		Backend:        sessionBinding.Backend,
		Plugin:         sessionBinding.Plugin,
		EnabledPlugins: enabledPlugins,
	})
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
	enabledPlugins := config.ResolveEnabledPlugins(cfg)
	if sessionPatch != nil {
		sessionBinding := config.ResolveStorageBinding(cfg, "session_store")
		sessionStore, err := authcache.OpenStoreWithOptions(authcache.StoreOptions{
			ConfigPath:     configPath,
			Backend:        sessionBinding.Backend,
			Plugin:         sessionBinding.Plugin,
			EnabledPlugins: enabledPlugins,
		})
		if err != nil {
			return err
		}
		session := sessionPatch.ToSession()
		session.AccountName = accountName
		session.Platform = account.Platform
		session.Subject = account.Subject
		session.GrantType = strings.TrimSpace(account.Auth.Method)
		if err := sessionStore.Save(session); err != nil {
			return err
		}
	}

	if len(secretPatches) > 0 {
		binding := config.ResolveStorageBinding(cfg, "secret_store")
		backend := strings.TrimSpace(binding.Backend)
		if backend == "" {
			backend = "auto"
		}
		secretStore, err := secretstore.Open(secretstore.Options{
			ConfigPath:      configPath,
			Backend:         backend,
			FallbackBackend: binding.FallbackBackend,
			Plugin:          binding.Plugin,
			EnabledPlugins:  enabledPlugins,
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

func cloneStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = strings.TrimSpace(value)
	}
	return cloned
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
