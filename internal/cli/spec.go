package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/clawrise/clawrise-cli/internal/output"
	"github.com/clawrise/clawrise-cli/internal/spec"
)

// runSpec handles `clawrise spec` subcommands.
func runSpec(args []string, stdout io.Writer, service *spec.Service) error {
	if len(args) == 0 || isHelpToken(args[0]) {
		printSpecHelp(stdout)
		return nil
	}

	switch args[0] {
	case "list":
		if len(args) > 2 {
			return fmt.Errorf("usage: clawrise spec list [path]")
		}

		path := ""
		if len(args) == 2 {
			path = strings.TrimSpace(args[1])
		}

		result, err := service.List(path)
		if err != nil {
			return writeSpecError(stdout, err)
		}
		return output.WriteJSON(stdout, map[string]any{
			"ok":   true,
			"data": result,
		})
	case "get":
		if len(args) != 2 {
			return fmt.Errorf("usage: clawrise spec get <operation>")
		}

		result, err := service.Get(args[1])
		if err != nil {
			return writeSpecError(stdout, err)
		}
		return output.WriteJSON(stdout, map[string]any{
			"ok":   true,
			"data": result,
		})
	case "status":
		if len(args) != 1 {
			return fmt.Errorf("usage: clawrise spec status")
		}

		result, err := service.Status()
		if err != nil {
			return writeSpecError(stdout, err)
		}
		return output.WriteJSON(stdout, map[string]any{
			"ok":   true,
			"data": result,
		})
	case "export":
		return output.WriteJSON(stdout, map[string]any{
			"ok": false,
			"error": map[string]any{
				"code":    "NOT_IMPLEMENTED",
				"message": fmt.Sprintf("spec %s is reserved for future implementation", args[0]),
			},
		})
	default:
		return fmt.Errorf("unknown spec command: %s", args[0])
	}
}

func printSpecHelp(stdout io.Writer) {
	_, _ = fmt.Fprintln(stdout, "Usage: clawrise spec [list|get|status|export]")
	_, _ = fmt.Fprintln(stdout, "")
	_, _ = fmt.Fprintln(stdout, "Examples:")
	_, _ = fmt.Fprintln(stdout, "  clawrise spec list")
	_, _ = fmt.Fprintln(stdout, "  clawrise spec list feishu.docs")
	_, _ = fmt.Fprintln(stdout, "  clawrise spec get notion.page.create")
}

func writeSpecError(stdout io.Writer, err error) error {
	specErr, ok := err.(*spec.Error)
	if !ok {
		return err
	}

	if writeErr := output.WriteJSON(stdout, map[string]any{
		"ok": false,
		"error": map[string]any{
			"code":    specErr.Code,
			"message": specErr.Message,
		},
	}); writeErr != nil {
		return writeErr
	}
	return ExitError{Code: 1}
}
