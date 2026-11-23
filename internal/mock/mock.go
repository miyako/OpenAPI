package mock

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/miyako/OpenAPI/internal/openapi"
)

// Server represents a mock 4D REST API server.
type Server struct {
	ListenAddr string
	Spec       *openapi.OpenAPISpec
	httpServer *http.Server
}

// NewServer initializes a new mock server.
func NewServer(listenAddr string, spec *openapi.OpenAPISpec) *Server {
	if listenAddr == "" {
		listenAddr = "127.0.0.1:8088"
	}
	return &Server{
		ListenAddr: listenAddr,
		Spec:       spec,
	}
}

// Start runs the mock HTTP server.
func (s *Server) Start() error {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":  "mock-ok",
			"version": "1.0.0",
		})
	})

	mux.HandleFunc("/openapi.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		data, _ := s.Spec.ToJSON()
		_, _ = w.Write(data)
	})

	mux.HandleFunc("/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/yaml; charset=utf-8")
		data, _ := s.Spec.ToYAML()
		_, _ = w.Write(data)
	})

	// Generic ORDA REST handler
	mux.HandleFunc("/rest/", s.handleREST)

	listener, err := net.Listen("tcp", s.ListenAddr)
	if err != nil {
		return fmt.Errorf("mock server failed to bind to %s: %w", s.ListenAddr, err)
	}

	s.httpServer = &http.Server{
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Printf("[4d-openapi mock] Mock REST server listening on http://%s\n", s.ListenAddr)
	log.Printf("[4d-openapi mock] Mock OpenAPI spec available at http://%s/openapi.json\n", s.ListenAddr)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	serverErr := make(chan error, 1)
	go func() {
		if err := s.httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	select {
	case <-stop:
		log.Println("[4d-openapi mock] Shutting down mock server...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.httpServer.Shutdown(ctx)
	case err := <-serverErr:
		return err
	}
}

func (s *Server) handleREST(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	path := strings.TrimPrefix(r.URL.Path, "/rest/")

	// 1. Catalog discovery
	if path == "$catalog" || strings.HasPrefix(path, "$catalog/") {
		dcList := make([]string, 0, len(s.Spec.Components.Schemas))
		for name := range s.Spec.Components.Schemas {
			if !strings.HasSuffix(name, "Selection") {
				dcList = append(dcList, name)
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"dataClasses": dcList,
			"fourDVersion": "21",
		})
		return
	}

	// 2. Entity Set release
	if strings.Contains(path, "/$entityset/") && r.Method == "DELETE" {
		_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
		return
	}

	// 3. Entity delete
	if r.Method == "DELETE" {
		_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
		return
	}

	// 4. Single entity (e.g. Company(1))
	if strings.Contains(path, "(") && strings.Contains(path, ")") {
		dcName := path[:strings.Index(path, "(")]
		entity := s.generateMockEntity(dcName, 1)
		_ = json.NewEncoder(w).Encode(entity)
		return
	}

	// 5. Collection query (e.g. /rest/Company)
	dcName := strings.Split(path, "/")[0]
	if sch, exists := s.Spec.Components.Schemas[dcName]; exists && sch != nil {
		entities := []map[string]interface{}{
			s.generateMockEntity(dcName, 1),
			s.generateMockEntity(dcName, 2),
		}
		selection := map[string]interface{}{
			"__COUNT":    2,
			"__SENT":     2,
			"__FIRST":    0,
			"__ENTITIES": entities,
		}
		_ = json.NewEncoder(w).Encode(selection)
		return
	}

	// Fallback mock function response
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"result": "mock_success",
	})
}

func (s *Server) generateMockEntity(dcName string, id int) map[string]interface{} {
	entity := make(map[string]interface{})
	entity["__KEY"] = fmt.Sprintf("%d", id)
	entity["__STAMP"] = 1
	entity["ID"] = id

	sch, exists := s.Spec.Components.Schemas[dcName]
	if !exists || sch == nil {
		return entity
	}

	for propName, prop := range sch.Properties {
		if propName == "ID" || propName == "__KEY" || propName == "__STAMP" {
			continue
		}
		if prop.Ref != "" {
			continue // skip expanded relations in mock entity
		}

		switch prop.Type {
		case "string":
			if prop.Format == "date" {
				entity[propName] = "2026-01-01"
			} else {
				entity[propName] = fmt.Sprintf("sample_%s_%d", propName, id)
			}
		case "integer":
			entity[propName] = id * 10
		case "number":
			entity[propName] = float64(id) * 123.45
		case "boolean":
			entity[propName] = true
		default:
			entity[propName] = fmt.Sprintf("sample_%s", propName)
		}
	}

	return entity
}
