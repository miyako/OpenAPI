package parser

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/miyako/OpenAPI/internal/diagnostics"
	"github.com/miyako/OpenAPI/internal/ir"
)

// Regex patterns for 4D class parsing
var (
	funcRegex = regexp.MustCompile(`(?i)^\s*(?:(exposed|onhttpget)\s+)*(?:(exposed|onhttpget)\s+)?function\s+([a-zA-Z0-9_]+)\s*(?:\((.*?)\))?\s*(?::\s*([a-zA-Z0-9_\.]+))?`)
	singletonRegex = regexp.MustCompile(`(?i)^\s*(?:shared\s+)?singleton\s+class\s+constructor`)
)

// ParseClassesDirectory parses all .4dm class files in a directory.
func ParseClassesDirectory(classesDir string, col *diagnostics.Collector) ([]ir.Function, error) {
	if classesDir == "" {
		return nil, nil
	}

	entries, err := os.ReadDir(classesDir)
	if err != nil {
		col.AddWarning("WARN_CLASSES_READ", err.Error(), classesDir, 0, diagnostics.ConfidenceUnknown)
		return nil, err
	}

	allFunctions := make([]ir.Function, 0)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".4dm") {
			continue
		}

		filePath := filepath.Join(classesDir, entry.Name())
		funcs, err := ParseClassFile(filePath, col)
		if err != nil {
			col.AddWarning("WARN_CLASS_PARSE", err.Error(), filePath, 0, diagnostics.ConfidenceInferred)
			continue
		}
		allFunctions = append(allFunctions, funcs...)
	}

	return allFunctions, nil
}

// ParseClassFile extracts exposed functions and openapi annotations from a 4D class file.
func ParseClassFile(filePath string, col *diagnostics.Collector) ([]ir.Function, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	baseName := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	target, className := determineTargetAndClassName(baseName)

	scanner := bufio.NewScanner(file)
	var commentBlock []string
	var functions []ir.Function
	lineNum := 0
	isSingleton := false

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if singletonRegex.MatchString(trimmed) {
			isSingleton = true
			target = ir.TargetSingleton
		}

		if strings.HasPrefix(trimmed, "//") {
			commentBlock = append(commentBlock, strings.TrimPrefix(trimmed, "//"))
			continue
		}

		if trimmed == "" {
			// keep comments unless separated by empty lines
			commentBlock = nil
			continue
		}

		// Check for function
		if strings.Contains(strings.ToLower(trimmed), "function ") {
			matches := funcRegex.FindStringSubmatch(trimmed)
			if len(matches) > 0 {
				fullPrefix := strings.ToLower(trimmed[:strings.Index(strings.ToLower(trimmed), "function ")])
				isExposed := strings.Contains(fullPrefix, "exposed")
				isOnHTTPGet := strings.Contains(fullPrefix, "onhttpget")

				// Only process exposed functions
				if isExposed {
					funcName := matches[3]
					rawParams := matches[4]
					rawReturn := matches[5]

					params := parseParameters(rawParams)
					retType := parseReturnType(rawReturn)

					// Parse doc annotations
					summary, description, tag, paramDocs, deprecated, hidden := parseAnnotations(commentBlock)
					if hidden {
						commentBlock = nil
						continue
					}
					for i := range params {
						if doc, ok := paramDocs[params[i].Name]; ok {
							params[i].Description = doc
						}
					}

					if isSingleton {
						target = ir.TargetSingleton
					}

					functions = append(functions, ir.Function{
						Name:        funcName,
						Target:      target,
						ClassName:   className,
						Exposed:     true,
						OnHTTPGet:   isOnHTTPGet,
						Summary:     summary,
						Description: description,
						Tag:         tag,
						Parameters:  params,
						ReturnType:  retType,
						Provenance: ir.Provenance{
							File:       filePath,
							Line:       lineNum,
							Confidence: diagnostics.ConfidenceInferred,
						},
					})

					if deprecated {
						col.AddInfo("INFO_DEPRECATED_FUNCTION", "Function marked as deprecated: "+funcName, filePath, diagnostics.ConfidenceInferred)
					}
				}
			}
		}

		// Reset comments after statement
		commentBlock = nil
	}

	return functions, scanner.Err()
}

