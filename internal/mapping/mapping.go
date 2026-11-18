package mapping

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/miyako/OpenAPI/internal/config"
	"github.com/miyako/OpenAPI/internal/diagnostics"
	"github.com/miyako/OpenAPI/internal/ir"
	"github.com/miyako/OpenAPI/internal/openapi"
)

// MapIRToOpenAPI converts normalized IR into an OpenAPI 3.1 specification.
func MapIRToOpenAPI(inputIR *ir.IR, cfg *config.Config, col *diagnostics.Collector) (*openapi.OpenAPISpec, error) {
	title := fmt.Sprintf("%s API", inputIR.Project.Name)
	if cfg != nil && cfg.Info.Title != "" {
		title = cfg.Info.Title
	}

	version := "1.0.0"
	if cfg != nil && cfg.Info.Version != "" {
		version = cfg.Info.Version
	}

	desc := "OpenAPI 3.1 specification generated from 4D project by 4d-openapi."
	if cfg != nil && cfg.Info.Description != "" {
		desc = cfg.Info.Description
	}

	spec := openapi.NewSpec(title, version, desc)

	// Servers
	serverSet := make(map[string]bool)
	if cfg != nil && len(cfg.Servers) > 0 {
		for _, s := range cfg.Servers {
			if !serverSet[s.URL] {
				serverSet[s.URL] = true
				spec.Servers = append(spec.Servers, openapi.Server{
					URL:         s.URL,
					Description: s.Description,
				})
			}
		}
	} else {
		spec.Servers = append(spec.Servers, openapi.Server{
			URL:         "http://127.0.0.1:8044",
			Description: "Default 4D REST Server",
		})
	}

	// 1. Map DataClasses (Surface A)
	for _, dc := range inputIR.DataClasses {
		if !dc.Exposed {
			continue
		}
		mapDataClassToSpec(spec, dc)
	}

	// 2. Map Exposed Class Functions (Surface B)
	for _, fn := range inputIR.Functions {
		if !fn.Exposed {
			continue
		}
		mapFunctionToSpec(spec, fn)
	}

	// 3. Map Custom HTTP Handlers (Surface C)
	for _, h := range inputIR.Handlers {
		mapHandlerToSpec(spec, h, col)
	}

	// 4. Apply Overlays if specified
	if cfg != nil && cfg.Overlay != "" {
		overlayPath := cfg.Overlay
		if !strings.HasPrefix(overlayPath, "/") && inputIR.Project.Path != "" {
			// Resolve relative to project directory
			projDir := inputIR.Project.Path
			if strings.HasSuffix(strings.ToLower(projDir), ".4dproject") {
				projDir = strings.TrimSuffix(projDir, "/Project/"+inputIR.Project.Name+".4DProject")
				projDir = strings.TrimSuffix(projDir, "/"+inputIR.Project.Name+".4DProject")
			}
			candidate := overlayPath
			if _, err := os.Stat(candidate); err != nil {
				// Try project dir
				cand2 := filepath.Join(projDir, overlayPath)
				if _, err2 := os.Stat(cand2); err2 == nil {
					overlayPath = cand2
				}
			}
		}
		_ = openapi.ApplyOverlay(spec, overlayPath, col)
	}

	return spec, nil
}

