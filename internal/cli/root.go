package cli

import (
	"context"
	"fmt"
	"io"
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
	case "subject":
		return runSubject(args[1:], store, deps.Stdout)
	case "profile":
		return runProfile(args[1:], store, deps.Stdout)
	case "version":
		return runVersion(deps.Version, deps.Stdout)
	case "doctor":
		return runDoctor(store, deps.Stdout)
	case "plugin":
		return runPlugin(args[1:], deps.Stdout)
	case "spec":
		manager, err := resolvePluginManager(deps)
		if err != nil {
			return err
		}
		specService := spec.NewServiceWithCatalog(manager.Registry(), manager.CatalogEntries())
		return runSpec(args[1:], deps.Stdout, specService)
	case "auth", "config", "batch", "completion":
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
	var subjectName string
	var inputJSON string
	var inputFile string
	var timeout time.Duration
	var dryRun bool
	var idempotencyKey string
	var outputFormat string
	var quiet bool

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
		if cfg.Defaults.Platform == "" {
			cfg.Defaults.Platform = profile.Platform
		}
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

func runDoctor(store *config.Store, stdout io.Writer) error {
	cfg, err := store.Load()
	if err != nil {
		return err
	}

	return output.WriteJSON(stdout, map[string]any{
		"ok":               true,
		"config_path":      store.Path(),
		"default_platform": cfg.Defaults.Platform,
		"default_subject":  cfg.Defaults.Subject,
		"default_profile":  cfg.Defaults.Profile,
		"profile_count":    len(cfg.Profiles),
		"go_version":       runtimeVersion(),
		"os":               runtimeOS(),
		"arch":             runtimeArch(),
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
	_, _ = fmt.Fprintln(w, "  clawrise subject [use|current|unset|list]")
	_, _ = fmt.Fprintln(w, "  clawrise profile [use|current|list]")
	_, _ = fmt.Fprintln(w, "  clawrise plugin [list|install|info|remove]")
	_, _ = fmt.Fprintln(w, "  clawrise spec [list|get|status|export]")
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
