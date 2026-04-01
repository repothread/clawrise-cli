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
	case "provider":
		return runConfigProvider(args[1:], store, stdout)
	case "auth-launcher":
		return runConfigAuthLauncher(args[1:], store, stdout, manager)
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
	_, _ = fmt.Fprintln(stdout, "       clawrise config provider use <platform> <plugin>")
	_, _ = fmt.Fprintln(stdout, "       clawrise config provider unset <platform>")
	_, _ = fmt.Fprintln(stdout, "       clawrise config auth-launcher prefer <action_type> <launcher_id>")
	_, _ = fmt.Fprintln(stdout, "       clawrise config auth-launcher unset <action_type> [launcher_id]")
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
	cfg.Auth.SecretStore.Plugin = "builtin"
	if backend == "auto" {
		fallbackBackend = strings.TrimSpace(fallbackBackend)
		if fallbackBackend == "" {
			fallbackBackend = "encrypted_file"
		}
		cfg.Auth.SecretStore.FallbackBackend = fallbackBackend
	} else {
		cfg.Auth.SecretStore.FallbackBackend = ""
	}
	cfg.Ensure()
	cfg.Plugins.Bindings.Storage.SecretStore = config.StoragePluginBinding{
		Backend:         cfg.Auth.SecretStore.Backend,
		Plugin:          "builtin",
		FallbackBackend: cfg.Auth.SecretStore.FallbackBackend,
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

func runConfigProvider(args []string, store *config.Store, stdout io.Writer) error {
	if len(args) == 0 || isHelpToken(args[0]) {
		printConfigProviderHelp(stdout)
		return nil
	}

	switch strings.TrimSpace(args[0]) {
	case "use":
		return runConfigProviderUse(args[1:], store, stdout)
	case "unset":
		return runConfigProviderUnset(args[1:], store, stdout)
	default:
		return fmt.Errorf("unknown config provider command: %s", args[0])
	}
}

func runConfigProviderUse(args []string, store *config.Store, stdout io.Writer) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: clawrise config provider use <platform> <plugin>")
	}

	platform := strings.TrimSpace(args[0])
	pluginName := strings.TrimSpace(args[1])
	if platform == "" || pluginName == "" {
		return fmt.Errorf("platform and plugin must not be empty")
	}

	cfg, err := store.Load()
	if err != nil {
		return err
	}
	discoveryOptions := buildPluginDiscoveryOptions(cfg)

	candidates, err := pluginruntime.DiscoverProviderCandidatesWithOptions(discoveryOptions)
	if err != nil {
		return err
	}
	allCandidates, err := pluginruntime.DiscoverProviderCandidates()
	if err != nil {
		return err
	}
	matched := false
	available := make([]string, 0)
	for _, candidate := range candidates {
		if candidate.Platform != platform {
			continue
		}
		available = append(available, candidate.Plugin)
		if candidate.Plugin == pluginName {
			matched = true
		}
	}
	if !matched {
		disabledByRule := false
		for _, candidate := range allCandidates {
			if candidate.Platform != platform {
				continue
			}
			if candidate.Plugin == pluginName {
				disabledByRule = true
				break
			}
		}
		if disabledByRule {
			return fmt.Errorf("plugin %s supports platform %s, but it is disabled by plugins.enabled", pluginName, platform)
		}
		if len(available) == 0 {
			return fmt.Errorf("no provider plugin currently supports platform %s", platform)
		}
		sort.Strings(available)
		return fmt.Errorf("plugin %s does not support platform %s; available plugins: %s", pluginName, platform, strings.Join(available, ", "))
	}
	cfg.Ensure()
	cfg.Plugins.Bindings.Providers[platform] = config.ProviderPluginBinding{
		Plugin: pluginName,
	}
	if err := store.Save(cfg); err != nil {
		return err
	}

	return output.WriteJSON(stdout, map[string]any{
		"ok": true,
		"data": map[string]any{
			"config_path": store.Path(),
			"platform":    platform,
			"plugin":      pluginName,
		},
	})
}

