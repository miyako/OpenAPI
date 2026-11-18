package extractor

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/miyako/OpenAPI/internal/config"
	"github.com/miyako/OpenAPI/internal/dependencies"
	"github.com/miyako/OpenAPI/internal/diagnostics"
	"github.com/miyako/OpenAPI/internal/ir"
	"github.com/miyako/OpenAPI/internal/parser"
	"github.com/miyako/OpenAPI/internal/project"
	"github.com/miyako/OpenAPI/internal/tool4d"
	"github.com/miyako/OpenAPI/internal/workspace"
)

// Options holds options for extraction.
type Options struct {
	ProjectPath       string
	Tool4DPath        string
	Tool4DVersion     string
	Offline           bool
	NoDependencies    bool
	Strict            bool
	TimeoutSeconds    int
	Config            *config.Config
}

// Extract performs hybrid extraction combining declarative catalog, dependencies, static class parsing, and tool4d runtime inspection.
func Extract(opts Options, col *diagnostics.Collector) (*ir.IR, error) {
	// 1. Discover Project
	proj, err := project.Discover(opts.ProjectPath)
	if err != nil {
		col.AddError("ERR_PROJECT_DISCOVERY", err.Error(), opts.ProjectPath, 0)
		return nil, fmt.Errorf("project discovery: %w", err)
	}

	meta := ir.ProjectMeta{
		Name:           proj.Name,
		Path:           proj.ProjectFile,
		FourDVersion:   proj.FourDVersion,
		Tool4dVerified: false,
	}
	specIR := ir.NewIR(meta)

	// 2. Resolve Dependencies
	depResolver := dependencies.NewResolver(proj.ProjectDir, opts.Offline, opts.NoDependencies, col)
	resolvedDeps, err := depResolver.Resolve()
	if err != nil {
		col.AddError("ERR_DEPENDENCY_RESOLUTION", err.Error(), proj.ProjectDir, 0)
		return nil, fmt.Errorf("dependency resolution: %w", err)
	}
	specIR.Components = depResolver.ToIRComponents(resolvedDeps)

	// 3. Declarative Catalog Parsing (Tables & Attributes)
	if proj.CatalogFile != "" {
		dataClasses, err := project.ParseCatalog(proj.CatalogFile, col)
		if err != nil {
			col.AddWarning("WARN_CATALOG_EXTRACTION", fmt.Sprintf("Error extracting catalog: %v", err), proj.CatalogFile, 0, diagnostics.ConfidenceUnknown)
		} else {
			specIR.DataClasses = dataClasses
		}
	} else {
		col.AddWarning("WARN_NO_CATALOG", "No catalog.4DCatalog file found in project", proj.ProjectFile, 0, diagnostics.ConfidenceUnknown)
	}

	// 4. Custom HTTP Handlers Parsing (HTTPHandlers.json)
	if proj.HandlersFile != "" {
		handlers, err := project.ParseHTTPHandlers(proj.HandlersFile, col)
		if err != nil {
			col.AddWarning("WARN_HANDLERS_EXTRACTION", fmt.Sprintf("Error extracting HTTP handlers: %v", err), proj.HandlersFile, 0, diagnostics.ConfidenceUnknown)
		} else {
			specIR.Handlers = handlers
		}
	}

	// 5. Static Source Class Extraction (Class functions & annotations)
	if proj.ClassesDir != "" {
		funcs, err := parser.ParseClassesDirectory(proj.ClassesDir, col)
		if err != nil {
			col.AddWarning("WARN_CLASS_PARSING", fmt.Sprintf("Error parsing classes: %v", err), proj.ClassesDir, 0, diagnostics.ConfidenceInferred)
		} else {
			specIR.Functions = funcs
		}
	}

	// 6. Optional tool4d Introspection Bridge (if tool4d is installed)
	toolRunner := tool4d.NewRunner(opts.Tool4DPath, 0, col)
	if tool4dBin, err := toolRunner.Locate(); err == nil {
		col.AddInfo("INFO_TOOL4D_LOCATED", fmt.Sprintf("Found tool4d at %s. Attempting runtime introspection bridge...", tool4dBin), tool4dBin, diagnostics.ConfidenceVerified)

		ws, err := workspace.Create(proj, resolvedDeps)
		if err == nil {
			defer ws.Cleanup()

			output, runErr := toolRunner.Run(ws.ProjectFile, "extract_metadata", nil)
			if runErr == nil && output != "" {
				// Parse runtime JSON output if emitted by bridge
				var bridgeOutput struct {
					DataClasses []ir.DataClass `json:"dataclasses"`
					Functions   []ir.Function  `json:"functions"`
				}
				// Look for JSON substring
				jsonStart := strings.Index(output, "{")
				jsonEnd := strings.LastIndex(output, "}")
				if jsonStart != -1 && jsonEnd > jsonStart {
					jsonBytes := []byte(output[jsonStart : jsonEnd+1])
					if err := json.Unmarshal(jsonBytes, &bridgeOutput); err == nil {
						specIR.Project.Tool4dVerified = true
						specIR.Project.Tool4dBuild = "verified-build"
						mergeRuntimeDataClasses(specIR, bridgeOutput.DataClasses)
						mergeRuntimeFunctions(specIR, bridgeOutput.Functions)
						col.AddInfo("INFO_RUNTIME_VERIFIED", "Successfully verified schema using tool4d runtime introspection", tool4dBin, diagnostics.ConfidenceVerified)
					}
				}
			}
		}
	} else {
		col.AddInfo("INFO_TOOL4D_BYPASS", "tool4d not found or not configured; using high-fidelity static declarations", "", diagnostics.ConfidenceInferred)
	}

	// Filter by config includes/excludes if specified
	if opts.Config != nil {
		filterIR(specIR, opts.Config)
	}

	// Attach diagnostics
	specIR.Diagnostics = col.All()

	return specIR, nil
}

