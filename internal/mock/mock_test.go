package mock

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/miyako/OpenAPI/internal/openapi"
)

func TestMockServerREST(t *testing.T) {
	spec := openapi.NewSpec("TestAPI", "1.0.0", "Mock test")
	spec.Components.Schemas["Company"] = &openapi.Schema{
		Type: "object",
		Properties: map[string]*openapi.Schema{
			"ID":   {Type: "integer"},
			"name": {Type: "string"},
		},
	}

	server := NewServer("127.0.0.1:0", spec)

	// Test collection GET
	req := httptest.NewRequest("GET", "/rest/Company", nil)
	rec := httptest.NewRecorder()
	server.handleREST(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for /rest/Company, got %d", rec.Code)
	}

	var sel map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &sel); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	if sel["__COUNT"] != float64(2) {
		t.Errorf("Expected __COUNT: 2, got %v", sel["__COUNT"])
	}

	// Test single entity GET
	reqSingle := httptest.NewRequest("GET", "/rest/Company(1)", nil)
	recSingle := httptest.NewRecorder()
	server.handleREST(recSingle, reqSingle)

	if recSingle.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for /rest/Company(1), got %d", recSingle.Code)
	}

	var ent map[string]interface{}
	if err := json.Unmarshal(recSingle.Body.Bytes(), &ent); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	if ent["ID"] != float64(1) {
		t.Errorf("Expected ID: 1, got %v", ent["ID"])
	}
}
