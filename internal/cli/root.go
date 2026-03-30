package cli

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/pflag"

	"github.com/clawrise/clawrise-cli/internal/config"
	"github.com/clawrise/clawrise-cli/internal/locator"
	"github.com/clawrise/clawrise-cli/internal/metadata"
	"github.com/clawrise/clawrise-cli/internal/output"
	pluginruntime "github.com/clawrise/clawrise-cli/internal/plugin"
	"github.com/clawrise/clawrise-cli/internal/runtime"
)

// Dependencies describes the base dependencies used by the CLI runtime.
type Dependencies struct {
	Version       string
	Stdout        io.Writer
	Stderr        io.Writer
	PluginManager *pluginruntime.Manager
}

// ExitError carries a process exit code without printing another error line.
type ExitError struct {
	Code int
}

func (e ExitError) Error() string {
	return ""
}

// Run dispatches all CLI behavior from raw process arguments.
func Run(args []string, deps Dependencies) error {
	store, err := config.ResolveStore()
	if err != nil {
		return err
	}

	if len(args) == 0 {
		printRootHelp(deps.Stdout)
		return nil
	}
	if isHelpToken(args[0]) {
		printRootHelp(deps.Stdout)
		return nil
	}

	switch args[0] {
	case "platform":
		return runPlatform(args[1:], store, deps.Stdout)
	case "account":
		var manager *pluginruntime.Manager
		if deps.PluginManager != nil {
			manager = deps.PluginManager
		} else {
			manager, _ = resolvePluginManager(deps)
		}
		return runAccount(args[1:], store, deps.Stdout, manager)
	case "subject":
		return runSubject(args[1:], store, deps.Stdout)
	case "version":
		return runVersion(deps.Version, deps.Stdout)
	case "doctor":
		var manager *pluginruntime.Manager
		if deps.PluginManager != nil {
			manager = deps.PluginManager
		} else {
			manager, _ = resolvePluginManager(deps)
		}
		return runDoctor(store, deps.Stdout, manager)
	case "plugin":
		return runPlugin(args[1:], deps.Stdout, deps.Version)
	case "spec":
		manager, err := resolvePluginManager(deps)
		if err != nil {
			return err
		}
		metadataService := metadata.NewServiceWithCatalog(manager.Registry(), manager.CatalogEntries())
		return runSpec(args[1:], deps.Stdout, metadataService.Spec())
	case "docs":
		manager, err := resolvePluginManager(deps)
		if err != nil {
			return err
		}
		metadataService := metadata.NewServiceWithCatalog(manager.Registry(), manager.CatalogEntries())
		return runDocs(args[1:], deps.Stdout, metadataService.Spec())
	case "auth":
		var manager *pluginruntime.Manager
		if deps.PluginManager != nil {
			manager = deps.PluginManager
		} else {
			manager, _ = resolvePluginManager(deps)
		}
		return runAuth(args[1:], store, deps.Stdout, manager)
	case "secret":
		cfg, err := store.Load()
		if err != nil {
			return err
		}
		return runAuthSecret(args[1:], cfg, store, deps.Stdout)
	case "config":
		var manager *pluginruntime.Manager
		if deps.PluginManager != nil {
			manager = deps.PluginManager
		} else {
			manager, _ = resolvePluginManager(deps)
		}
		return runConfig(args[1:], store, deps.Stdout, manager)
	case "completion":
		manager, err := resolvePluginManager(deps)
		if err != nil {
			return err
		}
		metadataService := metadata.NewServiceWithCatalog(manager.Registry(), manager.CatalogEntries())
		return runCompletion(args[1:], deps.Stdout, metadataService.Spec())
	case "batch":
		manager, err := resolvePluginManager(deps)
		if err != nil {
			return err
		}
		executor := runtime.NewExecutorWithManager(store, manager)
		return runBatch(args[1:], deps.Stdout, deps.Stderr, executor)
	default:
		manager, err := resolvePluginManager(deps)
		if err != nil {
			return err
		}
		executor := runtime.NewExecutorWithManager(store, manager)
		return runOperation(args, deps.Stdout, deps.Stderr, executor)
	}
}

