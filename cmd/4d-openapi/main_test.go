package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/miyako/OpenAPI/internal/config"
	"github.com/miyako/OpenAPI/internal/diagnostics"
	"github.com/miyako/OpenAPI/internal/extractor"
	"github.com/miyako/OpenAPI/internal/mapping"
)

func TestGoldenDeterminism(t *testing.T) {
	fixtureDir := filepath.Join("..", "..", "testdata", "fixtures", "TestApp")
	goldenDir := filepath.Join("..", "..", "testdata", "golden")

	col1 := diagnostics.NewCollector()
	cfg1, _ := config.LoadConfig(filepath.Join(fixtureDir, "openapi.yaml"))
	ir1, err := extractor.Extract(extractor.Options{ProjectPath: fixtureDir, Config: cfg1}, col1)
	if err != nil {
		t.Fatalf("Extract 1 failed: %v", err)
	}

	spec1, err := mapping.MapIRToOpenAPI(ir1, cfg1, col1)
	if err != nil {
		t.Fatalf("MapIRToOpenAPI 1 failed: %v", err)
	}

	yaml1, err := spec1.ToYAML()
	if err != nil {
		t.Fatalf("ToYAML 1 failed: %v", err)
	}

	json1, err := spec1.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON 1 failed: %v", err)
	}

	// Run 2: ensure identical bytes (determinism requirement)
	col2 := diagnostics.NewCollector()
	cfg2, _ := config.LoadConfig(filepath.Join(fixtureDir, "openapi.yaml"))
	ir2, err := extractor.Extract(extractor.Options{ProjectPath: fixtureDir, Config: cfg2}, col2)
	if err != nil {
		t.Fatalf("Extract 2 failed: %v", err)
	}

	spec2, err := mapping.MapIRToOpenAPI(ir2, cfg2, col2)
	if err != nil {
		t.Fatalf("MapIRToOpenAPI 2 failed: %v", err)
	}

	yaml2, err := spec2.ToYAML()
	if err != nil {
		t.Fatalf("ToYAML 2 failed: %v", err)
	}

	json2, err := spec2.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON 2 failed: %v", err)
	}

	if !bytes.Equal(yaml1, yaml2) {
		t.Errorf("Determinism failure: YAML outputs between two runs are not byte-identical")
	}

	if !bytes.Equal(json1, json2) {
		t.Errorf("Determinism failure: JSON outputs between two runs are not byte-identical")
	}

	// Verify against golden files
	expectedYAML, err := os.ReadFile(filepath.Join(goldenDir, "expected-openapi.yaml"))
	if err == nil {
		if !bytes.Equal(yaml1, expectedYAML) {
			t.Errorf("Generated YAML does not match expected-openapi.yaml golden file")
		}
	}

	expectedJSON, err := os.ReadFile(filepath.Join(goldenDir, "expected-openapi.json"))
	if err == nil {
		if !bytes.Equal(json1, expectedJSON) {
			t.Errorf("Generated JSON does not match expected-openapi.json golden file")
		}
	}
}
