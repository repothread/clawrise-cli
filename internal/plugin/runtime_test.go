package plugin

import (
	"context"
	"testing"
	"time"

	"github.com/clawrise/clawrise-cli/internal/adapter"
	"github.com/clawrise/clawrise-cli/internal/apperr"
	"github.com/clawrise/clawrise-cli/internal/config"
	speccatalog "github.com/clawrise/clawrise-cli/internal/spec/catalog"
)

func TestManagerRegistersOperationsThroughRuntimeBoundary(t *testing.T) {
	runtime := NewRegistryRuntime("demo", "test", []string{"demo"}, buildDemoRegistry(), []speccatalog.Entry{
		{Operation: "demo.page.get"},
	})

	manager, err := NewManager(context.Background(), []Runtime{runtime})
	if err != nil {
		t.Fatalf("NewManager returned error: %v", err)
	}

	definition, ok := manager.Registry().Resolve("demo.page.get")
	if !ok {
		t.Fatal("expected demo.page.get to be registered")
	}
	if definition.Handler == nil {
		t.Fatal("expected aggregated definition handler to be wired")
	}

	data, appErr := definition.Handler(context.Background(), adapter.Call{
		Profile: config.Profile{
			Platform: "demo",
			Subject:  "integration",
		},
		Input: map[string]any{
			"id": "page_demo",
		},
	})
	if appErr != nil {
		t.Fatalf("expected successful execution, got: %+v", appErr)
	}
	if data["id"] != "page_demo" {
		t.Fatalf("unexpected handler data: %+v", data)
	}
}

func TestManagerAggregatesCatalogEntries(t *testing.T) {
	manager, err := NewManager(context.Background(), []Runtime{
		NewRegistryRuntime("demo", "test", []string{"demo"}, buildDemoRegistry(), []speccatalog.Entry{
			{Operation: "demo.page.get"},
		}),
	})
	if err != nil {
		t.Fatalf("NewManager returned error: %v", err)
	}

	entries := manager.CatalogEntries()
	if len(entries) != 1 || entries[0].Operation != "demo.page.get" {
		t.Fatalf("unexpected catalog entries: %+v", entries)
	}
}

