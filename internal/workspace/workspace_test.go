package workspace

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/miyako/OpenAPI/internal/project"
)

func TestWorkspaceCreateAndCleanup(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ws-test-src-*")
	if err != nil {
		t.Fatalf("Failed to create temp src: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	projDir := filepath.Join(tmpDir, "Project")
	_ = os.MkdirAll(filepath.Join(projDir, "Sources"), 0755)
	_ = os.MkdirAll(filepath.Join(projDir, ".git"), 0755)

	projFile := filepath.Join(projDir, "TestApp.4DProject")
	if err := os.WriteFile(projFile, []byte("<project/>"), 0644); err != nil {
		t.Fatalf("Failed to write project file: %v", err)
	}

	gitFile := filepath.Join(projDir, ".git", "config")
	if err := os.WriteFile(gitFile, []byte("[core]"), 0644); err != nil {
		t.Fatalf("Failed to write git file: %v", err)
	}

	sourceFile := filepath.Join(projDir, "Sources", "Test.4dm")
	if err := os.WriteFile(sourceFile, []byte("// code"), 0644); err != nil {
		t.Fatalf("Failed to write source file: %v", err)
	}

	projInfo := &project.ProjectInfo{
		Name:        "TestApp",
		ProjectDir:  projDir,
		ProjectFile: projFile,
	}

	ws, err := Create(projInfo, nil)
	if err != nil {
		t.Fatalf("Workspace Create failed: %v", err)
	}

	if ws.Dir == "" {
		t.Fatalf("Workspace Dir is empty")
	}

	// Verify project file exists in isolated workspace
	if _, err := os.Stat(ws.ProjectFile); err != nil {
		t.Errorf("Project file missing in workspace: %v", err)
	}

	// Verify source was copied
	if _, err := os.Stat(filepath.Join(ws.Dir, "Sources", "Test.4dm")); err != nil {
		t.Errorf("Source file not copied to workspace: %v", err)
	}

	// Verify .git was skipped
	if _, err := os.Stat(filepath.Join(ws.Dir, ".git")); !os.IsNotExist(err) {
		t.Errorf("Expected .git to be excluded from workspace")
	}

	// Cleanup
	wsDir := ws.Dir
	ws.Cleanup()

	if _, err := os.Stat(wsDir); !os.IsNotExist(err) {
		t.Errorf("Workspace directory was not cleaned up")
	}
}
