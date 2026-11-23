package codegen

import (
	"fmt"
	"sort"
	"strings"

	"github.com/miyako/OpenAPI/internal/openapi"
)

// GenerateKotlin exports Kotlin data classes with kotlinx.serialization annotations from an OpenAPI 3.1 specification.
func GenerateKotlin(spec *openapi.OpenAPISpec, packageName string) string {
	if packageName == "" {
		packageName = "fourd.openapi.models"
	}

	var sb strings.Builder

	sb.WriteString("//\n")
	sb.WriteString(fmt.Sprintf("// %s — Kotlin / Android Models\n", spec.Info.Title))
	sb.WriteString(fmt.Sprintf("// Version: %s\n", spec.Info.Version))
	sb.WriteString("// Generated automatically by 4d-openapi\n")
	sb.WriteString("//\n\n")
	sb.WriteString(fmt.Sprintf("package %s\n\n", packageName))
	sb.WriteString("import kotlinx.serialization.Serializable\n")
	sb.WriteString("import kotlinx.serialization.SerialName\n\n")

	schemaNames := make([]string, 0, len(spec.Components.Schemas))
	for name := range spec.Components.Schemas {
		schemaNames = append(schemaNames, name)
	}
	sort.Strings(schemaNames)

	for _, name := range schemaNames {
		sch := spec.Components.Schemas[name]
		sb.WriteString(renderKotlinDataClass(name, sch))
		sb.WriteString("\n\n")
	}

	return sb.String()
}

func renderKotlinDataClass(name string, sch *openapi.Schema) string {
	var sb strings.Builder

	if sch.Description != "" {
		sb.WriteString("/**\n")
		sb.WriteString(fmt.Sprintf(" * %s\n", sch.Description))
		sb.WriteString(" */\n")
	}
	sb.WriteString("@Serializable\n")
	sb.WriteString(fmt.Sprintf("data class %s(\n", name))

	requiredMap := make(map[string]bool)
	for _, r := range sch.Required {
		requiredMap[r] = true
	}

	propNames := make([]string, 0, len(sch.Properties))
	for pn := range sch.Properties {
		propNames = append(propNames, pn)
	}
	sort.Strings(propNames)

	for i, pn := range propNames {
		prop := sch.Properties[pn]
		isRequired := requiredMap[pn]
		ktType := mapSchemaToKotlinType(prop, isRequired)
		fieldName := toKotlinIdentifier(pn)

		comma := ","
		if i == len(propNames)-1 {
			comma = ""
		}

		defaultVal := ""
		if !isRequired {
			defaultVal = " = null"
		}

		if prop.Description != "" {
			sb.WriteString(fmt.Sprintf("    /** %s */\n", prop.Description))
		}
		sb.WriteString(fmt.Sprintf("    @SerialName(\"%s\")\n", pn))
		sb.WriteString(fmt.Sprintf("    val %s: %s%s%s\n", fieldName, ktType, defaultVal, comma))
	}

	sb.WriteString(")")
	return sb.String()
}

func toKotlinIdentifier(name string) string {
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

func mapSchemaToKotlinType(sch *openapi.Schema, required bool) string {
	if sch == nil {
		return "Any?"
	}

	nullableSuffix := "?"
	if required {
		nullableSuffix = ""
	}

	if sch.Ref != "" {
		parts := strings.Split(sch.Ref, "/")
		return parts[len(parts)-1] + nullableSuffix
	}

	if sch.Type != nil {
		switch t := sch.Type.(type) {
		case string:
			return mapSingleKotlinType(t, sch) + nullableSuffix
		case []string:
			nonNullType := "String"
			for _, st := range t {
				if st != "null" {
					nonNullType = mapSingleKotlinType(st, sch)
				}
			}
			return nonNullType + "?"
		}
	}

	return "Any?"
}

func mapSingleKotlinType(t string, sch *openapi.Schema) string {
	switch t {
	case "string":
		return "String"
	case "integer":
		if sch.Format == "int64" {
			return "Long"
		}
		return "Int"
	case "number":
		return "Double"
	case "boolean":
		return "Boolean"
	case "array":
		if sch.Items != nil {
			itemType := mapSchemaToKotlinType(sch.Items, true)
			return fmt.Sprintf("List<%s>", itemType)
		}
		return "List<Any>"
	case "object":
		return "Map<String, Any>"
	default:
		return "Any"
	}
}
