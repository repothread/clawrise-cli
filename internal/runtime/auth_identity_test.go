package runtime

import (
	"context"
	"errors"
	"testing"

	"github.com/clawrise/clawrise-cli/internal/apperr"
	authcache "github.com/clawrise/clawrise-cli/internal/auth"
	"github.com/clawrise/clawrise-cli/internal/config"
	pluginruntime "github.com/clawrise/clawrise-cli/internal/plugin"
	"github.com/clawrise/clawrise-cli/internal/secretstore"
)

func TestNewExecutorWithManagerAndBuildFatalEnvelope(t *testing.T) {
	store := newTestStore(t, config.New())

	executor := NewExecutorWithManager(store, nil)
	if executor.manager != nil || executor.registry != nil {
		t.Fatalf("expected nil-manager executor to fall back to direct executor: %+v", executor)
	}

	manager := newRuntimeTestManager(t, "demo", &testRuntimeAuthProvider{})
	executor = NewExecutorWithManager(store, manager)
	if executor.manager == nil || executor.registry == nil {
		t.Fatalf("expected manager-backed executor to expose registry: %+v", executor)
	}

	envelope := executor.buildFatalEnvelope("req_demo", true, "demo", "demo.page.get", apperr.New("BROKEN", "boom").WithRetryable(false))
	if envelope.OK || envelope.Error == nil || envelope.Error.Code != "BROKEN" || !envelope.Meta.DryRun {
		t.Fatalf("unexpected fatal envelope: %+v", envelope)
	}
}

func TestExecutorResolveExecutionIdentityPersistsAuthPatches(t *testing.T) {
	t.Setenv("CLAWRISE_MASTER_KEY", "test-master-key")

	cfg := config.New()
	cfg.Ensure()
	cfg.Auth.SecretStore.Backend = "encrypted_file"
	cfg.Auth.SecretStore.FallbackBackend = "encrypted_file"
	cfg.Auth.SessionStore.Backend = "file"
	cfg.Accounts["notion_live"] = config.Account{Platform: "notion", Subject: "integration", Auth: config.AccountAuth{Method: "notion.oauth_public", Public: map[string]any{"client_id": "demo-client"}}}
	store := newTestStore(t, cfg)
	manager := newRuntimeTestManager(t, "notion", &testRuntimeAuthProvider{resolve: func(ctx context.Context, params pluginruntime.AuthResolveParams) (pluginruntime.AuthResolveResult, error) {
		return pluginruntime.AuthResolveResult{
			Ready:         true,
			Status:        "ready",
			ExecutionAuth: map[string]any{"bearer": "resolved-token"},
			SessionPatch:  &pluginruntime.AuthSessionPayload{AccessToken: "session-access", RefreshToken: "session-refresh", TokenType: "Bearer"},
			SecretPatches: map[string]string{"refresh_token": "persisted-refresh"},
		}, nil
	}})

	executor := NewExecutorWithManager(store, manager)
	identity, appErr := executor.resolveExecutionIdentity(context.Background(), cfg, "notion_live", cfg.Accounts["notion_live"])
	if appErr != nil {
		t.Fatalf("resolveExecutionIdentity returned error: %+v", appErr)
	}
	if identity.ExecutionAuth["bearer"] != "resolved-token" {
		t.Fatalf("unexpected resolved execution auth: %+v", identity)
	}

	sessionStore, err := authcache.OpenStoreWithOptions(authcache.StoreOptions{ConfigPath: store.Path(), Backend: "file"})
	if err != nil {
		t.Fatalf("failed to open session store: %v", err)
	}
	session, err := sessionStore.Load("notion_live")
	if err != nil {
		t.Fatalf("failed to load persisted session: %v", err)
	}
	if session.AccessToken != "session-access" || session.Platform != "notion" || session.Subject != "integration" || session.GrantType != "notion.oauth_public" {
		t.Fatalf("unexpected persisted session: %+v", session)
	}

	secretStore, err := secretstore.Open(secretstore.Options{ConfigPath: store.Path(), Backend: "encrypted_file", FallbackBackend: "encrypted_file"})
	if err != nil {
		t.Fatalf("failed to open secret store: %v", err)
	}
	refreshToken, err := secretStore.Get("notion_live", "refresh_token")
	if err != nil {
		t.Fatalf("failed to load persisted secret patch: %v", err)
	}
	if refreshToken != "persisted-refresh" {
		t.Fatalf("unexpected refresh token patch: %q", refreshToken)
	}
}

