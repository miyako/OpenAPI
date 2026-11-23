package completion

import (
	"fmt"
	"strings"
)

// GenerateBash outputs Bash completion script for 4d-openapi.
func GenerateBash() string {
	return `#!/usr/bin/env bash
# Bash completion for 4d-openapi

_4d_openapi_completions() {
    local cur prev commands flags
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"

    commands="generate serve inspect dependencies validate diff export init mock completion version help"
    flags="--project --output --format --config --tool4d --tool4d-version --offline --no-dependencies --emit-ir --emit-manifest --strict --stdout --watch --timeout --file --listen --token --target --old --new --json"

    if [[ ${COMP_CWORD} -eq 1 ]]; then
        COMPREPLY=( $(compgen -W "${commands}" -- "${cur}") )
        return 0
    fi

    case "${prev}" in
        --format)
            COMPREPLY=( $(compgen -W "yaml json" -- "${cur}") )
            return 0
            ;;
        --target)
            COMPREPLY=( $(compgen -W "typescript dart swift" -- "${cur}") )
            return 0
            ;;
        --project|--output|--file|--config|--tool4d|--old|--new)
            COMPREPLY=( $(compgen -f -- "${cur}") )
            return 0
            ;;
    esac

    if [[ ${cur} == -* ]]; then
        COMPREPLY=( $(compgen -W "${flags}" -- "${cur}") )
        return 0
    fi
}

complete -F _4d_openapi_completions 4d-openapi
`
}

// GenerateZsh outputs Zsh completion script for 4d-openapi.
func GenerateZsh() string {
	return `#compdef 4d-openapi

_4d_openapi() {
    local -a commands
    commands=(
        'generate:Extract 4D metadata and generate OpenAPI 3.1 specification'
        'serve:Serve specification and interactive documentation UI'
        'inspect:Inspect project API surface and diagnostics'
        'dependencies:Manage and verify project component dependencies'
        'validate:Validate an OpenAPI 3.1 specification file'
        'diff:Compare two specifications and detect breaking changes'
        'export:Export client models (typescript, dart, swift)'
        'init:Initialize openapi.yaml project configuration template'
        'mock:Run an embedded mock REST server matching the specification'
        'completion:Generate shell autocompletion script'
        'version:Print version information'
    )

    if (( CURRENT == 2 )); then
        _describe 'command' commands
        return
    fi

    case "$words[2]" in
        generate)
            _arguments \
                '--project[Path to 4D project folder]:project:_files -/' \
                '--output[Output file path]:output:_files' \
                '--format[Output format]:format:(yaml json)' \
                '--config[Path to openapi.yaml]:config:_files' \
                '--tool4d[Path to tool4d binary]:tool4d:_files' \
                '--offline[Strictly offline mode]' \
                '--watch[Watch project files for automatic regeneration]' \
                '--strict[Strict mode]' \
                '--emit-ir[Emit intermediate representation JSON]' \
                '--emit-manifest[Emit generation manifest]'
            ;;
        export)
            _arguments \
                '--project[Path to 4D project folder]:project:_files -/' \
                '--target[Export target]:target:(typescript dart swift)' \
                '--output[Output file path]:output:_files'
            ;;
        diff)
            _arguments \
                '--old[Baseline specification]:old:_files' \
                '--new[Candidate specification]:new:_files' \
                '--strict[Strict mode for breaking changes]'
            ;;
        serve|mock)
            _arguments \
                '--project[Path to 4D project folder]:project:_files -/' \
                '--file[OpenAPI specification file]:file:_files' \
                '--listen[Listen host and port]:listen:'
            ;;
        validate)
            _arguments \
                '--file[OpenAPI specification file to validate]:file:_files'
            ;;
    esac
}

_4d_openapi "$@"
`
}

// GenerateFish outputs Fish completion script for 4d-openapi.
func GenerateFish() string {
	var sb strings.Builder
	sb.WriteString("# Fish completion for 4d-openapi\n\n")
	commands := []struct{ name, desc string }{
		{"generate", "Extract 4D metadata and generate OpenAPI 3.1 specification"},
		{"serve", "Serve specification and interactive documentation UI"},
		{"inspect", "Inspect project API surface and diagnostics"},
		{"dependencies", "Manage and verify project component dependencies"},
		{"validate", "Validate an OpenAPI 3.1 specification file"},
		{"diff", "Compare two specifications and detect breaking changes"},
		{"export", "Export client models (typescript, dart, swift)"},
		{"init", "Initialize openapi.yaml project configuration template"},
		{"mock", "Run an embedded mock REST server"},
		{"completion", "Generate shell autocompletion script"},
		{"version", "Print version information"},
	}

	for _, c := range commands {
		sb.WriteString(fmt.Sprintf("complete -c 4d-openapi -n '__fish_use_subcommand' -a %s -d '%s'\n", c.name, c.desc))
	}
	return sb.String()
}
