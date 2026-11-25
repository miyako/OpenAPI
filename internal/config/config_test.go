package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg == nil {
		t.Fatal("Expected default config to be non-nil")
	}
	if len(cfg.Servers) == 0 {
		t.Errorf("Expected default server in config")
	}
	if cfg.Servers[0].URL != "http://127.0.0.1:8044" {
		t.Errorf("Unexpected default server URL: %s", cfg.Servers[0].URL)
	}
}

func TestFindConfigFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "config-find-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if found := FindConfigFile(tmpDir); found != "" {
		t.Errorf("Expected no config file, found: %s", found)
	}

	cfgPath := filepath.Join(tmpDir, "openapi.yaml")
	if err := os.WriteFile(cfgPath, []byte("strict: true\n"), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	if found := FindConfigFile(tmpDir); found != cfgPath {
		t.Errorf("Expected %s, got %s", cfgPath, found)
	}
}

func TestLoadConfigJSONAndYAML(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "config-load-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// JSON test
	jsonPath := filepath.Join(tmpDir, "openapi.json")
	jsonData := `{
		"strict": true,
		"info": {
			"title": "JSON API",
			"version": "1.2.3"
		}
	}`
	if err := os.WriteFile(jsonPath, []byte(jsonData), 0644); err != nil {
		t.Fatalf("Failed to write json config: %v", err)
	}

	cfg, err := LoadConfig(jsonPath)
	if err != nil {
		t.Fatalf("Failed to load JSON config: %v", err)
	}
	if !cfg.Strict {
		t.Errorf("Expected strict: true")
	}
	if cfg.Info.Title != "JSON API" {
		t.Errorf("Expected title 'JSON API', got '%s'", cfg.Info.Title)
	}

	// YAML test
	yamlPath := filepath.Join(tmpDir, "openapi.yaml")
	yamlData := `strict: true
title: YAML API
version: 2.0.0
`
	if err := os.WriteFile(yamlPath, []byte(yamlData), 0644); err != nil {
		t.Fatalf("Failed to write yaml config: %v", err)
	}

	cfgYAML, err := LoadConfig(yamlPath)
	if err != nil {
		t.Fatalf("Failed to load YAML config: %v", err)
	}
	if !cfgYAML.Strict {
		t.Errorf("Expected strict: true in YAML")
	}
}