func mapDataClassToSpec(spec *openapi.OpenAPISpec, dc ir.DataClass) {
	tag := dc.Name
	spec.Tags = append(spec.Tags, openapi.Tag{
		Name:        tag,
		Description: fmt.Sprintf("ORDA REST operations for %s DataClass", dc.Name),
	})

	// Generate Schemas: {DataClass} and {DataClass}Selection
	entitySchema := &openapi.Schema{
		Type:        "object",
		Description: fmt.Sprintf("Entity representation of %s", dc.Name),
		Properties:  make(map[string]*openapi.Schema),
		Required:    make([]string, 0),
	}

	// Internal ORDA stamps
	entitySchema.Properties["__KEY"] = &openapi.Schema{
		Type:        "string",
		Description: "Primary key identifier of entity in 4D",
		ReadOnly:    true,
	}
	entitySchema.Properties["__STAMP"] = &openapi.Schema{
		Type:        "integer",
		Format:      "int64",
		Description: "Revision stamp for optimistic locking",
		ReadOnly:    true,
	}

	for _, attr := range dc.Attributes {
		if !attr.Exposed {
			continue
		}

		var propSchema *openapi.Schema
		if attr.TargetDataClass != "" {
			if attr.RelationType == "one-to-many" {
				propSchema = &openapi.Schema{
					Type:        "array",
					Description: attr.Description,
					ReadOnly:    true,
					Items: &openapi.Schema{
						Ref: fmt.Sprintf("#/components/schemas/%s", attr.TargetDataClass),
					},
				}
			} else {
				propSchema = &openapi.Schema{
					Description: attr.Description,
					ReadOnly:    true,
					Ref:         fmt.Sprintf("#/components/schemas/%s", attr.TargetDataClass),
				}
			}
		} else {
			var propType interface{} = attr.Type
			if attr.Nullable {
				propType = []string{attr.Type, "null"}
			}

			propSchema = &openapi.Schema{
				Type:        propType,
				Format:      attr.Format,
				Description: attr.Description,
				ReadOnly:    attr.ReadOnly,
			}
		}

		entitySchema.Properties[attr.Name] = propSchema
		if attr.PrimaryKey {
			entitySchema.Required = append(entitySchema.Required, attr.Name)
		}
	}

	spec.Components.Schemas[dc.Name] = entitySchema

	// Selection schema
	selectionSchema := &openapi.Schema{
		Type:        "object",
		Description: fmt.Sprintf("Entity selection collection response for %s", dc.Name),
		Properties: map[string]*openapi.Schema{
			"__COUNT": {Type: "integer", Description: "Total count of matching entities"},
			"__SENT":  {Type: "integer", Description: "Count of entities sent in page"},
			"__FIRST": {Type: "integer", Description: "First entity index in selection"},
			"__ENTITIES": {
				Type: "array",
				Items: &openapi.Schema{
					Ref: fmt.Sprintf("#/components/schemas/%s", dc.Name),
				},
			},
		},
	}
	spec.Components.Schemas[fmt.Sprintf("%sSelection", dc.Name)] = selectionSchema

	// Collection Path: /rest/{DataClass}
	collectionPath := fmt.Sprintf("/rest/%s", dc.Name)
	spec.Paths[collectionPath] = openapi.PathItem{
		Summary: fmt.Sprintf("%s collection endpoint", dc.Name),
		Get: &openapi.Operation{
			OperationID: fmt.Sprintf("list%s", dc.Name),
			Summary:     fmt.Sprintf("Retrieve %s entities", dc.Name),
			Description: fmt.Sprintf("Query and retrieve entity collection for %s with standard ORDA query parameters.", dc.Name),
			Tags:        []string{tag},
			Parameters:  buildORDAParameters(),
			Responses: map[string]openapi.Response{
				"200": {
					Description: "Successful query",
					Content: map[string]openapi.MediaType{
						"application/json": {
							Schema: &openapi.Schema{
								Ref: fmt.Sprintf("#/components/schemas/%sSelection", dc.Name),
							},
						},
					},
				},
			},
		},
		Post: &openapi.Operation{
			OperationID: fmt.Sprintf("create%s", dc.Name),
			Summary:     fmt.Sprintf("Create a new %s entity", dc.Name),
			Tags:        []string{tag},
			RequestBody: &openapi.RequestBody{
				Required:    true,
				Description: fmt.Sprintf("Payload for new %s", dc.Name),
				Content: map[string]openapi.MediaType{
					"application/json": {
						Schema: &openapi.Schema{
							Ref: fmt.Sprintf("#/components/schemas/%s", dc.Name),
						},
					},
				},
			},
			Responses: map[string]openapi.Response{
				"200": {
					Description: "Created entity",
					Content: map[string]openapi.MediaType{
						"application/json": {
							Schema: &openapi.Schema{
								Ref: fmt.Sprintf("#/components/schemas/%s", dc.Name),
							},
						},
					},
				},
			},
		},
	}

	// Entity Path: /rest/{DataClass}({key})
	entityPath := fmt.Sprintf("/rest/%s({key})", dc.Name)
	spec.Paths[entityPath] = openapi.PathItem{
		Summary: fmt.Sprintf("Single %s entity endpoint", dc.Name),
		Parameters: []openapi.Parameter{
			{
				Name:        "key",
				In:          "path",
				Required:    true,
				Description: "Primary key identifier of entity",
				Schema:      &openapi.Schema{Type: "string"},
			},
		},
		Get: &openapi.Operation{
			OperationID: fmt.Sprintf("get%s", dc.Name),
			Summary:     fmt.Sprintf("Retrieve single %s by key", dc.Name),
			Tags:        []string{tag},
			Responses: map[string]openapi.Response{
				"200": {
					Description: "Entity found",
					Content: map[string]openapi.MediaType{
						"application/json": {
							Schema: &openapi.Schema{
								Ref: fmt.Sprintf("#/components/schemas/%s", dc.Name),
							},
						},
					},
				},
				"404": {Description: "Entity not found"},
			},
		},
		Put: &openapi.Operation{
			OperationID: fmt.Sprintf("update%s", dc.Name),
			Summary:     fmt.Sprintf("Update %s entity", dc.Name),
			Tags:        []string{tag},
			RequestBody: &openapi.RequestBody{
				Required: true,
				Content: map[string]openapi.MediaType{
					"application/json": {
						Schema: &openapi.Schema{
							Ref: fmt.Sprintf("#/components/schemas/%s", dc.Name),
						},
					},
				},
			},
			Responses: map[string]openapi.Response{
				"200": {
					Description: "Entity updated",
					Content: map[string]openapi.MediaType{
						"application/json": {
							Schema: &openapi.Schema{
								Ref: fmt.Sprintf("#/components/schemas/%s", dc.Name),
							},
						},
					},
				},
				"404": {Description: "Entity not found"},
			},
		},
		Delete: &openapi.Operation{
			OperationID: fmt.Sprintf("delete%s", dc.Name),
			Summary:     fmt.Sprintf("Delete %s entity", dc.Name),
			Tags:        []string{tag},
			Responses: map[string]openapi.Response{
				"200": {
					Description: "Entity deleted successfully",
					Content: map[string]openapi.MediaType{
						"application/json": {
							Schema: &openapi.Schema{
								Type: "object",
								Properties: map[string]*openapi.Schema{
									"ok": {Type: "boolean"},
								},
							},
						},
					},
				},
				"404": {Description: "Entity not found"},
			},
		},
	}
}

