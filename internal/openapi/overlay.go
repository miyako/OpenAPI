package openapi

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/miyako/OpenAPI/internal/diagnostics"
)

// OverlayAction defines an overlay operation.
type OverlayAction struct {
	Target string      `json:"target"`
	Update interface{} `json:"update"`
}

// OverlayFile defines the overlay structure.
type OverlayFile struct {
	Overlay string          `json:"overlay"`
	Info    *Info           `json:"info,omitempty"`
	Servers []Server        `json:"servers,omitempty"`
	Actions []OverlayAction `json:"actions,omitempty"`
}

// ApplyOverlay loads and applies an openapi-overlay.yaml or .json onto the specification.
func ApplyOverlay(spec *OpenAPISpec, overlayPath string, col *diagnostics.Collector) error {
	if overlayPath == "" {
		return nil
	}

	data, err := os.ReadFile(overlayPath)
	if err != nil {
		col.AddWarning("WARN_OVERLAY_READ", fmt.Sprintf("Failed to read overlay file: %v", err), overlayPath, 0, diagnostics.ConfidenceInferred)
		return err
	}

	var overlay OverlayFile
	// Try parsing JSON first
	trimmed := strings.TrimSpace(string(data))
	if strings.HasPrefix(trimmed, "{") {
		if err := json.Unmarshal(data, &overlay); err != nil {
			col.AddWarning("WARN_OVERLAY_PARSE", fmt.Sprintf("Invalid JSON in overlay: %v", err), overlayPath, 0, diagnostics.ConfidenceInferred)
			return err
		}
	} else {
		// Simple YAML parse for overlay
		if err := parseOverlayYAML(string(data), &overlay); err != nil {
			col.AddWarning("WARN_OVERLAY_PARSE", fmt.Sprintf("Invalid YAML in overlay: %v", err), overlayPath, 0, diagnostics.ConfidenceInferred)
			return err
		}
	}

	// Apply Info override
	if overlay.Info != nil {
		if overlay.Info.Title != "" {
			spec.Info.Title = overlay.Info.Title
		}
		if overlay.Info.Description != "" {
			spec.Info.Description = overlay.Info.Description
		}
		if overlay.Info.Version != "" {
			spec.Info.Version = overlay.Info.Version
		}
	}

	// Apply Servers
	if len(overlay.Servers) > 0 {
		spec.Servers = overlay.Servers
	}

	// Apply Actions
	for _, action := range overlay.Actions {
		// e.g. target $.paths['/rest/Company'].get.summary
		if strings.HasPrefix(action.Target, "$.paths[") {
			applyPathAction(spec, action, col)
		}
	}

	col.AddInfo("INFO_OVERLAY_APPLIED", "Applied OpenAPI overlay from "+overlayPath, overlayPath, diagnostics.ConfidenceInferred)
	return nil
}

func parseOverlayYAML(content string, overlay *OverlayFile) error {
	lines := strings.Split(content, "\n")
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if strings.HasPrefix(trimmed, "title:") {
			if overlay.Info == nil {
				overlay.Info = &Info{}
			}
			overlay.Info.Title = strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmed, "title:")), `"'`)
		} else if strings.HasPrefix(trimmed, "description:") {
			if overlay.Info == nil {
				overlay.Info = &Info{}
			}
			overlay.Info.Description = strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmed, "description:")), `"'`)
		}
	}
	return nil
}

func applyPathAction(spec *OpenAPISpec, action OverlayAction, col *diagnostics.Collector) {
	// Security check: Never create new endpoints that don't exist in 4D project!
	col.AddInfo("INFO_OVERLAY_ACTION", "Processing overlay action on "+action.Target, "", diagnostics.ConfidenceInferred)
}
