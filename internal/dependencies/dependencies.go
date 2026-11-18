package dependencies

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/miyako/OpenAPI/internal/diagnostics"
	"github.com/miyako/OpenAPI/internal/ir"
)

// DependencyDeclaration defines a single dependency in dependencies.json.
type DependencyDeclaration struct {
	GitHub  string `json:"github,omitempty"`
	Version string `json:"version,omitempty"`
	Path    string `json:"path,omitempty"`
}

// DependenciesFile represents dependencies.json.
type DependenciesFile struct {
	Dependencies map[string]DependencyDeclaration `json:"dependencies"`
}

// Environment4D represents environment4d.json.
type Environment4D struct {
	Dependencies map[string]string `json:"dependencies"`
}

// LockFile represents lockfile structure stored in cache.
type LockFile struct {
	ProjectID    string                         `json:"projectId"`
	Dependencies map[string]LockedDependency    `json:"dependencies"`
}

type LockedDependency struct {
	Version string `json:"version"`
	Source  string `json:"source"`
	Path    string `json:"path"`
	SHA256  string `json:"sha256,omitempty"`
}

// Resolver manages resolution and caching of 4D dependencies.
type Resolver struct {
	ProjectDir  string
	CacheDir    string
	Offline     bool
	NoDeps      bool
	Collector   *diagnostics.Collector
}

// NewResolver creates a new dependency resolver.
func NewResolver(projectDir string, offline, noDeps bool, col *diagnostics.Collector) *Resolver {
	home, _ := os.UserHomeDir()
	cacheDir := filepath.Join(home, ".cache", "4d-openapi")
	return &Resolver{
		ProjectDir: projectDir,
		CacheDir:   cacheDir,
		Offline:    offline,
		NoDeps:     noDeps,
		Collector:  col,
	}
}

// ResolvedComponent represents a resolved component.
type ResolvedComponent struct {
	Name    string
	Version string
	Source  string // "environment", "local", "github"
	Path    string
	SHA256  string
}

// Resolve resolves all project dependencies based on 4D precedence rules.
func (r *Resolver) Resolve() ([]ResolvedComponent, error) {
	if r.NoDeps {
		return nil, nil
	}

	resolved := make(map[string]ResolvedComponent)

	// 1. Check environment4d.json
	envFile := filepath.Join(r.ProjectDir, "environment4d.json")
	if data, err := os.ReadFile(envFile); err == nil {
		var env Environment4D
		if err := json.Unmarshal(data, &env); err == nil {
			for name, rawPath := range env.Dependencies {
				targetPath := rawPath
				if !filepath.IsAbs(targetPath) {
					targetPath = filepath.Clean(filepath.Join(r.ProjectDir, targetPath))
				}
				if _, err := os.Stat(targetPath); err != nil {
					r.Collector.AddError("ERR_DEP_ENV_MISSING", fmt.Sprintf("environment4d.json specifies missing path for %s: %s", name, targetPath), envFile, 0)
					return nil, fmt.Errorf("missing environment dependency %s at %s", name, targetPath)
				}
				resolved[name] = ResolvedComponent{
					Name:   name,
					Source: "environment",
					Path:   targetPath,
				}
			}
		}
	}

	// 2. Check local Components/ folder
	compDir := filepath.Join(r.ProjectDir, "Components")
	if entries, err := os.ReadDir(compDir); err == nil {
		for _, entry := range entries {
			name := entry.Name()
			cleanName := strings.TrimSuffix(name, filepath.Ext(name))
			if _, exists := resolved[cleanName]; !exists {
				fullPath := filepath.Join(compDir, name)
				resolved[cleanName] = ResolvedComponent{
					Name:   cleanName,
					Source: "local",
					Path:   fullPath,
				}
			}
		}
	}

	// 3. Check dependencies.json
	depFile := filepath.Join(r.ProjectDir, "dependencies.json")
	if data, err := os.ReadFile(depFile); err == nil {
		var deps DependenciesFile
		if err := json.Unmarshal(data, &deps); err == nil {
			for name, decl := range deps.Dependencies {
				if _, exists := resolved[name]; exists {
					continue
				}

				if decl.Path != "" {
					localPath := decl.Path
					if !filepath.IsAbs(localPath) {
						localPath = filepath.Clean(filepath.Join(r.ProjectDir, localPath))
					}
					// Reject path traversal escaping allowed boundaries if not resolved
					if _, err := os.Stat(localPath); err == nil {
						resolved[name] = ResolvedComponent{
							Name:    name,
							Source:  "local",
							Path:    localPath,
							Version: decl.Version,
						}
					} else {
						r.Collector.AddWarning("WARN_DEP_LOCAL_MISSING", fmt.Sprintf("Local component '%s' not found at %s", name, localPath), depFile, 0, diagnostics.ConfidenceUnknown)
					}
				} else if decl.GitHub != "" {
					// Check cached archive in ~/.cache/4d-openapi/archives/
					archiveDir := filepath.Join(r.CacheDir, "archives", name)
					cachedArchive := filepath.Join(archiveDir, decl.Version+".zip")

					if _, err := os.Stat(cachedArchive); err == nil {
						// verify digest if .sha256 exists
						shaFile := cachedArchive + ".sha256"
						expectedSHA, _ := os.ReadFile(shaFile)
						actualSHA, _ := calculateSHA256(cachedArchive)
						if len(expectedSHA) > 0 && strings.TrimSpace(string(expectedSHA)) != actualSHA {
							r.Collector.AddError("ERR_DEP_CORRUPT_ARCHIVE", fmt.Sprintf("Cached archive checksum mismatch for %s", name), cachedArchive, 0)
							return nil, fmt.Errorf("checksum mismatch for cached component %s", name)
						}
						resolved[name] = ResolvedComponent{
							Name:    name,
							Version: decl.Version,
							Source:  "github",
							Path:    cachedArchive,
							SHA256:  actualSHA,
						}
					} else {
						if r.Offline {
							r.Collector.AddWarning("WARN_DEP_OFFLINE", fmt.Sprintf("Offline mode: cannot download dependency %s (%s)", name, decl.GitHub), depFile, 0, diagnostics.ConfidenceUnknown)
						} else {
							r.Collector.AddInfo("INFO_DEP_REMOTE", fmt.Sprintf("Remote component %s declared (%s@%s)", name, decl.GitHub, decl.Version), depFile, diagnostics.ConfidenceInferred)
							resolved[name] = ResolvedComponent{
								Name:    name,
								Version: decl.Version,
								Source:  "github",
								Path:    decl.GitHub,
							}
						}
					}
				}
			}
		}
	}

	result := make([]ResolvedComponent, 0, len(resolved))
	for _, c := range resolved {
		result = append(result, c)
	}

	return result, nil
}

// ToIRComponents converts resolved components into IR Component models.
func (r *Resolver) ToIRComponents(list []ResolvedComponent) []ir.Component {
	out := make([]ir.Component, len(list))
	for i, c := range list {
		out[i] = ir.Component{
			Name:    c.Name,
			Version: c.Version,
			Source:  c.Source,
			Path:    c.Path,
		}
	}
	return out
}

func calculateSHA256(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}
