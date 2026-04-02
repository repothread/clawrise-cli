package plugin

import "testing"

func TestFilterManifestsByEnabledRulesSkipsExplicitlyDisabledPlugin(t *testing.T) {
	manifests := []Manifest{
		{Name: "demo-a"},
		{Name: "demo-b"},
	}

	filtered := filterManifestsByEnabledRules(manifests, map[string]string{
		"demo-a": "disabled",
		"demo-b": "^0.4.0",
	})
	if len(filtered) != 1 {
		t.Fatalf("unexpected filtered manifests: %+v", filtered)
	}
	if filtered[0].Name != "demo-b" {
		t.Fatalf("unexpected selected manifest: %+v", filtered[0])
	}
}

func TestResolveManifestSelectionStateMarksProviderFilteredByBinding(t *testing.T) {
	manifest := Manifest{
		Name: "demo-a",
		Capabilities: []CapabilityDescriptor{{
			Type:      CapabilityTypeProvider,
			Platforms: []string{"demo"},
		}},
	}

	state := resolveManifestSelectionState(manifest, DiscoveryOptions{
		ProviderBindings: map[string]string{
			"demo": "demo-b",
		},
	})
	if !state.Enabled {
		t.Fatalf("expected manifest to remain enabled: %+v", state)
	}
	if state.Selected {
		t.Fatalf("expected provider-only plugin to be filtered: %+v", state)
	}
	if state.SelectionReason != "filtered_by_provider_binding" {
		t.Fatalf("unexpected selection reason: %+v", state)
	}
}

func TestResolveManifestSelectionStateKeepsMultiCapabilityPluginSelected(t *testing.T) {
	manifest := Manifest{
		Name: "demo-suite",
		Capabilities: []CapabilityDescriptor{
			{
				Type:      CapabilityTypeProvider,
				Platforms: []string{"demo"},
			},
			{
				Type: CapabilityTypeAuthLauncher,
				ID:   "browser",
			},
		},
	}

	state := resolveManifestSelectionState(manifest, DiscoveryOptions{
		ProviderBindings: map[string]string{
			"demo": "demo-b",
		},
	})
	if !state.Selected {
		t.Fatalf("expected multi-capability plugin to stay selected: %+v", state)
	}
	if state.SelectionReason != "partially_selected_by_provider_binding" {
		t.Fatalf("unexpected selection reason: %+v", state)
	}
}
