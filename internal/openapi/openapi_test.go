package openapi

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestOpenAPISerializationAndDeterminism(t *testing.T) {
	spec := NewSpec("DeterminismTest", "1.0.0", "Testing 100 consecutive serializations for determinism")
	spec.Servers = append(spec.Servers, Server{URL: "http://127.0.0.1:8044", Description: "Local server"})

	spec.Tags = append(spec.Tags, Tag{Name: "Zeta", Description: "Last tag"})
	spec.Tags = append(spec.Tags, Tag{Name: "Alpha", Description: "First tag"})

	spec.Components.Schemas["User"] = &Schema{
		Type:        "object",
		Description: "User entity",
		Required:    []string{"id", "email"},
		Properties: map[string]*Schema{
			"id":    {Type: "integer", Format: "int64"},
			"email": {Type: "string"},
			"age":   {Type: []string{"integer", "null"}},
		},
	}
	spec.Components.Schemas["Account"] = &Schema{
		Type:        "object",
		Description: "Account entity",
		Properties: map[string]*Schema{
			"balance": {Type: "number"},
		},
	}

	spec.Components.SecuritySchemes["4DSessionHeader"] = &SecurityScheme{
		Type:        "apiKey",
		Name:        "4D-Session-ID",
		In:          "header",
		Description: "4D Session ID header",
	}

	spec.Paths["/rest/User"] = PathItem{
		Get: &Operation{
			OperationID: "listUser",
			Summary:     "List Users",
			Tags:        []string{"User"},
			Responses: map[string]Response{
				"200": {Description: "Success"},
			},
		},
	}
	spec.Paths["/rest/Account"] = PathItem{
		Get: &Operation{
			OperationID: "listAccount",
			Summary:     "List Accounts",
			Tags:        []string{"Account"},
			Responses: map[string]Response{
				"200": {Description: "Success"},
			},
		},
	}

	// 1. Initial generation
	firstYAML, err := spec.ToYAML()
	if err != nil {
		t.Fatalf("Failed to generate YAML: %v", err)
	}
	firstJSON, err := spec.ToJSON()
	if err != nil {
		t.Fatalf("Failed to generate JSON: %v", err)
	}

	// Verify valid JSON
	var parsedJSON map[string]interface{}
	if err := json.Unmarshal(firstJSON, &parsedJSON); err != nil {
		t.Fatalf("Invalid JSON produced: %v", err)
	}

	// Verify YAML content
	yamlStr := string(firstYAML)
	if !strings.Contains(yamlStr, "openapi: 3.1.0") {
		t.Errorf("YAML missing openapi version")
	}
	if !strings.Contains(yamlStr, "4DSessionHeader:") {
		t.Errorf("YAML missing security schemes")
	}

	// 2. Acceptance Criteria 1: 100 consecutive runs produce byte-identical YAML and JSON outputs
	for i := 0; i < 100; i++ {
		iterYAML, err := spec.ToYAML()
		if err != nil {
			t.Fatalf("Iteration %d YAML error: %v", i, err)
		}
		if !bytes.Equal(firstYAML, iterYAML) {
			t.Fatalf("Non-deterministic YAML output detected on iteration %d", i)
		}

		iterJSON, err := spec.ToJSON()
		if err != nil {
			t.Fatalf("Iteration %d JSON error: %v", i, err)
		}
		if !bytes.Equal(firstJSON, iterJSON) {
			t.Fatalf("Non-deterministic JSON output detected on iteration %d", i)
		}
	}
}
