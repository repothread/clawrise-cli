package plugin

import "testing"

func TestValidateProviderBindingsFromCandidatesRequiresExplicitBindingOnConflict(t *testing.T) {
	err := ValidateProviderBindingsFromCandidates([]ProviderCandidate{
		{Platform: "demo", Plugin: "demo-a", Version: "0.1.0"},
		{Platform: "demo", Plugin: "demo-b", Version: "0.1.0"},
	}, map[string]string{})
	if err == nil {
		t.Fatal("expected binding conflict error")
	}
}

func TestValidateProviderBindingsFromCandidatesRejectsUnknownPlugin(t *testing.T) {
	err := ValidateProviderBindingsFromCandidates([]ProviderCandidate{
		{Platform: "demo", Plugin: "demo-a", Version: "0.1.0"},
	}, map[string]string{
		"demo": "demo-b",
	})
	if err == nil {
		t.Fatal("expected unknown plugin binding error")
	}
}

func TestValidateProviderBindingsFromCandidatesAcceptsMatchingBinding(t *testing.T) {
	err := ValidateProviderBindingsFromCandidates([]ProviderCandidate{
		{Platform: "demo", Plugin: "demo-a", Version: "0.1.0"},
		{Platform: "demo", Plugin: "demo-b", Version: "0.1.0"},
	}, map[string]string{
		"demo": "demo-b",
	})
	if err != nil {
		t.Fatalf("expected binding to be valid, got: %v", err)
	}
}

func TestValidateProviderBindingsWithEnabledRulesRejectsDisabledPlugin(t *testing.T) {
	manifests := []Manifest{
		{
			Name: "demo-a",
			Capabilities: []CapabilityDescriptor{{
				Type:      CapabilityTypeProvider,
				Platforms: []string{"demo"},
			}},
		},
		{
			Name: "demo-b",
			Capabilities: []CapabilityDescriptor{{
				Type:      CapabilityTypeProvider,
				Platforms: []string{"demo"},
			}},
		},
	}

	err := ValidateProviderBindingsWithEnabledRules(manifests, map[string]string{
		"demo": "demo-a",
	}, map[string]string{
		"demo-a": "disabled",
	})
	if err == nil {
		t.Fatal("expected disabled plugin binding error")
	}
	if err.Error() != "provider binding for platform demo points to demo-a, but the plugin is disabled by plugins.enabled" {
		t.Fatalf("unexpected error: %v", err)
	}
}