func runConfigProviderUnset(args []string, store *config.Store, stdout io.Writer) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: clawrise config provider unset <platform>")
	}

	platform := strings.TrimSpace(args[0])
	if platform == "" {
		return fmt.Errorf("platform must not be empty")
	}

	cfg, err := store.Load()
	if err != nil {
		return err
	}
	cfg.Ensure()
	delete(cfg.Plugins.Bindings.Providers, platform)
	if err := store.Save(cfg); err != nil {
		return err
	}

	return output.WriteJSON(stdout, map[string]any{
		"ok": true,
		"data": map[string]any{
			"config_path": store.Path(),
			"platform":    platform,
			"unset":       true,
		},
	})
}

func printConfigProviderHelp(stdout io.Writer) {
	_, _ = fmt.Fprintln(stdout, "Usage: clawrise config provider use <platform> <plugin>")
	_, _ = fmt.Fprintln(stdout, "       clawrise config provider unset <platform>")
}

func runConfigAuthLauncher(args []string, store *config.Store, stdout io.Writer, manager *pluginruntime.Manager) error {
	if len(args) == 0 || isHelpToken(args[0]) {
		printConfigAuthLauncherHelp(stdout)
		return nil
	}

	switch strings.TrimSpace(args[0]) {
	case "prefer":
		return runConfigAuthLauncherPrefer(args[1:], store, stdout, manager)
	case "unset":
		return runConfigAuthLauncherUnset(args[1:], store, stdout)
	default:
		return fmt.Errorf("unknown config auth-launcher command: %s", args[0])
	}
}

func runConfigAuthLauncherPrefer(args []string, store *config.Store, stdout io.Writer, manager *pluginruntime.Manager) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: clawrise config auth-launcher prefer <action_type> <launcher_id>")
	}
	if manager == nil {
		return fmt.Errorf("plugin manager is required for config auth-launcher prefer")
	}

	actionType := strings.TrimSpace(args[0])
	launcherID := strings.TrimSpace(args[1])
	if actionType == "" || launcherID == "" {
		return fmt.Errorf("action_type and launcher_id must not be empty")
	}

	launchers := manager.AuthLaunchers()
	available := make([]string, 0)
	matched := false
	for _, launcher := range launchers {
		if !launcherSupportsConfiguredAction(launcher, actionType) {
			continue
		}
		available = append(available, launcher.ID)
		if launcher.ID == launcherID {
			matched = true
		}
	}
	if !matched {
		if len(available) == 0 {
			return fmt.Errorf("no auth launcher currently supports action %s", actionType)
		}
		sort.Strings(available)
		return fmt.Errorf("auth launcher %s does not support action %s; available launchers: %s", launcherID, actionType, strings.Join(available, ", "))
	}

	cfg, err := store.Load()
	if err != nil {
		return err
	}
	preferences := config.SetAuthLauncherPreference(cfg, actionType, launcherID)
	if err := store.Save(cfg); err != nil {
		return err
	}

	return output.WriteJSON(stdout, map[string]any{
		"ok": true,
		"data": map[string]any{
			"config_path": store.Path(),
			"action_type": actionType,
			"launcher_id": launcherID,
			"preferences": preferences,
		},
	})
}

func runConfigAuthLauncherUnset(args []string, store *config.Store, stdout io.Writer) error {
	if len(args) != 1 && len(args) != 2 {
		return fmt.Errorf("usage: clawrise config auth-launcher unset <action_type> [launcher_id]")
	}

	actionType := strings.TrimSpace(args[0])
	launcherID := ""
	if len(args) == 2 {
		launcherID = strings.TrimSpace(args[1])
	}
	if actionType == "" {
		return fmt.Errorf("action_type must not be empty")
	}

	cfg, err := store.Load()
	if err != nil {
		return err
	}
	preferences := config.UnsetAuthLauncherPreference(cfg, actionType, launcherID)
	if err := store.Save(cfg); err != nil {
		return err
	}

	return output.WriteJSON(stdout, map[string]any{
		"ok": true,
		"data": map[string]any{
			"config_path": store.Path(),
			"action_type": actionType,
			"launcher_id": launcherID,
			"preferences": preferences,
			"unset":       true,
		},
	})
}

