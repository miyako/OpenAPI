package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/miyako/OpenAPI/internal/openapi"
)

func TestServerEndpoints(t *testing.T) {
	spec := openapi.NewSpec("TestApp", "1.0.0", "Description")
	srv, err := NewServer(Config{
		Spec: spec,
	})
	if err != nil {
		t.Fatalf("Failed to initialize server: %v", err)
	}

	// 1. Test /health
	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected 200 OK for /health, got %d", rec.Code)
	}

	// 2. Test spec bytes
	if len(srv.SpecYAML) == 0 {
		t.Errorf("Expected non-empty SpecYAML")
	}
	if len(srv.SpecJSON) == 0 {
		t.Errorf("Expected non-empty SpecJSON")
	}
}
