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
