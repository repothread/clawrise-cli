package plugin

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/clawrise/clawrise-cli/internal/adapter"
	speccatalog "github.com/clawrise/clawrise-cli/internal/spec/catalog"
)

type stubPluginRuntime struct {
	name         string
	handshake    HandshakeResult
	handshakeErr error
	definitions  []adapter.Definition
	listErr      error
	catalog      []speccatalog.Entry
	catalogErr   error
}

func (r *stubPluginRuntime) Name() string { return r.name }
func (r *stubPluginRuntime) Handshake(ctx context.Context) (HandshakeResult, error) {
	if r.handshakeErr != nil {
		return HandshakeResult{}, r.handshakeErr
	}
	return r.handshake, nil
}
func (r *stubPluginRuntime) ListOperations(ctx context.Context) ([]adapter.Definition, error) {
	if r.listErr != nil {
		return nil, r.listErr
	}
	return r.definitions, nil
}
func (r *stubPluginRuntime) GetCatalog(ctx context.Context) ([]speccatalog.Entry, error) {
	if r.catalogErr != nil {
		return nil, r.catalogErr
	}
	return r.catalog, nil
}
func (r *stubPluginRuntime) ListAuthMethods(ctx context.Context) ([]AuthMethodDescriptor, error) {
	return nil, nil
}
func (r *stubPluginRuntime) ListAuthPresets(ctx context.Context) ([]AuthPresetDescriptor, error) {
	return nil, nil
}
func (r *stubPluginRuntime) InspectAuth(ctx context.Context, params AuthInspectParams) (AuthInspectResult, error) {
	return AuthInspectResult{}, nil
}
func (r *stubPluginRuntime) BeginAuth(ctx context.Context, params AuthBeginParams) (AuthBeginResult, error) {
	return AuthBeginResult{}, nil
}
func (r *stubPluginRuntime) CompleteAuth(ctx context.Context, params AuthCompleteParams) (AuthCompleteResult, error) {
	return AuthCompleteResult{}, nil
}
func (r *stubPluginRuntime) ResolveAuth(ctx context.Context, params AuthResolveParams) (AuthResolveResult, error) {
	return AuthResolveResult{}, nil
}
func (r *stubPluginRuntime) Execute(ctx context.Context, req ExecuteRequest) (ExecuteResult, error) {
	return ExecuteResult{}, nil
}
func (r *stubPluginRuntime) Health(ctx context.Context) (HealthResult, error) {
	return HealthResult{OK: true}, nil
}

type stubAuthLauncherRuntime struct {
	name         string
	handshake    HandshakeResult
	handshakeErr error
	descriptor   AuthLauncherDescriptor
	describeErr  error
	launch       func(AuthLaunchParams) (AuthLaunchResult, error)
}

func (r *stubAuthLauncherRuntime) Name() string { return r.name }
func (r *stubAuthLauncherRuntime) Handshake(ctx context.Context) (HandshakeResult, error) {
	if r.handshakeErr != nil {
		return HandshakeResult{}, r.handshakeErr
	}
	return r.handshake, nil
}
func (r *stubAuthLauncherRuntime) DescribeAuthLauncher(ctx context.Context) (AuthLauncherDescriptor, error) {
	if r.describeErr != nil {
		return AuthLauncherDescriptor{}, r.describeErr
	}
	return r.descriptor, nil
}
func (r *stubAuthLauncherRuntime) LaunchAuth(ctx context.Context, params AuthLaunchParams) (AuthLaunchResult, error) {
	if r.launch == nil {
		return AuthLaunchResult{Handled: false}, nil
	}
	return r.launch(params)
}

