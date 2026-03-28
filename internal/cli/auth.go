package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/clawrise/clawrise-cli/internal/config"
	"github.com/clawrise/clawrise-cli/internal/output"
	pluginruntime "github.com/clawrise/clawrise-cli/internal/plugin"
)

// runAuth 处理最小可用的授权检查命令。
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
	case "begin", "status", "continue":
		return runAuthFlow(args, cfg, store, stdout)
	case "list":
		if len(args) != 1 {
			return fmt.Errorf("usage: clawrise auth list")
		}
		return output.WriteJSON(stdout, map[string]any{
			"ok": true,
			"data": map[string]any{
				"connections": config.SortedProfileInspections(cfg),
			},
		})
	case "inspect":
		if len(args) != 2 {
			return fmt.Errorf("usage: clawrise auth inspect <connection>")
		}
		name, profile, ok := lookupConnection(cfg, args[1])
		if !ok {
			return writeCLIError(stdout, "CONNECTION_NOT_FOUND", "the selected connection does not exist")
		}
		return output.WriteJSON(stdout, map[string]any{
			"ok":   true,
			"data": config.InspectProfile(name, profile),
		})
	case "check":
		if len(args) > 2 {
			return fmt.Errorf("usage: clawrise auth check [connection]")
		}

		name := ""
		if len(args) == 2 {
			name = strings.TrimSpace(args[1])
		} else if platform := strings.TrimSpace(cfg.Defaults.Platform); platform != "" {
			name = strings.TrimSpace(cfg.Defaults.Connections[platform])
			if name == "" {
				name = strings.TrimSpace(cfg.Defaults.Profile)
			}
		}
		if strings.TrimSpace(name) == "" {
			return writeCLIError(stdout, "CONNECTION_REQUIRED", "no connection was provided and no default connection is configured")
		}

		resolvedName, profile, ok := lookupConnection(cfg, name)
		if !ok {
			return writeCLIError(stdout, "CONNECTION_NOT_FOUND", "the selected connection does not exist")
		}

		inspection := config.InspectProfile(resolvedName, profile)
		summary := buildAuthOperationSummary(profile, manager)
		valid := inspection.ShapeValid && inspection.ResolvedValid

		if err := output.WriteJSON(stdout, map[string]any{
			"ok": valid,
			"data": map[string]any{
				"profile":    inspection,
				"operations": summary,
			},
		}); err != nil {
			return err
		}
		if !valid {
			return ExitError{Code: 1}
		}
		return nil
	case "session":
		return runAuthSession(args[1:], cfg, store, stdout)
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

func lookupConnection(cfg *config.Config, name string) (string, config.Profile, bool) {
	cfg.Ensure()
	profile, ok := cfg.Connections[strings.TrimSpace(name)]
	return strings.TrimSpace(name), profile, ok
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
	_, _ = fmt.Fprintln(stdout, "Usage: clawrise auth [list|inspect|check|begin|status|continue|session|secret]")
	_, _ = fmt.Fprintln(stdout, "       clawrise auth begin [connection] [--mode <name>] [--redirect-uri <uri>]")
	_, _ = fmt.Fprintln(stdout, "       clawrise auth status <flow_id>")
	_, _ = fmt.Fprintln(stdout, "       clawrise auth continue <flow_id> [--callback-url <url> | --code <text>]")
	_, _ = fmt.Fprintln(stdout, "       clawrise auth session [inspect|clear|refresh] [connection]")
	_, _ = fmt.Fprintln(stdout, "       clawrise auth secret [set|delete] <connection> <field>")
}
