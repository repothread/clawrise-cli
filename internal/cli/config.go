package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/spf13/pflag"

	"github.com/clawrise/clawrise-cli/internal/config"
	"github.com/clawrise/clawrise-cli/internal/output"
	pluginruntime "github.com/clawrise/clawrise-cli/internal/plugin"
)

// runConfig handles config bootstrap commands.
func runConfig(args []string, store *config.Store, stdout io.Writer, manager *pluginruntime.Manager) error {
	if len(args) == 0 || isHelpToken(args[0]) {
		printConfigHelp(stdout)
		return nil
	}

	switch args[0] {
	case "init":
		return runConfigInit(args[1:], store, stdout, manager)
	case "secret-store":
		return runConfigSecretStore(args[1:], store, stdout)
	default:
		return fmt.Errorf("unknown config command: %s", args[0])
	}
}

func runConfigInit(args []string, store *config.Store, stdout io.Writer, manager *pluginruntime.Manager) error {
	flags := pflag.NewFlagSet("clawrise config init", pflag.ContinueOnError)
	flags.SetOutput(stdout)

	var platform string
	var presetID string
	var subject string
	var account string
	var method string
	var scopes []string
	var force bool

	flags.StringVar(&platform, "platform", "", "set the platform for the default account")
	flags.StringVar(&presetID, "preset", "", "select the account preset id")
	flags.StringVar(&subject, "subject", "", "set the subject for the default account")
	flags.StringVar(&account, "account", "", "set the account name")
	flags.StringVar(&method, "method", "", "override the auth method")
	flags.StringSliceVar(&scopes, "scope", nil, "append auth scopes for interactive OAuth")
	flags.BoolVar(&force, "force", false, "overwrite the existing config file")

	if err := flags.Parse(args); err != nil {
		if err == pflag.ErrHelp {
			return nil
		}
		return err
	}
	if len(flags.Args()) != 0 {
		return fmt.Errorf("usage: clawrise config init --platform <name> [--preset <id>] [--subject <name>] [--account <name>] [--method <name>] [--scope <name>] [--force]")
	}
	if manager == nil {
		return fmt.Errorf("plugin manager is required for config init")
	}
	platform = strings.TrimSpace(platform)
	if platform == "" {
		return fmt.Errorf("config init requires --platform")
	}

	if _, err := os.Stat(store.Path()); err == nil && !force {
		return fmt.Errorf("config file already exists at %s; rerun with --force to overwrite it", store.Path())
	} else if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to stat config file: %w", err)
	}

	result, err := buildInitConfigFromMetadata(context.Background(), manager, initConfigOptions{
		Platform: platform,
		PresetID: strings.TrimSpace(presetID),
		Subject:  strings.TrimSpace(subject),
		Account:  strings.TrimSpace(account),
		Method:   strings.TrimSpace(method),
		Scopes:   scopes,
	})
	if err != nil {
		return err
	}
	if err := store.Save(result.Config); err != nil {
		return err
	}

	return output.WriteJSON(stdout, map[string]any{
		"ok": true,
		"data": map[string]any{
			"config_path":      store.Path(),
			"account":          result.AccountName,
			"platform":         result.Platform,
			"subject":          result.Subject,
			"method":           result.Method,
			"scopes":           initAccountScopes(result.Config.Accounts[result.AccountName]),
			"secret_backend":   result.SecretBackend,
			"session_backend":  result.SessionBackend,
			"required_secrets": result.SecretFields,
		},
	})
}

func printConfigHelp(stdout io.Writer) {
	_, _ = fmt.Fprintln(stdout, "Usage: clawrise config init --platform <name> [--preset <id>] [--subject <name>] [--account <name>] [--method <name>] [--scope <name>] [--force]")
	_, _ = fmt.Fprintln(stdout, "       clawrise config secret-store use <backend> [--fallback-backend <backend>]")
}

func runConfigSecretStore(args []string, store *config.Store, stdout io.Writer) error {
	if len(args) == 0 || isHelpToken(args[0]) {
		printConfigSecretStoreHelp(stdout)
		return nil
	}

	switch strings.TrimSpace(args[0]) {
	case "use":
		return runConfigSecretStoreUse(args[1:], store, stdout)
	default:
		return fmt.Errorf("unknown config secret-store command: %s", args[0])
	}
}

