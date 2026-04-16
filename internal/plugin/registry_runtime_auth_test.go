package plugin

import (
	"context"
	"strings"
	"testing"
	"time"

	authcache "github.com/clawrise/clawrise-cli/internal/auth"
)

type testAuthProvider struct {
	methods  []AuthMethodDescriptor
	presets  []AuthPresetDescriptor
	inspect  AuthInspectResult
	begin    AuthBeginResult
	complete AuthCompleteResult
	resolve  AuthResolveResult
}

func (p *testAuthProvider) ListMethods(ctx context.Context) ([]AuthMethodDescriptor, error) {
	return append([]AuthMethodDescriptor(nil), p.methods...), nil
}

func (p *testAuthProvider) ListPresets(ctx context.Context) ([]AuthPresetDescriptor, error) {
	return append([]AuthPresetDescriptor(nil), p.presets...), nil
}

func (p *testAuthProvider) Inspect(ctx context.Context, params AuthInspectParams) (AuthInspectResult, error) {
	return p.inspect, nil
}

func (p *testAuthProvider) Begin(ctx context.Context, params AuthBeginParams) (AuthBeginResult, error) {
	return p.begin, nil
}

func (p *testAuthProvider) Complete(ctx context.Context, params AuthCompleteParams) (AuthCompleteResult, error) {
	return p.complete, nil
}

func (p *testAuthProvider) Resolve(ctx context.Context, params AuthResolveParams) (AuthResolveResult, error) {
	return p.resolve, nil
}

func TestRegistryRuntimeAuthProviderPassthrough(t *testing.T) {
	expiresAt := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	provider := &testAuthProvider{
		methods: []AuthMethodDescriptor{{ID: "demo.token", Platform: "demo"}},
		presets: []AuthPresetDescriptor{{ID: "machine", Platform: "demo", AuthMethod: "demo.token"}},
		inspect: AuthInspectResult{Ready: true, Status: "ready"},
		begin: AuthBeginResult{
			HumanRequired: true,
			Flow: AuthFlowPayload{
				ID:     "flow_demo",
				Method: "demo.token",
			},
		},
		complete: AuthCompleteResult{
			Ready:  true,
			Status: "ready",
		},
		resolve: AuthResolveResult{
			Ready:         true,
			Status:        "ready",
			ExecutionAuth: map[string]any{"token": "demo-token"},
			SessionPatch: &AuthSessionPayload{
				AccessToken: "session-token",
				ExpiresAt:   expiresAt.Format(time.RFC3339),
			},
		},
	}

	runtime := NewRegistryRuntimeWithOptions("demo", "test", []string{"demo"}, buildDemoRegistry(), nil, RegistryRuntimeOptions{
		AuthProvider: provider,
	})

	methods, err := runtime.ListAuthMethods(context.Background())
	if err != nil {
		t.Fatalf("ListAuthMethods returned error: %v", err)
	}
	if len(methods) != 1 || methods[0].ID != "demo.token" {
		t.Fatalf("unexpected auth methods: %+v", methods)
	}

	presets, err := runtime.ListAuthPresets(context.Background())
	if err != nil {
		t.Fatalf("ListAuthPresets returned error: %v", err)
	}
	if len(presets) != 1 || presets[0].ID != "machine" {
		t.Fatalf("unexpected auth presets: %+v", presets)
	}

	inspect, err := runtime.InspectAuth(context.Background(), AuthInspectParams{})
	if err != nil {
		t.Fatalf("InspectAuth returned error: %v", err)
	}
	if !inspect.Ready || inspect.Status != "ready" {
		t.Fatalf("unexpected inspect result: %+v", inspect)
	}

	begin, err := runtime.BeginAuth(context.Background(), AuthBeginParams{})
	if err != nil {
		t.Fatalf("BeginAuth returned error: %v", err)
	}
	if !begin.HumanRequired || begin.Flow.ID != "flow_demo" {
		t.Fatalf("unexpected begin result: %+v", begin)
	}

	complete, err := runtime.CompleteAuth(context.Background(), AuthCompleteParams{})
	if err != nil {
		t.Fatalf("CompleteAuth returned error: %v", err)
	}
	if !complete.Ready || complete.Status != "ready" {
		t.Fatalf("unexpected complete result: %+v", complete)
	}

	resolve, err := runtime.ResolveAuth(context.Background(), AuthResolveParams{})
	if err != nil {
		t.Fatalf("ResolveAuth returned error: %v", err)
	}
	if !resolve.Ready || resolve.ExecutionAuth["token"] != "demo-token" || resolve.SessionPatch.AccessToken != "session-token" {
		t.Fatalf("unexpected resolve result: %+v", resolve)
	}

	health, err := runtime.Health(context.Background())
	if err != nil {
		t.Fatalf("Health returned error: %v", err)
	}
	if !health.OK || health.Details["name"] != "demo" {
		t.Fatalf("unexpected health result: %+v", health)
	}
}

