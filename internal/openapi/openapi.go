package openapi

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// OpenAPISpec represents the root OpenAPI 3.1.0 document.
type OpenAPISpec struct {
	OpenAPI    string                 `json:"openapi"`
	Info       Info                   `json:"info"`
	Servers    []Server               `json:"servers,omitempty"`
	Paths      map[string]PathItem    `json:"paths"`
	Components Components             `json:"components"`
	Tags       []Tag                  `json:"tags,omitempty"`
	Extensions map[string]interface{} `json:"-"`
}

type Info struct {
	Title          string  `json:"title"`
	Version        string  `json:"version"`
	Description    string  `json:"description,omitempty"`
	TermsOfService string  `json:"termsOfService,omitempty"`
	Contact        *Contact `json:"contact,omitempty"`
	License        *License `json:"license,omitempty"`
}

type Contact struct {
	Name  string `json:"name,omitempty"`
	URL   string `json:"url,omitempty"`
	Email string `json:"email,omitempty"`
}

type License struct {
	Name string `json:"name"`
	URL  string `json:"url,omitempty"`
}

type Server struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

type Tag struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type PathItem struct {
	Summary     string      `json:"summary,omitempty"`
	Description string      `json:"description,omitempty"`
	Get         *Operation  `json:"get,omitempty"`
	Post        *Operation  `json:"post,omitempty"`
	Put         *Operation  `json:"put,omitempty"`
	Delete      *Operation  `json:"delete,omitempty"`
	Patch       *Operation  `json:"patch,omitempty"`
	Parameters  []Parameter `json:"parameters,omitempty"`
}

