package validation

import (
	"fmt"
	"strings"

	"github.com/miyako/OpenAPI/internal/diagnostics"
	"github.com/miyako/OpenAPI/internal/ir"
	"github.com/miyako/OpenAPI/internal/openapi"
)

// ValidateIR checks the structural validity and integrity of the Intermediate Representation.
func ValidateIR(inputIR *ir.IR, col *diagnostics.Collector) error {
	if inputIR.IRVersion != ir.CurrentIRVersion {
		col.AddError("ERR_IR_VERSION", fmt.Sprintf("Unsupported IR version: %d (expected %d)", inputIR.IRVersion, ir.CurrentIRVersion), "", 0)
		return fmt.Errorf("invalid IR version: %d", inputIR.IRVersion)
	}

	if inputIR.Project.Name == "" {
		col.AddError("ERR_IR_PROJECT_NAME", "Project name cannot be empty in IR", "", 0)
		return fmt.Errorf("project name cannot be empty")
	}

	// Validate DataClasses
	dcNames := make(map[string]bool)
	for _, dc := range inputIR.DataClasses {
		if dc.Name == "" {
			col.AddError("ERR_IR_DATACLASS_NAME", "DataClass name cannot be empty", dc.Provenance.File, dc.Provenance.Line)
			return fmt.Errorf("dataclass without name")
		}
		if dcNames[dc.Name] {
			col.AddWarning("WARN_IR_DUPLICATE_DC", "Duplicate DataClass name: "+dc.Name, dc.Provenance.File, dc.Provenance.Line, dc.Provenance.Confidence)
		}
		dcNames[dc.Name] = true
	}

	return nil
}

// ValidateOpenAPI checks that an OpenAPI 3.1 specification conforms to basic specification rules.
func ValidateOpenAPI(spec *openapi.OpenAPISpec, col *diagnostics.Collector) error {
	if !strings.HasPrefix(spec.OpenAPI, "3.1") {
		col.AddError("ERR_OPENAPI_VERSION", fmt.Sprintf("Specification openapi version must be 3.1.x, got %s", spec.OpenAPI), "", 0)
		return fmt.Errorf("invalid openapi version: %s", spec.OpenAPI)
	}

	if spec.Info.Title == "" {
		col.AddError("ERR_OPENAPI_INFO_TITLE", "OpenAPI info.title cannot be empty", "", 0)
		return fmt.Errorf("info.title cannot be empty")
	}

	if spec.Info.Version == "" {
		col.AddError("ERR_OPENAPI_INFO_VERSION", "OpenAPI info.version cannot be empty", "", 0)
		return fmt.Errorf("info.version cannot be empty")
	}

	// Validate paths
	opIDs := make(map[string]string)
	for path, item := range spec.Paths {
		if !strings.HasPrefix(path, "/") {
			col.AddError("ERR_OPENAPI_PATH", fmt.Sprintf("Path must begin with slash: %s", path), "", 0)
			return fmt.Errorf("invalid path: %s", path)
		}

		ops := []*openapi.Operation{item.Get, item.Post, item.Put, item.Delete, item.Patch}
		for _, op := range ops {
			if op == nil {
				continue
			}
			if op.OperationID != "" {
				if existingPath, found := opIDs[op.OperationID]; found {
					col.AddWarning("WARN_OPENAPI_DUPLICATE_OPID", fmt.Sprintf("Duplicate operationId '%s' on %s (already on %s)", op.OperationID, path, existingPath), path, 0, diagnostics.ConfidenceInferred)
				}
				opIDs[op.OperationID] = path
			}

			if len(op.Responses) == 0 {
				col.AddError("ERR_OPENAPI_RESPONSES", fmt.Sprintf("Operation '%s' on %s must have at least one response", op.OperationID, path), path, 0)
				return fmt.Errorf("missing responses for operation %s", op.OperationID)
			}
		}
	}

	return nil
}
