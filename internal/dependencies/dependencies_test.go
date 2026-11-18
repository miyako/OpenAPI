package dependencies

import (
	"path/filepath"
	"testing"

	"github.com/miyako/OpenAPI/internal/diagnostics"
)

func TestDependencyResolution(t *testing.T) {
	projDir := filepath.Join("..", "..", "testdata", "fixtures", "TestApp")
	col := diagnostics.NewCollector()
	res := NewResolver(projDir, false, false, col)
	deps, err := res.Resolve()
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if len(deps) != 1 {
		t.Fatalf("Expected 1 dependency, got %d", len(deps))
	}

	if deps[0].Name != "my-auth-component" {
		t.Errorf("Expected my-auth-component, got %s", deps[0].Name)
	}
}

func TestNoDependenciesFlag(t *testing.T) {
	projDir := filepath.Join("..", "..", "testdata", "fixtures", "TestApp")
	col := diagnostics.NewCollector()
	res := NewResolver(projDir, false, true, col)
	deps, err := res.Resolve()
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if len(deps) != 0 {
		t.Errorf("Expected 0 dependencies when NoDeps is true, got %d", len(deps))
	}
}