func runConfigSecretStoreUse(args []string, store *config.Store, stdout io.Writer) error {
	flags := pflag.NewFlagSet("clawrise config secret-store use", pflag.ContinueOnError)
	flags.SetOutput(stdout)

	var fallbackBackend string
	flags.StringVar(&fallbackBackend, "fallback-backend", "", "set the fallback backend used only when backend=auto")

	if err := flags.Parse(args); err != nil {
		if err == pflag.ErrHelp {
			return nil
		}
		return err
	}
	if len(flags.Args()) != 1 {
		return fmt.Errorf("usage: clawrise config secret-store use <backend> [--fallback-backend <backend>]")
	}

	backend := strings.TrimSpace(flags.Args()[0])
	if backend == "" {
		return fmt.Errorf("secret store backend must not be empty")
	}

	cfg, err := store.Load()
	if err != nil {
		return err
	}

	cfg.Auth.SecretStore.Backend = backend
	if backend == "auto" {
		fallbackBackend = strings.TrimSpace(fallbackBackend)
		if fallbackBackend == "" {
			fallbackBackend = "encrypted_file"
		}
		cfg.Auth.SecretStore.FallbackBackend = fallbackBackend
	} else {
		cfg.Auth.SecretStore.FallbackBackend = ""
	}

	if err := store.Save(cfg); err != nil {
		return err
	}

	return output.WriteJSON(stdout, map[string]any{
		"ok": true,
		"data": map[string]any{
			"config_path":      store.Path(),
			"secret_backend":   cfg.Auth.SecretStore.Backend,
			"fallback_backend": cfg.Auth.SecretStore.FallbackBackend,
		},
	})
}

func printConfigSecretStoreHelp(stdout io.Writer) {
	_, _ = fmt.Fprintln(stdout, "Usage: clawrise config secret-store use <backend> [--fallback-backend <backend>]")
}

type initConfigOptions struct {
	Platform string
	PresetID string
	Subject  string
	Account  string
	Method   string
	Scopes   []string
}

type initConfigResult struct {
	Config         *config.Config
	AccountName    string
	Platform       string
	Subject        string
	Method         string
	SecretFields   []string
	SessionBackend string
	SecretBackend  string
}

func buildInitConfigFromMetadata(ctx context.Context, manager *pluginruntime.Manager, opts initConfigOptions) (initConfigResult, error) {
	presets, err := manager.ListAuthPresets(ctx, opts.Platform)
	if err != nil {
		return initConfigResult{}, err
	}
	methods, err := manager.ListAuthMethods(ctx, opts.Platform)
	if err != nil {
		return initConfigResult{}, err
	}
	methodIndex := make(map[string]pluginruntime.AuthMethodDescriptor, len(methods))
	for _, method := range methods {
		methodIndex[strings.TrimSpace(method.ID)] = method
	}

	preset, err := selectInitPreset(presets, methodIndex, opts)
	if err != nil {
		return initConfigResult{}, err
	}

	accountName := strings.TrimSpace(opts.Account)
	if accountName == "" {
		accountName = strings.TrimSpace(preset.DefaultAccountName)
	}
	if accountName == "" {
		accountName = opts.Platform + "_" + preset.Subject + "_default"
	}

	account := config.Account{
		Title:    preset.DisplayName,
		Platform: preset.Platform,
		Subject:  preset.Subject,
		Auth: config.AccountAuth{
			Method:     preset.AuthMethod,
			Public:     buildInitPublicFields(preset, methodIndex[strings.TrimSpace(preset.AuthMethod)], opts.Scopes),
			SecretRefs: map[string]string{},
		},
	}
	for _, field := range preset.SecretFields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		account.Auth.SecretRefs[field] = config.SecretRef(accountName, field)
	}

	cfg := config.New()
	cfg.Ensure()
	cfg.Defaults.Platform = account.Platform
	cfg.Defaults.Subject = account.Subject
	cfg.Defaults.Account = accountName
	cfg.Defaults.PlatformAccounts[account.Platform] = accountName
	applyBootstrapDefaults(cfg)
	cfg.Accounts[accountName] = account

	secretFields := append([]string(nil), preset.SecretFields...)
	sort.Strings(secretFields)
	return initConfigResult{
		Config:         cfg,
		AccountName:    accountName,
		Platform:       account.Platform,
		Subject:        account.Subject,
		Method:         account.Auth.Method,
		SecretFields:   secretFields,
		SessionBackend: cfg.Auth.SessionStore.Backend,
		SecretBackend:  cfg.Auth.SecretStore.Backend,
	}, nil
}

