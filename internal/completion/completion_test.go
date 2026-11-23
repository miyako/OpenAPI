package completion

import (
	"strings"
	"testing"
)

func TestCompletions(t *testing.T) {
	bash := GenerateBash()
	if !strings.Contains(bash, "_4d_openapi_completions") {
		t.Errorf("Expected bash completion function, got:\n%s", bash)
	}

	zsh := GenerateZsh()
	if !strings.Contains(zsh, "_4d_openapi") {
		t.Errorf("Expected zsh completion function, got:\n%s", zsh)
	}

	fish := GenerateFish()
	if !strings.Contains(fish, "complete -c 4d-openapi") {
		t.Errorf("Expected fish completion script, got:\n%s", fish)
	}
}
