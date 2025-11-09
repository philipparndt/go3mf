package cmd

import (
	"fmt"
	"os"
)

type CompletionCmd struct {
	Shell string `arg:"" help:"Shell type: bash, zsh, or fish"`
}

func (c *CompletionCmd) Run() error {
	switch c.Shell {
	case "bash":
		return c.generateBash()
	case "zsh":
		return c.generateZsh()
	case "fish":
		return c.generateFish()
	default:
		return fmt.Errorf("unsupported shell: %s (supported: bash, zsh, fish)", c.Shell)
	}
}

func (c *CompletionCmd) generateBash() error {
	script := `# bash completion for go3mf

_go3mf_completions() {
    local cur prev opts
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"

    # Main commands
    if [[ ${COMP_CWORD} -eq 1 ]]; then
        opts="combine inspect version completion"
        COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
        return 0
    fi

    # Options for combine command
    if [[ ${COMP_WORDS[1]} == "combine" ]]; then
        case "${prev}" in
            -o|--output)
                COMPREPLY=( $(compgen -f -X '!*.3mf' -- ${cur}) )
                return 0
                ;;
            -n|--name)
                return 0
                ;;
            -c|--color|--filament)
                COMPREPLY=( $(compgen -W "1 2 3 4" -- ${cur}) )
                return 0
                ;;
            *)
                if [[ ${cur} == -* ]]; then
                    opts="-o --output --object -n --name -c --color --filament --open -h --help"
                    COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
                else
                    COMPREPLY=( $(compgen -f -X '!*.@(scad|3mf|stl|yaml|yml)' -- ${cur}) )
                fi
                return 0
                ;;
        esac
    fi

    # Options for inspect command
    if [[ ${COMP_WORDS[1]} == "inspect" ]]; then
        if [[ ${cur} == -* ]]; then
            opts="-h --help"
            COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
        else
            COMPREPLY=( $(compgen -f -X '!*.3mf' -- ${cur}) )
        fi
        return 0
    fi

    # Options for completion command
    if [[ ${COMP_WORDS[1]} == "completion" ]]; then
        if [[ ${COMP_CWORD} -eq 2 ]]; then
            opts="bash zsh fish"
            COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
        fi
        return 0
    fi
}

complete -F _go3mf_completions go3mf
`
	fmt.Print(script)
	return nil
}

func (c *CompletionCmd) generateZsh() error {
	script := `#compdef go3mf

_go3mf() {
    local -a commands
    commands=(
        'combine:Combine files into single 3MF'
        'inspect:Inspect a 3MF file and show its contents'
        'version:Show version information'
        'completion:Generate shell completion script'
    )

    local -a combine_opts
    combine_opts=(
        '(-o --output)'{-o,--output}'[Output file path]:output file:_files -g "*.3mf"'
        '--object[Start a new object group]'
        '(-n --name)'{-n,--name}'[Set object name]:name:'
        '(-c --color --filament)'{-c,--color,--filament}'[Set filament slot]:slot:(1 2 3 4)'
        '--open[Open the result file in the default application]'
        '(-h --help)'{-h,--help}'[Show help]'
        '*:input files:_files -g "*.{scad,3mf,stl,yaml,yml}"'
    )

    local -a inspect_opts
    inspect_opts=(
        '(-h --help)'{-h,--help}'[Show help]'
        '*:3mf file:_files -g "*.3mf"'
    )

    local -a completion_shells
    completion_shells=(
        'bash:Generate bash completion'
        'zsh:Generate zsh completion'
        'fish:Generate fish completion'
    )

    _arguments -C \
        '1: :->command' \
        '*:: :->args'

    case $state in
        command)
            _describe 'command' commands
            ;;
        args)
            case $words[1] in
                combine)
                    _arguments $combine_opts
                    ;;
                inspect)
                    _arguments $inspect_opts
                    ;;
                completion)
                    _describe 'shell' completion_shells
                    ;;
                version)
                    _arguments '(-h --help)'{-h,--help}'[Show help]'
                    ;;
            esac
            ;;
    esac
}

_go3mf
`
	fmt.Print(script)
	return nil
}

