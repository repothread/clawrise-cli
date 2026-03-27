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
	case "list":
		if len(args) != 1 {
			return fmt.Errorf("usage: clawrise auth list")
		}
		return output.WriteJSON(stdout, map[string]any{
			"ok": true,
			"data": map[string]any{
				"profiles": config.SortedProfileInspections(cfg),
			},
		})
	case "inspect":
		if len(args) != 2 {
			return fmt.Errorf("usage: clawrise auth inspect <profile>")
		}
		name, profile, ok := lookupProfile(cfg, args[1])
		if !ok {
			return writeCLIError(stdout, "PROFILE_NOT_FOUND", "the selected profile does not exist")
		}
		return output.WriteJSON(stdout, map[string]any{
			"ok":   true,
			"data": config.InspectProfile(name, profile),
		})
	case "check":
		if len(args) > 2 {
			return fmt.Errorf("usage: clawrise auth check [profile]")
		}

		name := cfg.Defaults.Profile
		if len(args) == 2 {
			name = strings.TrimSpace(args[1])
		}
		if strings.TrimSpace(name) == "" {
			return writeCLIError(stdout, "PROFILE_REQUIRED", "no profile was provided and no default profile is configured")
		}

		resolvedName, profile, ok := lookupProfile(cfg, name)
		if !ok {
			return writeCLIError(stdout, "PROFILE_NOT_FOUND", "the selected profile does not exist")
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

func lookupProfile(cfg *config.Config, name string) (string, config.Profile, bool) {
	cfg.Ensure()
	profile, ok := cfg.Profiles[strings.TrimSpace(name)]
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
	_, _ = fmt.Fprintln(stdout, "Usage: clawrise auth [list|inspect|check]")
}
