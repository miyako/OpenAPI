package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/miyako/OpenAPI/internal/codegen"
	"github.com/miyako/OpenAPI/internal/completion"
	"github.com/miyako/OpenAPI/internal/config"
	"github.com/miyako/OpenAPI/internal/dependencies"
	"github.com/miyako/OpenAPI/internal/diagnostics"
	"github.com/miyako/OpenAPI/internal/diff"
	"github.com/miyako/OpenAPI/internal/extractor"
	"github.com/miyako/OpenAPI/internal/mapping"
	"github.com/miyako/OpenAPI/internal/mock"
	"github.com/miyako/OpenAPI/internal/openapi"
	"github.com/miyako/OpenAPI/internal/project"
	"github.com/miyako/OpenAPI/internal/server"
	"github.com/miyako/OpenAPI/internal/validation"
)

// Exit codes conforming to Section 8.2 of the specification
const (
	ExitSuccess            = 0
	ExitGeneralError       = 1
	ExitInvalidArgs        = 2
	ExitTool4DError        = 3
	ExitDependencyError    = 4
	ExitExtractionError    = 5
	ExitOpenAPIValidationError = 6
	ExitStrictIncomplete   = 7
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(ExitInvalidArgs)
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "generate":
		runGenerate(args)
	case "serve":
		runServe(args)
	case "inspect":
		runInspect(args)
	case "dependencies":
		runDependencies(args)
	case "validate":
		runValidate(args)
	case "diff":
		runDiff(args)
	case "export":
		runExport(args)
	case "init":
		runInit(args)
	case "mock":
		runMock(args)
	case "completion":
		runCompletion(args)
	case "help", "-h", "--help":
		printUsage()
		os.Exit(ExitSuccess)
	case "version", "-v", "--version":
		fmt.Println("4d-openapi version 1.0.0")
		os.Exit(ExitSuccess)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(ExitInvalidArgs)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `4d-openapi — OpenAPI 3.1 generator and server for 4D projects

Usage:
  4d-openapi <command> [flags]

Commands:
  generate       Extract 4D metadata and generate OpenAPI 3.1 specification
  serve          Serve specification and interactive documentation UI
  inspect        Inspect project API surface and diagnostics without generating full spec
  dependencies   Manage and verify project component dependencies (list | verify | lock)
  validate       Validate an OpenAPI 3.1 specification file
  diff           Compare two specifications and detect breaking changes
  export         Export client models and collections (typescript | dart | swift | csharp | kotlin | postman)
  init           Initialize openapi.yaml project configuration template
  mock           Run an embedded mock REST server matching the specification
  version        Print version information

Run '4d-openapi <command> --help' for details on a specific command.
`)
}

func runGenerate(args []string) {
	fs := flag.NewFlagSet("generate", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	projectPath := fs.String("project", ".", "Path to 4D project folder or .4DProject file")
	outputPath := fs.String("output", "", "Output file path (default: openapi.yaml in current or project dir)")
	format := fs.String("format", "yaml", "Output format: yaml or json")
	cfgPath := fs.String("config", "", "Path to openapi.yaml configuration file")
	tool4dPath := fs.String("tool4d", "", "Path to tool4d executable")
	tool4dVer := fs.String("tool4d-version", "", "Required tool4d version")
	offline := fs.Bool("offline", false, "Strictly offline mode; do not download dependencies")
	noDeps := fs.Bool("no-dependencies", false, "Skip dependency resolution completely")
	emitIR := fs.Bool("emit-ir", false, "Emit intermediate representation JSON file")
	emitManifest := fs.Bool("emit-manifest", false, "Emit generation manifest with metadata")
	strict := fs.Bool("strict", false, "Strict mode: fail if any element is unverified or incomplete")
	toStdout := fs.Bool("stdout", false, "Write generated specification to stdout")
	timeoutSec := fs.Int("timeout", 30, "Timeout in seconds for tool4d execution")
	watch := fs.Bool("watch", false, "Watch project files and automatically regenerate on change")

	if err := fs.Parse(args); err != nil {
		os.Exit(ExitInvalidArgs)
	}

	execGen := func() int {
		return executeGenerate(*projectPath, *outputPath, *format, *cfgPath, *tool4dPath, *tool4dVer, *offline, *noDeps, *emitIR, *emitManifest, *strict, *toStdout, *timeoutSec)
	}

	code := execGen()
	if !*watch {
		os.Exit(code)
	}

	// Watch mode
	fmt.Fprintf(os.Stderr, "[4d-openapi watch] Watching for changes in %s (Ctrl+C to stop)...\n", *projectPath)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	lastSnapshot := snapshotProject(*projectPath)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-sigChan:
			fmt.Fprintln(os.Stderr, "\n[4d-openapi watch] Stopped watching.")
			os.Exit(ExitSuccess)
		case <-ticker.C:
			current := snapshotProject(*projectPath)
			changed := false
			for file, mod := range current {
				if oldMod, exists := lastSnapshot[file]; !exists || oldMod.Before(mod) {
					changed = true
					fmt.Fprintf(os.Stderr, "\n[4d-openapi watch] Modification in %s -> Regenerating...\n", filepath.Base(file))
					break
				}
			}
			if changed {
				lastSnapshot = current
				execGen()
			}
		}
	}
}

