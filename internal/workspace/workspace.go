package workspace

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/miyako/OpenAPI/internal/dependencies"
	"github.com/miyako/OpenAPI/internal/project"
)

// Workspace encapsulates an isolated copy of the project for safe tool4d execution.
type Workspace struct {
	Dir         string
	ProjectFile string
}

// Create prepares an isolated analysis workspace in a temporary directory.
func Create(proj *project.ProjectInfo, resolvedDeps []dependencies.ResolvedComponent) (*Workspace, error) {
	tempDir, err := os.MkdirTemp("", "4d-openapi-ws-*")
	if err != nil {
		return nil, fmt.Errorf("create temporary workspace: %w", err)
	}

	ws := &Workspace{
		Dir: tempDir,
	}

	// Copy project contents safely (avoiding .git, userPreferences, etc.)
	srcBase := proj.ProjectDir
	err = filepath.Walk(srcBase, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(srcBase, path)
		if err != nil {
			return err
		}

		// Skip VCS and local caches
		if strings.HasPrefix(rel, ".git") || strings.HasPrefix(rel, "userPreferences") || strings.Contains(rel, ".DS_Store") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		destPath := filepath.Join(tempDir, rel)
		if info.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}

		// Don't follow arbitrary symlinks outside workspace
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}

		return copyFile(path, destPath)
	})

	if err != nil {
		ws.Cleanup()
		return nil, fmt.Errorf("populate workspace: %w", err)
	}

	// Locate copied .4DProject in workspace
	relProjFile, err := filepath.Rel(srcBase, proj.ProjectFile)
	if err == nil {
		ws.ProjectFile = filepath.Join(tempDir, relProjFile)
	}

	// Ensure Components directory exists and link/copy dependencies
	if len(resolvedDeps) > 0 {
		wsCompDir := filepath.Join(tempDir, "Components")
		_ = os.MkdirAll(wsCompDir, 0755)
		for _, dep := range resolvedDeps {
			if dep.Source == "environment" || dep.Source == "local" {
				depDest := filepath.Join(wsCompDir, dep.Name)
				// Copy or link local component
				if stat, err := os.Stat(dep.Path); err == nil && stat.IsDir() {
					_ = copyDir(dep.Path, depDest)
				}
			}
		}
	}

	return ws, nil
}

// Cleanup safely removes the temporary analysis workspace.
func (ws *Workspace) Cleanup() {
	if ws.Dir != "" && strings.Contains(ws.Dir, "4d-openapi-ws-") {
		_ = os.RemoveAll(ws.Dir)
	}
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		return copyFile(path, target)
	})
}
