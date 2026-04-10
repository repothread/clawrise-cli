package cli

import (
	"fmt"
	"strings"
)

var (
	rootCompletionCommands = []string{
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
	platformCompletionCommands   = []string{"use", "current", "unset"}
	accountCompletionCommands    = []string{"list", "inspect", "use", "current", "add", "ensure", "remove"}
	subjectCompletionCommands    = []string{"use", "current", "unset", "list"}
	authCompletionCommands       = []string{"list", "methods", "presets", "inspect", "check", "login", "complete", "logout", "secret"}
	authSecretCompletionCommands = []string{"set", "put", "delete"}
	configCompletionCommands     = []string{"init", "secret-store", "provider", "auth-launcher", "policy", "audit"}
	pluginCompletionCommands     = []string{"list", "install", "info", "remove", "verify", "upgrade"}
	specCompletionCommands       = []string{"list", "get", "status", "export"}
	docsCompletionCommands       = []string{"generate"}
)

func commandUsageLine(path string, commands []string) string {
	return fmt.Sprintf("%s %s", path, commandAlternatives(commands))
}

func commandAlternatives(commands []string) string {
	return "[" + strings.Join(commands, "|") + "]"
}