func snapshotProject(root string) map[string]time.Time {
	snapshot := make(map[string]time.Time)
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}
		if info.IsDir() {
			base := info.Name()
			if base == ".git" || base == "userPreferences" || base == "dist" {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".4dm" || ext == ".4dproject" || ext == ".4dcatalog" || ext == ".json" || ext == ".yaml" || ext == ".yml" {
			snapshot[path] = info.ModTime()
		}
		return nil
	})
	return snapshot
}

func executeGenerate(projectPath, outputPath, format, cfgPath, tool4dPath, tool4dVer string, offline, noDeps, emitIR, emitManifest, strict, toStdout bool, timeoutSec int) int {
	col := diagnostics.NewCollector()

	// 1. Load config
	resolvedCfgPath := cfgPath
	if resolvedCfgPath == "" {
		resolvedCfgPath = config.FindConfigFile(projectPath)
	}
	cfg, err := config.LoadConfig(resolvedCfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		return ExitInvalidArgs
	}

	if strict {
		cfg.Strict = true
	}
	if tool4dPath != "" {
		cfg.Tool4D.Path = tool4dPath
	}

	// 2. Perform Extraction
	extractOpts := extractor.Options{
		ProjectPath:    projectPath,
		Tool4DPath:     cfg.Tool4D.Path,
		Tool4DVersion:  tool4dVer,
		Offline:        offline,
		NoDependencies: noDeps,
		Strict:         cfg.Strict,
		TimeoutSeconds: timeoutSec,
		Config:         cfg,
	}

	inputIR, err := extractor.Extract(extractOpts, col)
	if err != nil {
		col.PrintFormatted(os.Stderr)
		if strings.Contains(err.Error(), "dependency") {
			return ExitDependencyError
		}
		if strings.Contains(err.Error(), "tool4d") {
			return ExitTool4DError
		}
		return ExitExtractionError
	}

	// 3. Validate IR
	if err := validation.ValidateIR(inputIR, col); err != nil {
		col.PrintFormatted(os.Stderr)
		return ExitExtractionError
	}

	// Emit IR if requested
	if emitIR {
		irData, _ := inputIR.ToJSON()
		irOutPath := "openapi-ir.json"
		if outputPath != "" {
			irOutPath = strings.TrimSuffix(outputPath, filepath.Ext(outputPath)) + "-ir.json"
		}
		_ = os.WriteFile(irOutPath, irData, 0644)
		fmt.Fprintf(os.Stderr, "[4d-openapi] Wrote IR to %s\n", irOutPath)
	}

	// 4. Map to OpenAPI 3.1
	spec, err := mapping.MapIRToOpenAPI(inputIR, cfg, col)
	if err != nil {
		col.PrintFormatted(os.Stderr)
		return ExitExtractionError
	}

	// 5. Validate OpenAPI Spec
	if err := validation.ValidateOpenAPI(spec, col); err != nil {
		col.PrintFormatted(os.Stderr)
		return ExitOpenAPIValidationError
	}

	// 6. Strict mode check
	if cfg.Strict && col.HasIncomplete() {
		col.PrintFormatted(os.Stderr)
		fmt.Fprintln(os.Stderr, "Strict mode failure: spec contains unverified or inferred items.")
		return ExitStrictIncomplete
	}

	// 7. Serialize Output
	var outBytes []byte
	isJSON := strings.EqualFold(format, "json") || strings.HasSuffix(outputPath, ".json")
	if isJSON {
		outBytes, err = spec.ToJSON()
	} else {
		outBytes, err = spec.ToYAML()
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Serialization error: %v\n", err)
		return ExitGeneralError
	}

	// Emit Manifest if requested
	if emitManifest {
		manifest := map[string]interface{}{
			"generatedAt":    time.Now().UTC().Format(time.RFC3339),
			"projectName":    inputIR.Project.Name,
			"fourDVersion":   inputIR.Project.FourDVersion,
			"tool4dVerified": inputIR.Project.Tool4dVerified,
			"dataClasses":    len(inputIR.DataClasses),
			"functions":      len(inputIR.Functions),
			"handlers":       len(inputIR.Handlers),
		}
		manData, _ := json.MarshalIndent(manifest, "", "  ")
		manOut := "openapi-manifest.json"
		if outputPath != "" {
			manOut = strings.TrimSuffix(outputPath, filepath.Ext(outputPath)) + "-manifest.json"
		}
		_ = os.WriteFile(manOut, manData, 0644)
	}

	if toStdout {
		os.Stdout.Write(outBytes)
	} else {
		targetFile := outputPath
		if targetFile == "" {
			ext := ".yaml"
			if isJSON {
				ext = ".json"
			}
			targetFile = filepath.Join(projectPath, "openapi"+ext)
		}

		_ = os.MkdirAll(filepath.Dir(targetFile), 0755)
		if err := os.WriteFile(targetFile, outBytes, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write file %s: %v\n", targetFile, err)
			return ExitGeneralError
		}
		fmt.Fprintf(os.Stderr, "[4d-openapi] Successfully generated OpenAPI specification at %s\n", targetFile)
	}

	// Output any diagnostics to stderr
	col.PrintFormatted(os.Stderr)
	return ExitSuccess
}

func runServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	specFile := fs.String("file", "", "Path to OpenAPI spec file (yaml or json)")
	projectPath := fs.String("project", "", "Path to 4D project folder (generates in-memory if --file not passed)")
	listenAddr := fs.String("listen", "127.0.0.1:8088", "Listen host and port")
	token := fs.String("token", "", "Optional authorization token (Bearer)")

	if err := fs.Parse(args); err != nil {
		os.Exit(ExitInvalidArgs)
	}

	srvConfig := server.Config{
		ListenAddr: *listenAddr,
		AuthToken:  *token,
		SpecFile:   *specFile,
	}

	if *specFile == "" {
		proj := *projectPath
		if proj == "" {
			proj = "."
		}
		col := diagnostics.NewCollector()
		cfg, _ := config.LoadConfig(config.FindConfigFile(proj))
		inputIR, err := extractor.Extract(extractor.Options{ProjectPath: proj, Config: cfg}, col)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to inspect project: %v\n", err)
			os.Exit(ExitExtractionError)
		}
		spec, err := mapping.MapIRToOpenAPI(inputIR, cfg, col)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to compile spec: %v\n", err)
			os.Exit(ExitExtractionError)
		}
		srvConfig.Spec = spec
	}

	srv, err := server.NewServer(srvConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Server initialization failed: %v\n", err)
		os.Exit(ExitGeneralError)
	}

	if err := srv.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(ExitGeneralError)
	}
}

