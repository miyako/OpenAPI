package server

import (
	"context"
	_ "embed"
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

//go:embed assets/index.html
var defaultIndexHTML []byte

// Server represents the independent HTTP documentation server.
type Server struct {
	ListenAddr string
	AuthToken  string
	SpecYAML   []byte
	SpecJSON   []byte
	HTML       []byte
	httpServer *http.Server
}

// Config holds options for starting the documentation server.
type Config struct {
	ListenAddr string
	AuthToken  string
	SpecFile   string
	Spec       *openapi.OpenAPISpec
}

// NewServer initializes an independent HTTP server.
func NewServer(cfg Config) (*Server, error) {
	listen := cfg.ListenAddr
	if listen == "" {
		listen = "127.0.0.1:8088"
	}

	var yamlBytes []byte
	var jsonBytes []byte

	if cfg.Spec != nil {
		var err error
		yamlBytes, err = cfg.Spec.ToYAML()
		if err != nil {
			return nil, fmt.Errorf("serialize spec to yaml: %w", err)
		}
		jsonBytes, err = cfg.Spec.ToJSON()
		if err != nil {
			return nil, fmt.Errorf("serialize spec to json: %w", err)
		}
	} else if cfg.SpecFile != "" {
		data, err := os.ReadFile(cfg.SpecFile)
		if err != nil {
			return nil, fmt.Errorf("read spec file: %w", err)
		}
		if strings.HasSuffix(cfg.SpecFile, ".json") {
			jsonBytes = data
			// Could convert to YAML if needed, or serve as is
			yamlBytes = data
		} else {
			yamlBytes = data
			jsonBytes = data
		}
	} else {
		return nil, fmt.Errorf("neither spec nor specFile provided")
	}

	return &Server{
		ListenAddr: listen,
		AuthToken:  cfg.AuthToken,
		SpecYAML:   yamlBytes,
		SpecJSON:   jsonBytes,
		HTML:       defaultIndexHTML,
	}, nil
}

// Start runs the HTTP server and blocks until context is cancelled or SIGINT/SIGTERM is received.
func (s *Server) Start() error {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":  "ok",
			"version": "1.0.0",
		})
	})

	mux.HandleFunc("/openapi.yaml", s.authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/yaml; charset=utf-8")
		_, _ = w.Write(s.SpecYAML)
	}))

	mux.HandleFunc("/openapi.json", s.authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write(s.SpecJSON)
	}))

	mux.HandleFunc("/docs", s.authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(s.HTML)
	}))

	mux.HandleFunc("/", s.authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/docs", http.StatusTemporaryRedirect)
			return
		}
		http.NotFound(w, r)
	}))

	listener, err := net.Listen("tcp", s.ListenAddr)
	if err != nil {
		return fmt.Errorf("failed to bind to %s: %w", s.ListenAddr, err)
	}

	s.httpServer = &http.Server{
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Printf("[4d-openapi serve] Listening on http://%s\n", s.ListenAddr)
	log.Printf("[4d-openapi serve] Documentation available at: http://%s/docs\n", s.ListenAddr)

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
		log.Println("[4d-openapi serve] Shutting down server gracefully...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.httpServer.Shutdown(ctx)
	case err := <-serverErr:
		return err
	}
}

func (s *Server) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.AuthToken != "" {
			authHeader := r.Header.Get("Authorization")
			token := strings.TrimPrefix(authHeader, "Bearer ")
			if token != s.AuthToken {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}
		next(w, r)
	}
}