type Operation struct {
	OperationID string                 `json:"operationId"`
	Summary     string                 `json:"summary,omitempty"`
	Description string                 `json:"description,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	Parameters  []Parameter            `json:"parameters,omitempty"`
	RequestBody *RequestBody           `json:"requestBody,omitempty"`
	Responses   map[string]Response    `json:"responses"`
	Deprecated  bool                   `json:"deprecated,omitempty"`
	Extensions  map[string]interface{} `json:"-"`
}

type Parameter struct {
	Name        string  `json:"name"`
	In          string  `json:"in"` // "query", "header", "path", "cookie"
	Description string  `json:"description,omitempty"`
	Required    bool    `json:"required,omitempty"`
	Schema      *Schema `json:"schema,omitempty"`
}

type RequestBody struct {
	Description string               `json:"description,omitempty"`
	Required    bool                 `json:"required,omitempty"`
	Content     map[string]MediaType `json:"content"`
}

type Response struct {
	Description string               `json:"description"`
	Content     map[string]MediaType `json:"content,omitempty"`
}

type MediaType struct {
	Schema *Schema `json:"schema,omitempty"`
}

type Schema struct {
	Type        interface{}        `json:"type,omitempty"` // string or []string for nullable union in OpenAPI 3.1
	Format      string             `json:"format,omitempty"`
	Description string             `json:"description,omitempty"`
	Properties  map[string]*Schema `json:"properties,omitempty"`
	Required    []string           `json:"required,omitempty"`
	Items       *Schema            `json:"items,omitempty"`
	Ref         string             `json:"$ref,omitempty"`
	Enum        []string           `json:"enum,omitempty"`
	ReadOnly    bool               `json:"readOnly,omitempty"`
	Extensions  map[string]interface{} `json:"-"`
}

type Components struct {
	Schemas       map[string]*Schema `json:"schemas"`
	SecuritySchemes map[string]interface{} `json:"securitySchemes,omitempty"`
}

// NewSpec initializes an empty OpenAPI 3.1.0 specification.
func NewSpec(title, version, description string) *OpenAPISpec {
	return &OpenAPISpec{
		OpenAPI: "3.1.0",
		Info: Info{
			Title:       title,
			Version:     version,
			Description: description,
		},
		Servers:    make([]Server, 0),
		Paths:      make(map[string]PathItem),
		Components: Components{
			Schemas: make(map[string]*Schema),
		},
		Tags:       make([]Tag, 0),
		Extensions: make(map[string]interface{}),
	}
}

// ToJSON produces deterministic, byte-reproducible indented JSON.
func (s *OpenAPISpec) ToJSON() ([]byte, error) {
	// Custom marshaling for sorted keys and OpenAPI 3.1 extensions
	ordered := s.buildOrderedMap()
	return json.MarshalIndent(ordered, "", "  ")
}

// ToYAML produces deterministic, byte-reproducible YAML.
func (s *OpenAPISpec) ToYAML() ([]byte, error) {
	var sb strings.Builder
	sb.WriteString("openapi: 3.1.0\n")
	sb.WriteString("info:\n")
	sb.WriteString(fmt.Sprintf("  title: %s\n", escapeYAMLString(s.Info.Title)))
	sb.WriteString(fmt.Sprintf("  version: %s\n", escapeYAMLString(s.Info.Version)))
	if s.Info.Description != "" {
		sb.WriteString(fmt.Sprintf("  description: %s\n", escapeYAMLString(s.Info.Description)))
	}

	if len(s.Servers) > 0 {
		sb.WriteString("servers:\n")
		for _, srv := range s.Servers {
			sb.WriteString(fmt.Sprintf("  - url: %s\n", escapeYAMLString(srv.URL)))
			if srv.Description != "" {
				sb.WriteString(fmt.Sprintf("    description: %s\n", escapeYAMLString(srv.Description)))
			}
		}
	}

	if len(s.Tags) > 0 {
		sb.WriteString("tags:\n")
		// Stable sort tags
		tagsCopy := make([]Tag, len(s.Tags))
		copy(tagsCopy, s.Tags)
		sort.Slice(tagsCopy, func(i, j int) bool {
			return tagsCopy[i].Name < tagsCopy[j].Name
		})
		for _, tag := range tagsCopy {
			sb.WriteString(fmt.Sprintf("  - name: %s\n", escapeYAMLString(tag.Name)))
			if tag.Description != "" {
				sb.WriteString(fmt.Sprintf("    description: %s\n", escapeYAMLString(tag.Description)))
			}
		}
	}

	// Paths in stable alphabetical order
	sb.WriteString("paths:\n")
	pathKeys := make([]string, 0, len(s.Paths))
	for p := range s.Paths {
		pathKeys = append(pathKeys, p)
	}
	sort.Strings(pathKeys)

	for _, p := range pathKeys {
		item := s.Paths[p]
		sb.WriteString(fmt.Sprintf("  %s:\n", quoteYAMLKey(p)))
		if item.Summary != "" {
			sb.WriteString(fmt.Sprintf("    summary: %s\n", escapeYAMLString(item.Summary)))
		}

		// Operations in standard order: get, post, put, delete, patch
		writeOpYAML(&sb, "get", item.Get, 4)
		writeOpYAML(&sb, "post", item.Post, 4)
		writeOpYAML(&sb, "put", item.Put, 4)
		writeOpYAML(&sb, "delete", item.Delete, 4)
		writeOpYAML(&sb, "patch", item.Patch, 4)
	}

	// Components
	sb.WriteString("components:\n")
	sb.WriteString("  schemas:\n")
	schemaKeys := make([]string, 0, len(s.Components.Schemas))
	for k := range s.Components.Schemas {
		schemaKeys = append(schemaKeys, k)
	}
	sort.Strings(schemaKeys)

	for _, k := range schemaKeys {
		sch := s.Components.Schemas[k]
		sb.WriteString(fmt.Sprintf("    %s:\n", k))
		writeSchemaYAML(&sb, sch, 6)
	}

	return []byte(sb.String()), nil
}

func (s *OpenAPISpec) buildOrderedMap() map[string]interface{} {
	m := make(map[string]interface{})
	m["openapi"] = s.OpenAPI
	m["info"] = s.Info
	if len(s.Servers) > 0 {
		m["servers"] = s.Servers
	}
	if len(s.Tags) > 0 {
		m["tags"] = s.Tags
	}
	m["paths"] = s.Paths
	m["components"] = s.Components
	return m
}

func writeOpYAML(sb *strings.Builder, method string, op *Operation, indent int) {
	if op == nil {
		return
	}
	spaces := strings.Repeat(" ", indent)
	sb.WriteString(fmt.Sprintf("%s%s:\n", spaces, method))
	sb.WriteString(fmt.Sprintf("%s  operationId: %s\n", spaces, escapeYAMLString(op.OperationID)))
	if op.Summary != "" {
		sb.WriteString(fmt.Sprintf("%s  summary: %s\n", spaces, escapeYAMLString(op.Summary)))
	}
	if op.Description != "" {
		sb.WriteString(fmt.Sprintf("%s  description: %s\n", spaces, escapeYAMLString(op.Description)))
	}
	if len(op.Tags) > 0 {
		sb.WriteString(fmt.Sprintf("%s  tags:\n", spaces))
		for _, t := range op.Tags {
			sb.WriteString(fmt.Sprintf("%s    - %s\n", spaces, escapeYAMLString(t)))
		}
	}

	// Extensions (e.g. x-4d-regexPattern)
	if len(op.Extensions) > 0 {
		extKeys := make([]string, 0, len(op.Extensions))
		for k := range op.Extensions {
			extKeys = append(extKeys, k)
		}
		sort.Strings(extKeys)
		for _, k := range extKeys {
			sb.WriteString(fmt.Sprintf("%s  %s: %s\n", spaces, k, escapeYAMLString(fmt.Sprintf("%v", op.Extensions[k]))))
		}
	}

	if len(op.Parameters) > 0 {
		sb.WriteString(fmt.Sprintf("%s  parameters:\n", spaces))
		// Sort parameters stably by 'in' then 'name'
		sortedParams := make([]Parameter, len(op.Parameters))
		copy(sortedParams, op.Parameters)
		sort.Slice(sortedParams, func(i, j int) bool {
			if sortedParams[i].In != sortedParams[j].In {
				return sortedParams[i].In < sortedParams[j].In
			}
			return sortedParams[i].Name < sortedParams[j].Name
		})

		for _, p := range sortedParams {
			sb.WriteString(fmt.Sprintf("%s    - name: %s\n", spaces, p.Name))
			sb.WriteString(fmt.Sprintf("%s      in: %s\n", spaces, p.In))
			if p.Required {
				sb.WriteString(fmt.Sprintf("%s      required: true\n", spaces))
			}
			if p.Description != "" {
				sb.WriteString(fmt.Sprintf("%s      description: %s\n", spaces, escapeYAMLString(p.Description)))
			}
			if p.Schema != nil {
				sb.WriteString(fmt.Sprintf("%s      schema:\n", spaces))
				writeSchemaYAML(sb, p.Schema, indent+8)
			}
		}
	}

	if op.RequestBody != nil {
		sb.WriteString(fmt.Sprintf("%s  requestBody:\n", spaces))
		if op.RequestBody.Required {
			sb.WriteString(fmt.Sprintf("%s    required: true\n", spaces))
		}
		if op.RequestBody.Description != "" {
			sb.WriteString(fmt.Sprintf("%s    description: %s\n", spaces, escapeYAMLString(op.RequestBody.Description)))
		}
		sb.WriteString(fmt.Sprintf("%s    content:\n", spaces))
		writeContentYAML(sb, op.RequestBody.Content, indent+6)
	}

	sb.WriteString(fmt.Sprintf("%s  responses:\n", spaces))
	respCodes := make([]string, 0, len(op.Responses))
	for c := range op.Responses {
		respCodes = append(respCodes, c)
	}
	sort.Strings(respCodes)

	for _, c := range respCodes {
		resp := op.Responses[c]
		sb.WriteString(fmt.Sprintf("%s    %s:\n", spaces, quoteYAMLKey(c)))
		sb.WriteString(fmt.Sprintf("%s      description: %s\n", spaces, escapeYAMLString(resp.Description)))
		if len(resp.Content) > 0 {
			sb.WriteString(fmt.Sprintf("%s      content:\n", spaces))
			writeContentYAML(sb, resp.Content, indent+8)
		}
	}
}

func writeContentYAML(sb *strings.Builder, content map[string]MediaType, indent int) {
	spaces := strings.Repeat(" ", indent)
	mTypes := make([]string, 0, len(content))
	for mt := range content {
		mTypes = append(mTypes, mt)
	}
	sort.Strings(mTypes)

	for _, mt := range mTypes {
		sb.WriteString(fmt.Sprintf("%s%s:\n", spaces, mt))
		if content[mt].Schema != nil {
			sb.WriteString(fmt.Sprintf("%s  schema:\n", spaces))
			writeSchemaYAML(sb, content[mt].Schema, indent+4)
		}
	}
}

func writeSchemaYAML(sb *strings.Builder, sch *Schema, indent int) {
	if sch == nil {
		return
	}
	spaces := strings.Repeat(" ", indent)

	if sch.Ref != "" {
		sb.WriteString(fmt.Sprintf("%s$ref: %s\n", spaces, escapeYAMLString(sch.Ref)))
		return
	}

	if sch.Type != nil {
		switch t := sch.Type.(type) {
		case string:
			sb.WriteString(fmt.Sprintf("%stype: %s\n", spaces, t))
		case []string:
			sb.WriteString(fmt.Sprintf("%stype:\n", spaces))
			for _, item := range t {
				sb.WriteString(fmt.Sprintf("%s  - %s\n", spaces, item))
			}
		}
	}

	if sch.Format != "" {
		sb.WriteString(fmt.Sprintf("%sformat: %s\n", spaces, sch.Format))
	}
	if sch.Description != "" {
		sb.WriteString(fmt.Sprintf("%sdescription: %s\n", spaces, escapeYAMLString(sch.Description)))
	}
	if sch.ReadOnly {
		sb.WriteString(fmt.Sprintf("%sreadOnly: true\n", spaces))
	}

	if len(sch.Required) > 0 {
		reqCopy := make([]string, len(sch.Required))
		copy(reqCopy, sch.Required)
		sort.Strings(reqCopy)
		sb.WriteString(fmt.Sprintf("%srequired:\n", spaces))
		for _, r := range reqCopy {
			sb.WriteString(fmt.Sprintf("%s  - %s\n", spaces, r))
		}
	}

	if len(sch.Properties) > 0 {
		sb.WriteString(fmt.Sprintf("%sproperties:\n", spaces))
		propKeys := make([]string, 0, len(sch.Properties))
		for pk := range sch.Properties {
			propKeys = append(propKeys, pk)
		}
		sort.Strings(propKeys)

		for _, pk := range propKeys {
			sb.WriteString(fmt.Sprintf("%s  %s:\n", spaces, pk))
			writeSchemaYAML(sb, sch.Properties[pk], indent+4)
		}
	}

	if sch.Items != nil {
		sb.WriteString(fmt.Sprintf("%sitems:\n", spaces))
		writeSchemaYAML(sb, sch.Items, indent+2)
	}
}

func escapeYAMLString(val string) string {
	if strings.ContainsAny(val, "\n:{}[]#&*!|>'\"%@`") || strings.TrimSpace(val) != val {
		b, _ := json.Marshal(val)
		return string(b)
	}
	return val
}

func quoteYAMLKey(k string) string {
	if strings.ContainsAny(k, "{}:/[]") {
		return fmt.Sprintf(`"%s"`, k)
	}
	return k
}
