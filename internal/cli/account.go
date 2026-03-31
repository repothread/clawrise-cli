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
	case "ensure":
		return runAccountEnsure(args[1:], cfg, store, stdout, manager)
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
	var subject string
	var method string
	var scopes []string
	var title string
	var useAsDefault bool

	flags.StringVar(&platform, "platform", "", "set the platform for the new account")
	flags.StringVar(&presetID, "preset", "", "select the account preset id")
	flags.StringVar(&subject, "subject", "", "set the subject for the new account")
	flags.StringVar(&method, "method", "", "override the auth method")
	flags.StringSliceVar(&scopes, "scope", nil, "append auth scopes for interactive OAuth")
	flags.StringVar(&title, "title", "", "set the account title")
	flags.BoolVar(&useAsDefault, "use", false, "set the new account as default")

	if err := flags.Parse(args); err != nil {
		if err == pflag.ErrHelp {
			return nil
		}
		return err
	}
	if len(flags.Args()) > 1 {
		return fmt.Errorf("usage: clawrise account add [name] --platform <name> [--preset <id>] [--subject <name>] [--method <name>] [--scope <name>] [--use]")
	}
	if strings.TrimSpace(platform) == "" {
		return fmt.Errorf("account platform is required")
	}
	if manager == nil {
		return fmt.Errorf("plugin manager is required for account add")
	}

	accountName := ""
	if len(flags.Args()) == 1 {
		accountName = strings.TrimSpace(flags.Args()[0])
	}

	initResult, err := buildInitConfigFromMetadata(context.Background(), manager, initConfigOptions{
		Platform: strings.TrimSpace(platform),
		PresetID: strings.TrimSpace(presetID),
		Subject:  strings.TrimSpace(subject),
		Account:  accountName,
		Method:   strings.TrimSpace(method),
		Scopes:   scopes,
	})
	if err != nil {
		return err
	}

	accountName = initResult.AccountName
	account := initResult.Config.Accounts[accountName]
	if strings.TrimSpace(title) != "" {
		account.Title = strings.TrimSpace(title)
	}

	cfg.Ensure()
	if _, exists := cfg.Accounts[accountName]; exists {
		return writeCLIError(stdout, "ACCOUNT_ALREADY_EXISTS", "the selected account name already exists")
	}

	applyBootstrapDefaults(cfg)
	cfg.Accounts[accountName] = account

	shouldUseAsDefault := useAsDefault || len(cfg.Accounts) == 1 || strings.TrimSpace(cfg.Defaults.Account) == ""
	if shouldUseAsDefault {
		cfg.Defaults.Account = accountName
		cfg.Defaults.Platform = account.Platform
		cfg.Defaults.Subject = account.Subject
		cfg.Defaults.PlatformAccounts[account.Platform] = accountName
	} else if strings.TrimSpace(cfg.Defaults.PlatformAccounts[account.Platform]) == "" {
		cfg.Defaults.PlatformAccounts[account.Platform] = accountName
	}

	if err := store.Save(cfg); err != nil {
		return err
	}

	return output.WriteJSON(stdout, map[string]any{
		"ok": true,
		"data": map[string]any{
			"name":            accountName,
			"platform":        account.Platform,
			"subject":         account.Subject,
			"auth_method":     account.Auth.Method,
			"secret_fields":   initResult.SecretFields,
			"public":          account.Auth.Public,
			"used_as_default": shouldUseAsDefault,
		},
	})
}