func TestManagerLaunchAuthUsesHighestPriorityMatchingLauncher(t *testing.T) {
	calls := []string{}
	manager, err := NewManagerWithOptions(context.Background(), []Runtime{
		NewRegistryRuntime("demo", "test", []string{"demo"}, buildDemoRegistry(), nil),
	}, ManagerOptions{
		AuthLaunchers: []AuthLauncherRuntime{
			&testLauncherRuntime{
				descriptor: AuthLauncherDescriptor{
					ID:          "low",
					DisplayName: "low",
					ActionTypes: []string{"open_url"},
					Priority:    10,
				},
				launch: func(params AuthLaunchParams) (AuthLaunchResult, error) {
					calls = append(calls, "low")
					return AuthLaunchResult{Handled: true, Status: "launched", LauncherID: "low"}, nil
				},
			},
			&testLauncherRuntime{
				descriptor: AuthLauncherDescriptor{
					ID:          "high",
					DisplayName: "high",
					ActionTypes: []string{"open_url"},
					Priority:    100,
				},
				launch: func(params AuthLaunchParams) (AuthLaunchResult, error) {
					calls = append(calls, "high")
					return AuthLaunchResult{Handled: true, Status: "launched", LauncherID: "high"}, nil
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("NewManagerWithOptions returned error: %v", err)
	}

	result, err := manager.LaunchAuth(context.Background(), AuthLaunchParams{
		Context: AuthLaunchContext{
			AccountName: "demo_account",
			Platform:    "demo",
		},
		Action: AuthAction{
			Type: "open_url",
			URL:  "https://example.com/auth",
		},
	})
	if err != nil {
		t.Fatalf("LaunchAuth returned error: %v", err)
	}
	if !result.Handled || result.LauncherID != "high" {
		t.Fatalf("unexpected launch result: %+v", result)
	}
	if len(calls) != 1 || calls[0] != "high" {
		t.Fatalf("expected only high priority launcher to run, got: %+v", calls)
	}
}

func TestManagerLaunchAuthReturnsNoMatchingLauncher(t *testing.T) {
	manager, err := NewManagerWithOptions(context.Background(), []Runtime{
		NewRegistryRuntime("demo", "test", []string{"demo"}, buildDemoRegistry(), nil),
	}, ManagerOptions{
		AuthLaunchers: []AuthLauncherRuntime{
			&testLauncherRuntime{
				descriptor: AuthLauncherDescriptor{
					ID:          "device-only",
					DisplayName: "device-only",
					ActionTypes: []string{"device_code"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("NewManagerWithOptions returned error: %v", err)
	}

	result, err := manager.LaunchAuth(context.Background(), AuthLaunchParams{
		Context: AuthLaunchContext{
			AccountName: "demo_account",
			Platform:    "demo",
		},
		Action: AuthAction{
			Type: "open_url",
			URL:  "https://example.com/auth",
		},
	})
	if err != nil {
		t.Fatalf("LaunchAuth returned error: %v", err)
	}
	if result.Handled || result.Status != "no_matching_launcher" {
		t.Fatalf("unexpected launch result: %+v", result)
	}
}

func TestBuildExecuteIdentityUsesMethodAndExecutionAuthShape(t *testing.T) {
	identity := buildExecuteIdentity("demo_account", config.Profile{
		Platform: "demo",
		Subject:  "integration",
		Method:   "demo.token",
	}, "demo.token", map[string]any{
		"type":         "resolved_access_token",
		"access_token": "demo-token",
	})

	if identity.Platform != "demo" || identity.Subject != "integration" {
		t.Fatalf("unexpected identity header: %+v", identity)
	}
	if identity.Auth.Method != "demo.token" {
		t.Fatalf("expected auth method in identity payload, got: %+v", identity.Auth)
	}
	executionAuth := identity.Auth.ExecutionAuth
	if executionAuth["access_token"] != "demo-token" {
		t.Fatalf("unexpected execution_auth payload: %+v", executionAuth)
	}
}

func TestBuildProfileFromIdentitySupportsNestedExecutionAuth(t *testing.T) {
	profile := buildProfileFromIdentity(ExecuteIdentity{
		Platform:    "notion",
		Subject:     "integration",
		AccountName: "demo_account",
		Auth: ExecuteAuth{
			Method: "notion.oauth_public",
			ExecutionAuth: map[string]any{
				"type":           "resolved_access_token",
				"access_token":   "nested-token",
				"notion_version": "2026-03-11",
			},
		},
	})

	if profile.Method != "notion.oauth_public" {
		t.Fatalf("unexpected profile method: %+v", profile)
	}
	if profile.Grant.AccessToken != "nested-token" || profile.Grant.NotionVer != "2026-03-11" {
		t.Fatalf("unexpected nested execution auth result: %+v", profile)
	}
}

func buildDemoRegistry() *adapter.Registry {
	registry := adapter.NewRegistry()
	registry.Register(adapter.Definition{
		Operation:       "demo.page.get",
		Platform:        "demo",
		Mutating:        false,
		DefaultTimeout:  time.Second,
		AllowedSubjects: []string{"integration"},
		Spec: adapter.OperationSpec{
			Summary: "Get one demo page.",
			Input: adapter.InputSpec{
				Sample: map[string]any{
					"id": "page_demo",
				},
			},
		},
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return map[string]any{
				"id": call.Input["id"],
			}, nil
		},
	})
	return registry
}

type testLauncherRuntime struct {
	descriptor AuthLauncherDescriptor
	launch     func(params AuthLaunchParams) (AuthLaunchResult, error)
}

func (r *testLauncherRuntime) Name() string {
	return r.descriptor.ID
}

func (r *testLauncherRuntime) Handshake(ctx context.Context) (HandshakeResult, error) {
	return HandshakeResult{
		ProtocolVersion: ProtocolVersion,
		Name:            r.descriptor.ID,
		Version:         "test",
	}, nil
}

func (r *testLauncherRuntime) DescribeAuthLauncher(ctx context.Context) (AuthLauncherDescriptor, error) {
	return r.descriptor, nil
}

func (r *testLauncherRuntime) LaunchAuth(ctx context.Context, params AuthLaunchParams) (AuthLaunchResult, error) {
	if r.launch == nil {
		return AuthLaunchResult{Handled: false, Status: "skipped"}, nil
	}
	return r.launch(params)
}
