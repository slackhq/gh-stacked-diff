package commands

import (
	"flag"
	"fmt"
	"log/slog"
	"strings"

	"github.com/slackhq/gh-stacked-diff/v2/util"
)

func createCompletionCommand() Command {
	flagSet := flag.NewFlagSet("completion", flag.ContinueOnError)
	shell := flagSet.String("shell", "", "Shell to generate completions for. Supported: zsh, bash, fish, powershell")
	return Command{
		FlagSet:         flagSet,
		DefaultLogLevel: slog.LevelError,
		Summary:         "Generates shell autocomplete script",
		Description: "Generates a shell autocomplete script.\n\nTo enable completions, add the output to your shell configuration.\n\n" +
			"For zsh, add the following to your ~/.zshrc:\n\n" +
			"   eval \"$(sd completion --shell zsh)\"\n\n" +
			"For bash, add the following to your ~/.bashrc:\n\n" +
			"   eval \"$(sd completion --shell bash)\"\n\n" +
			"For fish, add the following to ~/.config/fish/completions/sd.fish:\n\n" +
			"   sd completion --shell fish | source\n\n" +
			"For powershell, add the following to your $PROFILE:\n\n" +
			"   sd completion --shell powershell | Out-String | Invoke-Expression",
		Usage:         "sd " + flagSet.Name() + " --shell zsh|bash|fish|powershell",
		SkipRepoCheck: true,
		OnSelected: func(appConfig util.AppConfig, command Command) {
			if flagSet.NArg() != 0 {
				commandError(appConfig, flagSet, "too many args", command.Usage)
			}
			if *shell == "" {
				commandError(appConfig, flagSet, "--shell is required", command.Usage)
			}
			switch *shell {
			case "zsh":
				util.Fprint(appConfig.Io.Out, generateZshCompletion())
			case "bash":
				util.Fprint(appConfig.Io.Out, generateBashCompletion())
			case "fish":
				util.Fprint(appConfig.Io.Out, generateFishCompletion())
			case "powershell":
				util.Fprint(appConfig.Io.Out, generatePowershellCompletion())
			default:
				panic(fmt.Sprint("unsupported shell: ", *shell, ". Supported shells: zsh, bash, fish, powershell"))
			}
		}}
}

// completableCommands returns all commands that should appear in shell completions.
// Excludes hidden commands and the completion command itself.
func completableCommands() []Command {
	var result []Command
	for _, cmd := range allCommands() {
		if cmd.Hidden || cmd.FlagSet.Name() == "completion" {
			continue
		}
		result = append(result, cmd)
	}
	return result
}

func generateZshCompletion() string {
	// Build the list of commands and their summaries for zsh completion.
	commands := completableCommands()
	var subcommands []string
	for _, cmd := range commands {
		// Escape colons and single quotes in summaries for zsh.
		summary := strings.ReplaceAll(cmd.Summary, "'", "'\\''")
		summary = strings.ReplaceAll(summary, ":", "\\:")
		name := cmd.FlagSet.Name()
		subcommands = append(subcommands, fmt.Sprintf("'%s:%s'", name, summary))
	}

	var flagCases []string
	for _, cmd := range commands {
		var flags []string
		cmd.FlagSet.VisitAll(func(f *flag.Flag) {
			usage := strings.ReplaceAll(f.Usage, "'", "'\\''")
			usage = strings.ReplaceAll(usage, ":", "\\:")
			// Remove newlines from usage for the completion description.
			usage = strings.ReplaceAll(usage, "\n", " ")
			flags = append(flags, fmt.Sprintf("'-%s[%s]'", f.Name, usage))
		})
		if len(flags) > 0 {
			flagCases = append(flagCases, fmt.Sprintf(
				"            %s)\n                _arguments -s \\\n                    %s\n                ;;",
				cmd.FlagSet.Name(),
				strings.Join(flags, " \\\n                    "),
			))
		}
	}

	return fmt.Sprintf(`#compdef sd

_sd() {
    local -a commands
    commands=(
        %s
    )

    _arguments -C \
        '-log-level[Set log level]:level:(debug info warn error)' \
        '1:command:->command' \
        '*::arg:->args' && return

    case "$state" in
        command)
            _describe -t commands 'sd command' commands
            ;;
        args)
            case "${words[1]}" in
%s
            esac
            ;;
    esac
}

compdef _sd sd
`, strings.Join(subcommands, "\n        "), strings.Join(flagCases, "\n"))
}