func (c *CompletionCmd) generateFish() error {
	script := `# fish completion for go3mf

# Main commands
complete -c go3mf -f -n "__fish_use_subcommand" -a "combine" -d "Combine files into single 3MF"
complete -c go3mf -f -n "__fish_use_subcommand" -a "inspect" -d "Inspect a 3MF file and show its contents"
complete -c go3mf -f -n "__fish_use_subcommand" -a "version" -d "Show version information"
complete -c go3mf -f -n "__fish_use_subcommand" -a "completion" -d "Generate shell completion script"

# combine command options
complete -c go3mf -f -n "__fish_seen_subcommand_from combine" -s o -l output -d "Output file path" -r -a "(__fish_complete_suffix .3mf)"
complete -c go3mf -f -n "__fish_seen_subcommand_from combine" -l object -d "Start a new object group"
complete -c go3mf -f -n "__fish_seen_subcommand_from combine" -s n -l name -d "Set object name" -r
complete -c go3mf -f -n "__fish_seen_subcommand_from combine" -s c -l color -l filament -d "Set filament slot" -r -a "1 2 3 4"
complete -c go3mf -f -n "__fish_seen_subcommand_from combine" -l open -d "Open the result file in the default application"
complete -c go3mf -f -n "__fish_seen_subcommand_from combine" -s h -l help -d "Show help"
complete -c go3mf -n "__fish_seen_subcommand_from combine" -a "(__fish_complete_suffix .scad)" -d "SCAD file"
complete -c go3mf -n "__fish_seen_subcommand_from combine" -a "(__fish_complete_suffix .3mf)" -d "3MF file"
complete -c go3mf -n "__fish_seen_subcommand_from combine" -a "(__fish_complete_suffix .stl)" -d "STL file"
complete -c go3mf -n "__fish_seen_subcommand_from combine" -a "(__fish_complete_suffix .yaml)" -d "YAML config"
complete -c go3mf -n "__fish_seen_subcommand_from combine" -a "(__fish_complete_suffix .yml)" -d "YAML config"

# inspect command options
complete -c go3mf -f -n "__fish_seen_subcommand_from inspect" -s h -l help -d "Show help"
complete -c go3mf -n "__fish_seen_subcommand_from inspect" -a "(__fish_complete_suffix .3mf)" -d "3MF file"

# completion command options
complete -c go3mf -f -n "__fish_seen_subcommand_from completion" -a "bash" -d "Generate bash completion"
complete -c go3mf -f -n "__fish_seen_subcommand_from completion" -a "zsh" -d "Generate zsh completion"
complete -c go3mf -f -n "__fish_seen_subcommand_from completion" -a "fish" -d "Generate fish completion"

# version command options
complete -c go3mf -f -n "__fish_seen_subcommand_from version" -s h -l help -d "Show help"
`
	fmt.Print(script)
	return nil
}

func (c *CompletionCmd) Help() string {
	return `
Generate shell completion scripts for go3mf.

Examples:
  # Bash
  go3mf completion bash > /etc/bash_completion.d/go3mf
  # or
  go3mf completion bash > ~/.local/share/bash-completion/completions/go3mf

  # Zsh
  go3mf completion zsh > ~/.zsh/completion/_go3mf
  # or add to .zshrc:
  autoload -U compinit && compinit

  # Fish
  go3mf completion fish > ~/.config/fish/completions/go3mf.fish
`
}

// For testing purposes
func generateCompletionToFile(shell, filepath string) error {
	// Save current stdout
	oldStdout := os.Stdout

	// Create file
	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Redirect stdout to file
	os.Stdout = file

	// Generate completion
	cmd := &CompletionCmd{Shell: shell}
	err = cmd.Run()

	// Restore stdout
	os.Stdout = oldStdout

	return err
}