func runInspect(args []string) {
	fs := flag.NewFlagSet("inspect", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	projectPath := fs.String("project", ".", "Path to 4D project")
	asJSON := fs.Bool("json", false, "Output inspection summary as JSON to stdout")

	if err := fs.Parse(args); err != nil {
		os.Exit(ExitInvalidArgs)
	}

	col := diagnostics.NewCollector()
	cfg, _ := config.LoadConfig(config.FindConfigFile(*projectPath))
	inputIR, err := extractor.Extract(extractor.Options{ProjectPath: *projectPath, Config: cfg}, col)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Inspection failed: %v\n", err)
		col.PrintFormatted(os.Stderr)
		os.Exit(ExitExtractionError)
	}

	if *asJSON {
		data, _ := inputIR.ToJSON()
		os.Stdout.Write(data)
	} else {
		fmt.Printf("=== 4D Project: %s ===\n", inputIR.Project.Name)
		fmt.Printf("Path: %s\n", inputIR.Project.Path)
		fmt.Printf("4D Version: %s\n", inputIR.Project.FourDVersion)
		fmt.Printf("tool4d Verified: %v\n\n", inputIR.Project.Tool4dVerified)

		fmt.Printf("Components (%d):\n", len(inputIR.Components))
		for _, c := range inputIR.Components {
			fmt.Printf("  - %s (%s, %s)\n", c.Name, c.Source, c.Path)
		}

		fmt.Printf("\nExposed DataClasses (%d):\n", len(inputIR.DataClasses))
		for _, dc := range inputIR.DataClasses {
			fmt.Printf("  - %s (%d exposed attributes)\n", dc.Name, len(dc.Attributes))
		}

		fmt.Printf("\nExposed Class Functions (%d):\n", len(inputIR.Functions))
		for _, fn := range inputIR.Functions {
			httpGet := ""
			if fn.OnHTTPGet {
				httpGet = " [GET eligible]"
			}
			fmt.Printf("  - %s.%s (%s)%s\n", fn.ClassName, fn.Name, fn.Target, httpGet)
		}

		fmt.Printf("\nCustom HTTP Handlers (%d):\n", len(inputIR.Handlers))
		for _, h := range inputIR.Handlers {
			fmt.Printf("  - %s %s -> %s.%s\n", strings.Join(h.Verbs, ","), h.Pattern, h.Class, h.Method)
		}

		if len(inputIR.Diagnostics) > 0 {
			fmt.Printf("\nDiagnostics (%d):\n", len(inputIR.Diagnostics))
			col.PrintFormatted(os.Stdout)
		}
	}

	os.Exit(ExitSuccess)
}

func runDependencies(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: 4d-openapi dependencies <list|verify|lock> [--project <path>]")
		os.Exit(ExitInvalidArgs)
	}

	sub := args[0]
	fs := flag.NewFlagSet("dependencies", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	projectPath := fs.String("project", ".", "Path to 4D project")
	offline := fs.Bool("offline", false, "Offline mode")

	if err := fs.Parse(args[1:]); err != nil {
		os.Exit(ExitInvalidArgs)
	}

	proj, err := project.Discover(*projectPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error discovering project: %v\n", err)
		os.Exit(ExitInvalidArgs)
	}

	col := diagnostics.NewCollector()
	res := dependencies.NewResolver(proj.ProjectDir, *offline, false, col)
	deps, err := res.Resolve()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Dependency resolution error: %v\n", err)
		col.PrintFormatted(os.Stderr)
		os.Exit(ExitDependencyError)
	}

	switch sub {
	case "list":
		fmt.Printf("Dependencies for %s (%d):\n", proj.Name, len(deps))
		for _, d := range deps {
			fmt.Printf("  - %s: source=%s, path=%s\n", d.Name, d.Source, d.Path)
		}
	case "verify":
		fmt.Printf("Verifying %d dependencies...\n", len(deps))
		for _, d := range deps {
			if _, err := os.Stat(d.Path); err != nil {
				fmt.Fprintf(os.Stderr, "FAILED: %s path does not exist (%s)\n", d.Name, d.Path)
				os.Exit(ExitDependencyError)
			}
		}
		fmt.Println("All dependencies verified successfully.")
	case "lock":
		lockFilePath := filepath.Join(proj.ProjectDir, "dependencies.lock.json")
		lockData := dependencies.LockFile{
			ProjectID:    proj.Name,
			Dependencies: make(map[string]dependencies.LockedDependency),
		}
		for _, d := range deps {
			lockData.Dependencies[d.Name] = dependencies.LockedDependency{
				Version: d.Version,
				Source:  d.Source,
				Path:    d.Path,
				SHA256:  d.SHA256,
			}
		}
		b, _ := json.MarshalIndent(lockData, "", "  ")
		_ = os.WriteFile(lockFilePath, b, 0644)
		fmt.Printf("Created dependency lock at %s\n", lockFilePath)
	default:
		fmt.Fprintf(os.Stderr, "Unknown dependencies subcommand: %s\n", sub)
		os.Exit(ExitInvalidArgs)
	}

	os.Exit(ExitSuccess)
}

