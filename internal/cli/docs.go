package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/pflag"

	"github.com/clawrise/clawrise-cli/internal/output"
	"github.com/clawrise/clawrise-cli/internal/spec"
)

// runDocs 处理文档生成相关命令。
func runDocs(args []string, stdout io.Writer, service *spec.Service) error {
	if len(args) == 0 || isHelpToken(args[0]) {
		printDocsHelp(stdout)
		return nil
	}

	switch args[0] {
	case "generate":
		return runDocsGenerate(args[1:], stdout, service)
	default:
		return fmt.Errorf("unknown docs command: %s", args[0])
	}
}

func runDocsGenerate(args []string, stdout io.Writer, service *spec.Service) error {
	flags := pflag.NewFlagSet("clawrise docs generate", pflag.ContinueOnError)
	flags.SetOutput(stdout)

	var outDir string
	flags.StringVar(&outDir, "out-dir", "./docs/generated", "write generated markdown docs into the target directory")

	if err := flags.Parse(args); err != nil {
		if err == pflag.ErrHelp {
			return nil
		}
		return err
	}
	if len(flags.Args()) > 1 {
		return fmt.Errorf("usage: clawrise docs generate [path] [--out-dir <dir>]")
	}

	path := ""
	if len(flags.Args()) == 1 {
		path = strings.TrimSpace(flags.Args()[0])
	}

	// docs 生成继续复用 spec 元数据导出，避免形成第二套事实源。
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

func printDocsHelp(stdout io.Writer) {
	_, _ = fmt.Fprintln(stdout, "Usage: clawrise docs generate [path] [--out-dir <dir>]")
	_, _ = fmt.Fprintln(stdout, "")
	_, _ = fmt.Fprintln(stdout, "Examples:")
	_, _ = fmt.Fprintln(stdout, "  clawrise docs generate")
	_, _ = fmt.Fprintln(stdout, "  clawrise docs generate notion.page")
	_, _ = fmt.Fprintln(stdout, "  clawrise docs generate feishu.docs --out-dir ./docs/generated")
}
