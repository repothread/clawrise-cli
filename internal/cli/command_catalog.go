package cli

import "strings"

var (
	rootCLICommands = []string{
		"platform",
		"account",
		"subject",
		"auth",
		"secret",
		"config",
		"plugin",
		"spec",
		"docs",
		"completion",
		"doctor",
		"version",
		"batch",
	}
	platformCLICommands           = []string{"use", "current", "unset"}
	accountCLICommands            = []string{"list", "inspect", "use", "current", "add", "ensure", "remove"}
	subjectCLICommands            = []string{"use", "current", "unset", "list"}
	authCLICommands               = []string{"list", "methods", "presets", "inspect", "check", "login", "complete", "logout", "secret"}
	authSecretCLICommands         = []string{"set", "put", "delete"}
	configCLICommands             = []string{"init", "secret-store", "provider", "auth-launcher", "policy", "audit"}
	configSecretStoreCLICommands  = []string{"use"}
	configProviderCLICommands     = []string{"use", "unset"}
	configAuthLauncherCLICommands = []string{"prefer", "unset"}
	configPolicyCLICommands       = []string{"mode", "use", "remove"}
	configAuditCLICommands        = []string{"mode", "add", "remove"}
	configAuditTargetCLICommands  = []string{"stdout", "webhook", "plugin"}
	pluginCLICommands             = []string{"list", "install", "info", "remove", "verify", "upgrade"}
	specCLICommands               = []string{"list", "get", "status", "export"}
	docsCLICommands               = []string{"generate"}
	completionShellCLICommands    = []string{"bash", "zsh", "fish"}
)

func commandAlternatives(values []string) string {
	return "[" + strings.Join(values, "|") + "]"
}
