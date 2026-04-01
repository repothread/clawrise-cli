package plugin

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	goRuntime "runtime"
	"strings"
	"testing"
)

func TestProjectSamplePolicyPluginCompatibility(t *testing.T) {
	prepareProjectSamplePluginEnv(t)

	manifest := loadProjectSampleManifest(t, "sample-policy", "0.1.0")

	runtime := NewProcessRuntime(manifest)
	defer func() {
		_ = runtime.Close()
	}()

	handshake, err := runtime.Handshake(context.Background())
	if err != nil {
		t.Fatalf("Handshake returned error: %v", err)
	}
	if handshake.Name != "sample-policy" || handshake.Version != "0.1.0" {
		t.Fatalf("unexpected handshake result: %+v", handshake)
	}

	capabilities, err := runtime.ListCapabilities(context.Background())
	if err != nil {
		t.Fatalf("ListCapabilities returned error: %v", err)
	}
	if len(capabilities) != 1 || capabilities[0].Type != CapabilityTypePolicy || capabilities[0].ID != "require_reason_for_mutations" {
		t.Fatalf("unexpected capabilities: %+v", capabilities)
	}

	policy := NewProcessPolicy(manifest, manifest.CapabilitiesByType(CapabilityTypePolicy)[0])
	defer func() {
		_ = policy.Close()
	}()

	approvalResult, err := policy.Evaluate(context.Background(), PolicyEvaluateParams{
		Request: PolicyEvaluationRequest{
			RequestID: "req_missing_reason",
			Operation: "notion.page.update",
			Mutating:  true,
			Input: map[string]any{
				"title": "Release Notes",
			},
		},
	})
	if err != nil {
		t.Fatalf("Evaluate returned error for missing reason: %v", err)
	}
	if approvalResult.Decision != "require_approval" {
		t.Fatalf("expected require_approval, got: %+v", approvalResult)
	}

	annotationResult, err := policy.Evaluate(context.Background(), PolicyEvaluateParams{
		Request: PolicyEvaluationRequest{
			RequestID: "req_with_reason",
			Operation: "notion.page.update",
			Mutating:  true,
			Input: map[string]any{
				"change_reason": "sync the weekly summary",
			},
		},
	})
	if err != nil {
		t.Fatalf("Evaluate returned error for annotated request: %v", err)
	}
	if annotationResult.Decision != "annotate" {
		t.Fatalf("expected annotate, got: %+v", annotationResult)
	}
	if annotationResult.Annotations["change_reason"] != "sync the weekly summary" {
		t.Fatalf("unexpected policy annotations: %+v", annotationResult.Annotations)
	}

	allowResult, err := policy.Evaluate(context.Background(), PolicyEvaluateParams{
		Request: PolicyEvaluationRequest{
			RequestID: "req_read_only",
			Operation: "notion.page.get",
			Mutating:  false,
		},
	})
	if err != nil {
		t.Fatalf("Evaluate returned error for read request: %v", err)
	}
	if allowResult.Decision != "allow" {
		t.Fatalf("expected allow, got: %+v", allowResult)
	}
}