func generateBashCompletion() string {
	commands := completableCommands()
	var commandNames []string
	for _, cmd := range commands {
		commandNames = append(commandNames, cmd.FlagSet.Name())
	}

	var flagCases []string
	for _, cmd := range commands {
		var flags []string
		cmd.FlagSet.VisitAll(func(f *flag.Flag) {
			flags = append(flags, "-"+f.Name)
		})
		if len(flags) > 0 {
			flagCases = append(flagCases, fmt.Sprintf(
				"        %s)\n            COMPREPLY=($(compgen -W \"%s\" -- \"$cur\"))\n            return 0\n            ;;",
				cmd.FlagSet.Name(),
				strings.Join(flags, " "),
			))
		}
	}

	return fmt.Sprintf(`_sd() {
    local cur prev commands cmd i
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    commands="%s"

    # Find the subcommand by scanning past flags.
    cmd=""
    for ((i=1; i < COMP_CWORD; i++)); do
        case "${COMP_WORDS[i]}" in
            -log-level)
                ((i++))
                ;;
            -*)
                ;;
            *)
                cmd="${COMP_WORDS[i]}"
                break
                ;;
        esac
    done

    # Complete -log-level values regardless of position.
    if [[ "$prev" == "-log-level" ]]; then
        COMPREPLY=($(compgen -W "debug info warn error" -- "$cur"))
        return 0
    fi

    if [[ -z "$cmd" ]]; then
        COMPREPLY=($(compgen -W "$commands -log-level" -- "$cur"))
        return 0
    fi

    case "$cmd" in
%s
        *)
            ;;
    esac
}

complete -F _sd sd
`, strings.Join(commandNames, " "), strings.Join(flagCases, "\n"))
}

func generateFishCompletion() string {
	commands := completableCommands()
	var lines []string
	lines = append(lines, "# Top-level flags")
	lines = append(lines, "complete -c sd -n '__fish_use_subcommand' -o log-level -xa 'debug info warn error' -d 'Set log level'")
	lines = append(lines, "")
	lines = append(lines, "# Subcommands")
	for _, cmd := range commands {
		// Escape single quotes in summary.
		summary := strings.ReplaceAll(cmd.Summary, "'", "'\\''")
		lines = append(lines, fmt.Sprintf(
			"complete -c sd -n '__fish_use_subcommand' -a '%s' -d '%s'",
			cmd.FlagSet.Name(), summary,
		))
	}
	lines = append(lines, "")
	lines = append(lines, "# Subcommand flags")
	for _, cmd := range commands {
		cmd.FlagSet.VisitAll(func(f *flag.Flag) {
			// Remove newlines from usage.
			usage := strings.ReplaceAll(f.Usage, "\n", " ")
			usage = strings.ReplaceAll(usage, "'", "'\\''")
			lines = append(lines, fmt.Sprintf(
				"complete -c sd -n '__fish_seen_subcommand_from %s' -o %s -d '%s'",
				cmd.FlagSet.Name(), f.Name, usage,
			))
		})
	}
	lines = append(lines, "")
	return strings.Join(lines, "\n")
}

func generatePowershellCompletion() string {
	commands := completableCommands()
	var commandEntries []string
	for _, cmd := range commands {
		// Escape single quotes in summary.
		summary := strings.ReplaceAll(cmd.Summary, "'", "''")
		commandEntries = append(commandEntries, fmt.Sprintf(
			"        [CompletionResult]::new('%s', '%s', [CompletionResultType]::ParameterValue, '%s')",
			cmd.FlagSet.Name(), cmd.FlagSet.Name(), summary,
		))
	}

	var flagCases []string
	for _, cmd := range commands {
		var flags []string
		cmd.FlagSet.VisitAll(func(f *flag.Flag) {
			// Remove newlines and escape single quotes.
			usage := strings.ReplaceAll(f.Usage, "\n", " ")
			usage = strings.ReplaceAll(usage, "'", "''")
			flags = append(flags, fmt.Sprintf(
				"                [CompletionResult]::new('-%s', '-%s', [CompletionResultType]::ParameterName, '%s')",
				f.Name, f.Name, usage,
			))
		})
		if len(flags) > 0 {
			flagCases = append(flagCases, fmt.Sprintf(
				"            '%s' {\n%s\n            }",
				cmd.FlagSet.Name(),
				strings.Join(flags, "\n"),
			))
		}
	}

	return fmt.Sprintf(`Register-ArgumentCompleter -Native -CommandName sd -ScriptBlock {
    param($wordToComplete, $commandAst, $cursorPosition)
    $elements = $commandAst.CommandElements | Select-Object -Skip 1
    $args = @($elements | ForEach-Object { $_.ToString() })
    # Find the subcommand by scanning past flags.
    $cmd = $null
    for ($i = 0; $i -lt $args.Count; $i++) {
        if ($args[$i] -eq '-log-level') {
            $i++
        } elseif ($args[$i].StartsWith('-')) {
            continue
        } else {
            $cmd = $args[$i]
            break
        }
    }
    # Complete -log-level values.
    $prev = if ($args.Count -gt 0) { $args[$args.Count - 1] } else { $null }
    if ($prev -eq '-log-level') {
        'debug', 'info', 'warn', 'error' | ForEach-Object {
            [CompletionResult]::new($_, $_, [CompletionResultType]::ParameterValue, $_)
        }
    } elseif ($null -eq $cmd) {
        # Complete commands and top-level flags.
%s
        [CompletionResult]::new('-log-level', '-log-level', [CompletionResultType]::ParameterName, 'Set log level')
    } else {
        # Complete subcommand flags.
        switch ($cmd) {
%s
        }
    } | Where-Object { $_.CompletionText -like "$wordToComplete*" }
}
`, strings.Join(commandEntries, "\n"), strings.Join(flagCases, "\n"))
}
