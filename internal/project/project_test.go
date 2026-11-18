package project

import (
	"path/filepath"
	"testing"

	"github.com/miyako/OpenAPI/internal/diagnostics"
)

func TestProjectDiscovery(t *testing.T) {
	fixtureDir := filepath.Join("..", "..", "testdata", "fixtures", "TestApp")
	proj, err := Discover(fixtureDir)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if proj.Name != "TestApp" {
		t.Errorf("Expected project name 'TestApp', got '%s'", proj.Name)
	}

	if proj.CatalogFile == "" {
		t.Errorf("Expected non-empty CatalogFile")
	}

	if proj.ClassesDir == "" {
		t.Errorf("Expected non-empty ClassesDir")
	}

	if proj.HandlersFile == "" {
		t.Errorf("Expected non-empty HandlersFile")
	}
}

func TestParseCatalog(t *testing.T) {
	catalogPath := filepath.Join("..", "..", "testdata", "fixtures", "TestApp", "Project", "Database", "catalog.4DCatalog")
	col := diagnostics.NewCollector()
	dataClasses, err := ParseCatalog(catalogPath, col)
	if err != nil {
		t.Fatalf("ParseCatalog failed: %v", err)
	}

	if len(dataClasses) != 2 {
		t.Fatalf("Expected 2 DataClasses, got %d", len(dataClasses))
	}

	dcMap := make(map[string]int)
	for _, dc := range dataClasses {
		dcMap[dc.Name] = len(dc.Attributes)
	}

	if dcMap["Company"] != 7 {
		t.Errorf("Expected 7 attributes for Company (including relations), got %d", dcMap["Company"])
	}
	if dcMap["Employee"] != 6 {
		t.Errorf("Expected 6 attributes for Employee (including relations), got %d", dcMap["Employee"])
	}
}

func TestParseHTTPHandlers(t *testing.T) {
	handlersPath := filepath.Join("..", "..", "testdata", "fixtures", "TestApp", "Project", "Sources", "HTTPHandlers.json")
	col := diagnostics.NewCollector()
	handlers, err := ParseHTTPHandlers(handlersPath, col)
	if err != nil {
		t.Fatalf("ParseHTTPHandlers failed: %v", err)
	}

	if len(handlers) != 2 {
		t.Fatalf("Expected 2 handlers, got %d", len(handlers))
	}

	var hasRegex bool
	for _, h := range handlers {
		if h.IsRegex {
			hasRegex = true
		}
	}

	if !hasRegex {
		t.Errorf("Expected at least one regex handler")
	}
}
