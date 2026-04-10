package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/clawrise/clawrise-cli/internal/spec"
)

var (
	completionShells            = []string{"bash", "zsh", "fish"}
	operationCompletionFlags    = []string{"--account", "--subject", "--json", "--input", "--timeout", "--dry-run", "--debug-provider-payload", "--verify", "--idempotency-key", "--output", "--quiet", "--help", "-h"}
	batchCompletionFlags        = []string{"--json", "--input", "--help", "-h"}
	specExportCompletionFlags   = []string{"--format", "--out-dir", "--help", "-h"}
	docsGenerateCompletionFlags = []string{"--out-dir", "--help", "-h"}
	configInitCompletionFlags   = []string{"--platform", "--preset", "--subject", "--account", "--method", "--scope", "--force", "--help", "-h"}
)

// runCompletion prints the shell completion script.
func runCompletion(args []string, stdout io.Writer, service *spec.Service) error {
	if len(args) == 0 || isHelpToken(args[0]) {
		printCompletionHelp(stdout)
		return nil
	}
	if len(args) != 1 {
		return fmt.Errorf("usage: clawrise completion <bash|zsh|fish>")
	}

	shell := strings.TrimSpace(args[0])
	data := service.CompletionData()

	switch shell {
	case "bash":
		_, _ = io.WriteString(stdout, buildBashCompletionScript(data))
		return nil
	case "zsh":
		_, _ = io.WriteString(stdout, buildZshCompletionScript(data))
		return nil
	case "fish":
		_, _ = io.WriteString(stdout, buildFishCompletionScript(data))
		return nil
	default:
		return fmt.Errorf("unsupported completion shell: %s", shell)
	}
}

func printCompletionHelp(stdout io.Writer) {
	_, _ = fmt.Fprintln(stdout, "Usage: clawrise completion <bash|zsh|fish>")
	_, _ = fmt.Fprintln(stdout, "")
	_, _ = fmt.Fprintln(stdout, "Examples:")
	_, _ = fmt.Fprintln(stdout, "  clawrise completion bash")
	_, _ = fmt.Fprintln(stdout, "  clawrise completion zsh")
	_, _ = fmt.Fprintln(stdout, "  clawrise completion fish")
}

func buildBashCompletionScript(data spec.CompletionData) string {
	rootWords := shellWords(append(append([]string{}, rootCompletionCommands...), data.Operations...))
	specPaths := shellWords(data.SpecPaths)

	return fmt.Sprintf(`# bash completion for clawrise
_clawrise_completion() {
  local cur prev first second
  cur="${COMP_WORDS[COMP_CWORD]}"
  prev=""
  if [[ ${COMP_CWORD} -gt 0 ]]; then
    prev="${COMP_WORDS[COMP_CWORD-1]}"
  fi
  first=""
  second=""
  if [[ ${#COMP_WORDS[@]} -gt 1 ]]; then
    first="${COMP_WORDS[1]}"
  fi
  if [[ ${#COMP_WORDS[@]} -gt 2 ]]; then
    second="${COMP_WORDS[2]}"
  fi

  if [[ ${COMP_CWORD} -eq 1 ]]; then
    COMPREPLY=( $(compgen -W '%s' -- "$cur") )
    return 0
  fi

  case "$first" in
    platform)
      COMPREPLY=( $(compgen -W '%s' -- "$cur") )
      return 0
      ;;
    subject)
      COMPREPLY=( $(compgen -W '%s' -- "$cur") )
      return 0
      ;;
    account)
      COMPREPLY=( $(compgen -W '%s' -- "$cur") )
      return 0
      ;;
    auth)
      if [[ ${COMP_CWORD} -eq 2 ]]; then
        COMPREPLY=( $(compgen -W '%s' -- "$cur") )
        return 0
      fi
      if [[ "$second" == "secret" ]]; then
        COMPREPLY=( $(compgen -W '%s' -- "$cur") )
        return 0
      fi
      COMPREPLY=()
      return 0
      ;;
    config)
      if [[ ${COMP_CWORD} -eq 2 ]]; then
        COMPREPLY=( $(compgen -W '%s' -- "$cur") )
      else
        COMPREPLY=( $(compgen -W '%s' -- "$cur") )
      fi
      return 0
      ;;
    plugin)
      COMPREPLY=( $(compgen -W '%s' -- "$cur") )
      return 0
      ;;
    spec)
      if [[ ${COMP_CWORD} -eq 2 ]]; then
        COMPREPLY=( $(compgen -W '%s' -- "$cur") )
        return 0
      fi
      case "$second" in
        list|get|export)
          COMPREPLY=( $(compgen -W '%s %s' -- "$cur") )
          return 0
          ;;
        status)
          COMPREPLY=()
          return 0
          ;;
      esac
      ;;
    docs)
      if [[ ${COMP_CWORD} -eq 2 ]]; then
        COMPREPLY=( $(compgen -W '%s' -- "$cur") )
        return 0
      fi
      case "$second" in
        generate)
          COMPREPLY=( $(compgen -W '%s %s' -- "$cur") )
          return 0
          ;;
      esac
      ;;
    completion)
      COMPREPLY=( $(compgen -W '%s' -- "$cur") )
      return 0
      ;;
    batch)
      COMPREPLY=( $(compgen -W '%s' -- "$cur") )
      return 0
      ;;
    doctor|version)
      COMPREPLY=()
      return 0
      ;;
    *)
      COMPREPLY=( $(compgen -W '%s' -- "$cur") )
      return 0
      ;;
  esac
}

complete -F _clawrise_completion clawrise
`,
		rootWords,
		shellWords(platformCompletionCommands),
		shellWords(subjectCompletionCommands),
		shellWords(accountCompletionCommands),
		shellWords(authCompletionCommands),
		shellWords(authSecretCompletionCommands),
		shellWords(configCompletionCommands),
		shellWords(configInitCompletionFlags),
		shellWords(pluginCompletionCommands),
		shellWords(specCompletionCommands),
		specPaths,
		shellWords(specExportCompletionFlags),
		shellWords(docsCompletionCommands),
		specPaths,
		shellWords(docsGenerateCompletionFlags),
		shellWords(completionShells),
		shellWords(batchCompletionFlags),
		shellWords(operationCompletionFlags),
	)
}

