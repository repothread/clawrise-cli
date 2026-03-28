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
	"github.com/clawrise/clawrise-cli/internal/output"
	pluginruntime "github.com/clawrise/clawrise-cli/internal/plugin"
	"github.com/clawrise/clawrise-cli/internal/runtime"
	"github.com/clawrise/clawrise-cli/internal/spec"
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
	case "connection":
		return runConnection(args[1:], store, deps.Stdout)
	case "subject":
		return runSubject(args[1:], store, deps.Stdout)
	case "profile":
		return runConnection(args[1:], store, deps.Stdout)
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
		specService := spec.NewServiceWithCatalog(manager.Registry(), manager.CatalogEntries())
		return runSpec(args[1:], deps.Stdout, specService)
	case "auth":
		var manager *pluginruntime.Manager
		if deps.PluginManager != nil {
			manager = deps.PluginManager
		} else {
			manager, _ = resolvePluginManager(deps)
		}
		return runAuth(args[1:], store, deps.Stdout, manager)
	case "config":
		return runConfig(args[1:], store, deps.Stdout)
	case "completion":
		manager, err := resolvePluginManager(deps)
		if err != nil {
			return err
		}
		specService := spec.NewServiceWithCatalog(manager.Registry(), manager.CatalogEntries())
		return runCompletion(args[1:], deps.Stdout, specService)
	case "batch":
		return runPlaceholder(args[0], deps.Stdout)
	default:
		manager, err := resolvePluginManager(deps)
		if err != nil {
			return err
		}
		executor := runtime.NewExecutor(store, manager.Registry())
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

	var profileName string
	var connectionName string
	var subjectName string
	var inputJSON string
	var inputFile string
	var timeout time.Duration
	var dryRun bool
	var idempotencyKey string
	var outputFormat string
	var quiet bool

	flags.StringVar(&connectionName, "connection", "", "select the connection for this execution")
	flags.StringVar(&profileName, "profile", "", "select the profile for this execution")
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
		ConnectionName: connectionName,
		ProfileName:    profileName,
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

		clearedProfile := ""
		if cfg.Defaults.Profile != "" {
			if profile, ok := cfg.Profiles[cfg.Defaults.Profile]; ok && profile.Subject != subject {
				clearedProfile = cfg.Defaults.Profile
				cfg.Defaults.Profile = ""
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
		if clearedProfile != "" {
			result["cleared_profile"] = clearedProfile
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
		return output.WriteJSON(stdout, map[string]any{
			"subjects": []string{"bot", "user", "integration"},
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

func runProfile(args []string, store *config.Store, stdout io.Writer) error {
	if len(args) == 0 || isHelpToken(args[0]) {
		printProfileHelp(stdout)
		return nil
	}

	cfg, err := store.Load()
	if err != nil {
		return err
	}

	switch args[0] {
	case "use":
		if len(args) != 2 {
			return fmt.Errorf("usage: clawrise profile use <profile>")
		}
		profile, ok := cfg.Profiles[args[1]]
		if !ok {
			return output.WriteJSON(stdout, map[string]any{
				"ok": false,
				"error": map[string]any{
					"code":    "PROFILE_NOT_FOUND",
					"message": "the selected profile does not exist",
				},
			})
		}

		cfg.Defaults.Profile = args[1]
		// profile 是唯一能完整表达执行身份的默认值，切换后需要同步平台，
		// 否则裸 operation 会继续沿用旧平台并产生平台漂移。
		cfg.Defaults.Platform = profile.Platform
		cfg.Defaults.Subject = profile.Subject
		if err := store.Save(cfg); err != nil {
			return err
		}
		return output.WriteJSON(stdout, map[string]any{
			"ok":      true,
			"profile": args[1],
			"subject": profile.Subject,
			"platform": map[string]any{
				"name": profile.Platform,
			},
		})
	case "current":
		var profile any
		if cfg.Defaults.Profile != "" {
			profile = cfg.Defaults.Profile
		}
		return output.WriteJSON(stdout, map[string]any{
			"profile": profile,
		})
	case "list":
		names := make([]string, 0, len(cfg.Profiles))
		for name := range cfg.Profiles {
			names = append(names, name)
		}
		sort.Strings(names)

		items := make([]map[string]any, 0, len(cfg.Profiles))
		for _, name := range names {
			profile := cfg.Profiles[name]
			items = append(items, map[string]any{
				"name":     name,
				"platform": profile.Platform,
				"subject":  profile.Subject,
				"grant": map[string]any{
					"type": profile.Grant.Type,
				},
			})
		}
		return output.WriteJSON(stdout, map[string]any{
			"profiles": items,
		})
	default:
		return fmt.Errorf("unknown profile command: %s", args[0])
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

	profileInspections := config.SortedProfileInspections(cfg)
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

	defaultProfileOK := true
	if cfg.Defaults.Profile == "" {
		defaultProfileOK = false
		checks = append(checks, map[string]any{
			"code":    "DEFAULT_PROFILE_MISSING",
			"status":  "warn",
			"message": "no default profile is configured",
		})
		nextSteps = append(nextSteps, "run `clawrise config init` or `clawrise profile use <name>` to set a default profile")
	} else if _, ok := cfg.Profiles[cfg.Defaults.Profile]; !ok {
		defaultProfileOK = false
		checks = append(checks, map[string]any{
			"code":    "DEFAULT_PROFILE_NOT_FOUND",
			"status":  "error",
			"message": fmt.Sprintf("default profile %s does not exist in config", cfg.Defaults.Profile),
		})
		nextSteps = append(nextSteps, "update the config file or run `clawrise profile use <name>` to select an existing profile")
	}

	invalidProfiles := 0
	for _, inspection := range profileInspections {
		if inspection.ShapeValid && inspection.ResolvedValid {
			continue
		}
		invalidProfiles++
	}
	if invalidProfiles > 0 {
		checks = append(checks, map[string]any{
			"code":    "INVALID_PROFILES",
			"status":  "warn",
			"message": fmt.Sprintf("%d configured profiles have invalid or unresolved auth fields", invalidProfiles),
		})
		nextSteps = append(nextSteps, "run `clawrise auth check <profile>` to inspect invalid profile details")
	}

	runtimeSummary := map[string]any{
		"registered_operation_count": 0,
		"catalog_entry_count":        0,
		"storage": map[string]any{
			"root_dir":        filepath.Join(filepath.Dir(store.Path()), "runtime"),
			"idempotency_dir": filepath.Join(filepath.Dir(store.Path()), "runtime", "idempotency"),
			"audit_dir":       filepath.Join(filepath.Dir(store.Path()), "runtime", "audit"),
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

	return output.WriteJSON(stdout, map[string]any{
		"ok": true,
		"data": map[string]any{
			"config_path": store.Path(),
			"defaults": map[string]any{
				"platform":           cfg.Defaults.Platform,
				"subject":            cfg.Defaults.Subject,
				"profile":            cfg.Defaults.Profile,
				"default_profile_ok": defaultProfileOK,
			},
			"profiles": map[string]any{
				"count": len(profileInspections),
				"items": profileInspections,
			},
			"plugins": discovery,
			"runtime": runtimeSummary,
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

func runPlaceholder(name string, stdout io.Writer) error {
	return output.WriteJSON(stdout, map[string]any{
		"ok": false,
		"error": map[string]any{
			"code":    "NOT_IMPLEMENTED",
			"message": fmt.Sprintf("%s is reserved for future implementation", name),
		},
	})
}

func printRootHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, "Clawrise is an agent-native CLI execution layer.")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Usage:")
	_, _ = fmt.Fprintln(w, "  clawrise <operation> [flags]")
	_, _ = fmt.Fprintln(w, "  clawrise platform [use|current|unset]")
	_, _ = fmt.Fprintln(w, "  clawrise connection [use|current|list]")
	_, _ = fmt.Fprintln(w, "  clawrise subject [use|current|unset|list]")
	_, _ = fmt.Fprintln(w, "  clawrise auth [list|inspect|check|begin|connect|status|continue|session|secret]")
	_, _ = fmt.Fprintln(w, "  clawrise config init")
	_, _ = fmt.Fprintln(w, "  clawrise plugin [list|install|info|remove|verify]")
	_, _ = fmt.Fprintln(w, "  clawrise spec [list|get|status|export]")
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

func printProfileHelp(stdout io.Writer) {
	_, _ = fmt.Fprintln(stdout, "Usage: clawrise profile [use|current|list]")
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
	switch subject {
	case "bot", "user", "integration":
		return true
	default:
		return false
	}
}