func TestProjectSampleAuditPluginCompatibility(t *testing.T) {
	prepareProjectSamplePluginEnv(t)

	auditLogPath := filepath.Join(t.TempDir(), "sample-audit.ndjson")
	t.Setenv("CLAWRISE_SAMPLE_AUDIT_LOG", auditLogPath)

	manifest := loadProjectSampleManifest(t, "sample-audit", "0.1.0")

	runtime := NewProcessRuntime(manifest)
	defer func() {
		_ = runtime.Close()
	}()

	handshake, err := runtime.Handshake(context.Background())
	if err != nil {
		t.Fatalf("Handshake returned error: %v", err)
	}
	if handshake.Name != "sample-audit" || handshake.Version != "0.1.0" {
		t.Fatalf("unexpected handshake result: %+v", handshake)
	}

	capabilities, err := runtime.ListCapabilities(context.Background())
	if err != nil {
		t.Fatalf("ListCapabilities returned error: %v", err)
	}
	if len(capabilities) != 1 || capabilities[0].Type != CapabilityTypeAuditSink || capabilities[0].ID != "file_capture" {
		t.Fatalf("unexpected capabilities: %+v", capabilities)
	}

	sink := NewProcessAuditSink(manifest, manifest.CapabilitiesByType(CapabilityTypeAuditSink)[0])
	defer func() {
		_ = sink.Close()
	}()

	if err := sink.Emit(context.Background(), AuditEmitParams{
		Record: GovernanceAuditRecord{
			RequestID: "req_demo",
			Operation: "notion.page.update",
			OK:        true,
		},
	}); err != nil {
		t.Fatalf("Emit returned error: %v", err)
	}

	data, err := os.ReadFile(auditLogPath)
	if err != nil {
		t.Fatalf("failed to read audit log: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected one audit entry, got %d: %q", len(lines), string(data))
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &payload); err != nil {
		t.Fatalf("failed to decode audit log entry: %v", err)
	}
	if payload["sink_id"] != "file_capture" {
		t.Fatalf("unexpected audit entry sink id: %+v", payload)
	}

	record, ok := payload["record"].(map[string]any)
	if !ok {
		t.Fatalf("unexpected audit entry record payload: %+v", payload)
	}
	if record["request_id"] != "req_demo" || record["operation"] != "notion.page.update" {
		t.Fatalf("unexpected audit record payload: %+v", record)
	}
}

func TestProjectSamplePluginsAreVisibleToInspectDiscovery(t *testing.T) {
	prepareProjectSamplePluginEnv(t)
	t.Setenv("CLAWRISE_PLUGIN_PATHS", filepath.Join(projectRoot(t), "examples", "plugins"))
	t.Setenv("HOME", t.TempDir())

	report, err := InspectDiscoveryWithOptions(context.Background(), DiscoveryOptions{
		PolicyMode: "manual",
		PolicySelectors: []PolicyCapabilitySelector{
			{Plugin: "sample-policy", PolicyID: "require_reason_for_mutations"},
		},
		AuditMode: "manual",
		AuditSinks: []AuditSinkSelector{
			{Type: "plugin", Plugin: "sample-audit", SinkID: "file_capture"},
		},
	})
	if err != nil {
		t.Fatalf("InspectDiscoveryWithOptions returned error: %v", err)
	}

	if len(report.Plugins) != 2 {
		t.Fatalf("expected two sample plugins, got: %+v", report.Plugins)
	}

	policyItem := findPluginInspection(t, report.Plugins, "sample-policy")
	policyRoute := findCapabilityRoute(t, policyItem.CapabilityRoutes, CapabilityTypePolicy, "require_reason_for_mutations")
	if !policyRoute.Active || policyRoute.Source != "configured" {
		t.Fatalf("unexpected policy route: %+v", policyRoute)
	}
	if policyItem.InspectionError != "" {
		t.Fatalf("unexpected policy inspection error: %+v", policyItem)
	}

	auditItem := findPluginInspection(t, report.Plugins, "sample-audit")
	auditRoute := findCapabilityRoute(t, auditItem.CapabilityRoutes, CapabilityTypeAuditSink, "file_capture")
	if !auditRoute.Active || auditRoute.Source != "configured" {
		t.Fatalf("unexpected audit route: %+v", auditRoute)
	}
	if auditItem.InspectionError != "" {
		t.Fatalf("unexpected audit inspection error: %+v", auditItem)
	}
}

func prepareProjectSamplePluginEnv(t *testing.T) {
	t.Helper()

	t.Setenv("GOCACHE", filepath.Join(t.TempDir(), "go-build"))
}

func loadProjectSampleManifest(t *testing.T, pluginName string, version string) Manifest {
	t.Helper()

	manifestPath := filepath.Join(projectRoot(t), "examples", "plugins", pluginName, version, ManifestFileName)
	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("LoadManifest returned error: %v", err)
	}
	return manifest
}

func projectRoot(t *testing.T) string {
	t.Helper()

	_, filename, _, ok := goRuntime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve test file path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
}

func findPluginInspection(t *testing.T, items []DiscoveredPluginInspection, name string) DiscoveredPluginInspection {
	t.Helper()

	for _, item := range items {
		if item.Name == name {
			return item
		}
	}
	t.Fatalf("failed to find plugin inspection for %s", name)
	return DiscoveredPluginInspection{}
}
