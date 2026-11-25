package ir

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestIRSerialization(t *testing.T) {
	meta := ProjectMeta{
		Name:         "DemoProject",
		Path:         "/path/to/DemoProject.4DProject",
		FourDVersion: "20 R7",
	}

	irObj := NewIR(meta)
	irObj.DataClasses = append(irObj.DataClasses, DataClass{
		Name:    "Customer",
		Exposed: true,
		Attributes: []Attribute{
			{
				Name:       "ID",
				Type:       "integer",
				Format:     "int64",
				PrimaryKey: true,
				Exposed:    true,
			},
			{
				Name:    "name",
				Type:    "string",
				Exposed: true,
			},
		},
	})

	irObj.Handlers = append(irObj.Handlers, Handler{
		Pattern: "/api/health",
		Verbs:   []string{"GET"},
		Class:   "HealthCheck",
		Method:  "status",
	})

	jsonData, err := irObj.ToJSON()
	if err != nil {
		t.Fatalf("Failed to serialize IR to JSON: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(jsonData, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal serialized IR JSON: %v", err)
	}

	str := string(jsonData)
	if !strings.Contains(str, "\"irVersion\": 1") {
		t.Errorf("Missing irVersion in IR JSON")
	}
	if !strings.Contains(str, "\"Customer\"") {
		t.Errorf("Missing Customer in IR JSON")
	}
	if !strings.Contains(str, "\"/api/health\"") {
		t.Errorf("Missing /api/health in IR JSON")
	}
}
