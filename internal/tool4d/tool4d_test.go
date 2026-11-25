package tool4d

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/miyako/OpenAPI/internal/diagnostics"
)

func TestTool4DRunner(t *testing.T) {
	col := diagnostics.NewCollector()

	// 1. Default timeout
	runner := NewRunner("", 0, col)
	if runner.Timeout != 30*time.Second {
		t.Errorf("Expected default timeout 30s, got %v", runner.Timeout)
	}

	// 2. Non-existent explicit path
	badRunner := NewRunner("/non/existent/path/tool4d", 10*time.Second, col)
	if _, err := badRunner.Locate(); err == nil {
		t.Errorf("Expected error locating non-existent tool4d")
	}

	// 3. Existing mock executable path
	tmpDir, err := os.MkdirTemp("", "tool4d-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mockBin := filepath.Join(tmpDir, "mock_tool4d")
	if err := os.WriteFile(mockBin, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatalf("Failed to create mock binary: %v", err)
	}

	mockRunner := NewRunner(mockBin, 5*time.Second, col)
	located, err := mockRunner.Locate()
	if err != nil {
		t.Fatalf("Failed to locate mock tool4d: %v", err)
	}
	if located != mockBin {
		t.Errorf("Expected located path %s, got %s", mockBin, located)
	}
}
