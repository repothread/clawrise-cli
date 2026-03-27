package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/clawrise/clawrise-cli/internal/output"
	pluginruntime "github.com/clawrise/clawrise-cli/internal/plugin"
)

func runPlugin(args []string, stdout io.Writer) error {
	if len(args) == 0 || isHelpToken(args[0]) {
		printPluginHelp(stdout)
		return nil
	}

	switch args[0] {
	case "list":
		if len(args) != 1 {
			return fmt.Errorf("usage: clawrise plugin list")
		}
		items, err := pluginruntime.ListInstalled()
		if err != nil {
			return err
		}
		return output.WriteJSON(stdout, map[string]any{
			"plugins": items,
		})
	case "install":
		if len(args) != 2 {
			return fmt.Errorf("usage: clawrise plugin install <source>")
		}
		result, err := pluginruntime.Install(strings.TrimSpace(args[1]))
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
		result, err := pluginruntime.InfoInstalled(args[1], args[2])
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
	default:
		return fmt.Errorf("unknown plugin command: %s", args[0])
	}
}

func printPluginHelp(stdout io.Writer) {
	_, _ = fmt.Fprintln(stdout, "Usage: clawrise plugin [list|install|info|remove]")
	_, _ = fmt.Fprintln(stdout, "")
	_, _ = fmt.Fprintln(stdout, "Examples:")
	_, _ = fmt.Fprintln(stdout, "  clawrise plugin list")
	_, _ = fmt.Fprintln(stdout, "  clawrise plugin info demo 0.1.0")
	_, _ = fmt.Fprintln(stdout, "  clawrise plugin install file:///tmp/demo-plugin.tar.gz")
	_, _ = fmt.Fprintln(stdout, "  clawrise plugin install https://example.com/demo-plugin.tar.gz")
	_, _ = fmt.Fprintln(stdout, "  clawrise plugin install npm://@clawrise/plugin-feishu")
	_, _ = fmt.Fprintln(stdout, "  clawrise plugin remove demo 0.1.0")
}
