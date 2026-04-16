package runtime

import (
	"testing"

	"github.com/clawrise/clawrise-cli/internal/config"
)

func TestResolveSelectedPolicyRuntimesModesAndSelectors(t *testing.T) {
	policyRoot := t.TempDir()
	t.Setenv("CLAWRISE_PLUGIN_PATHS", policyRoot)
	t.Setenv("HOME", t.TempDir())
	writeRuntimePluginManifest(t, policyRoot, "policy-b", `[
  {
    "type": "policy",
    "id": "review-b",
    "platforms": ["notion"],
    "priority": 10
  }
]`)
	writeRuntimePluginManifest(t, policyRoot, "policy-a", `[
  {
    "type": "policy",
    "id": "review-a",
    "platforms": ["notion"],
    "priority": 20
  }
]`)

	autoCfg := config.New()
	autoCfg.Ensure()
	selections, warnings, err := resolveSelectedPolicyRuntimes(autoCfg)
	if err != nil {
		t.Fatalf("resolveSelectedPolicyRuntimes returned error: %v", err)
	}
	if len(warnings) != 0 || len(selections) != 2 {
		t.Fatalf("unexpected auto policy selections: selections=%+v warnings=%+v", selections, warnings)
	}
	if selections[0].Runtime.Name() != "policy-a" || selections[0].Source != "auto" || selections[1].Runtime.Name() != "policy-b" {
		t.Fatalf("expected auto policy selections to be priority sorted, got %+v", summarizePolicySelections(selections))
	}

	manualCfg := config.New()
	manualCfg.Ensure()
	manualCfg.Runtime.Policy.Mode = config.RuntimeSelectionModeManual
	selections, warnings, err = resolveSelectedPolicyRuntimes(manualCfg)
	if err != nil {
		t.Fatalf("resolveSelectedPolicyRuntimes returned error in manual mode: %v", err)
	}
	if selections != nil || warnings != nil {
		t.Fatalf("expected manual mode without selectors to return nils, got selections=%+v warnings=%+v", selections, warnings)
	}

	configuredCfg := config.New()
	configuredCfg.Ensure()
	configuredCfg.Runtime.Policy.Mode = config.RuntimeSelectionModeManual
	configuredCfg.Runtime.Policy.Plugins = []config.PolicyPluginBinding{
		{Plugin: "policy-b", PolicyID: "review-b"},
		{Plugin: "missing-policy", PolicyID: "missing"},
		{Plugin: "policy-b", PolicyID: "review-b"},
	}
	selections, warnings, err = resolveSelectedPolicyRuntimes(configuredCfg)
	if err != nil {
		t.Fatalf("resolveSelectedPolicyRuntimes returned error for configured selectors: %v", err)
	}
	if len(selections) != 1 || selections[0].Runtime.Name() != "policy-b" || selections[0].Source != "configured" {
		t.Fatalf("unexpected configured policy selections: %+v", summarizePolicySelections(selections))
	}
	if len(warnings) != 1 || warnings[0] == "" {
		t.Fatalf("expected one missing-selector warning, got %+v", warnings)
	}

	disabledCfg := config.New()
	disabledCfg.Ensure()
	disabledCfg.Runtime.Policy.Mode = config.RuntimeSelectionModeDisabled
	selections, warnings, err = resolveSelectedPolicyRuntimes(disabledCfg)
	if err != nil {
		t.Fatalf("resolveSelectedPolicyRuntimes returned error in disabled mode: %v", err)
	}
	if selections != nil || warnings != nil {
		t.Fatalf("expected disabled mode to return nils, got selections=%+v warnings=%+v", selections, warnings)
	}
}

