package project

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/miyako/OpenAPI/internal/diagnostics"
	"github.com/miyako/OpenAPI/internal/ir"
)

// ProjectInfo contains resolved paths and metadata for a 4D project.
type ProjectInfo struct {
	Name           string
	ProjectFile    string
	ProjectDir     string
	SourcesDir     string
	ClassesDir     string
	CatalogFile    string
	HandlersFile   string
	FourDVersion   string
	Tool4DVerified bool
}

// Discover locates and validates a 4D project structure.
func Discover(targetPath string) (*ProjectInfo, error) {
	if targetPath == "" {
		targetPath = "."
	}

	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		return nil, fmt.Errorf("resolve project path: %w", err)
	}

	fi, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("project path does not exist: %s", absPath)
	}

	var projectFile string
	var projectDir string

	if !fi.IsDir() && strings.HasSuffix(strings.ToLower(absPath), ".4dproject") {
		projectFile = absPath
		projectDir = filepath.Dir(absPath)
	} else {
		// Look in current dir and Project/ subfolder
		projectDir = absPath
		matches, _ := filepath.Glob(filepath.Join(projectDir, "*.4DProject"))
		if len(matches) == 0 {
			matches, _ = filepath.Glob(filepath.Join(projectDir, "Project", "*.4DProject"))
		}
		if len(matches) == 0 {
			// Walk shallowly
			entries, _ := os.ReadDir(projectDir)
			for _, e := range entries {
				if strings.HasSuffix(strings.ToLower(e.Name()), ".4dproject") {
					matches = append(matches, filepath.Join(projectDir, e.Name()))
					break
				}
			}
		}

		if len(matches) > 0 {
			projectFile = matches[0]
		} else {
			// Try finding .4DProject recursively up to 2 levels
			filepath.Walk(projectDir, func(path string, info os.FileInfo, err error) error {
				if err == nil && !info.IsDir() && strings.HasSuffix(strings.ToLower(path), ".4dproject") {
					projectFile = path
					return filepath.SkipAll
				}
				return nil
			})
		}
	}

	if projectFile == "" {
		return nil, fmt.Errorf("no .4DProject file found in %s", targetPath)
	}

	projectName := strings.TrimSuffix(filepath.Base(projectFile), filepath.Ext(projectFile))
	baseDir := filepath.Dir(projectFile)
	if filepath.Base(baseDir) == "Project" {
		baseDir = filepath.Dir(baseDir)
	}

	// Locate Sources and Classes directories
	sourcesDir := ""
	sourcesCandidates := []string{
		filepath.Join(filepath.Dir(projectFile), "Sources"),
		filepath.Join(baseDir, "Project", "Sources"),
		filepath.Join(baseDir, "Sources"),
	}
	for _, c := range sourcesCandidates {
		if stat, err := os.Stat(c); err == nil && stat.IsDir() {
			sourcesDir = c
			break
		}
	}

	classesDir := ""
	if sourcesDir != "" {
		cd := filepath.Join(sourcesDir, "Classes")
		if stat, err := os.Stat(cd); err == nil && stat.IsDir() {
			classesDir = cd
		}
	}

	// Locate Catalog
	catalogFile := ""
	catalogCandidates := []string{
		filepath.Join(filepath.Dir(projectFile), "Database", "catalog.4DCatalog"),
		filepath.Join(filepath.Dir(projectFile), "Sources", "Database", "catalog.4DCatalog"),
		filepath.Join(baseDir, "Project", "Sources", "Database", "catalog.4DCatalog"),
		filepath.Join(baseDir, "Database", "catalog.4DCatalog"),
	}
	for _, c := range catalogCandidates {
		if stat, err := os.Stat(c); err == nil && !stat.IsDir() {
			catalogFile = c
			break
		}
	}

	// Locate HTTPHandlers.json
	handlersFile := ""
	handlersCandidates := []string{
		filepath.Join(filepath.Dir(projectFile), "HTTPHandlers.json"),
		filepath.Join(sourcesDir, "HTTPHandlers.json"),
		filepath.Join(baseDir, "Project", "Sources", "HTTPHandlers.json"),
		filepath.Join(baseDir, "HTTPHandlers.json"),
		filepath.Join(baseDir, "Project", "HTTPHandlers.json"),
	}
	for _, c := range handlersCandidates {
		if stat, err := os.Stat(c); err == nil && !stat.IsDir() {
			handlersFile = c
			break
		}
	}

	return &ProjectInfo{
		Name:         projectName,
		ProjectFile:  projectFile,
		ProjectDir:   baseDir,
		SourcesDir:   sourcesDir,
		ClassesDir:   classesDir,
		CatalogFile:  catalogFile,
		HandlersFile: handlersFile,
		FourDVersion: "21", // default minimum modern 4D
	}, nil
}

