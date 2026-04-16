package runtime

import (
	"testing"

	"github.com/clawrise/clawrise-cli/internal/config"
	pluginruntime "github.com/clawrise/clawrise-cli/internal/plugin"
)

func TestInspectPolicyChainReportsConfiguredSelectorsAndWarnings(t *testing.T) {
	policyRoot := t.TempDir()
	t.Setenv("CLAWRISE_PLUGIN_PATHS", policyRoot)
	t.Setenv("HOME", t.TempDir())
	writeRuntimePluginManifest(t, policyRoot, "policy-demo", `[
  {
    "type": "policy",
    "id": "review",
    "platforms": ["notion"],
    "priority": 20
  }
]`)

	cfg := config.New()
	cfg.Ensure()
	cfg.Runtime.Policy.Mode = config.RuntimeSelectionModeManual
	cfg.Runtime.Policy.Plugins = []config.PolicyPluginBinding{{Plugin: "policy-demo", PolicyID: "review"}, {Plugin: "missing-policy", PolicyID: "missing"}}
	cfg.Runtime.Policy.DenyOperations = []string{"notion.page.delete"}
	cfg.Runtime.Policy.RequireApprovalOperations = []string{"notion.page.update"}
	cfg.Runtime.Policy.AnnotateOperations = map[string]string{" notion.page.create ": " annotate create "}

	inspection := InspectPolicyChain(cfg)
	if inspection.Mode != config.RuntimeSelectionModeManual {
		t.Fatalf("unexpected policy mode: %+v", inspection)
	}
	if len(inspection.ConfiguredPlugins) != 2 {
		t.Fatalf("expected configured selector views, got: %+v", inspection.ConfiguredPlugins)
	}
	if inspection.Local.RuleCount != 3 {
		t.Fatalf("unexpected local policy summary: %+v", inspection.Local)
	}
	if inspection.Local.AnnotateOperations["notion.page.create"] != "annotate create" {
		t.Fatalf("expected trimmed annotate rule, got: %+v", inspection.Local.AnnotateOperations)
	}
	if len(inspection.ActiveChain) != 1 {
		t.Fatalf("expected one active policy runtime, got: %+v", inspection.ActiveChain)
	}
	if inspection.ActiveChain[0].Plugin != "policy-demo" || inspection.ActiveChain[0].PolicyID != "review" || inspection.ActiveChain[0].Source != "configured" {
		t.Fatalf("unexpected active policy chain item: %+v", inspection.ActiveChain[0])
	}
	if !containsWarning(inspection.Warnings, "missing-policy/missing") {
		t.Fatalf("expected missing selector warning, got: %+v", inspection.Warnings)
	}
}

func TestInspectAuditSinksReportsConfiguredSelectionsAndWarnings(t *testing.T) {
	auditRoot := t.TempDir()
	t.Setenv("CLAWRISE_PLUGIN_PATHS", auditRoot)
	t.Setenv("HOME", t.TempDir())
	writeRuntimePluginManifest(t, auditRoot, "audit-demo", `[
  {
    "type": "audit_sink",
    "id": "sink-main",
    "priority": 30
  }
]`)

	cfg := config.New()
	cfg.Ensure()
	cfg.Runtime.Audit.Mode = config.RuntimeSelectionModeManual
	cfg.Runtime.Audit.Sinks = []config.AuditSinkConfig{
		{Type: config.AuditSinkTypeStdout},
		{URL: "https://example.com/audit", Headers: map[string]string{" Authorization ": "Bearer demo", " ": "drop"}, TimeoutMS: 1500},
		{Plugin: "audit-demo", SinkID: "sink-main"},
		{Plugin: "missing-audit", SinkID: "missing"},
		{Type: "custom"},
	}

	inspection := InspectAuditSinks(cfg)
	if inspection.Mode != config.RuntimeSelectionModeManual {
		t.Fatalf("unexpected audit mode: %+v", inspection)
	}
	if len(inspection.ConfiguredSinks) != 5 {
		t.Fatalf("expected configured audit sink views, got: %+v", inspection.ConfiguredSinks)
	}
	if len(inspection.ActiveSinks) != 3 {
		t.Fatalf("expected stdout/webhook/plugin active sinks, got: %+v", inspection.ActiveSinks)
	}
	if inspection.ActiveSinks[0].Type != config.AuditSinkTypeStdout {
		t.Fatalf("expected stdout sink first, got: %+v", inspection.ActiveSinks)
	}
	if inspection.ActiveSinks[1].Type != config.AuditSinkTypeWebhook || len(inspection.ActiveSinks[1].HeaderNames) != 1 || inspection.ActiveSinks[1].HeaderNames[0] != "Authorization" {
		t.Fatalf("unexpected webhook summary: %+v", inspection.ActiveSinks[1])
	}
	if inspection.ActiveSinks[2].Plugin != "audit-demo" || inspection.ActiveSinks[2].SinkID != "sink-main" || inspection.ActiveSinks[2].Source != "configured" {
		t.Fatalf("unexpected plugin sink summary: %+v", inspection.ActiveSinks[2])
	}
	if !containsWarning(inspection.Warnings, "missing-audit/missing") {
		t.Fatalf("expected missing audit sink warning, got: %+v", inspection.Warnings)
	}
	if !containsWarning(inspection.Warnings, "type custom is not supported") {
		t.Fatalf("expected unsupported audit sink warning, got: %+v", inspection.Warnings)
	}
}