func printConfigAuthLauncherHelp(stdout io.Writer) {
	_, _ = fmt.Fprintln(stdout, "Usage: clawrise config auth-launcher prefer <action_type> <launcher_id>")
	_, _ = fmt.Fprintln(stdout, "       clawrise config auth-launcher unset <action_type> [launcher_id]")
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
	if runtime, ok := manager.RuntimeForPlatform(account.Platform); ok {
		if manifestRuntime, ok := runtime.(interface{ Manifest() pluginruntime.Manifest }); ok {
			manifest := manifestRuntime.Manifest()
			if strings.TrimSpace(manifest.Name) != "" {
				cfg.Plugins.Bindings.Providers[account.Platform] = config.ProviderPluginBinding{
					Plugin: manifest.Name,
				}
			}
		}
	}
	cfg.Accounts[accountName] = account

	secretFields := append([]string(nil), preset.SecretFields...)
	sort.Strings(secretFields)
	secretBinding := config.ResolveStorageBinding(cfg, "secret_store")
	sessionBinding := config.ResolveStorageBinding(cfg, "session_store")
	return initConfigResult{
		Config:         cfg,
		AccountName:    accountName,
		Platform:       account.Platform,
		Subject:        account.Subject,
		Method:         account.Auth.Method,
		SecretFields:   secretFields,
		SessionBackend: sessionBinding.Backend,
		SecretBackend:  secretBinding.Backend,
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
	if strings.TrimSpace(cfg.Auth.AuthFlowStore.Backend) == "" {
		cfg.Auth.AuthFlowStore.Backend = "file"
	}
	if strings.TrimSpace(cfg.Runtime.Governance.Backend) == "" {
		cfg.Runtime.Governance.Backend = "file"
	}
	if !cfg.Plugins.Bindings.Storage.SecretStore.HasValue() {
		cfg.Plugins.Bindings.Storage.SecretStore = bootstrapStorageBinding(cfg.Auth.SecretStore.Backend, cfg.Auth.SecretStore.Plugin, cfg.Auth.SecretStore.FallbackBackend)
	}
	if !cfg.Plugins.Bindings.Storage.SessionStore.HasValue() {
		cfg.Plugins.Bindings.Storage.SessionStore = bootstrapStorageBinding(cfg.Auth.SessionStore.Backend, cfg.Auth.SessionStore.Plugin, "")
	}
	if !cfg.Plugins.Bindings.Storage.AuthFlowStore.HasValue() {
		cfg.Plugins.Bindings.Storage.AuthFlowStore = bootstrapStorageBinding(cfg.Auth.AuthFlowStore.Backend, cfg.Auth.AuthFlowStore.Plugin, "")
	}
	if !cfg.Plugins.Bindings.Storage.Governance.HasValue() {
		cfg.Plugins.Bindings.Storage.Governance = bootstrapStorageBinding(cfg.Runtime.Governance.Backend, cfg.Runtime.Governance.Plugin, "")
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

func bootstrapStorageBinding(backend string, plugin string, fallbackBackend string) config.StoragePluginBinding {
	backend = strings.TrimSpace(backend)
	plugin = strings.TrimSpace(plugin)
	fallbackBackend = strings.TrimSpace(fallbackBackend)
	if plugin == "" && backend != "" {
		plugin = "builtin"
	}
	return config.StoragePluginBinding{
		Backend:         backend,
		Plugin:          plugin,
		FallbackBackend: fallbackBackend,
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

func launcherSupportsConfiguredAction(launcher pluginruntime.AuthLauncherDescriptor, actionType string) bool {
	actionType = strings.TrimSpace(actionType)
	if actionType == "" {
		return false
	}
	if len(launcher.ActionTypes) == 0 {
		return true
	}
	for _, item := range launcher.ActionTypes {
		if strings.TrimSpace(item) == actionType {
			return true
		}
	}
	return false
}
