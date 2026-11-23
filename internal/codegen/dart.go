package codegen

import (
	"fmt"
	"sort"
	"strings"

	"github.com/miyako/OpenAPI/internal/openapi"
)

// GenerateDart exports Dart class models from an OpenAPI 3.1 specification.
func GenerateDart(spec *openapi.OpenAPISpec) string {
	var sb strings.Builder

	sb.WriteString("//\n")
	sb.WriteString(fmt.Sprintf("// %s — Dart / Flutter Models\n", spec.Info.Title))
	sb.WriteString(fmt.Sprintf("// Version: %s\n", spec.Info.Version))
	sb.WriteString("// Generated automatically by 4d-openapi\n")
	sb.WriteString("//\n\n")

	schemaNames := make([]string, 0, len(spec.Components.Schemas))
	for name := range spec.Components.Schemas {
		schemaNames = append(schemaNames, name)
	}
	sort.Strings(schemaNames)

	for _, name := range schemaNames {
		sch := spec.Components.Schemas[name]
		sb.WriteString(renderDartClass(name, sch))
		sb.WriteString("\n\n")
	}

	return sb.String()
}

func renderDartClass(name string, sch *openapi.Schema) string {
	var sb strings.Builder

	if sch.Description != "" {
		sb.WriteString(fmt.Sprintf("/// %s\n", sch.Description))
	}
	sb.WriteString(fmt.Sprintf("class %s {\n", name))

	requiredMap := make(map[string]bool)
	for _, r := range sch.Required {
		requiredMap[r] = true
	}

	propNames := make([]string, 0, len(sch.Properties))
	for pn := range sch.Properties {
		propNames = append(propNames, pn)
	}
	sort.Strings(propNames)

	// Fields
	for _, pn := range propNames {
		prop := sch.Properties[pn]
		isRequired := requiredMap[pn]
		dartType := mapSchemaToDartType(prop, isRequired)
		fieldName := toDartIdentifier(pn)

		if prop.Description != "" {
			sb.WriteString(fmt.Sprintf("  /// %s\n", prop.Description))
		}
		sb.WriteString(fmt.Sprintf("  final %s %s;\n", dartType, fieldName))
	}

	// Constructor
	sb.WriteString(fmt.Sprintf("\n  %s({\n", name))
	for _, pn := range propNames {
		isRequired := requiredMap[pn]
		fieldName := toDartIdentifier(pn)
		reqPrefix := ""
		if isRequired {
			reqPrefix = "required "
		}
		sb.WriteString(fmt.Sprintf("    %sthis.%s,\n", reqPrefix, fieldName))
	}
	sb.WriteString("  });\n\n")

	// fromJson factory
	sb.WriteString(fmt.Sprintf("  factory %s.fromJson(Map<String, dynamic> json) {\n", name))
	sb.WriteString(fmt.Sprintf("    return %s(\n", name))
	for _, pn := range propNames {
		prop := sch.Properties[pn]
		fieldName := toDartIdentifier(pn)
		getter := renderDartFromJsonField(pn, prop)
		sb.WriteString(fmt.Sprintf("      %s: %s,\n", fieldName, getter))
	}
	sb.WriteString("    );\n  }\n\n")

	// toJson method
	sb.WriteString("  Map<String, dynamic> toJson() {\n")
	sb.WriteString("    return {\n")
	for _, pn := range propNames {
		prop := sch.Properties[pn]
		fieldName := toDartIdentifier(pn)
		renderDartToJsonField(&sb, pn, fieldName, prop)
	}
	sb.WriteString("    };\n  }\n")

	sb.WriteString("}")
	return sb.String()
}

func toDartIdentifier(name string) string {
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

func mapSchemaToDartType(sch *openapi.Schema, required bool) string {
	if sch == nil {
		return "dynamic"
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
			return mapSingleDartType(t, sch) + suffix
		case []string:
			// Nullable check
			nonNullType := "dynamic"
			for _, st := range t {
				if st != "null" {
					nonNullType = mapSingleDartType(st, sch)
				}
			}
			return nonNullType + "?"
		}
	}

	return "dynamic"
}

func mapSingleDartType(t string, sch *openapi.Schema) string {
	switch t {
	case "string":
		return "String"
	case "integer":
		return "int"
	case "number":
		return "double"
	case "boolean":
		return "bool"
	case "array":
		if sch.Items != nil {
			itemType := mapSchemaToDartType(sch.Items, true)
			return fmt.Sprintf("List<%s>", itemType)
		}
		return "List<dynamic>"
	case "object":
		return "Map<String, dynamic>"
	default:
		return "dynamic"
	}
}

func renderDartFromJsonField(key string, prop *openapi.Schema) string {
	raw := fmt.Sprintf("json['%s']", key)
	if prop.Ref != "" {
		parts := strings.Split(prop.Ref, "/")
		refName := parts[len(parts)-1]
		return fmt.Sprintf("%s != null ? %s.fromJson(%s as Map<String, dynamic>) : null", raw, refName, raw)
	}
	if prop.Type == "array" && prop.Items != nil && prop.Items.Ref != "" {
		parts := strings.Split(prop.Items.Ref, "/")
		refName := parts[len(parts)-1]
		return fmt.Sprintf("(%s as List<dynamic>?)?.map((e) => %s.fromJson(e as Map<String, dynamic>)).toList()", raw, refName)
	}
	if prop.Type == "number" {
		return fmt.Sprintf("(%s as num?)?.toDouble()", raw)
	}
	if prop.Type == "integer" {
		return fmt.Sprintf("%s as int?", raw)
	}
	if prop.Type == "boolean" {
		return fmt.Sprintf("%s as bool?", raw)
	}
	if prop.Type == "string" {
		return fmt.Sprintf("%s as String?", raw)
	}
	return raw
}

func renderDartToJsonField(sb *strings.Builder, key, fieldName string, prop *openapi.Schema) {
	if prop.Ref != "" {
		sb.WriteString(fmt.Sprintf("      if (%s != null) '%s': %s?.toJson(),\n", fieldName, key, fieldName))
	} else if prop.Type == "array" && prop.Items != nil && prop.Items.Ref != "" {
		sb.WriteString(fmt.Sprintf("      if (%s != null) '%s': %s?.map((e) => e.toJson()).toList(),\n", fieldName, key, fieldName))
	} else {
		sb.WriteString(fmt.Sprintf("      '%s': %s,\n", key, fieldName))
	}
}
