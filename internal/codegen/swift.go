package codegen

import (
	"fmt"
	"sort"
	"strings"

	"github.com/miyako/OpenAPI/internal/openapi"
)

// GenerateSwift exports Swift Codable structs from an OpenAPI 3.1 specification.
func GenerateSwift(spec *openapi.OpenAPISpec) string {
	var sb strings.Builder

	sb.WriteString("//\n")
	sb.WriteString(fmt.Sprintf("// %s — Swift Models\n", spec.Info.Title))
	sb.WriteString(fmt.Sprintf("// Version: %s\n", spec.Info.Version))
	sb.WriteString("// Generated automatically by 4d-openapi\n")
	sb.WriteString("//\n\n")
	sb.WriteString("import Foundation\n\n")

	schemaNames := make([]string, 0, len(spec.Components.Schemas))
	for name := range spec.Components.Schemas {
		schemaNames = append(schemaNames, name)
	}
	sort.Strings(schemaNames)

	for _, name := range schemaNames {
		sch := spec.Components.Schemas[name]
		sb.WriteString(renderSwiftStruct(name, sch))
		sb.WriteString("\n\n")
	}

	return sb.String()
}

func renderSwiftStruct(name string, sch *openapi.Schema) string {
	var sb strings.Builder

	if sch.Description != "" {
		sb.WriteString(fmt.Sprintf("/// %s\n", sch.Description))
	}

	protocols := "Codable"
	if _, hasID := sch.Properties["ID"]; hasID {
		protocols = "Codable, Identifiable"
	}
	sb.WriteString(fmt.Sprintf("public struct %s: %s {\n", name, protocols))

	requiredMap := make(map[string]bool)
	for _, r := range sch.Required {
		requiredMap[r] = true
	}

	propNames := make([]string, 0, len(sch.Properties))
	for pn := range sch.Properties {
		propNames = append(propNames, pn)
	}
	sort.Strings(propNames)

	hasCodingKeyMapping := false
	for _, pn := range propNames {
		prop := sch.Properties[pn]
		isRequired := requiredMap[pn]
		swiftType := mapSchemaToSwiftType(prop, isRequired)
		fieldName := toSwiftIdentifier(pn)
		if fieldName != pn {
			hasCodingKeyMapping = true
		}

		mutability := "var"
		if prop.ReadOnly {
			mutability = "let"
		}

		if prop.Description != "" {
			sb.WriteString(fmt.Sprintf("    /// %s\n", prop.Description))
		}
		sb.WriteString(fmt.Sprintf("    public %s %s: %s\n", mutability, fieldName, swiftType))
	}

	// CodingKeys enum
	sb.WriteString("\n    public enum CodingKeys: String, CodingKey {\n")
	for _, pn := range propNames {
		fieldName := toSwiftIdentifier(pn)
		if fieldName != pn {
			sb.WriteString(fmt.Sprintf("        case %s = \"%s\"\n", fieldName, pn))
		} else {
			sb.WriteString(fmt.Sprintf("        case %s\n", fieldName))
		}
	}
	sb.WriteString("    }\n")

	sb.WriteString("}")
	_ = hasCodingKeyMapping
	return sb.String()
}

func toSwiftIdentifier(name string) string {
	if strings.HasPrefix(name, "__") {
		return strings.ToLower(strings.TrimPrefix(name, "__"))
	}
	if name == "ID" {
		return "id"
	}
	if len(name) > 0 {
		return strings.ToLower(name[:1]) + name[1:]
	}
	return name
}

func mapSchemaToSwiftType(sch *openapi.Schema, required bool) string {
	if sch == nil {
		return "AnyCodable?"
	}

	suffix := ""
	if !required {
		suffix = "?"
	}

	if sch.Ref != "" {
		parts := strings.Split(sch.Ref, "/")
		return parts[len(parts)-1] + suffix
	}

	if sch.Type != nil {
		switch t := sch.Type.(type) {
		case string:
			return mapSingleSwiftType(t, sch) + suffix
		case []string:
			nonNullType := "String"
			for _, st := range t {
				if st != "null" {
					nonNullType = mapSingleSwiftType(st, sch)
				}
			}
			return nonNullType + "?"
		}
	}

	return "String?"
}

func mapSingleSwiftType(t string, sch *openapi.Schema) string {
	switch t {
	case "string":
		if sch.Format == "date" || sch.Format == "date-time" {
			return "Date"
		}
		return "String"
	case "integer":
		if sch.Format == "int64" {
			return "Int64"
		}
		return "Int"
	case "number":
		return "Double"
	case "boolean":
		return "Bool"
	case "array":
		if sch.Items != nil {
			itemType := mapSchemaToSwiftType(sch.Items, true)
			return fmt.Sprintf("[%s]", itemType)
		}
		return "[String]"
	case "object":
		return "[String: String]"
	default:
		return "String"
	}
}
