package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/pflag"

	"github.com/clawrise/clawrise-cli/internal/config"
	"github.com/clawrise/clawrise-cli/internal/output"
	"github.com/clawrise/clawrise-cli/internal/secretstore"
)

// runAuthSecret 管理长期敏感信息。
func runAuthSecret(args []string, cfg *config.Config, store *config.Store, stdout io.Writer) error {
	if len(args) == 0 || isHelpToken(args[0]) {
		printAuthSecretHelp(stdout)
		return nil
	}

	secretBackend := strings.TrimSpace(cfg.Auth.SecretStore.Backend)
	if secretBackend == "" {
		secretBackend = "auto"
	}
	secretStore, err := secretstore.Open(secretstore.Options{
		ConfigPath:      store.Path(),
		Backend:         secretBackend,
		FallbackBackend: cfg.Auth.SecretStore.FallbackBackend,
	})
	if err != nil {
		return err
	}

	switch args[0] {
	case "set":
		return runAuthSecretSet(args[1:], cfg, stdout, secretStore)
	case "delete":
		return runAuthSecretDelete(args[1:], cfg, stdout, secretStore)
	default:
		return fmt.Errorf("unknown auth secret command: %s", args[0])
	}
}

func runAuthSecretSet(args []string, cfg *config.Config, stdout io.Writer, secretStore secretstore.Store) error {
	flags := pflag.NewFlagSet("clawrise auth secret set", pflag.ContinueOnError)
	flags.SetOutput(stdout)

	var value string
	var fromStdin bool

	flags.StringVar(&value, "value", "", "要写入 secret store 的敏感值")
	flags.BoolVar(&fromStdin, "stdin", false, "从标准输入读取敏感值")

	if err := flags.Parse(args); err != nil {
		if err == pflag.ErrHelp {
			return nil
		}
		return err
	}
	if len(flags.Args()) != 2 {
		return fmt.Errorf("usage: clawrise auth secret set <connection> <field> [--value <text> | --stdin]")
	}

	connectionName := strings.TrimSpace(flags.Args()[0])
	fieldName := strings.TrimSpace(flags.Args()[1])
	if _, ok := cfg.Connections[connectionName]; !ok {
		return writeCLIError(stdout, "CONNECTION_NOT_FOUND", "the selected connection does not exist")
	}

	if fromStdin {
		reader := readPipedInput()
		if reader == nil {
			return fmt.Errorf("no stdin data was provided")
		}
		input, err := io.ReadAll(reader)
		if err != nil {
			return err
		}
		value = strings.TrimSpace(string(input))
	}
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("secret value must not be empty")
	}

	if err := secretStore.Set(connectionName, fieldName, value); err != nil {
		return err
	}
	return output.WriteJSON(stdout, map[string]any{
		"ok": true,
		"data": map[string]any{
			"connection": connectionName,
			"field":      fieldName,
			"backend":    secretStore.Backend(),
		},
	})
}

func runAuthSecretDelete(args []string, cfg *config.Config, stdout io.Writer, secretStore secretstore.Store) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: clawrise auth secret delete <connection> <field>")
	}

	connectionName := strings.TrimSpace(args[0])
	fieldName := strings.TrimSpace(args[1])
	if _, ok := cfg.Connections[connectionName]; !ok {
		return writeCLIError(stdout, "CONNECTION_NOT_FOUND", "the selected connection does not exist")
	}

	if err := secretStore.Delete(connectionName, fieldName); err != nil {
		return err
	}
	return output.WriteJSON(stdout, map[string]any{
		"ok": true,
		"data": map[string]any{
			"connection": connectionName,
			"field":      fieldName,
			"backend":    secretStore.Backend(),
			"deleted":    true,
		},
	})
}

func printAuthSecretHelp(stdout io.Writer) {
	_, _ = fmt.Fprintln(stdout, "Usage: clawrise auth secret [set|delete] <connection> <field>")
}
