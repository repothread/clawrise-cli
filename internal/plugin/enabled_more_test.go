package plugin

import "testing"

func TestResolvePluginEnabledStateAndSelectionReasons(t *testing.T) {
	state := resolvePluginEnabledState("  demo-plugin  ", map[string]string{
		"demo-plugin": " disabled ",
	})
	if state.Enabled || !state.Explicit || state.Rule != "disabled" {
		t.Fatalf("unexpected disabled plugin state: %+v", state)
	}

	defaultState := resolvePluginEnabledState("", nil)
	if !defaultState.Enabled || defaultState.Explicit {
		t.Fatalf("unexpected default plugin state: %+v", defaultState)
	}

	manifest := Manifest{
		Name: "demo-plugin",
		Capabilities: []CapabilityDescriptor{{
			Type:      CapabilityTypeProvider,
			Platforms: []string{"demo"},
		}},
	}
	selection := resolveManifestSelectionState(manifest, DiscoveryOptions{
		EnabledPlugins: map[string]string{"demo-plugin": "disabled"},
	})
	if selection.Enabled || selection.Selected || selection.SelectionReason != "disabled_by_plugins.enabled" {
		t.Fatalf("unexpected disabled selection state: %+v", selection)
	}

	active := resolveManifestSelectionState(Manifest{Name: "active-plugin"}, DiscoveryOptions{})
	if !active.Enabled || !active.Selected || active.SelectionReason != "active" {
		t.Fatalf("unexpected active selection state: %+v", active)
	}
}

func TestCapabilityRouteSelectorsAndModes(t *testing.T) {
	policyCapability := CapabilityDescriptor{Type: CapabilityTypePolicy, ID: "review"}
	auditCapability := CapabilityDescriptor{Type: CapabilityTypeAuditSink, ID: "capture"}

	if !policyCapabilityMatchesSelector("demo", policyCapability, PolicyCapabilitySelector{Plugin: " demo "}) {
		t.Fatal("expected policy selector to match by plugin")
	}
	if !policyCapabilityMatchesSelector("demo", policyCapability, PolicyCapabilitySelector{PolicyID: " review "}) {
		t.Fatal("expected policy selector to match by policy id")
	}
	if policyCapabilityMatchesSelector("demo", policyCapability, PolicyCapabilitySelector{}) {
		t.Fatal("expected empty policy selector not to match")
	}
	if policyCapabilityMatchesSelector("demo", policyCapability, PolicyCapabilitySelector{Plugin: "other"}) {
		t.Fatal("expected mismatched policy selector not to match")
	}

	if !auditSinkCapabilityMatchesSelector("demo", auditCapability, AuditSinkSelector{Type: "plugin", Plugin: "demo", SinkID: "capture"}) {
		t.Fatal("expected audit sink selector to match plugin sink")
	}
	if auditSinkCapabilityMatchesSelector("demo", auditCapability, AuditSinkSelector{Type: "webhook", Plugin: "demo"}) {
		t.Fatal("expected non-plugin audit sink selector not to match plugin capability")
	}
	if auditSinkCapabilityMatchesSelector("demo", auditCapability, AuditSinkSelector{}) {
		t.Fatal("expected empty audit sink selector not to match")
	}

	if got := normalizeCapabilitySelectionMode(" manual "); got != selectionModeManual {
		t.Fatalf("unexpected normalized manual mode: %q", got)
	}
	if got := normalizeCapabilitySelectionMode("disabled"); got != selectionModeDisabled {
		t.Fatalf("unexpected normalized disabled mode: %q", got)
	}
	if got := normalizeCapabilitySelectionMode("surprise"); got != selectionModeAuto {
		t.Fatalf("expected unknown mode to normalize to auto, got %q", got)
	}
}
