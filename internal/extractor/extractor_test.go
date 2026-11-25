package extractor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/miyako/OpenAPI/internal/diagnostics"
)

func TestExtract(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "extractor-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	projDir := filepath.Join(tmpDir, "Project")
	sourcesDir := filepath.Join(projDir, "Sources")
	classesDir := filepath.Join(sourcesDir, "Classes")
	dbDir := filepath.Join(sourcesDir, "Database")
	_ = os.MkdirAll(classesDir, 0755)
	_ = os.MkdirAll(dbDir, 0755)

	projFile := filepath.Join(projDir, "TestApp.4DProject")
	if err := os.WriteFile(projFile, []byte("<project/>"), 0644); err != nil {
		t.Fatalf("Failed to write project file: %v", err)
	}

	catalogXML := `<?xml version="1.0" encoding="UTF-8"?>
<database>
  <table name="Customer" id="1" uuid="cust-uuid" exposed="true">
    <field name="ID" id="1" type="4" autosequence="true" primary_key="true"/>
    <field name="name" id="2" type="2" exposed="true"/>
  </table>
</database>`
	catalogFile := filepath.Join(dbDir, "catalog.4DCatalog")
	if err := os.WriteFile(catalogFile, []byte(catalogXML), 0644); err != nil {
		t.Fatalf("Failed to write catalog file: %v", err)
	}

	classCode := `// Customer entity class
exposed Function getFullName() : Text
    return "Customer Name"
`
	classFile := filepath.Join(classesDir, "Customer.4dm")
	if err := os.WriteFile(classFile, []byte(classCode), 0644); err != nil {
		t.Fatalf("Failed to write class file: %v", err)
	}

	col := diagnostics.NewCollector()
	opts := Options{
		ProjectPath:    projFile,
		NoDependencies: true,
		Offline:        true,
	}

	extractedIR, err := Extract(opts, col)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if extractedIR.Project.Name != "TestApp" {
		t.Errorf("Expected project name TestApp, got %s", extractedIR.Project.Name)
	}

	if len(extractedIR.DataClasses) != 1 {
		for _, d := range col.All() {
			t.Logf("DIAG: [%s] %s (%s)", d.Code, d.Message, d.Source)
		}
		t.Fatalf("Expected 1 DataClass, got %d", len(extractedIR.DataClasses))
	}

	if extractedIR.DataClasses[0].Name != "Customer" {
		t.Errorf("Expected Customer dataclass, got %s", extractedIR.DataClasses[0].Name)
	}

	if len(extractedIR.Functions) != 1 {
		t.Fatalf("Expected 1 Function, got %d", len(extractedIR.Functions))
	}

	if extractedIR.Functions[0].Name != "getFullName" {
		t.Errorf("Expected getFullName function, got %s", extractedIR.Functions[0].Name)
	}
}