// XML structures for catalog.4DCatalog
type XMLCatalog struct {
	XMLName   xml.Name      `xml:"database"`
	Tables    []XMLTable    `xml:"table"`
	Relations []XMLRelation `xml:"relation"`
}

type XMLRelation struct {
	Name        string `xml:"name,attr"`
	SourceTable string `xml:"source_table,attr"`
	SourceField string `xml:"source_field,attr"`
	DestTable   string `xml:"destination_table,attr"`
	DestField   string `xml:"destination_field,attr"`
	Type        string `xml:"type,attr"`
}

type XMLTable struct {
	Name       string     `xml:"name,attr"`
	ID         string     `xml:"id,attr"`
	Exposed    string     `xml:"exposed,attr"`
	Fields     []XMLField `xml:"field"`
	PrimaryKey *XMLPK     `xml:"primary_key"`
}

type XMLPK struct {
	Field string `xml:"field,attr"`
}

type XMLField struct {
	Name         string `xml:"name,attr"`
	ID           string `xml:"id,attr"`
	Type         string `xml:"type,attr"`
	Exposed      string `xml:"exposed,attr"`
	Autosequence string `xml:"autosequence,attr"`
	NotNull      string `xml:"not_null,attr"`
	ReadOnly     string `xml:"read_only,attr"`
}

