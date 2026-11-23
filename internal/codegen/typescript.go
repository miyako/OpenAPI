package codegen

import (
	"fmt"
	"sort"
	"strings"

	"github.com/miyako/OpenAPI/internal/openapi"
)

// GenerateTypeScript exports TypeScript interfaces from an OpenAPI 3.1 specification.
func GenerateTypeScript(spec *openapi.OpenAPISpec) string {
	var sb strings.Builder

	sb.WriteString("/**\n")
	sb.WriteString(fmt.Sprintf(" * %s — TypeScript Definitions\n", spec.Info.Title))
	sb.WriteString(fmt.Sprintf(" * Version: %s\n", spec.Info.Version))
	sb.WriteString(" * Generated automatically by 4d-openapi\n")
	sb.WriteString(" */\n\n")

	// Sort schema names for deterministic generation
	schemaNames := make([]string, 0, len(spec.Components.Schemas))
	for name := range spec.Components.Schemas {
		schemaNames = append(schemaNames, name)
	}
	sort.Strings(schemaNames)

	for _, name := range schemaNames {
		sch := spec.Components.Schemas[name]
		sb.WriteString(renderInterface(name, sch))
		sb.WriteString("\n\n")
	}

	return sb.String()
}

func renderInterface(name string, sch *openapi.Schema) string {
	var sb strings.Builder

	if sch.Description != "" {
		sb.WriteString(fmt.Sprintf("/** %s */\n", sch.Description))
	}
	sb.WriteString(fmt.Sprintf("export interface %s {\n", name))

	requiredMap := make(map[string]bool)
	for _, r := range sch.Required {
		requiredMap[r] = true
	}

	// Sort properties
	propNames := make([]string, 0, len(sch.Properties))
	for pn := range sch.Properties {
		propNames = append(propNames, pn)
	}
	sort.Strings(propNames)

	for _, pn := range propNames {
		prop := sch.Properties[pn]
		isRequired := requiredMap[pn]

		roPrefix := ""
		if prop.ReadOnly {
			roPrefix = "readonly "
		}

		optSuffix := ""
		if !isRequired {
			optSuffix = "?"
		}

		tsType := mapSchemaToTypeScript(prop)

		if prop.Description != "" {
			sb.WriteString(fmt.Sprintf("  /** %s */\n", prop.Description))
		}
		sb.WriteString(fmt.Sprintf("  %s%s%s: %s;\n", roPrefix, pn, optSuffix, tsType))
	}

	sb.WriteString("}")
	return sb.String()
}

func mapSchemaToTypeScript(sch *openapi.Schema) string {
	if sch == nil {
		return "unknown"
	}

	if sch.Ref != "" {
		parts := strings.Split(sch.Ref, "/")
		return parts[len(parts)-1]
	}

	if sch.Type != nil {
		switch t := sch.Type.(type) {
		case string:
			return mapSingleType(t, sch)
		case []string:
			var types []string
			for _, single := range t {
				types = append(types, mapSingleType(single, sch))
			}
			return strings.Join(types, " | ")
		}
	}

	return "unknown"
}

func mapSingleType(t string, sch *openapi.Schema) string {
	switch t {
	case "string":
		return "string"
	case "integer", "number":
		return "number"
	case "boolean":
		return "boolean"
	case "null":
		return "null"
	case "array":
		if sch.Items != nil {
			itemType := mapSchemaToTypeScript(sch.Items)
			return fmt.Sprintf("Array<%s>", itemType)
		}
		return "Array<unknown>"
	case "object":
		return "Record<string, unknown>"
	default:
		return "unknown"
	}
}