func TestNewManagerWithOptionsRejectsRuntimeAndLauncherSetupErrors(t *testing.T) {
	baseDef := adapter.Definition{Operation: "demo.page.get", Platform: "demo"}
	for _, tc := range []struct {
		name     string
		runtimes []Runtime
		options  ManagerOptions
		wantErr  string
	}{
		{
			name:     "runtime handshake error",
			runtimes: []Runtime{&stubPluginRuntime{name: "bad", handshakeErr: errors.New("boom")}},
			wantErr:  "failed to handshake provider runtime bad",
		},
		{
			name:     "runtime empty handshake name",
			runtimes: []Runtime{&stubPluginRuntime{name: "bad", handshake: HandshakeResult{Platforms: []string{"demo"}}}},
			wantErr:  "returned an empty handshake name",
		},
		{
			name: "duplicate platform",
			runtimes: []Runtime{
				&stubPluginRuntime{name: "a", handshake: HandshakeResult{Name: "a", Platforms: []string{"demo"}}},
				&stubPluginRuntime{name: "b", handshake: HandshakeResult{Name: "b", Platforms: []string{"demo"}}},
			},
			wantErr: "duplicate provider runtime registered for platform: demo",
		},
		{
			name:     "list operations error",
			runtimes: []Runtime{&stubPluginRuntime{name: "bad", handshake: HandshakeResult{Name: "bad"}, listErr: errors.New("list failed")}},
			wantErr:  "failed to list operations for provider runtime bad",
		},
		{
			name: "duplicate operation",
			runtimes: []Runtime{
				&stubPluginRuntime{name: "a", handshake: HandshakeResult{Name: "a"}, definitions: []adapter.Definition{baseDef}},
				&stubPluginRuntime{name: "b", handshake: HandshakeResult{Name: "b"}, definitions: []adapter.Definition{baseDef}},
			},
			wantErr: "duplicate operation registered by provider runtimes: demo.page.get",
		},
		{
			name:     "launcher handshake error",
			runtimes: []Runtime{&stubPluginRuntime{name: "ok", handshake: HandshakeResult{Name: "ok"}}},
			options:  ManagerOptions{AuthLaunchers: []AuthLauncherRuntime{&stubAuthLauncherRuntime{name: "launcher", handshakeErr: errors.New("bad handshake")}}},
			wantErr:  "failed to handshake auth launcher launcher",
		},
		{
			name:     "launcher empty handshake name",
			runtimes: []Runtime{&stubPluginRuntime{name: "ok", handshake: HandshakeResult{Name: "ok"}}},
			options:  ManagerOptions{AuthLaunchers: []AuthLauncherRuntime{&stubAuthLauncherRuntime{name: "launcher", handshake: HandshakeResult{}}}},
			wantErr:  "auth launcher launcher returned an empty handshake name",
		},
		{
			name:     "launcher describe error",
			runtimes: []Runtime{&stubPluginRuntime{name: "ok", handshake: HandshakeResult{Name: "ok"}}},
			options:  ManagerOptions{AuthLaunchers: []AuthLauncherRuntime{&stubAuthLauncherRuntime{name: "launcher", handshake: HandshakeResult{Name: "launcher"}, describeErr: errors.New("describe failed")}}},
			wantErr:  "failed to describe auth launcher launcher",
		},
		{
			name:     "catalog error",
			runtimes: []Runtime{&stubPluginRuntime{name: "bad", handshake: HandshakeResult{Name: "bad"}, catalogErr: errors.New("catalog failed")}},
			wantErr:  "failed to load catalog for provider runtime bad",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewManagerWithOptions(context.Background(), tc.runtimes, tc.options)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestNewManagerWithOptionsAppliesLauncherDescriptorFallbacksAndHelpers(t *testing.T) {
	manager, err := NewManagerWithOptions(context.Background(), []Runtime{
		&stubPluginRuntime{name: "demo", handshake: HandshakeResult{Name: "demo", Platforms: []string{"demo"}}},
	}, ManagerOptions{
		AuthLaunchers: []AuthLauncherRuntime{
			&stubAuthLauncherRuntime{
				name:      "launcher-runtime",
				handshake: HandshakeResult{Name: "launcher-runtime"},
				descriptor: AuthLauncherDescriptor{
					ActionTypes: []string{"open_url"},
				},
			},
		},
		AuthLauncherPreferences: map[string][]string{" open_url ": {"launcher-runtime"}},
	})
	if err != nil {
		t.Fatalf("NewManagerWithOptions returned error: %v", err)
	}

	launchers := manager.AuthLaunchers()
	if len(launchers) != 1 || launchers[0].ID != "launcher-runtime" || launchers[0].DisplayName != "launcher-runtime" {
		t.Fatalf("unexpected launcher descriptors: %+v", launchers)
	}
	launchers[0].ID = "mutated"
	if manager.AuthLaunchers()[0].ID != "launcher-runtime" {
		t.Fatal("expected AuthLaunchers to return a copy")
	}

	if runtime, ok := manager.RuntimeForPlatform(" demo "); !ok || runtime == nil {
		t.Fatalf("expected trimmed platform lookup to succeed, got runtime=%v ok=%v", runtime, ok)
	}
	var nilManager *Manager
	if runtime, ok := nilManager.RuntimeForPlatform("demo"); ok || runtime != nil {
		t.Fatalf("expected nil manager platform lookup to fail, got runtime=%v ok=%v", runtime, ok)
	}
	if launchers := nilManager.AuthLaunchers(); launchers != nil {
		t.Fatalf("expected nil manager auth launchers to be nil, got %+v", launchers)
	}
}

func TestManagerLaunchAuthNoLauncherSkippedAndFailed(t *testing.T) {
	manager, err := NewManager(context.Background(), []Runtime{&stubPluginRuntime{name: "demo", handshake: HandshakeResult{Name: "demo", Platforms: []string{"demo"}}}})
	if err != nil {
		t.Fatalf("NewManager returned error: %v", err)
	}
	result, err := manager.LaunchAuth(context.Background(), AuthLaunchParams{Action: AuthAction{Type: "open_url"}})
	if err != nil {
		t.Fatalf("LaunchAuth returned error: %v", err)
	}
	if result.Handled || result.Status != "no_launcher_available" {
		t.Fatalf("unexpected no-launcher result: %+v", result)
	}

	skippedManager, err := NewManagerWithOptions(context.Background(), []Runtime{&stubPluginRuntime{name: "demo", handshake: HandshakeResult{Name: "demo", Platforms: []string{"demo"}}}}, ManagerOptions{
		AuthLaunchers: []AuthLauncherRuntime{
			&testLauncherRuntime{descriptor: AuthLauncherDescriptor{ID: "skip-a", ActionTypes: []string{"open_url"}}},
			&testLauncherRuntime{descriptor: AuthLauncherDescriptor{ID: "skip-b", ActionTypes: []string{"open_url"}}},
		},
	})
	if err != nil {
		t.Fatalf("NewManagerWithOptions returned error: %v", err)
	}
	result, err = skippedManager.LaunchAuth(context.Background(), AuthLaunchParams{Action: AuthAction{Type: "open_url"}})
	if err != nil {
		t.Fatalf("LaunchAuth returned error for skipped launchers: %v", err)
	}
	if result.Handled || result.Status != "skipped" || result.LauncherID != "skip-a" {
		t.Fatalf("unexpected skipped-launcher result: %+v", result)
	}

	failedManager, err := NewManagerWithOptions(context.Background(), []Runtime{&stubPluginRuntime{name: "demo", handshake: HandshakeResult{Name: "demo", Platforms: []string{"demo"}}}}, ManagerOptions{
		AuthLaunchers: []AuthLauncherRuntime{
			&testLauncherRuntime{descriptor: AuthLauncherDescriptor{ID: "fail-a", ActionTypes: []string{"open_url"}}, launch: func(params AuthLaunchParams) (AuthLaunchResult, error) {
				return AuthLaunchResult{}, errors.New("first failure")
			}},
			&testLauncherRuntime{descriptor: AuthLauncherDescriptor{ID: "fail-b", ActionTypes: []string{"open_url"}}, launch: func(params AuthLaunchParams) (AuthLaunchResult, error) {
				return AuthLaunchResult{}, errors.New("second failure")
			}},
		},
	})
	if err != nil {
		t.Fatalf("NewManagerWithOptions returned error: %v", err)
	}
	result, err = failedManager.LaunchAuth(context.Background(), AuthLaunchParams{Action: AuthAction{Type: "open_url"}})
	if err == nil || !strings.Contains(err.Error(), "second failure") {
		t.Fatalf("expected launcher failure to bubble up, got result=%+v err=%v", result, err)
	}
	if result.Status != "launcher_failed" || result.Message != "second failure" {
		t.Fatalf("unexpected failed-launcher result: %+v", result)
	}
}
