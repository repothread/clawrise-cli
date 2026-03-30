package account

import (
	"testing"

	"github.com/clawrise/clawrise-cli/internal/config"
)

func TestResolveSelectionUsesDefaultAccount(t *testing.T) {
	cfg := &config.Config{
		Defaults: config.Defaults{
			Account: "demo_account",
		},
		Accounts: map[string]config.Account{
			"demo_account": {
				Platform: "demo",
				Subject:  "integration",
			},
		},
	}

	selection, err := ResolveSelection(cfg, "")
	if err != nil {
		t.Fatalf("ResolveSelection returned error: %v", err)
	}
	if !selection.Found || selection.Name != "demo_account" {
		t.Fatalf("unexpected selection: %+v", selection)
	}
}