func mapFunctionToSpec(spec *openapi.OpenAPISpec, fn ir.Function) {
	var path string
	tag := fn.ClassName

	switch fn.Target {
	case ir.TargetDataStore:
		path = fmt.Sprintf("/rest/$catalog/%s", fn.Name)
		tag = "DataStore"
	case ir.TargetDataClass:
		path = fmt.Sprintf("/rest/%s/%s", fn.ClassName, fn.Name)
	case ir.TargetEntity:
		path = fmt.Sprintf("/rest/%s({key})/%s", fn.ClassName, fn.Name)
	case ir.TargetEntitySelection:
		path = fmt.Sprintf("/rest/%s/%s", fn.ClassName, fn.Name)
	case ir.TargetSingleton:
		path = fmt.Sprintf("/rest/$singleton/%s/%s", fn.ClassName, fn.Name)
	default:
		path = fmt.Sprintf("/rest/%s/%s", fn.ClassName, fn.Name)
	}

	pathItem, exists := spec.Paths[path]
	if !exists {
		pathItem = openapi.PathItem{
			Summary: fmt.Sprintf("Class function %s.%s", fn.ClassName, fn.Name),
		}
	}

	summary := fn.Summary
	if summary == "" {
		summary = fmt.Sprintf("Call %s.%s", fn.ClassName, fn.Name)
	}

	// Always generate POST operation for ORDA class functions
	opID := fmt.Sprintf("%s_%s_post", fn.ClassName, fn.Name)
	postOp := &openapi.Operation{
		OperationID: opID,
		Summary:     summary,
		Description: fn.Description,
		Tags:        []string{tag},
		Responses:   buildFunctionResponse(fn),
	}

	if len(fn.Parameters) > 0 {
		postOp.RequestBody = buildFunctionRequestBody(fn)
	}
	pathItem.Post = postOp

	// If onHTTPGet is enabled, also generate GET operation
	if fn.OnHTTPGet {
		getOpID := fmt.Sprintf("%s_%s_get", fn.ClassName, fn.Name)
		getOp := &openapi.Operation{
			OperationID: getOpID,
			Summary:     summary + " (HTTP GET)",
			Description: fn.Description,
			Tags:        []string{tag},
			Responses:   buildFunctionResponse(fn),
		}

		if len(fn.Parameters) > 0 {
			getOp.Parameters = append(getOp.Parameters, openapi.Parameter{
				Name:        "$params",
				In:          "query",
				Description: "JSON array string of function parameters, e.g. [\"val1\", 2]",
				Required:    false,
				Schema:      &openapi.Schema{Type: "string"},
			})
		}
		pathItem.Get = getOp
	}

	spec.Paths[path] = pathItem
}

func mapHandlerToSpec(spec *openapi.OpenAPISpec, h ir.Handler, col *diagnostics.Collector) {
	rawPattern := h.Pattern
	path := rawPattern
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	// Translate simple regex (e.g. /users/([0-9]+)) to OpenAPI path template (/users/{id})
	extensions := make(map[string]interface{})
	if h.IsRegex {
		extensions["x-4d-regexPattern"] = rawPattern
		path = cleanRegexToOpenAPIPath(rawPattern)
	}

	tag := h.Class
	if tag == "" {
		tag = "CustomHandlers"
	}

	summary := h.Summary
	if summary == "" {
		summary = fmt.Sprintf("Handler %s.%s", h.Class, h.Method)
	}

	pathItem, exists := spec.Paths[path]
	if !exists {
		pathItem = openapi.PathItem{
			Summary: fmt.Sprintf("Custom handler route for %s", path),
		}
	}

	for _, verb := range h.Verbs {
		op := &openapi.Operation{
			OperationID: fmt.Sprintf("%s_%s_%s", h.Class, h.Method, strings.ToLower(verb)),
			Summary:     summary,
			Description: h.Description,
			Tags:        []string{tag},
			Extensions:  extensions,
			Responses: map[string]openapi.Response{
				"200": {
					Description: "Successful operation",
				},
			},
		}

		switch strings.ToUpper(verb) {
		case "GET":
			pathItem.Get = op
		case "POST":
			pathItem.Post = op
		case "PUT":
			pathItem.Put = op
		case "DELETE":
			pathItem.Delete = op
		case "PATCH":
			pathItem.Patch = op
		}
	}

	spec.Paths[path] = pathItem
}