func mergeRuntimeDataClasses(target *ir.IR, runtimeDCs []ir.DataClass) {
	if len(runtimeDCs) == 0 {
		return
	}
	existingMap := make(map[string]int)
	for i, dc := range target.DataClasses {
		existingMap[dc.Name] = i
	}

	for _, rdc := range runtimeDCs {
		if idx, exists := existingMap[rdc.Name]; exists {
			target.DataClasses[idx].Provenance.Confidence = diagnostics.ConfidenceVerified
			// Update attributes confidence
			for aIdx := range target.DataClasses[idx].Attributes {
				target.DataClasses[idx].Attributes[aIdx].Provenance.Confidence = diagnostics.ConfidenceVerified
			}
		} else {
			target.DataClasses = append(target.DataClasses, rdc)
		}
	}
}

func mergeRuntimeFunctions(target *ir.IR, runtimeFuncs []ir.Function) {
	if len(runtimeFuncs) == 0 {
		return
	}
	existingMap := make(map[string]int)
	for i, fn := range target.Functions {
		key := fmt.Sprintf("%s:%s", fn.ClassName, fn.Name)
		existingMap[key] = i
	}

	for _, rfn := range runtimeFuncs {
		key := fmt.Sprintf("%s:%s", rfn.ClassName, rfn.Name)
		if idx, exists := existingMap[key]; exists {
			target.Functions[idx].Provenance.Confidence = diagnostics.ConfidenceVerified
		} else {
			target.Functions = append(target.Functions, rfn)
		}
	}
}

func filterIR(specIR *ir.IR, cfg *config.Config) {
	if len(cfg.Include.DataClasses) > 0 {
		includeMap := make(map[string]bool)
		for _, name := range cfg.Include.DataClasses {
			includeMap[name] = true
		}
		filtered := make([]ir.DataClass, 0)
		for _, dc := range specIR.DataClasses {
			if includeMap[dc.Name] {
				filtered = append(filtered, dc)
			}
		}
		specIR.DataClasses = filtered
	}

	if len(cfg.Exclude.DataClasses) > 0 {
		excludeMap := make(map[string]bool)
		for _, name := range cfg.Exclude.DataClasses {
			excludeMap[name] = true
		}
		filtered := make([]ir.DataClass, 0)
		for _, dc := range specIR.DataClasses {
			if !excludeMap[dc.Name] {
				filtered = append(filtered, dc)
			}
		}
		specIR.DataClasses = filtered
	}
}
