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

// runConfig 处理配置初始化相关命令。
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
	var connection string
	var profileAlias string
	var method string
	var grantTypeAlias string
	var force bool

	flags.StringVar(&platform, "platform", "", "设置默认连接所属的平台")
	flags.StringVar(&subject, "subject", "", "设置默认连接使用的主体类型")
	flags.StringVar(&connection, "connection", "", "设置默认连接名")
	flags.StringVar(&profileAlias, "profile", "", "兼容旧参数，等同于 --connection")
	flags.StringVar(&method, "method", "", "覆盖连接使用的授权接入方式")
	flags.StringVar(&grantTypeAlias, "grant-type", "", "兼容旧参数，会自动映射到 method")
	flags.BoolVar(&force, "force", false, "overwrite the existing config file")

	if err := flags.Parse(args); err != nil {
		if err == pflag.ErrHelp {
			return nil
		}
		return err
	}
	if len(flags.Args()) != 0 {
		return fmt.Errorf("usage: clawrise config init [--platform <name>] [--subject <name>] [--connection <name>] [--method <name>] [--force]")
	}
	if strings.TrimSpace(connection) == "" {
		connection = strings.TrimSpace(profileAlias)
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
		Platform:   platform,
		Subject:    subject,
		Connection: connection,
		Method:     method,
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
			"connection_name":  result.ConnectionName,
			"profile_name":     result.ConnectionName,
			"platform":         result.Platform,
			"subject":          result.Subject,
			"method":           result.Method,
			"grant_type":       config.LegacyGrantTypeForMethod(result.Method),
			"secret_backend":   result.SecretBackend,
			"session_backend":  result.SessionBackend,
			"required_secrets": result.SecretFields,
		},
	})
}

func printConfigHelp(stdout io.Writer) {
	_, _ = fmt.Fprintln(stdout, "Usage: clawrise config init [--platform <name>] [--subject <name>] [--connection <name>] [--method <name>] [--force]")
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
