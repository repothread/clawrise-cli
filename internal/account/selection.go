package account

import (
	"fmt"
	"strings"

	"github.com/clawrise/clawrise-cli/internal/config"
)

// Selection describes one resolved account selection result.
type Selection struct {
	Name    string
	Account config.Account
	Found   bool
}

// ResolveSelection resolves an account in this order:
// explicit account, default account, then platform default account.
func ResolveSelection(cfg *config.Config, explicitName string) (Selection, error) {
	cfg.Ensure()

	name := strings.TrimSpace(explicitName)
	if name == "" && strings.TrimSpace(cfg.Defaults.Account) != "" {
		name = strings.TrimSpace(cfg.Defaults.Account)
	}
	if name == "" && strings.TrimSpace(cfg.Defaults.Platform) != "" {
		name = strings.TrimSpace(cfg.Defaults.PlatformAccounts[cfg.Defaults.Platform])
	}
	if name == "" {
		return Selection{}, fmt.Errorf("no account was provided and no default account is configured")
	}

	account, ok := cfg.Accounts[name]
	return Selection{
		Name:    name,
		Account: account,
		Found:   ok,
	}, nil
}
