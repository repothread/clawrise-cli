package plugin

import (
	"context"
	"testing"
	"time"

	"github.com/clawrise/clawrise-cli/internal/adapter"
)

func TestManagerAuthAggregationAcrossPlatforms(t *testing.T) {
	demoProvider := &testAuthProvider{
		methods: []AuthMethodDescriptor{{ID: "demo.token", Platform: "demo"}},
		presets: []AuthPresetDescriptor{{ID: "demo-machine", Platform: "demo", AuthMethod: "demo.token"}},
		inspect: AuthInspectResult{Ready: true, Status: "ready"},
		begin:   AuthBeginResult{Flow: AuthFlowPayload{ID: "flow_demo"}},
		complete: AuthCompleteResult{
			Ready:  true,
			Status: "ready",
		},
		resolve: AuthResolveResult{Ready: true, Status: "ready", ExecutionAuth: map[string]any{"token": "demo"}},
	}
	altProvider := &testAuthProvider{
		methods: []AuthMethodDescriptor{{ID: "alt.token", Platform: "alt"}},
		presets: []AuthPresetDescriptor{{ID: "alt-machine", Platform: "alt", AuthMethod: "alt.token"}},
	}

	manager, err := NewManager(context.Background(), []Runtime{
		NewRegistryRuntimeWithOptions("demo", "test", []string{"demo"}, buildDemoRegistry(), nil, RegistryRuntimeOptions{AuthProvider: demoProvider}),
		NewRegistryRuntimeWithOptions("alt", "test", []string{"alt"}, buildAltRegistry(), nil, RegistryRuntimeOptions{AuthProvider: altProvider}),
	})
	if err != nil {
		t.Fatalf("NewManager returned error: %v", err)
	}

	methods, err := manager.ListAuthMethods(context.Background(), "")
	if err != nil {
		t.Fatalf("ListAuthMethods returned error: %v", err)
	}
	if len(methods) != 2 {
		t.Fatalf("expected aggregated auth methods, got: %+v", methods)
	}

	presets, err := manager.ListAuthPresets(context.Background(), "")
	if err != nil {
		t.Fatalf("ListAuthPresets returned error: %v", err)
	}
	if len(presets) != 2 {
		t.Fatalf("expected aggregated auth presets, got: %+v", presets)
	}

	inspect, err := manager.InspectAuth(context.Background(), "demo", AuthInspectParams{})
	if err != nil {
		t.Fatalf("InspectAuth returned error: %v", err)
	}
	if !inspect.Ready || inspect.Status != "ready" {
		t.Fatalf("unexpected inspect result: %+v", inspect)
	}

	begin, err := manager.BeginAuth(context.Background(), "demo", AuthBeginParams{})
	if err != nil {
		t.Fatalf("BeginAuth returned error: %v", err)
	}
	if begin.Flow.ID != "flow_demo" {
		t.Fatalf("unexpected begin result: %+v", begin)
	}

	complete, err := manager.CompleteAuth(context.Background(), "demo", AuthCompleteParams{})
	if err != nil {
		t.Fatalf("CompleteAuth returned error: %v", err)
	}
	if !complete.Ready {
		t.Fatalf("unexpected complete result: %+v", complete)
	}

	resolve, err := manager.ResolveAuth(context.Background(), "demo", AuthResolveParams{})
	if err != nil {
		t.Fatalf("ResolveAuth returned error: %v", err)
	}
	if !resolve.Ready || resolve.ExecutionAuth["token"] != "demo" {
		t.Fatalf("unexpected resolve result: %+v", resolve)
	}
}

func buildAltRegistry() *adapter.Registry {
	registry := adapter.NewRegistry()
	registry.Register(adapter.Definition{
		Operation:       "alt.page.get",
		Platform:        "alt",
		Mutating:        false,
		DefaultTimeout:  time.Second,
		AllowedSubjects: []string{"integration"},
	})
	return registry
}

func TestManagerAuthMethodsRejectUnknownPlatform(t *testing.T) {
	manager, err := NewManager(context.Background(), []Runtime{
		NewRegistryRuntime("demo", "test", []string{"demo"}, buildDemoRegistry(), nil),
	})
	if err != nil {
		t.Fatalf("NewManager returned error: %v", err)
	}

	for _, tc := range []struct {
		name string
		call func() error
	}{
		{"ListAuthMethods", func() error { _, err := manager.ListAuthMethods(context.Background(), "missing"); return err }},
		{"ListAuthPresets", func() error { _, err := manager.ListAuthPresets(context.Background(), "missing"); return err }},
		{"InspectAuth", func() error {
			_, err := manager.InspectAuth(context.Background(), "missing", AuthInspectParams{})
			return err
		}},
		{"BeginAuth", func() error {
			_, err := manager.BeginAuth(context.Background(), "missing", AuthBeginParams{})
			return err
		}},
		{"CompleteAuth", func() error {
			_, err := manager.CompleteAuth(context.Background(), "missing", AuthCompleteParams{})
			return err
		}},
		{"ResolveAuth", func() error {
			_, err := manager.ResolveAuth(context.Background(), "missing", AuthResolveParams{})
			return err
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.call(); err == nil {
				t.Fatalf("expected %s to reject unknown platform", tc.name)
			}
		})
	}

	if runtime, ok := manager.RuntimeForPlatform("missing"); ok || runtime != nil {
		t.Fatalf("expected RuntimeForPlatform to miss unknown platform, got runtime=%v ok=%v", runtime, ok)
	}
}