func TestResolveSelectedAuditSinksAndHelpers(t *testing.T) {
	auditRoot := t.TempDir()
	t.Setenv("CLAWRISE_PLUGIN_PATHS", auditRoot)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("AUDIT_URL", "https://example.com/audit")
	t.Setenv("AUDIT_TOKEN", "Bearer demo")
	writeRuntimePluginManifest(t, auditRoot, "audit-b", `[
  {
    "type": "audit_sink",
    "id": "sink-b",
    "priority": 10
  }
]`)
	writeRuntimePluginManifest(t, auditRoot, "audit-a", `[
  {
    "type": "audit_sink",
    "id": "sink-a",
    "priority": 20
  }
]`)

	autoCfg := config.New()
	autoCfg.Ensure()
	selections, warnings := resolveSelectedAuditSinks(autoCfg)
	if len(warnings) != 0 || len(selections) != 2 {
		t.Fatalf("unexpected auto audit selections: selections=%+v warnings=%+v", selections, warnings)
	}
	if selections[0].Summary.Plugin != "audit-a" || selections[0].Summary.Source != "auto" || selections[1].Summary.Plugin != "audit-b" {
		t.Fatalf("expected auto audit sinks to be priority sorted, got %+v", summarizeSelectedAuditSinks(selections))
	}
	opened, openWarnings := openAuditSinks(autoCfg)
	if len(openWarnings) != 0 || len(opened) != 2 {
		t.Fatalf("unexpected openAuditSinks result: sinks=%+v warnings=%+v", opened, openWarnings)
	}

	manualCfg := config.New()
	manualCfg.Ensure()
	manualCfg.Runtime.Audit.Mode = config.RuntimeSelectionModeManual
	manualCfg.Runtime.Audit.Sinks = []config.AuditSinkConfig{
		{Type: config.AuditSinkTypeStdout},
		{URL: "env:AUDIT_URL", Headers: map[string]string{"Authorization": "env:AUDIT_TOKEN"}, TimeoutMS: 2500},
		{Plugin: "audit-b", SinkID: "sink-b"},
		{Plugin: "missing-audit", SinkID: "missing"},
		{Type: "custom"},
		{URL: ""},
	}
	selections, warnings = resolveSelectedAuditSinks(manualCfg)
	if len(selections) != 3 {
		t.Fatalf("expected stdout/webhook/plugin selections, got %+v", summarizeSelectedAuditSinks(selections))
	}
	if selections[0].Summary.Type != config.AuditSinkTypeStdout || selections[1].Summary.Type != config.AuditSinkTypeWebhook || selections[1].Summary.URL != "env:AUDIT_URL" || selections[2].Summary.Plugin != "audit-b" {
		t.Fatalf("unexpected manual audit selections: %+v", summarizeSelectedAuditSinks(selections))
	}
	if len(warnings) < 2 {
		t.Fatalf("expected selector warnings, got %+v", warnings)
	}

	if _, _, err := buildWebhookAuditSink(config.AuditSinkConfig{}); err == nil {
		t.Fatal("expected missing webhook url to fail")
	}
	if _, _, err := buildWebhookAuditSink(config.AuditSinkConfig{URL: "env:MISSING_AUDIT_URL"}); err == nil {
		t.Fatal("expected unresolved webhook env url to fail")
	}
	if _, _, err := buildWebhookAuditSink(config.AuditSinkConfig{URL: "env:AUDIT_URL", Headers: map[string]string{"Authorization": "env:MISSING_AUDIT_TOKEN"}}); err == nil {
		t.Fatal("expected unresolved webhook env header to fail")
	}

	if auditSinkRuntimeMatchesSelector(nil, config.AuditSinkConfig{Plugin: "audit-a"}) {
		t.Fatal("expected nil audit sink runtime not to match")
	}
	if !auditSinkRuntimeMatchesSelector(selections[2].Sink.(*pluginAuditSink).runtime, config.AuditSinkConfig{Plugin: "audit-b", SinkID: "sink-b"}) {
		t.Fatal("expected plugin audit sink selector to match discovered runtime")
	}
	if auditSinkRuntimeMatchesSelector(selections[2].Sink.(*pluginAuditSink).runtime, config.AuditSinkConfig{Plugin: "audit-a"}) {
		t.Fatal("expected mismatched plugin audit sink selector not to match")
	}

	if got := auditSelectionsToSinks([]selectedAuditSink{{Sink: nil}, {Sink: selections[0].Sink}}); len(got) != 1 {
		t.Fatalf("expected auditSelectionsToSinks to skip nil sink, got %+v", got)
	}
	if got := summarizeSelectedAuditSinks(nil); got != nil {
		t.Fatalf("expected nil summarizeSelectedAuditSinks result, got %+v", got)
	}
	if got := buildAuditSinkSelectorViews(nil); got != nil {
		t.Fatalf("expected nil buildAuditSinkSelectorViews result, got %+v", got)
	}
}
