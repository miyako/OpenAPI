package codegen

import (
	"strings"
	"testing"

	"github.com/miyako/OpenAPI/internal/openapi"
)

func TestGenerateKotlin(t *testing.T) {
	spec := openapi.NewSpec("TestApp", "1.0.0", "Kotlin test")
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

	ktCode := GenerateKotlin(spec, "com.example.app.models")

	if !strings.Contains(ktCode, "package com.example.app.models") {
		t.Errorf("Expected package declaration, got:\n%s", ktCode)
	}
	if !strings.Contains(ktCode, "@Serializable") {
		t.Errorf("Expected @Serializable annotation, got:\n%s", ktCode)
	}
	if !strings.Contains(ktCode, "data class Company(") {
		t.Errorf("Expected data class Company, got:\n%s", ktCode)
	}
	if !strings.Contains(ktCode, "@SerialName(\"ID\")") {
		t.Errorf("Expected @SerialName(\"ID\"), got:\n%s", ktCode)
	}
	if !strings.Contains(ktCode, "val id: Long") {
		t.Errorf("Expected val id: Long, got:\n%s", ktCode)
	}
	if !strings.Contains(ktCode, "val employees: List<Employee>? = null") {
		t.Errorf("Expected val employees: List<Employee>? = null, got:\n%s", ktCode)
	}
}
