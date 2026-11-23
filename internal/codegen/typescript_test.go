package codegen

import (
	"strings"
	"testing"

	"github.com/miyako/OpenAPI/internal/openapi"
)

func TestGenerateTypeScript(t *testing.T) {
	spec := openapi.NewSpec("TestApp", "1.0.0", "TypeScript test")
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

	tsCode := GenerateTypeScript(spec)

	if !strings.Contains(tsCode, "export interface Company {") {
		t.Errorf("Expected Company interface, got:\n%s", tsCode)
	}
	if !strings.Contains(tsCode, "readonly ID: number;") {
		t.Errorf("Expected readonly ID: number, got:\n%s", tsCode)
	}
	if !strings.Contains(tsCode, "name: string;") {
		t.Errorf("Expected name: string, got:\n%s", tsCode)
	}
	if !strings.Contains(tsCode, "revenue?: number | null;") {
		t.Errorf("Expected optional revenue?: number | null, got:\n%s", tsCode)
	}
	if !strings.Contains(tsCode, "employees?: Array<Employee>;") {
		t.Errorf("Expected employees?: Array<Employee>, got:\n%s", tsCode)
	}
}
