package cli

import (
	"fmt"
	"io"
	"os"

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
	var profile string
	var grantType string
	var force bool

	flags.StringVar(&platform, "platform", "", "set the platform for the generated default profile")
	flags.StringVar(&subject, "subject", "", "set the subject for the generated default profile")
	flags.StringVar(&profile, "profile", "", "set the profile name for the generated default profile")
	flags.StringVar(&grantType, "grant-type", "", "override the grant type used by the generated profile")
	flags.BoolVar(&force, "force", false, "overwrite the existing config file")

	if err := flags.Parse(args); err != nil {
		if err == pflag.ErrHelp {
			return nil
		}
		return err
	}
	if len(flags.Args()) != 0 {
		return fmt.Errorf("usage: clawrise config init [--platform <name>] [--subject <name>] [--profile <name>] [--grant-type <type>] [--force]")
	}

	if _, err := os.Stat(store.Path()); err == nil && !force {
		return fmt.Errorf("config file already exists at %s; rerun with --force to overwrite it", store.Path())
	} else if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to stat config file: %w", err)
	}

	result, err := config.BuildInitConfig(config.InitOptions{
		Platform:  platform,
		Subject:   subject,
		Profile:   profile,
		GrantType: grantType,
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
			"config_path":  store.Path(),
			"profile_name": result.ProfileName,
			"platform":     result.Platform,
			"subject":      result.Subject,
			"grant_type":   result.GrantType,
			"env_template": result.EnvTemplate,
		},
	})
}

func printConfigHelp(stdout io.Writer) {
	_, _ = fmt.Fprintln(stdout, "Usage: clawrise config init [--platform <name>] [--subject <name>] [--profile <name>] [--grant-type <type>] [--force]")
}