// determineTargetAndClassName determines target receiver from class file name.
func determineTargetAndClassName(baseName string) (ir.FunctionTarget, string) {
	if strings.EqualFold(baseName, "DataStore") {
		return ir.TargetDataStore, "DataStore"
	}
	if strings.HasSuffix(baseName, "Entity") {
		dataClassName := strings.TrimSuffix(baseName, "Entity")
		return ir.TargetEntity, dataClassName
	}
	if strings.HasSuffix(baseName, "Selection") {
		dataClassName := strings.TrimSuffix(baseName, "Selection")
		return ir.TargetEntitySelection, dataClassName
	}
	// Default is DataClass or Singleton
	return ir.TargetDataClass, baseName
}

// parseParameters parses "$param1 : Text; $param2 : Integer"
func parseParameters(raw string) []ir.Parameter {
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	parts := strings.Split(raw, ";")
	var params []ir.Parameter

	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}

		colonIdx := strings.Index(p, ":")
		paramName := strings.TrimSpace(p)
		paramType := "string"
		fourDType := "text"

		if colonIdx != -1 {
			paramName = strings.TrimSpace(p[:colonIdx])
			fourDType = strings.TrimSpace(p[colonIdx+1:])
			paramType = map4DTypeToOpenAPI(fourDType)
		}

		paramName = strings.TrimPrefix(paramName, "$")

		params = append(params, ir.Parameter{
			Name:      paramName,
			Type:      paramType,
			FourDType: fourDType,
			Required:  true,
		})
	}

	return params
}

// parseReturnType parses ": Object" or ": Text"
func parseReturnType(raw string) *ir.ReturnType {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return &ir.ReturnType{
			Type:      "object",
			FourDType: "object",
		}
	}

	return &ir.ReturnType{
		Type:      map4DTypeToOpenAPI(raw),
		FourDType: raw,
	}
}

// map4DTypeToOpenAPI converts 4D type names to OpenAPI type strings.
func map4DTypeToOpenAPI(raw string) string {
	t := strings.ToLower(strings.TrimSpace(raw))
	switch {
	case strings.Contains(t, "bool"):
		return "boolean"
	case strings.Contains(t, "int"):
		return "integer"
	case strings.Contains(t, "real") || strings.Contains(t, "number") || strings.Contains(t, "float"):
		return "number"
	case strings.Contains(t, "date") || strings.Contains(t, "time") || strings.Contains(t, "text"):
		return "string"
	case strings.Contains(t, "collection") || strings.Contains(t, "array"):
		return "array"
	case strings.Contains(t, "object") || strings.Contains(t, "entity") || strings.Contains(t, "cs.") || strings.Contains(t, "4d."):
		return "object"
	default:
		return "string"
	}
}

// parseAnnotations parses // @openapi.* comments
func parseAnnotations(comments []string) (summary, description, tag string, paramDocs map[string]string, deprecated, hidden bool) {
	paramDocs = make(map[string]string)
	var descLines []string

	for _, c := range comments {
		line := strings.TrimSpace(c)
		if strings.HasPrefix(line, "@openapi.summary") {
			summary = strings.TrimSpace(strings.TrimPrefix(line, "@openapi.summary"))
		} else if strings.HasPrefix(line, "@openapi.description") {
			description = strings.TrimSpace(strings.TrimPrefix(line, "@openapi.description"))
		} else if strings.HasPrefix(line, "@openapi.tag") {
			tag = strings.TrimSpace(strings.TrimPrefix(line, "@openapi.tag"))
		} else if strings.HasPrefix(line, "@openapi.deprecated") {
			deprecated = true
		} else if strings.HasPrefix(line, "@openapi.hidden") || strings.HasPrefix(line, "@openapi.ignore") {
			hidden = true
		} else if strings.HasPrefix(line, "@openapi.param") {
			rem := strings.TrimSpace(strings.TrimPrefix(line, "@openapi.param"))
			parts := strings.SplitN(rem, " ", 2)
			if len(parts) == 2 {
				pName := strings.TrimPrefix(parts[0], "$")
				paramDocs[pName] = parts[1]
			}
		} else if !strings.HasPrefix(line, "@") {
			descLines = append(descLines, line)
		}
	}

	if summary == "" && len(descLines) > 0 {
		summary = descLines[0]
		if len(descLines) > 1 && description == "" {
			description = strings.Join(descLines[1:], " ")
		}
	}

	return
}
