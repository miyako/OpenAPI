package codegen

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/miyako/OpenAPI/internal/openapi"
)

func TestGeneratePostmanCollection(t *testing.T) {
	spec := openapi.NewSpec("Test4DAPI", "1.0.0", "Postman Collection Test")
	spec.Servers = append(spec.Servers, openapi.Server{URL: "http://localhost:8044"})

	spec.Paths["/rest/Company"] = openapi.PathItem{
		Get: &openapi.Operation{
			OperationID: "listCompany",
			Summary:     "List companies",
			Description: "Retrieve a list of companies from 4D",
			Tags:        []string{"Company"},
			Parameters: []openapi.Parameter{
				{
					Name:        "$filter",
					In:          "query",
					Description: "Filter expression",
					Required:    false,
				},
				{
					Name:        "$top",
					In:          "query",
					Description: "Page size",
					Required:    false,
				},
			},
		},
		Post: &openapi.Operation{
			OperationID: "createCompany",
			Summary:     "Create company",
			Description: "Create a new company entity",
			Tags:        []string{"Company"},
			RequestBody: &openapi.RequestBody{
				Required: true,
			},
		},
	}

	spec.Paths["/rest/Company({key})"] = openapi.PathItem{
		Get: &openapi.Operation{
			OperationID: "getCompany",
			Summary:     "Get company by key",
			Tags:        []string{"Company"},
			Parameters: []openapi.Parameter{
				{
					Name:        "key",
					In:          "path",
					Description: "Primary key or UUID",
					Required:    true,
				},
			},
		},
	}

	rawJSON, err := GeneratePostmanCollection(spec, "")
	if err != nil {
		t.Fatalf("GeneratePostmanCollection returned error: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(rawJSON, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal generated Postman JSON: %v", err)
	}

	str := string(rawJSON)
	if !strings.Contains(str, "https://schema.getpostman.com/json/collection/v2.1.0/collection.json") {
		t.Errorf("Missing Postman v2.1.0 schema reference")
	}
	if !strings.Contains(str, "{{baseUrl}}/rest/Company(:key)") {
		t.Errorf("Expected parameterized URL in raw: %s", str)
	}
	if !strings.Contains(str, "\"name\": \"Company\"") {
		t.Errorf("Expected Company folder")
	}
	if !strings.Contains(str, "\"key\": \"$filter\"") {
		t.Errorf("Expected $filter query param")
	}
}