// ParseCatalog extracts exposed DataClasses and attributes from catalog.4DCatalog.
func ParseCatalog(catalogPath string, col *diagnostics.Collector) ([]ir.DataClass, error) {
	if catalogPath == "" {
		return nil, nil
	}

	data, err := os.ReadFile(catalogPath)
	if err != nil {
		col.AddWarning("WARN_CATALOG_READ", fmt.Sprintf("Failed to read catalog at %s: %v", catalogPath, err), catalogPath, 0, diagnostics.ConfidenceUnknown)
		return nil, err
	}

	var cat XMLCatalog
	if err := xml.Unmarshal(data, &cat); err != nil {
		col.AddError("ERR_CATALOG_PARSE", fmt.Sprintf("Failed to parse catalog XML: %v", err), catalogPath, 0)
		return nil, err
	}

	dataclasses := make([]ir.DataClass, 0)
	for _, t := range cat.Tables {
		// By default in 4D ORDA, tables are exposed unless explicitly exposed="false" or not enabled
		isExposed := true
		if strings.EqualFold(t.Exposed, "false") || strings.EqualFold(t.Exposed, "0") {
			isExposed = false
		}

		if !isExposed {
			continue
		}

		pkFieldName := ""
		if t.PrimaryKey != nil {
			pkFieldName = t.PrimaryKey.Field
		}

		attrs := make([]ir.Attribute, 0)
		for _, f := range t.Fields {
			fieldExposed := true
			if strings.EqualFold(f.Exposed, "false") || strings.EqualFold(f.Exposed, "0") {
				fieldExposed = false
			}

			if !fieldExposed {
				continue
			}

			isPK := (f.Name == pkFieldName) || strings.EqualFold(f.Autosequence, "true") || strings.EqualFold(f.Name, "ID")
			readOnly := strings.EqualFold(f.ReadOnly, "true") || isPK
			nullable := !strings.EqualFold(f.NotNull, "true") && !isPK

			openAPIType, openAPIFormat, fourDTypeName := map4DFieldType(f.Type)

			attrs = append(attrs, ir.Attribute{
				Name:         f.Name,
				Type:         openAPIType,
				Format:       openAPIFormat,
				FourDType:    fourDTypeName,
				PrimaryKey:   isPK,
				ReadOnly:     readOnly,
				Exposed:      true,
				Nullable:     nullable,
				Autosequence: strings.EqualFold(f.Autosequence, "true"),
				Provenance: ir.Provenance{
					File:       catalogPath,
					Confidence: diagnostics.ConfidenceVerified,
				},
			})
		}

		dataclasses = append(dataclasses, ir.DataClass{
			Name:    t.Name,
			Exposed: true,
			Scope:   "public",
			Attributes: attrs,
			Queries: ir.QueriesSupport{
				Filterable: true,
				Sortable:   true,
				Paginated:  true,
			},
			Provenance: ir.Provenance{
				File:       catalogPath,
				Confidence: diagnostics.ConfidenceVerified,
			},
		})
	}

	// Resolve relations between DataClasses
	tableMap := make(map[string]int) // table ID/name to index in dataclasses
	for idx, dc := range dataclasses {
		tableMap[dc.Name] = idx
		for _, t := range cat.Tables {
			if t.Name == dc.Name {
				tableMap[t.ID] = idx
			}
		}
	}

	for _, rel := range cat.Relations {
		srcIdx, srcOk := tableMap[rel.SourceTable]
		destIdx, destOk := tableMap[rel.DestTable]
		if !srcOk || !destOk {
			continue
		}

		srcDC := dataclasses[srcIdx].Name
		destDC := dataclasses[destIdx].Name

		relName := rel.Name
		if relName == "" {
			relName = strings.ToLower(destDC) + "s"
		}

		// Add one-to-many relation attribute to source DataClass
		dataclasses[srcIdx].Attributes = append(dataclasses[srcIdx].Attributes, ir.Attribute{
			Name:            relName,
			Type:            "array",
			FourDType:       "collection",
			Exposed:         true,
			Nullable:        true,
			ReadOnly:        true,
			TargetDataClass: destDC,
			RelationType:    "one-to-many",
			Description:     fmt.Sprintf("Relation to %s entities", destDC),
			Provenance: ir.Provenance{
				File:       catalogPath,
				Confidence: diagnostics.ConfidenceVerified,
			},
		})

		// Add many-to-one reverse relation attribute to destination DataClass
		reverseName := strings.ToLower(srcDC)
		dataclasses[destIdx].Attributes = append(dataclasses[destIdx].Attributes, ir.Attribute{
			Name:            reverseName,
			Type:            "object",
			FourDType:       "object",
			Exposed:         true,
			Nullable:        true,
			ReadOnly:        true,
			TargetDataClass: srcDC,
			RelationType:    "many-to-one",
			Description:     fmt.Sprintf("Relation to parent %s entity", srcDC),
			Provenance: ir.Provenance{
				File:       catalogPath,
				Confidence: diagnostics.ConfidenceVerified,
			},
		})
	}

	return dataclasses, nil
}

