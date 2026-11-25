package openapi

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/miyako/OpenAPI/internal/diagnostics"
)

func TestApplyOverlay(t *testing.T) {
	spec := NewSpec("OriginalTitle", "1.0.0", "OriginalDescription")
	spec.Paths["/rest/Company"] = PathItem{
		Get: &Operation{
			OperationID: "listCompany",
			Summary:     "Original summary",
		},
	}

	tmpDir, err := os.MkdirTemp("", "overlay-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	overlayYAML := `overlay: 1.0.0
info:
  title: Overridden Title
  description: Overridden Description
`
	overlayPath := filepath.Join(tmpDir, "openapi-overlay.yaml")
	if err := os.WriteFile(overlayPath, []byte(overlayYAML), 0644); err != nil {
		t.Fatalf("Failed to write overlay file: %v", err)
	}

	col := diagnostics.NewCollector()
	if err := ApplyOverlay(spec, overlayPath, col); err != nil {
		t.Fatalf("ApplyOverlay failed: %v", err)
	}

	if spec.Info.Title != "Overridden Title" {
		t.Errorf("Expected title 'Overridden Title', got '%s'", spec.Info.Title)
	}
	if spec.Info.Description != "Overridden Description" {
		t.Errorf("Expected description 'Overridden Description', got '%s'", spec.Info.Description)
	}
}
