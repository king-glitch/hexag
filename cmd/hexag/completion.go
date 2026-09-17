package main

import (
	"fmt"
)

const zshCompletionScript = `#compdef hexag

_hexag() {
    local -a commands
    commands=(
        'verify:Verify code adherence to strict AGENTS.md architecture & naming rules'
        'bruno:Generate Bruno API collections and Markdown API reference'
        'mongo:Generate Mongo fields and standalone models'
        'new:Scaffold a new hexagonal project from the official template'
        'update:Update AGENTS.md, CLAUDE.md, and configuration in an existing project'
        'completion:Generate shell autocompletion script (zsh, bash)'
        'version:Show hexag version'
        'help:Show help'
    )

    _arguments -C \
        '1: :->command' \
        '*:: :->args'

    case $state in
        command)
            _describe -t commands 'hexag command' commands
            ;;
        args)
            case $line[1] in
                verify)
                    _arguments \
                        '-v[verbose output]' \
                        '-dir[target directory to verify]:directory:_files -/'
                    ;;
                bruno)
                    _arguments \
                        '-routes[routes directory]:directory:_files -/' \
                        '-out[output directory for bruno collection]:directory:_files -/' \
                        '-name[collection name]:name:' \
                        '-base-url[base URL]:url:' \
                        '-api-md[path to write API.md]:file:_files'
                    ;;
                mongo)
                    _arguments \
                        '-file[domain.go source file]:file:_files' \
                        '-out[output directory]:directory:_files -/' \
                        '-pkg[output package name]:package:'
                    ;;
                new)
                    _arguments \
                        '1:module-path:' \
                        '2:directory:_files -/'
                    ;;
                completion)
                    _arguments \
                        '1:(zsh bash)'
                    ;;
            esac
            ;;
    esac
}

_hexag "$@"
`

const bashCompletionScript = `_hexag_completions() {
    local cur prev commands
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    commands="verify bruno mongo new update completion version help"

    if [ $COMP_CWORD -eq 1 ]; then
        COMPREPLY=( $(compgen -W "${commands}" -- ${cur}) )
        return 0
    fi

    case "${prev}" in
        -dir|-routes|-out)
            COMPREPLY=( $(compgen -d -- ${cur}) )
            return 0
            ;;
        -file|-api-md)
            COMPREPLY=( $(compgen -f -- ${cur}) )
            return 0
            ;;
        completion)
            COMPREPLY=( $(compgen -W "zsh bash" -- ${cur}) )
            return 0
            ;;
    esac
}
complete -F _hexag_completions hexag
`

func runCompletion(args []string) error {
	shell := "zsh"
	if len(args) > 0 {
		shell = args[0]
	}

	switch shell {
	case "zsh":
		fmt.Print(zshCompletionScript)
	case "bash":
		fmt.Print(bashCompletionScript)
	default:
		return fmt.Errorf("unsupported shell '%s'; supported shells are 'zsh' and 'bash'", shell)
	}

	return nil
}