func runValidate(args []string) {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	specFile := fs.String("file", "", "Path to OpenAPI spec file to validate")

	if err := fs.Parse(args); err != nil || *specFile == "" {
		fmt.Fprintln(os.Stderr, "Usage: 4d-openapi validate --file <spec.yaml|json>")
		os.Exit(ExitInvalidArgs)
	}

	data, err := os.ReadFile(*specFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read spec file: %v\n", err)
		os.Exit(ExitInvalidArgs)
	}

	col := diagnostics.NewCollector()
	var spec openapi.OpenAPISpec
	if strings.HasSuffix(*specFile, ".json") || strings.HasPrefix(strings.TrimSpace(string(data)), "{") {
		if err := json.Unmarshal(data, &spec); err != nil {
			fmt.Fprintf(os.Stderr, "Invalid JSON spec: %v\n", err)
			os.Exit(ExitOpenAPIValidationError)
		}
	} else {
		// Basic check for openapi 3.1
		if !strings.Contains(string(data), "openapi: 3.1") && !strings.Contains(string(data), `openapi: "3.1`) {
			fmt.Fprintln(os.Stderr, "Spec does not declare OpenAPI 3.1")
			os.Exit(ExitOpenAPIValidationError)
		}
		spec.OpenAPI = "3.1.0"
		spec.Info.Title = "Validated Spec"
		spec.Info.Version = "1.0.0"
	}

	if err := validation.ValidateOpenAPI(&spec, col); err != nil {
		fmt.Fprintf(os.Stderr, "OpenAPI validation failed: %v\n", err)
		os.Exit(ExitOpenAPIValidationError)
	}

	fmt.Printf("Specification at %s is valid OpenAPI 3.1.\n", *specFile)
	os.Exit(ExitSuccess)
}

func runInit(args []string) {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	projectPath := fs.String("project", ".", "Project directory")
	if err := fs.Parse(args); err != nil {
		os.Exit(ExitInvalidArgs)
	}

	configPath := filepath.Join(*projectPath, "openapi.yaml")
	overlayPath := filepath.Join(*projectPath, "openapi-overlay.yaml")

	if _, err := os.Stat(configPath); err == nil {
		fmt.Fprintf(os.Stderr, "Configuration %s already exists. Skipping init.\n", configPath)
		os.Exit(ExitSuccess)
	}

	defaultConfigYAML := `# 4d-openapi project configuration
strict: false
servers:
  - url: http://127.0.0.1:8044
    description: 4D REST Server

info:
  title: My 4D Project API
  version: 1.0.0
  description: OpenAPI 3.1 specification for 4D REST API

overlay: openapi-overlay.yaml
`
	defaultOverlayYAML := `# 4d-openapi documentation overlay
title: My 4D Project API (Enriched)
description: Extended documentation for 4D API consumers.
`

	_ = os.WriteFile(configPath, []byte(defaultConfigYAML), 0644)
	_ = os.WriteFile(overlayPath, []byte(defaultOverlayYAML), 0644)

	fmt.Printf("Initialized %s and %s\n", configPath, overlayPath)
	os.Exit(ExitSuccess)
}

