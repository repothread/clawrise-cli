package cli

import (
	"fmt"
	"io"

	"github.com/clawrise/clawrise-cli/internal/config"
	"github.com/clawrise/clawrise-cli/internal/output"
	pluginruntime "github.com/clawrise/clawrise-cli/internal/plugin"
)

// runAuth handles auth-related commands.
func runAuth(args []string, store *config.Store, stdout io.Writer, manager *pluginruntime.Manager) error {
	if len(args) == 0 || isHelpToken(args[0]) {
		printAuthHelp(stdout)
		return nil
	}

	cfg, err := store.Load()
	if err != nil {
		return err
	}

	switch args[0] {
	case "methods":
		return runAuthMethods(args[1:], stdout, manager)
	case "presets":
		return runAuthPresets(args[1:], stdout, manager)
	case "login":
		return runAuthLogin(args[1:], cfg, store, stdout, manager)
	case "complete":
		return runAuthCompleteV2(args[1:], cfg, store, stdout, manager)
	case "logout":
		return runAuthLogout(args[1:], cfg, store, stdout)
	case "list":
		if len(args) != 1 {
			return fmt.Errorf("usage: clawrise auth list")
		}
		cfg.Ensure()
		items := make([]map[string]any, 0, len(cfg.Accounts))
		for name, account := range cfg.Accounts {
			items = append(items, map[string]any{
				"name":        name,
				"platform":    account.Platform,
				"subject":     account.Subject,
				"auth_method": account.Auth.Method,
			})
		}
		return output.WriteJSON(stdout, map[string]any{
			"ok": true,
			"data": map[string]any{
				"accounts": items,
			},
		})
	case "inspect":
		return runAuthInspectV2(args[1:], cfg, store, stdout, manager)
	case "check":
		return runAuthCheckV2(args[1:], cfg, store, stdout, manager)
	case "secret":
		return runAuthSecret(args[1:], cfg, store, stdout)
	default:
		return fmt.Errorf("unknown auth command: %s", args[0])
	}
}

func buildAuthOperationSummary(profile config.Profile, manager *pluginruntime.Manager) map[string]any {
	summary := map[string]any{
		"platform": profile.Platform,
		"subject":  profile.Subject,
	}
	if manager == nil {
		return summary
	}

	definitions := manager.Registry().Definitions()
	platformCount := 0
	allowedCount := 0
	for _, definition := range definitions {
		if definition.Platform != profile.Platform {
			continue
		}
		platformCount++
		if stringSliceContains(definition.AllowedSubjects, profile.Subject) {
			allowedCount++
		}
	}
	summary["platform_operation_count"] = platformCount
	summary["subject_allowed_operation_count"] = allowedCount
	return summary
}

func stringSliceContains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func writeCLIError(stdout io.Writer, code string, message string) error {
	if err := output.WriteJSON(stdout, map[string]any{
		"ok": false,
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	}); err != nil {
		return err
	}
	return ExitError{Code: 1}
}

func printAuthHelp(stdout io.Writer) {
	_, _ = fmt.Fprintln(stdout, "Usage: clawrise auth [list|methods|presets|inspect|check|login|complete|logout|secret]")
	_, _ = fmt.Fprintln(stdout, "       clawrise auth methods [--platform <name>]")
	_, _ = fmt.Fprintln(stdout, "       clawrise auth presets [--platform <name>]")
	_, _ = fmt.Fprintln(stdout, "       clawrise auth inspect [account]")
	_, _ = fmt.Fprintln(stdout, "       clawrise auth check [account]")
	_, _ = fmt.Fprintln(stdout, "       clawrise auth login [account] [--mode <name>] [--redirect-uri <uri>] [--open-browser=true|false]")
	_, _ = fmt.Fprintln(stdout, "         --open-browser controls whether Clawrise should invoke an auth launcher automatically")
	_, _ = fmt.Fprintln(stdout, "       clawrise auth complete <flow_id> [--callback-url <url> | --code <text>]")
	_, _ = fmt.Fprintln(stdout, "       clawrise auth logout [account]")
	_, _ = fmt.Fprintln(stdout, "       clawrise auth secret [set|put|delete] <account> <field>")
}
