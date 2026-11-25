package mapping

import (
	"testing"

	"github.com/miyako/OpenAPI/internal/config"
	"github.com/miyako/OpenAPI/internal/diagnostics"
	"github.com/miyako/OpenAPI/internal/ir"
)

func TestMapIRToOpenAPI(t *testing.T) {
	col := diagnostics.NewCollector()
	cfg := &config.Config{
		Info: config.InfoConfig{
			Title:   "Custom 4D App",
			Version: "2.5.0",
		},
	}

	inputIR := &ir.IR{
		Project: ir.ProjectMeta{
			Name: "Custom 4D App",
			Path: "/path/to/project",
		},
		DataClasses: []ir.DataClass{
			{
				Name:    "Company",
				Exposed: true,
				Attributes: []ir.Attribute{
					{
						Name:       "ID",
						Type:       "integer",
						Format:     "int64",
						Exposed:    true,
						PrimaryKey: true,
					},
					{
						Name:    "name",
						Type:    "string",
						Exposed: true,
					},
					{
						Name:      "logo",
						Type:      "string",
						FourDType: "picture",
						Exposed:   true,
					},
					{
						Name:            "employees",
						Type:            "array",
						TargetDataClass: "Employee",
						RelationType:    "one-to-many",
						Exposed:         true,
					},
				},
			},
			{
				Name:    "Employee",
				Exposed: true,
				Attributes: []ir.Attribute{
					{
						Name:       "ID",
						Type:       "integer",
						Format:     "int64",
						Exposed:    true,
						PrimaryKey: true,
					},
					{
						Name:            "company",
						Type:            "object",
						TargetDataClass: "Company",
						RelationType:    "many-to-one",
						Exposed:         true,
					},
				},
			},
		},
		Handlers: []ir.Handler{
			{
				Verbs:   []string{"GET"},
				Pattern: "/api/status",
				Class:   "HealthService",
				Method:  "checkStatus",
			},
		},
	}

	spec, err := MapIRToOpenAPI(inputIR, cfg, col)
	if err != nil {
		t.Fatalf("MapIRToOpenAPI failed: %v", err)
	}

	if spec.Info.Title != "Custom 4D App" {
		t.Errorf("Expected title 'Custom 4D App', got '%s'", spec.Info.Title)
	}

	// Verify Security Schemes
	if spec.Components.SecuritySchemes["4DSessionCookie"] == nil {
		t.Errorf("Missing 4DSessionCookie security scheme")
	}
	if spec.Components.SecuritySchemes["4DSessionHeader"] == nil {
		t.Errorf("Missing 4DSessionHeader security scheme")
	}

	// Verify Schemas
	companySchema := spec.Components.Schemas["Company"]
	if companySchema == nil {
		t.Fatalf("Missing Company schema in components")
	}
	if companySchema.Properties["employees"] == nil {
		t.Errorf("Missing employees relation in Company schema")
	}

	employeeSchema := spec.Components.Schemas["Employee"]
	if employeeSchema == nil {
		t.Fatalf("Missing Employee schema in components")
	}
	if employeeSchema.Properties["company"] == nil {
		t.Errorf("Missing company relation in Employee schema")
	}

	// Verify ORDA Endpoints
	pathsToCheck := []string{
		"/rest/$catalog",
		"/rest/Company",
		"/rest/Company({key})",
		"/rest/Company({key})/logo",
		"/rest/Company/$entityset/{entitySetId}",
		"/api/status",
	}

	for _, p := range pathsToCheck {
		if _, ok := spec.Paths[p]; !ok {
			t.Errorf("Expected path '%s' in spec, but not found", p)
		}
	}
}