func cleanRegexToOpenAPIPath(regex string) string {
	res := regex
	// Replace simple captures like ([0-9]+) or ([^/]+) or (.*)
	if strings.Contains(res, "(.*)") {
		res = strings.ReplaceAll(res, "(.*)", "{param}")
	}
	if strings.Contains(res, "([0-9]+)") {
		res = strings.ReplaceAll(res, "([0-9]+)", "{id}")
	}
	if strings.HasSuffix(res, "/.*") {
		res = strings.TrimSuffix(res, "/.*") + "/{tail}"
	}
	res = strings.TrimPrefix(res, "^")
	res = strings.TrimSuffix(res, "$")
	if !strings.HasPrefix(res, "/") {
		res = "/" + res
	}
	return res
}

func buildORDAParameters() []openapi.Parameter {
	return []openapi.Parameter{
		{
			Name:        "$top",
			In:          "query",
			Description: "Maximum number of entities to return (pagination limit)",
			Schema:      &openapi.Schema{Type: "integer"},
		},
		{
			Name:        "$skip",
			In:          "query",
			Description: "Number of entities to skip from the beginning (pagination offset)",
			Schema:      &openapi.Schema{Type: "integer"},
		},
		{
			Name:        "$filter",
			In:          "query",
			Description: "ORDA filter expression, e.g. 'name = :1 and age > :2'",
			Schema:      &openapi.Schema{Type: "string"},
		},
		{
			Name:        "$params",
			In:          "query",
			Description: "JSON array of parameter values for filter placeholders",
			Schema:      &openapi.Schema{Type: "string"},
		},
		{
			Name:        "$orderby",
			In:          "query",
			Description: "Sorting expression, e.g. 'name desc, ID asc'",
			Schema:      &openapi.Schema{Type: "string"},
		},
		{
			Name:        "$attributes",
			In:          "query",
			Description: "Attributes to return, e.g. 'ID,name,company.name'",
			Schema:      &openapi.Schema{Type: "string"},
		},
		{
			Name:        "$method",
			In:          "query",
			Description: "Result format or entity set method, e.g. 'entityset'",
			Schema:      &openapi.Schema{Type: "string"},
		},
		{
			Name:        "$compute",
			In:          "query",
			Description: "Compute aggregate expression, e.g. 'salary:sum'",
			Schema:      &openapi.Schema{Type: "string"},
		},
		{
			Name:        "$entityset",
			In:          "query",
			Description: "Entity set identifier to reuse an existing query result",
			Schema:      &openapi.Schema{Type: "string"},
		},
	}
}

func buildFunctionResponse(fn ir.Function) map[string]openapi.Response {
	resp := map[string]openapi.Response{
		"200": {
			Description: "Successful function invocation",
			Content: map[string]openapi.MediaType{
				"application/json": {
					Schema: &openapi.Schema{
						Type: "object",
						Properties: map[string]*openapi.Schema{
							"result": {
								Type: "string",
							},
						},
					},
				},
			},
		},
	}

	if fn.ReturnType != nil {
		resp["200"].Content["application/json"].Schema.Properties["result"].Type = fn.ReturnType.Type
		if fn.ReturnType.Format != "" {
			resp["200"].Content["application/json"].Schema.Properties["result"].Format = fn.ReturnType.Format
		}
	}

	return resp
}

func buildFunctionRequestBody(fn ir.Function) *openapi.RequestBody {
	props := make(map[string]*openapi.Schema)
	var req []string

	for _, p := range fn.Parameters {
		props[p.Name] = &openapi.Schema{
			Type:        p.Type,
			Format:      p.Format,
			Description: p.Description,
		}
		if p.Required {
			req = append(req, p.Name)
		}
	}

	return &openapi.RequestBody{
		Description: fmt.Sprintf("Parameters for %s", fn.Name),
		Required:    true,
		Content: map[string]openapi.MediaType{
			"application/json": {
				Schema: &openapi.Schema{
					Type:       "object",
					Properties: props,
					Required:   req,
				},
			},
		},
	}
}
