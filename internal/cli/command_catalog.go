package cli

import (
	"fmt"
	"strings"
)

var (
	rootCommandNames = []string{
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
	platformCommandNames = []string{"use", "current", "unset"}
	accountCommandNames  = []string{"list", "inspect", "use", "current", "add", "ensure", "remove"}
	subjectCommandNames  = []string{"use", "current", "unset", "list"}
	authCommandNames     = []string{"list", "methods", "presets", "inspect", "check", "login", "complete", "logout", "secret"}
	secretCommandNames   = []string{"set", "put", "delete"}
	configCommandNames   = []string{"init", "secret-store", "provider", "auth-launcher", "policy", "audit"}
	pluginCommandNames   = []string{"list", "install", "info", "remove", "verify", "upgrade"}
	specCommandNames     = []string{"list", "get", "status", "export"}
	docsCommandNames     = []string{"generate"}
	completionShellNames = []string{"bash", "zsh", "fish"}
)

func commandUsage(noun string, commands []string) string {
	return fmt.Sprintf("Usage: clawrise %s %s", noun, bracketedCommandList(commands))
}

func bracketedCommandList(commands []string) string {
	return "[" + strings.Join(commands, "|") + "]"
}
