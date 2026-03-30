package cli

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/spf13/pflag"

	"github.com/clawrise/clawrise-cli/internal/config"
	"github.com/clawrise/clawrise-cli/internal/output"
	pluginruntime "github.com/clawrise/clawrise-cli/internal/plugin"
)

// runAccount manages account configuration.
func runAccount(args []string, store *config.Store, stdout io.Writer, manager *pluginruntime.Manager) error {
	if len(args) == 0 || isHelpToken(args[0]) {
		printAccountHelp(stdout)
		return nil
	}

	cfg, err := store.Load()
	if err != nil {
		return err
	}

	switch args[0] {
	case "list":
		return runAccountList(cfg, stdout)
	case "inspect":
		return runAccountInspect(args[1:], cfg, stdout)
	case "use":
		return runAccountUse(args[1:], cfg, store, stdout)
	case "current":
		return runAccountCurrent(cfg, stdout)
	case "remove":
		return runAccountRemove(args[1:], cfg, store, stdout)
	case "add":
		return runAccountAdd(args[1:], cfg, store, stdout, manager)
	default:
		return fmt.Errorf("unknown account command: %s", args[0])
	}
}

func runAccountList(cfg *config.Config, stdout io.Writer) error {
	cfg.Ensure()

	names := make([]string, 0, len(cfg.Accounts))
	for name := range cfg.Accounts {
		names = append(names, name)
	}
	sort.Strings(names)

	items := make([]map[string]any, 0, len(names))
	for _, name := range names {
		account := cfg.Accounts[name]
		items = append(items, map[string]any{
			"name":        name,
			"title":       account.Title,
			"platform":    account.Platform,
			"subject":     account.Subject,
			"auth_method": account.Auth.Method,
		})
	}
	return output.WriteJSON(stdout, map[string]any{
		"accounts": items,
	})
}

func runAccountInspect(args []string, cfg *config.Config, stdout io.Writer) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: clawrise account inspect <account>")
	}

	cfg.Ensure()
	accountName := strings.TrimSpace(args[0])
	account, ok := cfg.Accounts[accountName]
	if !ok {
		return writeCLIError(stdout, "ACCOUNT_NOT_FOUND", "the selected account does not exist")
	}

	return output.WriteJSON(stdout, map[string]any{
		"ok": true,
		"data": map[string]any{
			"name":     accountName,
			"title":    account.Title,
			"platform": account.Platform,
			"subject":  account.Subject,
			"auth": map[string]any{
				"method":      account.Auth.Method,
				"public":      account.Auth.Public,
				"secret_refs": account.Auth.SecretRefs,
			},
		},
	})
}

func runAccountUse(args []string, cfg *config.Config, store *config.Store, stdout io.Writer) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: clawrise account use <account>")
	}

	cfg.Ensure()
	accountName := strings.TrimSpace(args[0])
	account, ok := cfg.Accounts[accountName]
	if !ok {
		return writeCLIError(stdout, "ACCOUNT_NOT_FOUND", "the selected account does not exist")
	}

	cfg.Defaults.Account = accountName
	cfg.Defaults.Platform = account.Platform
	cfg.Defaults.PlatformAccounts[account.Platform] = accountName
	cfg.Defaults.Subject = account.Subject
	if err := store.Save(cfg); err != nil {
		return err
	}

	return output.WriteJSON(stdout, map[string]any{
		"ok": true,
		"account": map[string]any{
			"name":        accountName,
			"platform":    account.Platform,
			"subject":     account.Subject,
			"auth_method": account.Auth.Method,
		},
	})
}

func runAccountCurrent(cfg *config.Config, stdout io.Writer) error {
	cfg.Ensure()

	var current any
	if strings.TrimSpace(cfg.Defaults.Account) != "" {
		accountName := strings.TrimSpace(cfg.Defaults.Account)
		account, ok := cfg.Accounts[accountName]
		if ok {
			current = map[string]any{
				"name":        accountName,
				"platform":    account.Platform,
				"subject":     account.Subject,
				"auth_method": account.Auth.Method,
			}
		}
	}

	return output.WriteJSON(stdout, map[string]any{
		"account": current,
	})
}