func resolvePluginManager(deps Dependencies) (*pluginruntime.Manager, error) {
	if deps.PluginManager != nil {
		return deps.PluginManager, nil
	}
	return newDefaultPluginManager()
}

func newDefaultPluginManager() (*pluginruntime.Manager, error) {
	return pluginruntime.NewDiscoveredManager(context.Background())
}

func runOperation(args []string, stdout io.Writer, stderr io.Writer, executor *runtime.Executor) error {
	flags := pflag.NewFlagSet("clawrise", pflag.ContinueOnError)
	flags.SetInterspersed(true)
	flags.SetOutput(stderr)

	var accountName string
	var subjectName string
	var inputJSON string
	var inputFile string
	var timeout time.Duration
	var dryRun bool
	var idempotencyKey string
	var outputFormat string
	var quiet bool

	flags.StringVar(&accountName, "account", "", "select the account for this execution")
	flags.StringVar(&subjectName, "subject", "", "select the execution subject for this execution")
	flags.StringVar(&inputJSON, "json", "", "pass inline JSON input")
	flags.StringVar(&inputFile, "input", "", "read JSON input from a file")
	flags.DurationVar(&timeout, "timeout", 0, "override the timeout for this call")
	flags.BoolVar(&dryRun, "dry-run", false, "validate and resolve locally without executing the adapter")
	flags.StringVar(&idempotencyKey, "idempotency-key", "", "set an explicit idempotency key")
	flags.StringVar(&outputFormat, "output", "json", "set the output format, currently only json")
	flags.BoolVar(&quiet, "quiet", false, "print only data on successful execution")

	if err := flags.Parse(args); err != nil {
		if err == pflag.ErrHelp {
			return nil
		}
		return err
	}

	positionals := flags.Args()
	if len(positionals) == 0 {
		printRootHelp(stdout)
		return nil
	}
	if len(positionals) != 1 {
		return fmt.Errorf("expected exactly one operation, got %d", len(positionals))
	}
	if outputFormat != "json" {
		return fmt.Errorf("only --output json is supported right now")
	}

	envelope, err := executor.ExecuteContext(context.Background(), runtime.ExecuteOptions{
		OperationInput: positionals[0],
		AccountName:    accountName,
		SubjectName:    subjectName,
		InputJSON:      inputJSON,
		InputFile:      inputFile,
		Timeout:        timeout,
		DryRun:         dryRun,
		IdempotencyKey: idempotencyKey,
		Output:         outputFormat,
		Quiet:          quiet,
		Stdin:          readPipedInput(),
	})
	if err != nil {
		return err
	}

	if quiet && envelope.OK {
		if err := output.WriteJSON(stdout, envelope.Data); err != nil {
			return err
		}
	} else {
		if err := output.WriteJSON(stdout, envelope); err != nil {
			return err
		}
	}

	if !envelope.OK {
		return ExitError{Code: 1}
	}
	return nil
}

func runSubject(args []string, store *config.Store, stdout io.Writer) error {
	if len(args) == 0 || isHelpToken(args[0]) {
		printSubjectHelp(stdout)
		return nil
	}

	cfg, err := store.Load()
	if err != nil {
		return err
	}

	switch args[0] {
	case "use":
		if len(args) != 2 {
			return fmt.Errorf("usage: clawrise subject use <subject>")
		}
		subject := strings.TrimSpace(args[1])
		if !isSupportedSubject(subject) {
			return fmt.Errorf("unsupported subject: %s", subject)
		}

		clearedAccount := ""
		if cfg.Defaults.Account != "" {
			if account, ok := cfg.Accounts[cfg.Defaults.Account]; ok && account.Subject != subject {
				clearedAccount = cfg.Defaults.Account
				cfg.Defaults.Account = ""
			}
		}

		cfg.Defaults.Subject = subject
		if err := store.Save(cfg); err != nil {
			return err
		}

		result := map[string]any{
			"ok":      true,
			"subject": subject,
		}
		if clearedAccount != "" {
			result["cleared_account"] = clearedAccount
		}
		return output.WriteJSON(stdout, result)
	case "current":
		var subject any
		if cfg.Defaults.Subject != "" {
			subject = cfg.Defaults.Subject
		}
		return output.WriteJSON(stdout, map[string]any{
			"subject": subject,
		})
	case "unset":
		cfg.Defaults.Subject = ""
		if err := store.Save(cfg); err != nil {
			return err
		}
		return output.WriteJSON(stdout, map[string]any{
			"ok":      true,
			"message": "default subject has been cleared",
		})
	case "list":
		subjects := make([]string, 0)
		seen := map[string]struct{}{}
		appendSubject := func(value string) {
			value = strings.TrimSpace(value)
			if value == "" {
				return
			}
			if _, ok := seen[value]; ok {
				return
			}
			seen[value] = struct{}{}
			subjects = append(subjects, value)
		}
		appendSubject(cfg.Defaults.Subject)
		for _, account := range cfg.Accounts {
			appendSubject(account.Subject)
		}
		sort.Strings(subjects)
		return output.WriteJSON(stdout, map[string]any{
			"subjects": subjects,
		})
	default:
		return fmt.Errorf("unknown subject command: %s", args[0])
	}
}