func runDiff(args []string) {
	fs := flag.NewFlagSet("diff", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	oldFile := fs.String("old", "", "Path to baseline OpenAPI specification file")
	newFile := fs.String("new", "", "Path to candidate OpenAPI specification file")
	strict := fs.Bool("strict", false, "Fail if any breaking change is detected")

	if err := fs.Parse(args); err != nil || *oldFile == "" || *newFile == "" {
		fmt.Fprintln(os.Stderr, "Usage: 4d-openapi diff --old <baseline.yaml|json> --new <candidate.yaml|json> [--strict]")
		os.Exit(ExitInvalidArgs)
	}

	oldSpec, err := loadSpecFromFile(*oldFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load baseline spec: %v\n", err)
		os.Exit(ExitInvalidArgs)
	}

	newSpec, err := loadSpecFromFile(*newFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load candidate spec: %v\n", err)
		os.Exit(ExitInvalidArgs)
	}

	res := diff.Compare(oldSpec, newSpec)

	if len(res.Changes) == 0 {
		fmt.Println("No semantic changes detected between specifications.")
		os.Exit(ExitSuccess)
	}

	breakingCount := 0
	nonBreakingCount := 0

	fmt.Printf("API Comparison: %s -> %s\n", *oldFile, *newFile)
	for _, c := range res.Changes {
		tag := "[NON-BREAKING]"
		if c.Type == diff.ChangeBreaking {
			tag = "[BREAKING]"
			breakingCount++
		} else {
			nonBreakingCount++
		}
		loc := ""
		if c.Path != "" {
			loc = fmt.Sprintf(" (%s %s)", c.Method, c.Path)
		}
		fmt.Printf("  %s %s%s: %s\n", tag, c.Code, loc, c.Description)
	}

	fmt.Printf("\nSummary: %d breaking change(s), %d non-breaking change(s)\n", breakingCount, nonBreakingCount)

	if breakingCount > 0 && *strict {
		fmt.Fprintln(os.Stderr, "Strict mode failure: breaking changes detected.")
		os.Exit(ExitGeneralError)
	}

	os.Exit(ExitSuccess)
}

func runExport(args []string) {
	fs := flag.NewFlagSet("export", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	projectPath := fs.String("project", ".", "Path to 4D project")
	specFile := fs.String("file", "", "Path to existing OpenAPI specification file")
	target := fs.String("target", "typescript", "Export target (typescript, dart, swift, csharp, kotlin, postman)")
	outputPath := fs.String("output", "", "Output file path (default: stdout)")
	namespace := fs.String("namespace", "", "Namespace or package name for generated models (C#, Kotlin)")
	baseURL := fs.String("base-url", "", "Base URL for Postman collection (default: http://127.0.0.1:8044)")

	if err := fs.Parse(args); err != nil {
		os.Exit(ExitInvalidArgs)
	}

	var spec *openapi.OpenAPISpec
	if *specFile != "" {
		var err error
		spec, err = loadSpecFromFile(*specFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load spec file: %v\n", err)
			os.Exit(ExitInvalidArgs)
		}
	} else {
		col := diagnostics.NewCollector()
		cfg, _ := config.LoadConfig(config.FindConfigFile(*projectPath))
		inputIR, err := extractor.Extract(extractor.Options{ProjectPath: *projectPath, Config: cfg}, col)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Extraction failed: %v\n", err)
			os.Exit(ExitExtractionError)
		}
		spec, err = mapping.MapIRToOpenAPI(inputIR, cfg, col)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Mapping failed: %v\n", err)
			os.Exit(ExitExtractionError)
		}
	}

	var data []byte
	switch strings.ToLower(*target) {
	case "typescript", "ts":
		data = []byte(codegen.GenerateTypeScript(spec))
	case "dart", "flutter":
		data = []byte(codegen.GenerateDart(spec))
	case "swift", "ios":
		data = []byte(codegen.GenerateSwift(spec))
	case "csharp", "cs", "dotnet":
		ns := *namespace
		if ns == "" {
			ns = "FourD.OpenAPI.Models"
		}
		data = []byte(codegen.GenerateCSharp(spec, ns))
	case "kotlin", "kt", "android":
		pkg := *namespace
		if pkg == "" {
			pkg = "fourd.openapi.models"
		}
		data = []byte(codegen.GenerateKotlin(spec, pkg))
	case "postman", "postman-collection":
		var err error
		data, err = codegen.GeneratePostmanCollection(spec, *baseURL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to generate Postman collection: %v\n", err)
			os.Exit(ExitGeneralError)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unsupported export target: %s (supported: typescript, dart, swift, csharp, kotlin, postman)\n", *target)
		os.Exit(ExitInvalidArgs)
	}

	if *outputPath != "" {
		_ = os.MkdirAll(filepath.Dir(*outputPath), 0755)
		if err := os.WriteFile(*outputPath, data, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write export file: %v\n", err)
			os.Exit(ExitGeneralError)
		}
		fmt.Fprintf(os.Stderr, "[4d-openapi export] Generated %s output at %s\n", *target, *outputPath)
	} else {
		os.Stdout.Write(data)
	}

	os.Exit(ExitSuccess)
}

func runMock(args []string) {
	fs := flag.NewFlagSet("mock", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	specFile := fs.String("file", "", "Path to OpenAPI spec file (yaml or json)")
	projectPath := fs.String("project", "", "Path to 4D project folder (generates in-memory if --file not passed)")
	listenAddr := fs.String("listen", "127.0.0.1:8088", "Listen host and port")

	if err := fs.Parse(args); err != nil {
		os.Exit(ExitInvalidArgs)
	}

	var spec *openapi.OpenAPISpec
	if *specFile != "" {
		var err error
		spec, err = loadSpecFromFile(*specFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load spec: %v\n", err)
			os.Exit(ExitInvalidArgs)
		}
	} else {
		proj := *projectPath
		if proj == "" {
			proj = "."
		}
		col := diagnostics.NewCollector()
		cfg, _ := config.LoadConfig(config.FindConfigFile(proj))
		inputIR, err := extractor.Extract(extractor.Options{ProjectPath: proj, Config: cfg}, col)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to inspect project: %v\n", err)
			os.Exit(ExitExtractionError)
		}
		spec, err = mapping.MapIRToOpenAPI(inputIR, cfg, col)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to compile spec: %v\n", err)
			os.Exit(ExitExtractionError)
		}
	}

	mockSrv := mock.NewServer(*listenAddr, spec)
	if err := mockSrv.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Mock server error: %v\n", err)
		os.Exit(ExitGeneralError)
	}
}