func runAccountRemove(args []string, cfg *config.Config, store *config.Store, stdout io.Writer) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: clawrise account remove <account>")
	}

	cfg.Ensure()
	accountName := strings.TrimSpace(args[0])
	account, ok := cfg.Accounts[accountName]
	if !ok {
		return writeCLIError(stdout, "ACCOUNT_NOT_FOUND", "the selected account does not exist")
	}

	delete(cfg.Accounts, accountName)
	if strings.TrimSpace(cfg.Defaults.Account) == accountName {
		cfg.Defaults.Account = ""
	}
	if strings.TrimSpace(cfg.Defaults.PlatformAccounts[account.Platform]) == accountName {
		delete(cfg.Defaults.PlatformAccounts, account.Platform)
	}
	if err := store.Save(cfg); err != nil {
		return err
	}

	return output.WriteJSON(stdout, map[string]any{
		"ok": true,
		"data": map[string]any{
			"name":    accountName,
			"deleted": true,
		},
	})
}

func runAccountAdd(args []string, cfg *config.Config, store *config.Store, stdout io.Writer, manager *pluginruntime.Manager) error {
	flags := pflag.NewFlagSet("clawrise account add", pflag.ContinueOnError)
	flags.SetOutput(stdout)

	var platform string
	var presetID string
	var title string
	var useAsDefault bool

	flags.StringVar(&platform, "platform", "", "set the platform for the new account")
	flags.StringVar(&presetID, "preset", "", "select the account preset id")
	flags.StringVar(&title, "title", "", "set the account title")
	flags.BoolVar(&useAsDefault, "use", false, "set the new account as default")

	if err := flags.Parse(args); err != nil {
		if err == pflag.ErrHelp {
			return nil
		}
		return err
	}
	if len(flags.Args()) > 1 {
		return fmt.Errorf("usage: clawrise account add [name] --platform <name> --preset <id>")
	}
	if strings.TrimSpace(platform) == "" {
		return fmt.Errorf("account platform is required")
	}
	if strings.TrimSpace(presetID) == "" {
		return fmt.Errorf("account preset is required")
	}
	if manager == nil {
		return fmt.Errorf("plugin manager is required for account add")
	}

	presets, err := manager.ListAuthPresets(context.Background(), platform)
	if err != nil {
		return err
	}

	var preset *pluginruntime.AuthPresetDescriptor
	for index := range presets {
		if strings.TrimSpace(presets[index].ID) == strings.TrimSpace(presetID) {
			preset = &presets[index]
			break
		}
	}
	if preset == nil {
		return writeCLIError(stdout, "ACCOUNT_PRESET_NOT_FOUND", "the selected account preset does not exist")
	}

	accountName := ""
	if len(flags.Args()) == 1 {
		accountName = strings.TrimSpace(flags.Args()[0])
	}
	if accountName == "" {
		accountName = strings.TrimSpace(preset.DefaultAccountName)
	}
	if accountName == "" {
		return fmt.Errorf("account name is required")
	}

	cfg.Ensure()
	if _, exists := cfg.Accounts[accountName]; exists {
		return writeCLIError(stdout, "ACCOUNT_ALREADY_EXISTS", "the selected account name already exists")
	}

	account := config.Account{
		Title:    strings.TrimSpace(title),
		Platform: preset.Platform,
		Subject:  preset.Subject,
		Auth: config.AccountAuth{
			Method:     preset.AuthMethod,
			Public:     cloneAnyMap(preset.Public),
			SecretRefs: map[string]string{},
		},
	}
	if account.Title == "" {
		account.Title = preset.DisplayName
	}
	for _, field := range preset.SecretFields {
		account.Auth.SecretRefs[field] = config.SecretRef(accountName, field)
	}

	cfg.Accounts[accountName] = account
	cfg.Ensure()

	if useAsDefault {
		cfg.Defaults.Account = accountName
		cfg.Defaults.Platform = account.Platform
		cfg.Defaults.PlatformAccounts[account.Platform] = accountName
	}

	if err := store.Save(cfg); err != nil {
		return err
	}

	return output.WriteJSON(stdout, map[string]any{
		"ok": true,
		"data": map[string]any{
			"name":          accountName,
			"platform":      account.Platform,
			"subject":       account.Subject,
			"auth_method":   account.Auth.Method,
			"secret_fields": preset.SecretFields,
			"public":        account.Auth.Public,
		},
	})
}

func cloneAnyMap(values map[string]any) map[string]any {
	if len(values) == 0 {
		return nil
	}
	cloned := make(map[string]any, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func printAccountHelp(stdout io.Writer) {
	_, _ = fmt.Fprintln(stdout, "Usage: clawrise account [list|inspect|use|current|add|remove]")
}
