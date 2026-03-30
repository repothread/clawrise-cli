package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/pflag"

	"github.com/clawrise/clawrise-cli/internal/config"
	"github.com/clawrise/clawrise-cli/internal/output"
)

// runConfig handles config bootstrap commands.
func runConfig(args []string, store *config.Store, stdout io.Writer) error {
	if len(args) == 0 || isHelpToken(args[0]) {
		printConfigHelp(stdout)
		return nil
	}

	switch args[0] {
	case "init":
		return runConfigInit(args[1:], store, stdout)
	default:
		return fmt.Errorf("unknown config command: %s", args[0])
	}
}

func runConfigInit(args []string, store *config.Store, stdout io.Writer) error {
	flags := pflag.NewFlagSet("clawrise config init", pflag.ContinueOnError)
	flags.SetOutput(stdout)

	var platform string
	var subject string
	var account string
	var method string
	var grantTypeAlias string
	var scopes []string
	var force bool

	flags.StringVar(&platform, "platform", "", "set the platform for the default account")
	flags.StringVar(&subject, "subject", "", "set the subject for the default account")
	flags.StringVar(&account, "account", "", "set the account name")
	flags.StringVar(&method, "method", "", "override the auth method")
	flags.StringVar(&grantTypeAlias, "grant-type", "", "map a legacy grant type to an auth method")
	flags.StringSliceVar(&scopes, "scope", nil, "append auth scopes for interactive OAuth")
	flags.BoolVar(&force, "force", false, "overwrite the existing config file")

	if err := flags.Parse(args); err != nil {
		if err == pflag.ErrHelp {
			return nil
		}
		return err
	}
	if len(flags.Args()) != 0 {
		return fmt.Errorf("usage: clawrise config init [--platform <name>] [--subject <name>] [--account <name>] [--method <name>] [--scope <name>] [--force]")
	}
	if strings.TrimSpace(method) == "" {
		method = legacyGrantTypeToMethod(grantTypeAlias)
	}

	if _, err := os.Stat(store.Path()); err == nil && !force {
		return fmt.Errorf("config file already exists at %s; rerun with --force to overwrite it", store.Path())
	} else if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to stat config file: %w", err)
	}

	result, err := config.BuildInitConfig(config.InitOptions{
		Platform: platform,
		Subject:  subject,
		Account:  account,
		Method:   method,
		Scopes:   scopes,
	})
	if err != nil {
		return err
	}
	if err := store.Save(result.Config); err != nil {
		return err
	}

	return output.WriteJSON(stdout, map[string]any{
		"ok": true,
		"data": map[string]any{
			"config_path":      store.Path(),
			"account":          result.AccountName,
			"platform":         result.Platform,
			"subject":          result.Subject,
			"method":           result.Method,
			"grant_type":       config.LegacyGrantTypeForMethod(result.Method),
			"scopes":           result.Config.ResolvedAccount(result.AccountName).Params.Scopes,
			"secret_backend":   result.SecretBackend,
			"session_backend":  result.SessionBackend,
			"required_secrets": result.SecretFields,
		},
	})
}

func printConfigHelp(stdout io.Writer) {
	_, _ = fmt.Fprintln(stdout, "Usage: clawrise config init [--platform <name>] [--subject <name>] [--account <name>] [--method <name>] [--scope <name>] [--force]")
}

func legacyGrantTypeToMethod(grantType string) string {
	grantType = strings.TrimSpace(grantType)
	switch grantType {
	case "":
		return ""
	case "client_credentials":
		return "feishu.app_credentials"
	case "oauth_user":
		return "feishu.oauth_user"
	case "static_token":
		return "notion.internal_token"
	case "oauth_refreshable":
		return "notion.oauth_public"
	default:
		return ""
	}
}
