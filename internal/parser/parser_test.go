package parser

import (
	"path/filepath"
	"testing"

	"github.com/miyako/OpenAPI/internal/diagnostics"
	"github.com/miyako/OpenAPI/internal/ir"
)

func TestParseClassesDirectory(t *testing.T) {
	classesDir := filepath.Join("..", "..", "testdata", "fixtures", "TestApp", "Project", "Sources", "Classes")
	col := diagnostics.NewCollector()
	funcs, err := ParseClassesDirectory(classesDir, col)
	if err != nil {
		t.Fatalf("ParseClassesDirectory failed: %v", err)
	}

	if len(funcs) != 5 {
		t.Fatalf("Expected 5 exposed class functions, got %d", len(funcs))
	}

	targets := make(map[ir.FunctionTarget]bool)
	for _, fn := range funcs {
		targets[fn.Target] = true
	}

	// Verify all 5 target surfaces are discovered
	expectedTargets := []ir.FunctionTarget{
		ir.TargetDataStore,
		ir.TargetDataClass,
		ir.TargetEntity,
		ir.TargetEntitySelection,
		ir.TargetSingleton,
	}

	for _, exp := range expectedTargets {
		if !targets[exp] {
			t.Errorf("Expected function target %s to be parsed", exp)
		}
	}
}
