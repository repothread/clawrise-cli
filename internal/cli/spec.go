package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"

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
		return runSpecExport(args[1:], stdout, service)
	default:
		return fmt.Errorf("unknown spec command: %s", args[0])
	}
}

func runSpecExport(args []string, stdout io.Writer, service *spec.Service) error {
	flags := pflag.NewFlagSet("clawrise spec export", pflag.ContinueOnError)
	flags.SetOutput(stdout)

	var format string
	var outDir string
	flags.StringVar(&format, "format", "json", "set the export format: json or markdown")
	flags.StringVar(&outDir, "out-dir", "", "write exported files into the target directory")

	if err := flags.Parse(args); err != nil {
		if err == pflag.ErrHelp {
			return nil
		}
		return err
	}
	if len(flags.Args()) > 1 {
		return fmt.Errorf("usage: clawrise spec export [path] [--format <json|markdown>]")
	}

	path := ""
	if len(flags.Args()) == 1 {
		path = strings.TrimSpace(flags.Args()[0])
	}

	switch strings.TrimSpace(format) {
	case "json":
		if strings.TrimSpace(outDir) != "" {
			return fmt.Errorf("--out-dir is supported only when --format markdown is used")
		}
		result, err := service.Export(path)
		if err != nil {
			return writeSpecError(stdout, err)
		}
		return output.WriteJSON(stdout, map[string]any{
			"ok":   true,
			"data": result,
		})
	case "markdown":
		if strings.TrimSpace(outDir) != "" {
			documents, err := service.ExportMarkdownDocuments(path)
			if err != nil {
				return writeSpecError(stdout, err)
			}
			writtenFiles, err := writeSpecExportDocuments(outDir, documents)
			if err != nil {
				return err
			}
			return output.WriteJSON(stdout, map[string]any{
				"ok": true,
				"data": map[string]any{
					"path":          path,
					"format":        "markdown",
					"output_dir":    strings.TrimSpace(outDir),
					"written_files": writtenFiles,
				},
			})
		}
		document, err := service.ExportMarkdown(path)
		if err != nil {
			return writeSpecError(stdout, err)
		}
		_, err = io.WriteString(stdout, document)
		return err
	default:
		return fmt.Errorf("unsupported spec export format: %s", format)
	}
}

func printSpecHelp(stdout io.Writer) {
	_, _ = fmt.Fprintf(stdout, "Usage: clawrise spec %s\n", commandAlternatives(specCLICommands))
	_, _ = fmt.Fprintln(stdout, "")
	_, _ = fmt.Fprintln(stdout, "Examples:")
	_, _ = fmt.Fprintln(stdout, "  clawrise spec list")
	_, _ = fmt.Fprintln(stdout, "  clawrise spec list feishu.docs")
	_, _ = fmt.Fprintln(stdout, "  clawrise spec get notion.page.create")
	_, _ = fmt.Fprintln(stdout, "  clawrise spec export")
	_, _ = fmt.Fprintln(stdout, "  clawrise spec export notion.page.create --format markdown")
	_, _ = fmt.Fprintln(stdout, "  clawrise spec export notion.page --format markdown --out-dir ./docs/generated")
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

func writeSpecExportDocuments(rootDir string, documents map[string]string) ([]string, error) {
	rootDir = strings.TrimSpace(rootDir)
	if rootDir == "" {
		return nil, fmt.Errorf("output directory is required")
	}
	if err := os.MkdirAll(rootDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create spec export output dir: %w", err)
	}

	paths := make([]string, 0, len(documents))
	for relativePath, document := range documents {
		relativePath = strings.TrimSpace(relativePath)
		if relativePath == "" {
			continue
		}
		targetPath := filepath.Join(rootDir, relativePath)
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return nil, fmt.Errorf("failed to create spec export directory: %w", err)
		}
		if err := os.WriteFile(targetPath, []byte(document), 0o644); err != nil {
			return nil, fmt.Errorf("failed to write spec export document: %w", err)
		}
		paths = append(paths, targetPath)
	}
	return paths, nil
}
