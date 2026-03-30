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
		return runAuthComplete(args[1:], cfg, store, stdout, manager)
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
		return runAuthInspect(args[1:], cfg, store, stdout, manager)
	case "check":
		return runAuthCheck(args[1:], cfg, store, stdout, manager)
	case "secret":
		return runAuthSecret(args[1:], cfg, store, stdout)
	default:
		return fmt.Errorf("unknown auth command: %s", args[0])
	}
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

func buildAccountInspectionView(accountName string, account config.Account, result pluginruntime.AuthInspectResult) map[string]any {
	return map[string]any{
		"name":                  accountName,
		"title":                 account.Title,
		"platform":              account.Platform,
		"subject":               account.Subject,
		"auth_method":           account.Auth.Method,
		"ready":                 result.Ready,
		"status":                result.Status,
		"message":               result.Message,
		"missing_public_fields": result.MissingPublicFields,
		"missing_secret_fields": result.MissingSecretFields,
		"session_status":        result.SessionStatus,
		"human_required":        result.HumanRequired,
		"recommended_action":    result.RecommendedAction,
		"next_actions":          result.NextActions,
	}
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