func TestExecutorResolveExecutionIdentityMapsAuthStatuses(t *testing.T) {
	cfg := config.New()
	cfg.Ensure()
	cfg.Accounts["notion_live"] = config.Account{Platform: "notion", Subject: "integration", Auth: config.AccountAuth{Method: "notion.oauth_public"}}
	store := newTestStore(t, cfg)

	tests := []struct {
		name         string
		result       pluginruntime.AuthResolveResult
		resolveErr   error
		expectedCode string
	}{
		{name: "invalid auth config", result: pluginruntime.AuthResolveResult{Ready: false, Status: "invalid_auth_config", Message: "missing client_secret"}, expectedCode: "INVALID_AUTH_CONFIG"},
		{name: "authorization required", result: pluginruntime.AuthResolveResult{Ready: false, Status: "authorization_required", Message: "login required"}, expectedCode: "AUTHORIZATION_REQUIRED"},
		{name: "provider failure", resolveErr: errors.New("provider down"), expectedCode: "AUTH_RESOLVE_FAILED"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			manager := newRuntimeTestManager(t, "notion", &testRuntimeAuthProvider{resolve: func(ctx context.Context, params pluginruntime.AuthResolveParams) (pluginruntime.AuthResolveResult, error) {
				if tc.resolveErr != nil {
					return pluginruntime.AuthResolveResult{}, tc.resolveErr
				}
				return tc.result, nil
			}})
			executor := NewExecutorWithManager(store, manager)
			_, appErr := executor.resolveExecutionIdentity(context.Background(), cfg, "notion_live", cfg.Accounts["notion_live"])
			if appErr == nil || appErr.Code != tc.expectedCode {
				t.Fatalf("expected app error code %s, got: %+v", tc.expectedCode, appErr)
			}
		})
	}
}

func TestExecutorBuildFallbackExecutionIdentityIncludesResolvedSecretsAndSession(t *testing.T) {
	t.Setenv("RUNTIME_TEST_TOKEN", " notion-token ")

	cfg := config.New()
	cfg.Ensure()
	cfg.Auth.SessionStore.Backend = "file"
	cfg.Accounts["notion_live"] = config.Account{Platform: "notion", Subject: "integration", Auth: config.AccountAuth{Method: "notion.internal_token", Public: map[string]any{"notion_version": "2026-03-11"}, SecretRefs: map[string]string{"token": "env:RUNTIME_TEST_TOKEN"}}}
	store := newTestStore(t, cfg)

	sessionStore, err := authcache.OpenStoreWithOptions(authcache.StoreOptions{ConfigPath: store.Path(), Backend: "file"})
	if err != nil {
		t.Fatalf("failed to open session store: %v", err)
	}
	if err := sessionStore.Save(authcache.Session{AccountName: "notion_live", Platform: "notion", Subject: "integration", GrantType: "notion.internal_token", AccessToken: "session-access", RefreshToken: "session-refresh"}); err != nil {
		t.Fatalf("failed to seed session store: %v", err)
	}

	executor := &Executor{store: store}
	identity, appErr := executor.buildFallbackExecutionIdentity(cfg, "notion_live", cfg.Accounts["notion_live"])
	if appErr != nil {
		t.Fatalf("buildFallbackExecutionIdentity returned error: %+v", appErr)
	}
	if identity.Secrets["token"] != "notion-token" {
		t.Fatalf("expected resolved and trimmed token secret, got: %+v", identity.Secrets)
	}
	if identity.Session == nil || identity.Session.AccessToken != "session-access" || identity.Session.AccountName != "notion_live" {
		t.Fatalf("unexpected fallback session identity: %+v", identity.Session)
	}
}