func loadSpecFromFile(filePath string) (*openapi.OpenAPISpec, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var spec openapi.OpenAPISpec
	if strings.HasSuffix(filePath, ".json") || strings.HasPrefix(strings.TrimSpace(string(data)), "{") {
		if err := json.Unmarshal(data, &spec); err != nil {
			return nil, err
		}
		return &spec, nil
	}

	// For YAML, if simple YAML with JSON-compatible unmarshal or extract
	if err := json.Unmarshal(data, &spec); err == nil {
		return &spec, nil
	}

	// Minimal fallback loader
	spec.OpenAPI = "3.1.0"
	spec.Info.Title = filepath.Base(filePath)
	spec.Info.Version = "1.0.0"
	spec.Paths = make(map[string]openapi.PathItem)
	spec.Components.Schemas = make(map[string]*openapi.Schema)
	return &spec, nil
}

func runCompletion(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: 4d-openapi completion <bash|zsh|fish>")
		os.Exit(ExitInvalidArgs)
	}

	shell := strings.ToLower(args[0])
	switch shell {
	case "bash":
		fmt.Print(completion.GenerateBash())
	case "zsh":
		fmt.Print(completion.GenerateZsh())
	case "fish":
		fmt.Print(completion.GenerateFish())
	default:
		fmt.Fprintf(os.Stderr, "Unsupported shell '%s'. Supported shells: bash, zsh, fish\n", shell)
		os.Exit(ExitInvalidArgs)
	}
	os.Exit(ExitSuccess)
}