func applyBootstrapDefaults(cfg *config.Config) {
	cfg.Ensure()

	if strings.TrimSpace(cfg.Auth.SecretStore.Backend) == "" {
		cfg.Auth.SecretStore.Backend = "encrypted_file"
	}
	if strings.TrimSpace(cfg.Auth.SessionStore.Backend) == "" {
		cfg.Auth.SessionStore.Backend = "file"
	}
	if cfg.Runtime.Retry.MaxAttempts == 0 {
		cfg.Runtime.Retry.MaxAttempts = 1
	}
	if cfg.Runtime.Retry.BaseDelayMS == 0 {
		cfg.Runtime.Retry.BaseDelayMS = 200
	}
	if cfg.Runtime.Retry.MaxDelayMS == 0 {
		cfg.Runtime.Retry.MaxDelayMS = 1000
	}
}

func selectInitPreset(presets []pluginruntime.AuthPresetDescriptor, methodIndex map[string]pluginruntime.AuthMethodDescriptor, opts initConfigOptions) (pluginruntime.AuthPresetDescriptor, error) {
	filtered := make([]pluginruntime.AuthPresetDescriptor, 0, len(presets))
	for _, preset := range presets {
		if opts.PresetID != "" && strings.TrimSpace(preset.ID) != opts.PresetID {
			continue
		}
		if opts.Subject != "" && strings.TrimSpace(preset.Subject) != opts.Subject {
			continue
		}
		if opts.Method != "" && strings.TrimSpace(preset.AuthMethod) != opts.Method {
			continue
		}
		filtered = append(filtered, preset)
	}
	if len(filtered) == 0 {
		return pluginruntime.AuthPresetDescriptor{}, fmt.Errorf("no account preset matches platform=%s preset=%s subject=%s method=%s", opts.Platform, opts.PresetID, opts.Subject, opts.Method)
	}

	sort.SliceStable(filtered, func(i, j int) bool {
		leftMethod := methodIndex[strings.TrimSpace(filtered[i].AuthMethod)]
		rightMethod := methodIndex[strings.TrimSpace(filtered[j].AuthMethod)]
		leftMachine := strings.TrimSpace(leftMethod.Kind) == "machine"
		rightMachine := strings.TrimSpace(rightMethod.Kind) == "machine"
		if leftMachine != rightMachine {
			return leftMachine
		}
		if filtered[i].ID != filtered[j].ID {
			return filtered[i].ID < filtered[j].ID
		}
		return filtered[i].AuthMethod < filtered[j].AuthMethod
	})
	return filtered[0], nil
}

func buildInitPublicFields(preset pluginruntime.AuthPresetDescriptor, method pluginruntime.AuthMethodDescriptor, scopes []string) map[string]any {
	public := cloneAnyMap(preset.Public)
	if public == nil {
		public = map[string]any{}
	}
	supportsScopes := false
	for _, field := range method.PublicFields {
		name := strings.TrimSpace(field.Name)
		if name == "" {
			continue
		}
		if name == "scopes" {
			supportsScopes = true
		}
		if _, ok := public[name]; ok {
			continue
		}
		switch strings.TrimSpace(field.Type) {
		case "string_list":
			public[name] = []string{}
		default:
			public[name] = ""
		}
	}
	if _, ok := public["scopes"]; ok {
		supportsScopes = true
	}
	if len(scopes) > 0 && supportsScopes {
		public["scopes"] = normalizeInitScopes(scopes)
	}
	if len(public) == 0 {
		return nil
	}
	return public
}

func initAccountScopes(account config.Account) []string {
	if account.Auth.Public == nil {
		return nil
	}
	value, ok := account.Auth.Public["scopes"]
	if !ok {
		return nil
	}
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []any:
		items := make([]string, 0, len(typed))
		for _, raw := range typed {
			text, ok := raw.(string)
			if !ok {
				continue
			}
			text = strings.TrimSpace(text)
			if text == "" {
				continue
			}
			items = append(items, text)
		}
		return items
	default:
		return nil
	}
}

func normalizeInitScopes(scopes []string) []string {
	items := make([]string, 0, len(scopes))
	seen := map[string]struct{}{}
	for _, value := range scopes {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		items = append(items, value)
	}
	return items
}