func runPlatform(args []string, store *config.Store, stdout io.Writer) error {
	if len(args) == 0 || isHelpToken(args[0]) {
		printPlatformHelp(stdout)
		return nil
	}

	cfg, err := store.Load()
	if err != nil {
		return err
	}

	switch args[0] {
	case "use":
		if len(args) != 2 {
			return fmt.Errorf("usage: clawrise platform use <platform>")
		}
		cfg.Defaults.Platform = strings.TrimSpace(args[1])
		if err := store.Save(cfg); err != nil {
			return err
		}
		return output.WriteJSON(stdout, map[string]any{
			"ok":       true,
			"platform": cfg.Defaults.Platform,
		})
	case "current":
		var platform any
		if cfg.Defaults.Platform != "" {
			platform = cfg.Defaults.Platform
		}
		return output.WriteJSON(stdout, map[string]any{
			"platform": platform,
		})
	case "unset":
		cfg.Defaults.Platform = ""
		if err := store.Save(cfg); err != nil {
			return err
		}
		return output.WriteJSON(stdout, map[string]any{
			"ok":      true,
			"message": "default platform has been cleared",
		})
	default:
		return fmt.Errorf("unknown platform command: %s", args[0])
	}
}

func runVersion(version string, stdout io.Writer) error {
	return output.WriteJSON(stdout, map[string]any{
		"version": version,
	})
}