func TestRegistryRuntimeAuthMethodsReturnHelpfulErrorsWithoutProvider(t *testing.T) {
	runtime := NewRegistryRuntime("demo", "test", []string{"demo"}, buildDemoRegistry(), nil)

	methods, err := runtime.ListAuthMethods(context.Background())
	if err != nil {
		t.Fatalf("ListAuthMethods returned error: %v", err)
	}
	if len(methods) != 0 {
		t.Fatalf("expected empty auth methods without provider, got: %+v", methods)
	}

	presets, err := runtime.ListAuthPresets(context.Background())
	if err != nil {
		t.Fatalf("ListAuthPresets returned error: %v", err)
	}
	if len(presets) != 0 {
		t.Fatalf("expected empty auth presets without provider, got: %+v", presets)
	}

	for _, fn := range []struct {
		name string
		call func() error
	}{
		{"InspectAuth", func() error { _, err := runtime.InspectAuth(context.Background(), AuthInspectParams{}); return err }},
		{"BeginAuth", func() error { _, err := runtime.BeginAuth(context.Background(), AuthBeginParams{}); return err }},
		{"CompleteAuth", func() error { _, err := runtime.CompleteAuth(context.Background(), AuthCompleteParams{}); return err }},
		{"ResolveAuth", func() error { _, err := runtime.ResolveAuth(context.Background(), AuthResolveParams{}); return err }},
	} {
		t.Run(fn.name, func(t *testing.T) {
			err := fn.call()
			if err == nil || !strings.Contains(err.Error(), "does not expose auth") {
				t.Fatalf("expected auth provider error, got: %v", err)
			}
		})
	}
}

func TestAuthSessionPayloadRoundTrip(t *testing.T) {
	expiresAt := time.Date(2026, 4, 10, 12, 30, 0, 0, time.UTC)
	session := &authcache.Session{
		AccessToken:  "access-demo",
		RefreshToken: "refresh-demo",
		TokenType:    "Bearer",
		Metadata:     map[string]string{"workspace": "demo"},
		ExpiresAt:    &expiresAt,
	}

	payload := AuthSessionPayloadFromSession(session)
	if payload == nil || payload.AccessToken != "access-demo" || payload.ExpiresAt != expiresAt.Format(time.RFC3339) {
		t.Fatalf("unexpected payload from session: %+v", payload)
	}
	if AuthSessionPayloadFromSession(nil) != nil {
		t.Fatal("expected nil payload for nil session")
	}

	roundTrip := payload.ToSession()
	if roundTrip.AccessToken != session.AccessToken || roundTrip.RefreshToken != session.RefreshToken || roundTrip.TokenType != session.TokenType {
		t.Fatalf("unexpected round-trip session: %+v", roundTrip)
	}
	if roundTrip.ExpiresAt == nil || !roundTrip.ExpiresAt.Equal(expiresAt) {
		t.Fatalf("expected expires_at to survive round-trip, got: %+v", roundTrip.ExpiresAt)
	}

	invalid := AuthSessionPayload{
		AccessToken: "access-demo",
		ExpiresAt:   "not-a-time",
	}.ToSession()
	if invalid.ExpiresAt != nil {
		t.Fatalf("expected invalid expires_at to be ignored, got: %+v", invalid.ExpiresAt)
	}
}
