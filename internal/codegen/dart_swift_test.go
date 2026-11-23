package codegen

import (
	"strings"
	"testing"

	"github.com/miyako/OpenAPI/internal/openapi"
)

func TestGenerateDart(t *testing.T) {
	spec := openapi.NewSpec("TestApp", "1.0.0", "Dart test")
	spec.Components.Schemas["Company"] = &openapi.Schema{
		Type:        "object",
		Description: "A Company entity in 4D",
		Required:    []string{"ID", "name"},
		Properties: map[string]*openapi.Schema{
			"ID": {
				Type:     "integer",
				ReadOnly: true,
			},
			"name": {
				Type: "string",
			},
			"revenue": {
				Type: []string{"number", "null"},
			},
			"employees": {
				Type: "array",
				Items: &openapi.Schema{
					Ref: "#/components/schemas/Employee",
				},
			},
		},
	}

	dartCode := GenerateDart(spec)

	if !strings.Contains(dartCode, "class Company {") {
		t.Errorf("Expected Company class in Dart code, got:\n%s", dartCode)
	}
	if !strings.Contains(dartCode, "final int id;") {
		t.Errorf("Expected final int id;, got:\n%s", dartCode)
	}
	if !strings.Contains(dartCode, "factory Company.fromJson(Map<String, dynamic> json)") {
		t.Errorf("Expected factory Company.fromJson, got:\n%s", dartCode)
	}
}

func TestGenerateSwift(t *testing.T) {
	spec := openapi.NewSpec("TestApp", "1.0.0", "Swift test")
	spec.Components.Schemas["Company"] = &openapi.Schema{
		Type:        "object",
		Description: "A Company entity in 4D",
		Required:    []string{"ID", "name"},
		Properties: map[string]*openapi.Schema{
			"ID": {
				Type:     "integer",
				Format:   "int64",
				ReadOnly: true,
			},
			"name": {
				Type: "string",
			},
			"revenue": {
				Type: []string{"number", "null"},
			},
		},
	}

	swiftCode := GenerateSwift(spec)

	if !strings.Contains(swiftCode, "public struct Company: Codable, Identifiable {") {
		t.Errorf("Expected Identifiable struct in Swift code, got:\n%s", swiftCode)
	}
	if !strings.Contains(swiftCode, "public let id: Int64") {
		t.Errorf("Expected public let id: Int64, got:\n%s", swiftCode)
	}
	if !strings.Contains(swiftCode, "case id = \"ID\"") {
		t.Errorf("Expected case id = \"ID\" in CodingKeys, got:\n%s", swiftCode)
	}
}