func TestRuntimeHelperFormattingAndLabels(t *testing.T) {
	policyRuntime := &testPolicyRuntime{name: "policy-demo", id: "review"}
	if got := policyRuntimeLabel(policyRuntime); got != "policy-demo/review" {
		t.Fatalf("unexpected policy runtime label: %s", got)
	}
	if got := policySelectorLabel(config.PolicyPluginBinding{Plugin: "policy-demo", PolicyID: "review"}); got != "policy-demo/review" {
		t.Fatalf("unexpected policy selector label: %s", got)
	}
	if got := policySelectorLabel(config.PolicyPluginBinding{Plugin: "policy-demo"}); got != "policy-demo" {
		t.Fatalf("unexpected policy selector plugin-only label: %s", got)
	}
	if got := policySelectorLabel(config.PolicyPluginBinding{PolicyID: "review"}); got != "review" {
		t.Fatalf("unexpected policy selector id-only label: %s", got)
	}
	if got := buildPolicyDecisionMessage(policyRuntime, pluginruntime.PolicyEvaluateResult{Message: "blocked"}, "fallback"); got != "policy plugin policy-demo/review: blocked" {
		t.Fatalf("unexpected policy decision message: %s", got)
	}
	if got := buildPluginPolicyHitMessage(policyRuntime, pluginruntime.PolicyEvaluateResult{Annotations: map[string]any{"reviewer": "ops"}}, "fallback"); got != `{"reviewer":"ops"}` {
		t.Fatalf("unexpected annotation-based policy hit message: %s", got)
	}
	if got := buildPluginPolicyHitMessage(policyRuntime, pluginruntime.PolicyEvaluateResult{}, "fallback"); got != "fallback" {
		t.Fatalf("unexpected fallback policy hit message: %s", got)
	}

	auditRuntime := &testAuditRuntime{name: "audit-demo", id: "sink-main"}
	if got := (&pluginAuditSink{runtime: auditRuntime}).Name(); got != "audit-demo/sink-main" {
		t.Fatalf("unexpected plugin audit sink name: %s", got)
	}
	if got := (&stdoutAuditSink{}).Name(); got != "builtin/stdout" {
		t.Fatalf("unexpected stdout audit sink name: %s", got)
	}
	if got := (&webhookAuditSink{}).Name(); got != "builtin/webhook" {
		t.Fatalf("unexpected webhook audit sink name: %s", got)
	}
	if got := auditSinkRuntimeLabel(auditRuntime); got != "audit-demo/sink-main" {
		t.Fatalf("unexpected audit runtime label: %s", got)
	}
	if got := auditSinkSelectorLabel(config.AuditSinkConfig{Type: config.AuditSinkTypePlugin, Plugin: "audit-demo", SinkID: "sink-main"}); got != "audit-demo/sink-main" {
		t.Fatalf("unexpected audit selector label: %s", got)
	}
	if got := auditSinkSelectorLabel(config.AuditSinkConfig{Type: config.AuditSinkTypePlugin, Plugin: "audit-demo"}); got != "audit-demo" {
		t.Fatalf("unexpected plugin-only audit selector label: %s", got)
	}
	if got := auditSinkSelectorLabel(config.AuditSinkConfig{Type: config.AuditSinkTypePlugin, SinkID: "sink-main"}); got != "sink-main" {
		t.Fatalf("unexpected sink-id-only audit selector label: %s", got)
	}
	if got := auditSinkSelectorLabel(config.AuditSinkConfig{Type: config.AuditSinkTypeWebhook, URL: "https://example.com/audit"}); got != "https://example.com/audit" {
		t.Fatalf("unexpected webhook audit selector label: %s", got)
	}
	if got := auditSinkSelectorLabel(config.AuditSinkConfig{Type: "stdout"}); got != "stdout" {
		t.Fatalf("unexpected default audit selector label: %s", got)
	}
}
