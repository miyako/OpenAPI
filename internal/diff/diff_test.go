package diff

import (
	"testing"

	"github.com/miyako/OpenAPI/internal/openapi"
)

func TestDiffBreakingChanges(t *testing.T) {
	oldSpec := openapi.NewSpec("TestAPI", "1.0.0", "Old version")
	oldSpec.Paths["/rest/Company"] = openapi.PathItem{
		Get: &openapi.Operation{
			OperationID: "listCompany",
			Responses:   map[string]openapi.Response{"200": {Description: "OK"}},
		},
	}
	oldSpec.Components.Schemas["Company"] = &openapi.Schema{
		Type: "object",
		Properties: map[string]*openapi.Schema{
			"ID":   {Type: "integer"},
			"name": {Type: "string"},
		},
	}

	newSpec := openapi.NewSpec("TestAPI", "2.0.0", "New version")
	newSpec.Components.Schemas["Company"] = &openapi.Schema{
		Type: "object",
		Properties: map[string]*openapi.Schema{
			"ID": {Type: "integer"}, // 'name' was removed!
		},
	}
	// /rest/Company was removed!

	res := Compare(oldSpec, newSpec)
	if !res.HasBreaking() {
		t.Fatalf("Expected breaking changes, got none")
	}

	var hasPathRemoved, hasPropRemoved bool
	for _, c := range res.Changes {
		if c.Code == "PATH_REMOVED" {
			hasPathRemoved = true
		}
		if c.Code == "PROPERTY_REMOVED" {
			hasPropRemoved = true
		}
	}

	if !hasPathRemoved {
		t.Errorf("Expected PATH_REMOVED change")
	}
	if !hasPropRemoved {
		t.Errorf("Expected PROPERTY_REMOVED change")
	}
}

func TestDiffNonBreakingChanges(t *testing.T) {
	oldSpec := openapi.NewSpec("TestAPI", "1.0.0", "Old")
	oldSpec.Paths["/rest/Company"] = openapi.PathItem{
		Get: &openapi.Operation{
			OperationID: "listCompany",
			Responses:   map[string]openapi.Response{"200": {Description: "OK"}},
		},
	}

	newSpec := openapi.NewSpec("TestAPI", "1.1.0", "New")
	newSpec.Paths["/rest/Company"] = openapi.PathItem{
		Get: &openapi.Operation{
			OperationID: "listCompany",
			Responses:   map[string]openapi.Response{"200": {Description: "OK"}},
			Parameters: []openapi.Parameter{
				{
					Name:     "filter",
					In:       "query",
					Required: false,
				},
			},
		},
	}
	newSpec.Paths["/rest/Employee"] = openapi.PathItem{
		Get: &openapi.Operation{
			OperationID: "listEmployee",
			Responses:   map[string]openapi.Response{"200": {Description: "OK"}},
		},
	}

	res := Compare(oldSpec, newSpec)
	if res.HasBreaking() {
		t.Errorf("Expected only non-breaking changes, got breaking changes: %+v", res.Changes)
	}

	if len(res.Changes) != 2 {
		t.Errorf("Expected 2 non-breaking changes, got %d", len(res.Changes))
	}
}