func buildZshCompletionScript(data spec.CompletionData) string {
	return fmt.Sprintf(`#compdef clawrise

local -a root_commands
local -a platform_commands
local -a subject_commands
local -a account_commands
local -a auth_commands
local -a auth_secret_commands
local -a config_commands
local -a plugin_commands
local -a spec_commands
local -a docs_commands
local -a completion_shells
local -a operations
local -a spec_paths
local -a operation_flags
local -a config_init_flags
local -a spec_export_flags
local -a docs_generate_flags
local -a batch_flags

root_commands=(%s)
platform_commands=(%s)
subject_commands=(%s)
account_commands=(%s)
auth_commands=(%s)
auth_secret_commands=(%s)
config_commands=(%s)
plugin_commands=(%s)
spec_commands=(%s)
docs_commands=(%s)
completion_shells=(%s)
operations=(%s)
spec_paths=(%s)
operation_flags=(%s)
config_init_flags=(%s)
spec_export_flags=(%s)
docs_generate_flags=(%s)
batch_flags=(%s)

if (( CURRENT == 2 )); then
  compadd -- $root_commands $operations
  return
fi

case "$words[2]" in
  platform)
    compadd -- $platform_commands
    ;;
  account)
    compadd -- $account_commands
    ;;
  subject)
    compadd -- $subject_commands
    ;;
  auth)
    if (( CURRENT == 3 )); then
      compadd -- $auth_commands
    else
      case "$words[3]" in
        secret)
          compadd -- $auth_secret_commands
          ;;
      esac
    fi
    ;;
  config)
    if (( CURRENT == 3 )); then
      compadd -- $config_commands
    else
      compadd -- $config_init_flags
    fi
    ;;
  plugin)
    compadd -- $plugin_commands
    ;;
  spec)
    if (( CURRENT == 3 )); then
      compadd -- $spec_commands
    else
      case "$words[3]" in
        list|get|export)
          compadd -- $spec_paths $spec_export_flags
          ;;
      esac
    fi
    ;;
  docs)
    if (( CURRENT == 3 )); then
      compadd -- $docs_commands
    else
      case "$words[3]" in
        generate)
          compadd -- $spec_paths $docs_generate_flags
          ;;
      esac
    fi
    ;;
  completion)
    compadd -- $completion_shells
    ;;
  batch)
    compadd -- $batch_flags
    ;;
  doctor|version)
    ;;
  *)
    compadd -- $operation_flags
    ;;
esac
`, zshWords(append(append([]string{}, rootCompletionCommands...), data.Operations...)),
		zshWords(platformCompletionCommands),
		zshWords(subjectCompletionCommands),
		zshWords(accountCompletionCommands),
		zshWords(authCompletionCommands),
		zshWords(authSecretCompletionCommands),
		zshWords(configCompletionCommands),
		zshWords(pluginCompletionCommands),
		zshWords(specCompletionCommands),
		zshWords(docsCompletionCommands),
		zshWords(completionShells),
		zshWords(data.Operations),
		zshWords(data.SpecPaths),
		zshWords(operationCompletionFlags),
		zshWords(configInitCompletionFlags),
		zshWords(specExportCompletionFlags),
		zshWords(docsGenerateCompletionFlags),
		zshWords(batchCompletionFlags),
	)
}