func runAccountEnsure(args []string, cfg *config.Config, store *config.Store, stdout io.Writer, manager *pluginruntime.Manager) error {
	flags := pflag.NewFlagSet("clawrise account ensure", pflag.ContinueOnError)
	flags.SetOutput(stdout)

	var platform string
	var presetID string
	var subject string
	var method string
	var scopes []string
	var title string
	var useAsDefault bool
	var publicAssignments []string

	flags.StringVar(&platform, "platform", "", "设置账号所属平台")
	flags.StringVar(&presetID, "preset", "", "设置账号 preset")
	flags.StringVar(&subject, "subject", "", "设置账号 subject")
	flags.StringVar(&method, "method", "", "覆盖 auth method")
	flags.StringSliceVar(&scopes, "scope", nil, "为交互式 OAuth 追加 scope")
	flags.StringVar(&title, "title", "", "设置账号标题")
	flags.BoolVar(&useAsDefault, "use", false, "将账号设为默认账号")
	flags.StringArrayVar(&publicAssignments, "public", nil, "设置 public 字段，格式为 key=value")

	if err := flags.Parse(args); err != nil {
		if err == pflag.ErrHelp {
			return nil
		}
		return err
	}
	if len(flags.Args()) > 1 {
		return fmt.Errorf("usage: clawrise account ensure [name] --platform <name> [--preset <id>] [--subject <name>] [--method <name>] [--scope <name>] [--public <key=value>] [--use]")
	}
	if strings.TrimSpace(platform) == "" {
		return fmt.Errorf("account platform is required")
	}
	if manager == nil {
		return fmt.Errorf("plugin manager is required for account ensure")
	}

	publicOverrides, err := parseAccountPublicAssignments(publicAssignments)
	if err != nil {
		return err
	}

	accountName := ""
	if len(flags.Args()) == 1 {
		accountName = strings.TrimSpace(flags.Args()[0])
	}

	initResult, err := buildInitConfigFromMetadata(context.Background(), manager, initConfigOptions{
		Platform: strings.TrimSpace(platform),
		PresetID: strings.TrimSpace(presetID),
		Subject:  strings.TrimSpace(subject),
		Account:  accountName,
		Method:   strings.TrimSpace(method),
		Scopes:   scopes,
	})
	if err != nil {
		return err
	}

	accountName = initResult.AccountName
	desiredAccount := initResult.Config.Accounts[accountName]
	if strings.TrimSpace(title) != "" {
		desiredAccount.Title = strings.TrimSpace(title)
	}

	cfg.Ensure()
	action := "created"
	existingAccount, exists := cfg.Accounts[accountName]
	if exists {
		if strings.TrimSpace(existingAccount.Platform) != strings.TrimSpace(desiredAccount.Platform) {
			return writeCLIError(stdout, "ACCOUNT_PLATFORM_MISMATCH", "the selected account already exists with a different platform")
		}
		if strings.TrimSpace(existingAccount.Subject) != strings.TrimSpace(desiredAccount.Subject) {
			return writeCLIError(stdout, "ACCOUNT_SUBJECT_MISMATCH", "the selected account already exists with a different subject")
		}
		if strings.TrimSpace(existingAccount.Auth.Method) != strings.TrimSpace(desiredAccount.Auth.Method) {
			return writeCLIError(stdout, "ACCOUNT_AUTH_METHOD_MISMATCH", "the selected account already exists with a different auth method")
		}
		desiredAccount = mergeEnsuredAccount(existingAccount, desiredAccount, publicOverrides, strings.TrimSpace(title))
		action = "updated"
	} else {
		desiredAccount = mergeEnsuredAccount(config.Account{}, desiredAccount, publicOverrides, strings.TrimSpace(title))
	}

	applyBootstrapDefaults(cfg)
	cfg.Accounts[accountName] = desiredAccount

	shouldUseAsDefault := useAsDefault || len(cfg.Accounts) == 1 || strings.TrimSpace(cfg.Defaults.Account) == ""
	if shouldUseAsDefault {
		cfg.Defaults.Account = accountName
		cfg.Defaults.Platform = desiredAccount.Platform
		cfg.Defaults.Subject = desiredAccount.Subject
		cfg.Defaults.PlatformAccounts[desiredAccount.Platform] = accountName
	} else if strings.TrimSpace(cfg.Defaults.PlatformAccounts[desiredAccount.Platform]) == "" {
		cfg.Defaults.PlatformAccounts[desiredAccount.Platform] = accountName
	}

	if err := store.Save(cfg); err != nil {
		return err
	}

	return output.WriteJSON(stdout, map[string]any{
		"ok": true,
		"data": map[string]any{
			"name":            accountName,
			"platform":        desiredAccount.Platform,
			"subject":         desiredAccount.Subject,
			"auth_method":     desiredAccount.Auth.Method,
			"secret_fields":   initResult.SecretFields,
			"public":          desiredAccount.Auth.Public,
			"used_as_default": shouldUseAsDefault,
			"action":          action,
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

func parseAccountPublicAssignments(assignments []string) (map[string]string, error) {
	if len(assignments) == 0 {
		return nil, nil
	}

	result := make(map[string]string, len(assignments))
	for _, assignment := range assignments {
		text := strings.TrimSpace(assignment)
		if text == "" {
			continue
		}
		parts := strings.SplitN(text, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid public assignment %q: expected key=value", assignment)
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key == "" {
			return nil, fmt.Errorf("invalid public assignment %q: key must not be empty", assignment)
		}
		result[key] = value
	}
	if len(result) == 0 {
		return nil, nil
	}
	return result, nil
}

func mergeEnsuredAccount(existing config.Account, desired config.Account, publicOverrides map[string]string, explicitTitle string) config.Account {
	result := existing
	result.Platform = desired.Platform
	result.Subject = desired.Subject
	result.Auth.Method = desired.Auth.Method
	result.Auth.Public = mergeEnsuredPublicFields(existing.Auth.Public, desired.Auth.Public, publicOverrides)
	result.Auth.SecretRefs = mergeEnsuredSecretRefs(existing.Auth.SecretRefs, desired.Auth.SecretRefs)

	if strings.TrimSpace(explicitTitle) != "" {
		result.Title = strings.TrimSpace(explicitTitle)
	} else if strings.TrimSpace(result.Title) == "" {
		result.Title = desired.Title
	}

	return result
}

func mergeEnsuredPublicFields(existing map[string]any, desired map[string]any, overrides map[string]string) map[string]any {
	result := cloneAnyMap(existing)
	if result == nil {
		result = map[string]any{}
	}

	for key, value := range desired {
		if _, exists := result[key]; !exists {
			result[key] = value
			continue
		}
		if isEmptyPublicValue(value) {
			continue
		}
		result[key] = value
	}

	for key, value := range overrides {
		result[key] = value
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

func mergeEnsuredSecretRefs(existing map[string]string, desired map[string]string) map[string]string {
	result := map[string]string{}
	for key, value := range existing {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		result[key] = value
	}
	for key, value := range desired {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		result[key] = value
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func isEmptyPublicValue(value any) bool {
	switch typed := value.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(typed) == ""
	case []string:
		return len(typed) == 0
	case []any:
		return len(typed) == 0
	default:
		return false
	}
}

func printAccountHelp(stdout io.Writer) {
	_, _ = fmt.Fprintln(stdout, "Usage: clawrise account [list|inspect|use|current|add|remove]")
}