// map4DFieldType maps numeric or string 4D catalog types to OpenAPI 3.1 types.
func map4DFieldType(rawType string) (openAPIType, format, fourDTypeName string) {
	num, err := strconv.Atoi(rawType)
	if err == nil {
		switch num {
		case 1:
			return "boolean", "", "boolean"
		case 2:
			return "integer", "int32", "integer"
		case 3:
			return "integer", "int32", "longint"
		case 4:
			return "integer", "int64", "integer64"
		case 6:
			return "number", "double", "real"
		case 7:
			return "string", "date", "date"
		case 8:
			return "integer", "int64", "time" // 4D time is duration in ms or seconds
		case 9, 0:
			return "string", "", "text"
		case 10:
			return "string", "binary", "picture"
		case 14:
			return "object", "", "object"
		case 15:
			return "array", "", "collection"
		case 16:
			return "string", "byte", "blob"
		default:
			return "string", "", "text"
		}
	}

	// String type names
	t := strings.ToLower(rawType)
	switch {
	case strings.Contains(t, "bool"):
		return "boolean", "", "boolean"
	case strings.Contains(t, "64") || strings.Contains(t, "longint"):
		return "integer", "int64", "integer64"
	case strings.Contains(t, "int"):
		return "integer", "int32", "integer"
	case strings.Contains(t, "real") || strings.Contains(t, "number") || strings.Contains(t, "float"):
		return "number", "double", "real"
	case strings.Contains(t, "date"):
		return "string", "date", "date"
	case strings.Contains(t, "time"):
		return "integer", "int64", "time"
	case strings.Contains(t, "blob"):
		return "string", "byte", "blob"
	case strings.Contains(t, "picture") || strings.Contains(t, "image"):
		return "string", "binary", "picture"
	case strings.Contains(t, "object"):
		return "object", "", "object"
	case strings.Contains(t, "collection") || strings.Contains(t, "array"):
		return "array", "", "collection"
	default:
		return "string", "", "text"
	}
}

// JSON handler definition in HTTPHandlers.json
type rawHTTPHandler struct {
	Class        string   `json:"class"`
	Method       string   `json:"method"`
	Pattern      string   `json:"pattern"`
	RegexPattern string   `json:"regexPattern"`
	Verbs        []string `json:"verbs"`
	Verb         string   `json:"verb"`
	Summary      string   `json:"summary"`
	Description  string   `json:"description"`
}

// ParseHTTPHandlers reads and validates HTTPHandlers.json.
func ParseHTTPHandlers(handlersPath string, col *diagnostics.Collector) ([]ir.Handler, error) {
	if handlersPath == "" {
		return nil, nil
	}

	data, err := os.ReadFile(handlersPath)
	if err != nil {
		col.AddWarning("WARN_HANDLERS_READ", fmt.Sprintf("Failed to read handlers at %s: %v", handlersPath, err), handlersPath, 0, diagnostics.ConfidenceUnknown)
		return nil, err
	}

	var rawList []rawHTTPHandler
	if err := json.Unmarshal(data, &rawList); err != nil {
		// Try single object
		var single rawHTTPHandler
		if err2 := json.Unmarshal(data, &single); err2 == nil {
			rawList = []rawHTTPHandler{single}
		} else {
			col.AddError("ERR_HANDLERS_PARSE", fmt.Sprintf("Invalid JSON in HTTPHandlers.json: %v", err), handlersPath, 0)
			return nil, err
		}
	}

	handlers := make([]ir.Handler, 0, len(rawList))
	for _, r := range rawList {
		pattern := r.Pattern
		isRegex := false
		if r.RegexPattern != "" {
			pattern = r.RegexPattern
			isRegex = true
		} else if strings.ContainsAny(pattern, `*+?()[]^$`) {
			isRegex = true
		}

		verbs := r.Verbs
		if len(verbs) == 0 && r.Verb != "" {
			verbs = []string{strings.ToUpper(r.Verb)}
		}
		if len(verbs) == 0 {
			verbs = []string{"GET"}
		}
		for i, v := range verbs {
			verbs[i] = strings.ToUpper(strings.TrimSpace(v))
		}

		if isRegex {
			col.AddWarning("WARN_REGEX_HANDLER", fmt.Sprintf("Handler '%s' regex pattern '%s' preserved in x-4d-regexPattern", r.Class, pattern), handlersPath, 0, diagnostics.ConfidenceInferred)
		}

		handlers = append(handlers, ir.Handler{
			Class:       r.Class,
			Method:      r.Method,
			Pattern:     pattern,
			IsRegex:     isRegex,
			Verbs:       verbs,
			Summary:     r.Summary,
			Description: r.Description,
			Provenance: ir.Provenance{
				File:       handlersPath,
				Confidence: diagnostics.ConfidenceVerified,
			},
		})
	}

	return handlers, nil
}
