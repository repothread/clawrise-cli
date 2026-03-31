package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/pflag"

	"github.com/clawrise/clawrise-cli/internal/config"
	"github.com/clawrise/clawrise-cli/internal/output"
	"github.com/clawrise/clawrise-cli/internal/secretstore"
)

// runAuthSecret manages long-lived secret values.
func runAuthSecret(args []string, cfg *config.Config, store *config.Store, stdout io.Writer) error {
	if len(args) == 0 || isHelpToken(args[0]) {
		printAuthSecretHelp(stdout)
		return nil
	}

	secretBackend := strings.TrimSpace(cfg.Auth.SecretStore.Backend)
	if secretBackend == "" {
		secretBackend = "encrypted_file"
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
	case "set", "put":
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
	var fromEnv string
	var allowInsecureCLISecret bool

	flags.StringVar(&value, "value", "", "write the secret value directly")
	flags.BoolVar(&fromStdin, "stdin", false, "read the secret value from stdin")
	flags.StringVar(&fromEnv, "from-env", "", "read the secret value from the selected environment variable")
	flags.BoolVar(&allowInsecureCLISecret, "allow-insecure-cli-secret", false, "allow writing the secret directly from a CLI flag value")

	if err := flags.Parse(args); err != nil {
		if err == pflag.ErrHelp {
			return nil
		}
		return err
	}
	if len(flags.Args()) != 2 {
		return fmt.Errorf("usage: clawrise auth secret set <account> <field> [--stdin | --from-env <name> | --value <text> --allow-insecure-cli-secret]")
	}

	connectionName := strings.TrimSpace(flags.Args()[0])
	fieldName := strings.TrimSpace(flags.Args()[1])
	if _, ok := cfg.Accounts[connectionName]; !ok {
		return writeCLIError(stdout, "ACCOUNT_NOT_FOUND", "the selected account does not exist")
	}

	inputModeCount := 0
	if fromStdin {
		inputModeCount++
	}
	if strings.TrimSpace(fromEnv) != "" {
		inputModeCount++
	}
	if strings.TrimSpace(value) != "" {
		inputModeCount++
	}
	if inputModeCount == 0 {
		return fmt.Errorf("one secret input mode is required: use --stdin, --from-env, or --value with --allow-insecure-cli-secret")
	}
	if inputModeCount > 1 {
		return fmt.Errorf("secret input modes are mutually exclusive: choose only one of --stdin, --from-env, or --value")
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
	if envName := strings.TrimSpace(fromEnv); envName != "" {
		value = strings.TrimSpace(os.Getenv(envName))
		if value == "" {
			return fmt.Errorf("environment variable %s is empty or not set", envName)
		}
	}
	if strings.TrimSpace(value) != "" && !fromStdin && strings.TrimSpace(fromEnv) == "" && !allowInsecureCLISecret {
		return fmt.Errorf("--value requires --allow-insecure-cli-secret")
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
			"account": connectionName,
			"field":   fieldName,
			"backend": secretStore.Backend(),
		},
	})
}

func runAuthSecretDelete(args []string, cfg *config.Config, stdout io.Writer, secretStore secretstore.Store) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: clawrise auth secret delete <account> <field>")
	}

	connectionName := strings.TrimSpace(args[0])
	fieldName := strings.TrimSpace(args[1])
	if _, ok := cfg.Accounts[connectionName]; !ok {
		return writeCLIError(stdout, "ACCOUNT_NOT_FOUND", "the selected account does not exist")
	}

	if err := secretStore.Delete(connectionName, fieldName); err != nil {
		return err
	}
	return output.WriteJSON(stdout, map[string]any{
		"ok": true,
		"data": map[string]any{
			"account": connectionName,
			"field":   fieldName,
			"backend": secretStore.Backend(),
			"deleted": true,
		},
	})
}

func printAuthSecretHelp(stdout io.Writer) {
	_, _ = fmt.Fprintln(stdout, "Usage: clawrise auth secret [set|put|delete] <account> <field>")
	_, _ = fmt.Fprintln(stdout, "       clawrise auth secret set <account> <field> [--stdin | --from-env <name> | --value <text> --allow-insecure-cli-secret]")
}
