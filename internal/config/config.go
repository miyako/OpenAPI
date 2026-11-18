package config

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ServerConfig defines an API server.
type ServerConfig struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

// Tool4DConfig defines tool4d runner options.
type Tool4DConfig struct {
	Path    string `json:"path,omitempty"`
	Version string `json:"version,omitempty"`
	Timeout string `json:"timeout,omitempty"`
}

// FilterConfig specifies dataclasses or handlers to include/exclude.
type FilterConfig struct {
	DataClasses []string `json:"dataclasses,omitempty"`
	Handlers    []string `json:"handlers,omitempty"`
}

// InfoConfig specifies custom OpenAPI info block settings.
type InfoConfig struct {
	Title          string `json:"title,omitempty"`
	Version        string `json:"version,omitempty"`
	Description    string `json:"description,omitempty"`
	TermsOfService string `json:"termsOfService,omitempty"`
}

// Config represents the project configuration.
type Config struct {
	Servers []ServerConfig `json:"servers,omitempty"`
	Tool4D  Tool4DConfig   `json:"tool4d,omitempty"`
	Include FilterConfig   `json:"include,omitempty"`
	Exclude FilterConfig   `json:"exclude,omitempty"`
	Overlay string         `json:"overlay,omitempty"`
	Strict  bool           `json:"strict,omitempty"`
	Info    InfoConfig     `json:"info,omitempty"`
}

// DefaultConfig returns default configuration.
func DefaultConfig() *Config {
	return &Config{
		Servers: []ServerConfig{
			{
				URL:         "http://127.0.0.1:8044",
				Description: "Local 4D REST Server",
			},
		},
		Strict: false,
	}
}

// FindConfigFile looks for openapi.yaml, openapi.json, or .4d-openapi.yaml in the project directory.
func FindConfigFile(projectDir string) string {
	candidates := []string{
		filepath.Join(projectDir, "openapi.yaml"),
		filepath.Join(projectDir, "openapi.yml"),
		filepath.Join(projectDir, "openapi.json"),
		filepath.Join(projectDir, ".4d-openapi.yaml"),
		filepath.Join(projectDir, ".4d-openapi.json"),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}

// LoadConfig reads configuration from path.
func LoadConfig(path string) (*Config, error) {
	cfg := DefaultConfig()
	if path == "" {
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	// Try parsing as JSON first
	trimmed := strings.TrimSpace(string(data))
	if strings.HasPrefix(trimmed, "{") {
		if err := json.Unmarshal(data, cfg); err == nil {
			return cfg, nil
		}
	}

	// Parse simple YAML
	if err := parseSimpleYAML(string(data), cfg); err != nil {
		return nil, fmt.Errorf("parse config YAML: %w", err)
	}

	return cfg, nil
}

// parseSimpleYAML performs robust line-based YAML parsing for config files.
func parseSimpleYAML(content string, cfg *Config) error {
	scanner := bufio.NewScanner(strings.NewReader(content))
	var currentSection string
	var currentServer *ServerConfig

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		indent := len(line) - len(strings.TrimLeft(line, " "))

		// Top level keys
		if indent == 0 && strings.Contains(trimmed, ":") {
			parts := strings.SplitN(trimmed, ":", 2)
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])

			currentSection = key
			switch key {
			case "strict":
				cfg.Strict = (val == "true" || val == "yes" || val == "1")
			case "overlay":
				cfg.Overlay = strings.Trim(val, `"'`)
			}
			continue
		}

		// Nested items
		if currentSection == "servers" {
			if strings.HasPrefix(trimmed, "- ") {
				if len(cfg.Servers) == 1 && cfg.Servers[0].Description == "Local 4D REST Server" {
					// Replace default placeholder server
					cfg.Servers = nil
				}
				cfg.Servers = append(cfg.Servers, ServerConfig{})
				currentServer = &cfg.Servers[len(cfg.Servers)-1]
				rem := strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
				if strings.Contains(rem, ":") {
					kv := strings.SplitN(rem, ":", 2)
					k := strings.TrimSpace(kv[0])
					v := strings.Trim(strings.TrimSpace(kv[1]), `"'`)
					if k == "url" {
						currentServer.URL = v
					} else if k == "description" {
						currentServer.Description = v
					}
				}
			} else if currentServer != nil && strings.Contains(trimmed, ":") {
				kv := strings.SplitN(trimmed, ":", 2)
				k := strings.TrimSpace(kv[0])
				v := strings.Trim(strings.TrimSpace(kv[1]), `"'`)
				if k == "url" {
					currentServer.URL = v
				} else if k == "description" {
					currentServer.Description = v
				}
			}
		} else if currentSection == "tool4d" {
			kv := strings.SplitN(trimmed, ":", 2)
			if len(kv) == 2 {
				k := strings.TrimSpace(kv[0])
				v := strings.Trim(strings.TrimSpace(kv[1]), `"'`)
				if k == "path" {
					cfg.Tool4D.Path = v
				} else if k == "version" {
					cfg.Tool4D.Version = v
				} else if k == "timeout" {
					cfg.Tool4D.Timeout = v
				}
			}
		} else if currentSection == "info" {
			kv := strings.SplitN(trimmed, ":", 2)
			if len(kv) == 2 {
				k := strings.TrimSpace(kv[0])
				v := strings.Trim(strings.TrimSpace(kv[1]), `"'`)
				if k == "title" {
					cfg.Info.Title = v
				} else if k == "version" {
					cfg.Info.Version = v
				} else if k == "description" {
					cfg.Info.Description = v
				}
			}
		}
	}

	return scanner.Err()
}
