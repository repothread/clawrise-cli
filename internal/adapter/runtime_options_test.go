package adapter

import (
	"context"
	"testing"
)

func TestRuntimeOptionsAndRequestIDContextHelpers(t *testing.T) {
	ctx := WithRuntimeOptions(nil, RuntimeOptions{
		DebugProviderPayload: true,
		VerifyAfterWrite:     true,
	})
	options := RuntimeOptionsFromContext(ctx)
	if !options.DebugProviderPayload || !options.VerifyAfterWrite {
		t.Fatalf("unexpected runtime options: %+v", options)
	}
	if got := RuntimeOptionsFromContext(nil); got != (RuntimeOptions{}) {
		t.Fatalf("expected nil context to return zero runtime options, got %+v", got)
	}

	ctx = WithRequestID(ctx, "req-123")
	if got := RequestIDFromContext(ctx); got != "req-123" {
		t.Fatalf("unexpected request id from context: %q", got)
	}
	if got := RequestIDFromContext(nil); got != "" {
		t.Fatalf("expected nil context to return empty request id, got %q", got)
	}

	unchanged := WithRequestID(ctx, "")
	if got := RequestIDFromContext(unchanged); got != "req-123" {
		t.Fatalf("expected blank request id to keep original context value, got %q", got)
	}
}

func TestProviderDebugCaptureExportsClonedEntries(t *testing.T) {
	ctx, capture := WithProviderDebugCapture(nil)
	if capture == nil {
		t.Fatal("expected provider debug capture to be created")
	}
	ctxAgain, reused := WithProviderDebugCapture(ctx)
	if reused != capture {
		t.Fatal("expected existing provider debug capture to be reused")
	}
	if ctxAgain != ctx {
		t.Fatal("expected context to be reused when capture already exists")
	}

	AddProviderDebugEvent(ctx, nil)
	AddProviderDebugEvent(ctx, map[string]any{})
	AddProviderDebugEvent(ctx, map[string]any{
		"method": "GET",
		"path":   "/docs/1",
	})

	exported := ProviderDebugFromContext(ctx)
	items, ok := exported["provider_requests"].([]map[string]any)
	if !ok {
		t.Fatalf("expected provider_requests payload to be []map[string]any, got %#v", exported)
	}
	if len(items) != 1 {
		t.Fatalf("unexpected provider debug entry count: %d", len(items))
	}
	items[0]["method"] = "MUTATED"

	reloaded := ProviderDebugFromContext(ctx)
	reloadedItems := reloaded["provider_requests"].([]map[string]any)
	if got := reloadedItems[0]["method"]; got != "GET" {
		t.Fatalf("expected exported provider debug payload to be cloned, got %v", got)
	}
}

func TestProviderDebugCaptureFreshIsolationAndImport(t *testing.T) {
	baseCtx, _ := WithProviderDebugCapture(context.Background())
	AddProviderDebugEvent(baseCtx, map[string]any{"step": "base"})

	freshCtx, freshCapture := WithFreshProviderDebugCapture(baseCtx)
	if freshCapture == nil {
		t.Fatal("expected fresh provider debug capture to be created")
	}
	if payload := ProviderDebugFromContext(freshCtx); payload != nil {
		t.Fatalf("expected fresh capture to start empty, got %#v", payload)
	}

	ImportProviderDebug(freshCtx, map[string]any{
		"provider_requests": []any{
			map[string]any{"step": "imported-1"},
			"skip-non-map",
			map[string]any{"step": "imported-2"},
		},
	})

	payload := ProviderDebugFromContext(freshCtx)
	items := payload["provider_requests"].([]map[string]any)
	if len(items) != 2 {
		t.Fatalf("expected only map items to be imported, got %#v", items)
	}
	if items[0]["step"] != "imported-1" || items[1]["step"] != "imported-2" {
		t.Fatalf("unexpected imported provider debug entries: %#v", items)
	}

	basePayload := ProviderDebugFromContext(baseCtx)
	baseItems := basePayload["provider_requests"].([]map[string]any)
	if len(baseItems) != 1 || baseItems[0]["step"] != "base" {
		t.Fatalf("expected fresh capture to stay isolated from base capture, got %#v", baseItems)
	}
}
