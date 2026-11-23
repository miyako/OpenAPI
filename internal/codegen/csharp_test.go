package codegen

import (
	"strings"
	"testing"

	"github.com/miyako/OpenAPI/internal/openapi"
)

func TestGenerateCSharp(t *testing.T) {
	spec := openapi.NewSpec("TestApp", "1.0.0", "C# test")
	spec.Components.Schemas["Company"] = &openapi.Schema{
		Type:        "object",
		Description: "A Company entity in 4D",
		Required:    []string{"ID", "name"},
		Properties: map[string]*openapi.Schema{
			"ID": {
				Type:   "integer",
				Format: "int64",
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

	csCode := GenerateCSharp(spec, "MyProject.Models")

	if !strings.Contains(csCode, "namespace MyProject.Models") {
		t.Errorf("Expected namespace in C# code, got:\n%s", csCode)
	}
	if !strings.Contains(csCode, "public class Company") {
		t.Errorf("Expected public class Company in C# code, got:\n%s", csCode)
	}
	if !strings.Contains(csCode, "[JsonPropertyName(\"ID\")]") {
		t.Errorf("Expected JsonPropertyName(\"ID\"), got:\n%s", csCode)
	}
	if !strings.Contains(csCode, "public long Id { get; set; }") {
		t.Errorf("Expected public long Id, got:\n%s", csCode)
	}
	if !strings.Contains(csCode, "public List<Employee>? Employees { get; set; }") {
		t.Errorf("Expected public List<Employee>? Employees, got:\n%s", csCode)
	}
}
