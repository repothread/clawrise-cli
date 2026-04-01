package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/clawrise/clawrise-cli/internal/config"
	"github.com/clawrise/clawrise-cli/internal/output"
	pluginruntime "github.com/clawrise/clawrise-cli/internal/plugin"
)

func runPlugin(args []string, store *config.Store, stdout io.Writer, coreVersion string) error {
	if len(args) == 0 || isHelpToken(args[0]) {
		printPluginHelp(stdout)
		return nil
	}

	cfg := config.New()
	if store != nil {
		loaded, err := store.Load()
		if err != nil {
			return err
		}
		cfg = loaded
	}
	discoveryOptions := buildPluginDiscoveryOptions(cfg)
	installOptions := buildPluginInstallOptions(cfg, coreVersion)

	switch args[0] {
	case "list":
		if len(args) != 1 {
			return fmt.Errorf("usage: clawrise plugin list")
		}
		items, err := pluginruntime.ListInstalledWithOptions(discoveryOptions)
		if err != nil {
			return err
		}
		return output.WriteJSON(stdout, map[string]any{
			"plugins": items,
		})
	case "install":
		if len(args) != 2 {
			return fmt.Errorf("usage: clawrise plugin install <package-or-source>")
		}
		result, err := pluginruntime.InstallWithOptions(strings.TrimSpace(args[1]), installOptions)
		if err != nil {
			return err
		}
		return output.WriteJSON(stdout, map[string]any{
			"ok":   true,
			"data": result,
		})
	case "info":
		if len(args) != 3 {
			return fmt.Errorf("usage: clawrise plugin info <name> <version>")
		}
		result, err := pluginruntime.InfoInstalledWithOptions(args[1], args[2], discoveryOptions)
		if err != nil {
			return err
		}
		return output.WriteJSON(stdout, map[string]any{
			"ok":   true,
			"data": result,
		})
	case "remove":
		if len(args) != 3 {
			return fmt.Errorf("usage: clawrise plugin remove <name> <version>")
		}
		result, err := pluginruntime.RemoveInstalled(args[1], args[2])
		if err != nil {
			return err
		}
		return output.WriteJSON(stdout, map[string]any{
			"ok":   true,
			"data": result,
		})
	case "verify":
		if len(args) == 2 && strings.TrimSpace(args[1]) == "--all" {
			results, err := pluginruntime.VerifyAllInstalled(coreVersion)
			if err != nil {
				return err
			}
			allVerified := true
			for _, item := range results {
				if !item.Verified {
					allVerified = false
					break
				}
			}
			if err := output.WriteJSON(stdout, map[string]any{
				"ok": allVerified,
				"data": map[string]any{
					"all_verified": allVerified,
					"plugins":      results,
				},
			}); err != nil {
				return err
			}
			if !allVerified {
				return ExitError{Code: 1}
			}
			return nil
		}
		if len(args) != 3 {
			return fmt.Errorf("usage: clawrise plugin verify <name> <version> | clawrise plugin verify --all")
		}
		result, err := pluginruntime.VerifyInstalled(args[1], args[2], coreVersion)
		if err != nil {
			return err
		}
		if err := output.WriteJSON(stdout, map[string]any{
			"ok":   result.Verified,
			"data": result,
		}); err != nil {
			return err
		}
		if !result.Verified {
			return ExitError{Code: 1}
		}
		return nil
	case "upgrade":
		if len(args) != 3 {
			return fmt.Errorf("usage: clawrise plugin upgrade <name> <version>")
		}
		result, err := pluginruntime.UpgradeInstalled(args[1], args[2], installOptions)
		if err != nil {
			return err
		}
		return output.WriteJSON(stdout, map[string]any{
			"ok":   true,
			"data": result,
		})
	default:
		return fmt.Errorf("unknown plugin command: %s", args[0])
	}
}

func buildPluginInstallOptions(cfg *config.Config, coreVersion string) pluginruntime.InstallOptions {
	return pluginruntime.InstallOptions{
		CoreVersion:    strings.TrimSpace(coreVersion),
		AllowedSources: config.ResolvePluginInstallAllowedSources(cfg),
	}
}

func printPluginHelp(stdout io.Writer) {
	_, _ = fmt.Fprintln(stdout, "Usage: clawrise plugin [list|install|info|remove|verify|upgrade]")
	_, _ = fmt.Fprintln(stdout, "")
	_, _ = fmt.Fprintln(stdout, "Examples:")
	_, _ = fmt.Fprintln(stdout, "  clawrise plugin list")
	_, _ = fmt.Fprintln(stdout, "  clawrise plugin info demo 0.1.0")
	_, _ = fmt.Fprintln(stdout, "  clawrise plugin install @clawrise/clawrise-plugin-feishu")
	_, _ = fmt.Fprintln(stdout, "  clawrise plugin install file:///tmp/demo-plugin.tar.gz")
	_, _ = fmt.Fprintln(stdout, "  clawrise plugin install https://example.com/demo-plugin.tar.gz")
	_, _ = fmt.Fprintln(stdout, "  clawrise plugin install npm://@clawrise/clawrise-plugin-feishu")
	_, _ = fmt.Fprintln(stdout, "  clawrise plugin verify demo 0.1.0")
	_, _ = fmt.Fprintln(stdout, "  clawrise plugin verify --all")
	_, _ = fmt.Fprintln(stdout, "  clawrise plugin upgrade demo 0.1.0")
	_, _ = fmt.Fprintln(stdout, "  clawrise plugin remove demo 0.1.0")
}