func runDoctor(store *config.Store, stdout io.Writer, manager *pluginruntime.Manager) error {
	cfg, err := store.Load()
	if err != nil {
		return err
	}

	discovery, err := pluginruntime.InspectDiscovery(context.Background())
	if err != nil {
		return err
	}

	cfg.Ensure()
	checks := make([]map[string]any, 0)
	nextSteps := make([]string, 0)

	if len(discovery.Plugins) == 0 {
		checks = append(checks, map[string]any{
			"code":    "NO_DISCOVERED_PLUGINS",
			"status":  "warn",
			"message": "no discoverable plugins were found in the active plugin roots",
		})
		nextSteps = append(nextSteps, "run `clawrise plugin install <source>` to install at least one provider plugin")
	}

	defaultAccountOK := true
	defaultAccountName := strings.TrimSpace(cfg.Defaults.Account)
	if defaultAccountName == "" {
		defaultAccountOK = false
		checks = append(checks, map[string]any{
			"code":    "DEFAULT_ACCOUNT_MISSING",
			"status":  "warn",
			"message": "no default account is configured",
		})
		nextSteps = append(nextSteps, "run `clawrise account add ...` and `clawrise account use <name>` to set a default account")
	} else if _, ok := cfg.Accounts[defaultAccountName]; !ok {
		defaultAccountOK = false
		checks = append(checks, map[string]any{
			"code":    "DEFAULT_ACCOUNT_NOT_FOUND",
			"status":  "error",
			"message": fmt.Sprintf("default account %s does not exist in config", defaultAccountName),
		})
		nextSteps = append(nextSteps, "update the config file or run `clawrise account use <name>` to select an existing account")
	}

	accountNames := make([]string, 0, len(cfg.Accounts))
	for name := range cfg.Accounts {
		accountNames = append(accountNames, name)
	}
	sort.Strings(accountNames)

	accountInspections := make([]map[string]any, 0, len(accountNames))
	invalidAccounts := 0
	pendingAuthAccounts := 0
	readyAccounts := 0
	for _, name := range accountNames {
		account := cfg.Accounts[name]
		if manager == nil {
			return writeCLIError(stdout, "PLUGIN_MANAGER_REQUIRED", "plugin manager is required for doctor")
		}
		authAccount, err := buildPluginAuthAccount(cfg, store, name, account)
		if err != nil {
			return err
		}
		inspection, inspectErr := manager.InspectAuth(context.Background(), account.Platform, pluginruntime.AuthInspectParams{
			Account: authAccount,
		})
		if inspectErr != nil {
			return inspectErr
		}
		item := buildAccountInspectionView(name, account, inspection)
		if !inspection.Ready {
			if inspection.Status == "invalid_auth_config" {
				invalidAccounts++
			} else {
				pendingAuthAccounts++
			}
		} else {
			readyAccounts++
		}
		accountInspections = append(accountInspections, item)
	}

	if invalidAccounts > 0 {
		checks = append(checks, map[string]any{
			"code":    "INVALID_ACCOUNTS",
			"status":  "warn",
			"message": fmt.Sprintf("%d configured accounts have invalid or unresolved auth fields", invalidAccounts),
		})
		nextSteps = append(nextSteps, "run `clawrise auth inspect <account>` to inspect invalid account details")
	}
	if pendingAuthAccounts > 0 {
		checks = append(checks, map[string]any{
			"code":    "AUTHORIZATION_PENDING",
			"status":  "warn",
			"message": fmt.Sprintf("%d interactive auth accounts still need user authorization before they can execute", pendingAuthAccounts),
		})
		nextSteps = append(nextSteps, "run `clawrise auth login <account>` to finish interactive authorization")
	}

	configResolution, configResolutionErr := locator.ResolveConfigPathResolution()
	stateResolution, stateResolutionErr := locator.ResolveStateDirResolution(store.Path())
	runtimeResolution, runtimeResolutionErr := locator.ResolveRuntimeDirResolution(store.Path())

	pathsSummary := map[string]any{
		"config": map[string]any{
			"path": store.Path(),
		},
	}
	if configResolutionErr == nil {
		pathsSummary["config"] = map[string]any{
			"path":   configResolution.Path,
			"source": configResolution.Source,
		}
	}
	if stateResolutionErr == nil {
		pathsSummary["state"] = map[string]any{
			"path":   stateResolution.Path,
			"source": stateResolution.Source,
		}
		pathsSummary["sessions"] = map[string]any{
			"path": filepath.Join(stateResolution.Path, "auth", "sessions"),
		}
		pathsSummary["auth_flows"] = map[string]any{
			"path": filepath.Join(stateResolution.Path, "auth", "flows"),
		}
	}
	if runtimeResolutionErr == nil {
		pathsSummary["runtime"] = map[string]any{
			"path":   runtimeResolution.Path,
			"source": runtimeResolution.Source,
		}
	}

	runtimeRootDir := filepath.Join(filepath.Dir(store.Path()), "runtime")
	if runtimeResolutionErr == nil {
		runtimeRootDir = runtimeResolution.Path
	}
	runtimeSummary := map[string]any{
		"registered_operation_count": 0,
		"catalog_entry_count":        0,
		"storage": map[string]any{
			"root_dir":        runtimeRootDir,
			"idempotency_dir": filepath.Join(runtimeRootDir, "idempotency"),
			"audit_dir":       filepath.Join(runtimeRootDir, "audit"),
		},
		"retry_policy": map[string]any{
			"max_attempts":  cfg.Runtime.Retry.MaxAttempts,
			"base_delay_ms": cfg.Runtime.Retry.BaseDelayMS,
			"max_delay_ms":  cfg.Runtime.Retry.MaxDelayMS,
		},
	}
	if manager != nil {
		runtimeSummary["registered_operation_count"] = len(manager.Registry().Definitions())
		runtimeSummary["catalog_entry_count"] = len(manager.CatalogEntries())
	}

	playbookValidation := map[string]any{
		"path": metadata.DefaultPlaybookIndexPath(),
	}
	if manager != nil {
		metadataService := metadata.NewServiceWithCatalog(manager.Registry(), manager.CatalogEntries())
		validation, err := metadataService.ValidatePlaybooks()
		if err != nil {
			return err
		}
		playbookValidation = map[string]any{
			"path":          validation.Path,
			"ok":            validation.OK,
			"total":         validation.Total,
			"valid_count":   validation.ValidCount,
			"invalid_count": validation.InvalidCount,
			"missing_file":  validation.MissingFile,
			"issues":        validation.Issues,
		}
		if validation.InvalidCount > 0 {
			checks = append(checks, map[string]any{
				"code":    "PLAYBOOK_INDEX_INVALID",
				"status":  "warn",
				"message": fmt.Sprintf("%d playbooks reference missing docs or unknown operations", validation.InvalidCount),
			})
			nextSteps = append(nextSteps, "fix docs/playbooks/index.yaml so each playbook points to existing docs and registered operations")
		}
	}

	return output.WriteJSON(stdout, map[string]any{
		"ok": true,
		"data": map[string]any{
			"config_path": store.Path(),
			"defaults": map[string]any{
				"platform":           cfg.Defaults.Platform,
				"subject":            cfg.Defaults.Subject,
				"account":            defaultAccountName,
				"platform_accounts":  cfg.Defaults.PlatformAccounts,
				"default_account_ok": defaultAccountOK,
			},
			"accounts": map[string]any{
				"count":              len(accountInspections),
				"ready_count":        readyAccounts,
				"invalid_count":      invalidAccounts,
				"pending_auth_count": pendingAuthAccounts,
				"items":              accountInspections,
			},
			"plugins":   discovery,
			"runtime":   runtimeSummary,
			"paths":     pathsSummary,
			"playbooks": playbookValidation,
			"environment": map[string]any{
				"go_version": runtimeVersion(),
				"os":         runtimeOS(),
				"arch":       runtimeArch(),
			},
			"checks":     checks,
			"next_steps": nextSteps,
		},
	})
}

func printRootHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, "Clawrise is an agent-native CLI execution layer.")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Usage:")
	_, _ = fmt.Fprintln(w, "  clawrise <operation> [flags]")
	_, _ = fmt.Fprintln(w, "  clawrise platform [use|current|unset]")
	_, _ = fmt.Fprintln(w, "  clawrise account [list|inspect|use|current|add|remove]")
	_, _ = fmt.Fprintln(w, "  clawrise subject [use|current|unset|list]")
	_, _ = fmt.Fprintln(w, "  clawrise auth [list|methods|presets|inspect|check|login|complete|logout|secret]")
	_, _ = fmt.Fprintln(w, "  clawrise secret [put|delete]")
	_, _ = fmt.Fprintln(w, "  clawrise config init")
	_, _ = fmt.Fprintln(w, "  clawrise plugin [list|install|info|remove|verify]")
	_, _ = fmt.Fprintln(w, "  clawrise spec [list|get|status|export]")
	_, _ = fmt.Fprintln(w, "  clawrise docs generate [path] [--out-dir <dir>]")
	_, _ = fmt.Fprintln(w, "  clawrise batch [--json <payload> | --input <path>]")
	_, _ = fmt.Fprintln(w, "  clawrise completion [bash|zsh|fish]")
	_, _ = fmt.Fprintln(w, "  clawrise doctor")
	_, _ = fmt.Fprintln(w, "  clawrise version")
}

func printPlatformHelp(stdout io.Writer) {
	_, _ = fmt.Fprintln(stdout, "Usage: clawrise platform [use|current|unset]")
}

func printSubjectHelp(stdout io.Writer) {
	_, _ = fmt.Fprintln(stdout, "Usage: clawrise subject [use|current|unset|list]")
}

// isHelpToken keeps help-token detection consistent across subcommands.
func isHelpToken(token string) bool {
	switch strings.TrimSpace(token) {
	case "-h", "--help", "help":
		return true
	default:
		return false
	}
}

func isSupportedSubject(subject string) bool {
	return strings.TrimSpace(subject) != ""
}
