package validation

import (
	"testing"

	"github.com/miyako/OpenAPI/internal/diagnostics"
	"github.com/miyako/OpenAPI/internal/ir"
	"github.com/miyako/OpenAPI/internal/openapi"
)

func TestValidateIR(t *testing.T) {
	col := diagnostics.NewCollector()
	goodIR := ir.NewIR(ir.ProjectMeta{Name: "Sample", Path: "Sample.4DProject"})
	if err := ValidateIR(goodIR, col); err != nil {
		t.Fatalf("Expected valid IR, got: %v", err)
	}

	badIR := ir.NewIR(ir.ProjectMeta{Name: "", Path: "Sample.4DProject"})
	if err := ValidateIR(badIR, col); err == nil {
		t.Errorf("Expected error for empty project name")
	}
}

func TestValidateOpenAPI(t *testing.T) {
	col := diagnostics.NewCollector()
	spec := openapi.NewSpec("Test", "1.0.0", "Desc")
	spec.Paths["/test"] = openapi.PathItem{
		Get: &openapi.Operation{
			OperationID: "test_get",
			Responses: map[string]openapi.Response{
				"200": {Description: "OK"},
			},
		},
	}

	if err := ValidateOpenAPI(spec, col); err != nil {
		t.Fatalf("Expected valid OpenAPI spec, got: %v", err)
	}

	badSpec := openapi.NewSpec("Test", "1.0.0", "Desc")
	badSpec.OpenAPI = "2.0"
	if err := ValidateOpenAPI(badSpec, col); err == nil {
		t.Errorf("Expected error for non-3.1 openapi version")
	}
}