func buildFishCompletionScript(data spec.CompletionData) string {
	lines := []string{
		"# fish completion for clawrise",
		"complete -c clawrise -f",
	}

	for _, command := range rootCompletionCommands {
		lines = append(lines, fmt.Sprintf("complete -c clawrise -n '__fish_use_subcommand' -a '%s'", command))
	}
	for _, operation := range data.Operations {
		lines = append(lines, fmt.Sprintf("complete -c clawrise -n '__fish_use_subcommand' -a '%s'", operation))
	}

	lines = append(lines, fishCommandCompletions("platform", platformCompletionCommands)...)
	lines = append(lines, fishCommandCompletions("account", accountCompletionCommands)...)
	lines = append(lines, fishCommandCompletions("subject", subjectCompletionCommands)...)
	lines = append(lines, fishCommandCompletions("auth", authCompletionCommands)...)
	for _, value := range authSecretCompletionCommands {
		lines = append(lines, fmt.Sprintf("complete -c clawrise -n '__fish_seen_subcommand_from auth; and __fish_seen_subcommand_from secret' -a '%s'", value))
	}
	lines = append(lines, fishCommandCompletions("config", configCompletionCommands)...)
	lines = append(lines, fishCommandCompletions("plugin", pluginCompletionCommands)...)
	lines = append(lines, fishCommandCompletions("spec", specCompletionCommands)...)
	lines = append(lines, fishCommandCompletions("docs", docsCompletionCommands)...)
	lines = append(lines, fishCommandCompletions("completion", completionShells)...)

	for _, path := range data.SpecPaths {
		lines = append(lines, fmt.Sprintf("complete -c clawrise -n '__fish_seen_subcommand_from spec; and not __fish_seen_subcommand_from status' -a '%s'", path))
		lines = append(lines, fmt.Sprintf("complete -c clawrise -n '__fish_seen_subcommand_from docs generate' -a '%s'", path))
	}
	for _, flag := range specExportCompletionFlags {
		lines = append(lines, fmt.Sprintf("complete -c clawrise -n '__fish_seen_subcommand_from spec export' -l '%s'", strings.TrimLeft(flag, "-")))
	}
	for _, flag := range docsGenerateCompletionFlags {
		lines = append(lines, fmt.Sprintf("complete -c clawrise -n '__fish_seen_subcommand_from docs generate' -l '%s'", strings.TrimLeft(flag, "-")))
	}
	for _, flag := range batchCompletionFlags {
		if !strings.HasPrefix(flag, "--") {
			continue
		}
		lines = append(lines, fmt.Sprintf("complete -c clawrise -n '__fish_seen_subcommand_from batch' -l '%s'", strings.TrimPrefix(flag, "--")))
	}
	for _, flag := range configInitCompletionFlags {
		if !strings.HasPrefix(flag, "--") {
			continue
		}
		lines = append(lines, fmt.Sprintf("complete -c clawrise -n '__fish_seen_subcommand_from config init' -l '%s'", strings.TrimPrefix(flag, "--")))
	}
	for _, flag := range operationCompletionFlags {
		if strings.HasPrefix(flag, "--") {
			lines = append(lines, fmt.Sprintf("complete -c clawrise -n 'not __fish_seen_subcommand_from %s' -l '%s'", strings.Join(rootCompletionCommands, " "), strings.TrimPrefix(flag, "--")))
			continue
		}
		lines = append(lines, fmt.Sprintf("complete -c clawrise -n 'not __fish_seen_subcommand_from %s' -s '%s'", strings.Join(rootCompletionCommands, " "), strings.TrimPrefix(flag, "-")))
	}
	return strings.Join(lines, "\n") + "\n"
}

func fishCommandCompletions(command string, values []string) []string {
	lines := make([]string, 0, len(values))
	for _, value := range values {
		lines = append(lines, fmt.Sprintf("complete -c clawrise -n '__fish_seen_subcommand_from %s' -a '%s'", command, value))
	}
	return lines
}

func shellWords(values []string) string {
	return strings.Join(values, " ")
}

func zshWords(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, fmt.Sprintf("'%s'", strings.ReplaceAll(value, "'", `'\''`)))
	}
	return strings.Join(quoted, " ")
}
