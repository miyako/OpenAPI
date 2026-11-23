package codegen

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/miyako/OpenAPI/internal/openapi"
)

// PostmanCollection represents Postman Collection Format v2.1.0
type PostmanCollection struct {
	Info     PostmanInfo       `json:"info"`
	Item     []PostmanItem     `json:"item"`
	Variable []PostmanVariable `json:"variable,omitempty"`
}

type PostmanInfo struct {
	PostmanID   string `json:"_postman_id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Schema      string `json:"schema"`
	Version     string `json:"version,omitempty"`
}

type PostmanItem struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Item        []PostmanItem   `json:"item,omitempty"`
	Request     *PostmanRequest `json:"request,omitempty"`
	Response    []interface{}   `json:"response,omitempty"`
}

type PostmanRequest struct {
	Method      string          `json:"method"`
	Header      []PostmanHeader `json:"header"`
	Body        *PostmanBody    `json:"body,omitempty"`
	URL         PostmanURL      `json:"url"`
	Description string          `json:"description,omitempty"`
}

type PostmanHeader struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description,omitempty"`
}

type PostmanBody struct {
	Mode    string                 `json:"mode"`
	Raw     string                 `json:"raw"`
	Options map[string]interface{} `json:"options,omitempty"`
}

type PostmanURL struct {
	Raw      string            `json:"raw"`
	Host     []string          `json:"host"`
	Path     []string          `json:"path"`
	Query    []PostmanQuery    `json:"query,omitempty"`
	Variable []PostmanVariable `json:"variable,omitempty"`
}

type PostmanQuery struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description,omitempty"`
	Disabled    bool   `json:"disabled,omitempty"`
}

type PostmanVariable struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description,omitempty"`
	Type        string `json:"type,omitempty"`
}

var pathVarRegex = regexp.MustCompile(`\{([^}]+)\}`)

// GeneratePostmanCollection converts an OpenAPI 3.1 specification to Postman Collection v2.1.0 JSON.
func GeneratePostmanCollection(spec *openapi.OpenAPISpec, baseURL string) ([]byte, error) {
	if baseURL == "" {
		if len(spec.Servers) > 0 && spec.Servers[0].URL != "" {
			baseURL = strings.TrimRight(spec.Servers[0].URL, "/")
		} else {
			baseURL = "http://127.0.0.1:8044"
		}
	} else {
		baseURL = strings.TrimRight(baseURL, "/")
	}

	h := sha256.New()
	h.Write([]byte(spec.Info.Title + ":" + spec.Info.Version))
	hashHex := hex.EncodeToString(h.Sum(nil))
	postmanID := fmt.Sprintf("%s-%s-%s-%s-%s",
		hashHex[0:8], hashHex[8:12], hashHex[12:16], hashHex[16:20], hashHex[20:32])

	coll := PostmanCollection{
		Info: PostmanInfo{
			PostmanID:   postmanID,
			Name:        spec.Info.Title,
			Description: spec.Info.Description,
			Schema:      "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
			Version:     spec.Info.Version,
		},
		Item: make([]PostmanItem, 0),
		Variable: []PostmanVariable{
			{
				Key:         "baseUrl",
				Value:       baseURL,
				Description: "Base URL of the 4D REST Server",
				Type:        "string",
			},
		},
	}

	// Group requests by folder (first tag or "General")
	folderMap := make(map[string][]PostmanItem)

	// Ensure deterministic iteration over paths
	pathKeys := make([]string, 0, len(spec.Paths))
	for p := range spec.Paths {
		pathKeys = append(pathKeys, p)
	}
	sort.Strings(pathKeys)

	for _, pathStr := range pathKeys {
		pItem := spec.Paths[pathStr]
		methods := []struct {
			name string
			op   *openapi.Operation
		}{
			{"GET", pItem.Get},
			{"POST", pItem.Post},
			{"PUT", pItem.Put},
			{"DELETE", pItem.Delete},
			{"PATCH", pItem.Patch},
		}

		for _, m := range methods {
			if m.op == nil {
				continue
			}

			reqItem := buildPostmanRequest(pathStr, m.name, m.op, pItem.Parameters)
			folderName := "General"
			if len(m.op.Tags) > 0 && m.op.Tags[0] != "" {
				folderName = m.op.Tags[0]
			}
			folderMap[folderName] = append(folderMap[folderName], reqItem)
		}
	}

	// Stably sort folder names
	folderNames := make([]string, 0, len(folderMap))
	for fn := range folderMap {
		folderNames = append(folderNames, fn)
	}
	sort.Strings(folderNames)

	for _, fn := range folderNames {
		requests := folderMap[fn]
		// Sort requests within folder by Name
		sort.Slice(requests, func(i, j int) bool {
			return requests[i].Name < requests[j].Name
		})

		coll.Item = append(coll.Item, PostmanItem{
			Name:        fn,
			Description: fmt.Sprintf("Requests for %s", fn),
			Item:        requests,
		})
	}

	return json.MarshalIndent(coll, "", "  ")
}

func buildPostmanRequest(pathStr, method string, op *openapi.Operation, commonParams []openapi.Parameter) PostmanItem {
	name := op.Summary
	if name == "" {
		name = op.OperationID
	}
	if name == "" {
		name = fmt.Sprintf("%s %s", method, pathStr)
	}

	// Convert /rest/Company({key}) into /rest/Company(:key) for Postman url representation
	postmanPathRaw := pathVarRegex.ReplaceAllString(pathStr, ":$1")
	fullRawURL := "{{baseUrl}}" + postmanPathRaw

	// Path segments
	cleanPath := strings.Trim(pathStr, "/")
	var pathSegments []string
	if cleanPath != "" {
		for _, seg := range strings.Split(cleanPath, "/") {
			seg = pathVarRegex.ReplaceAllString(seg, ":$1")
			pathSegments = append(pathSegments, seg)
		}
	}

	// Collect parameters
	paramMap := make(map[string]openapi.Parameter)
	for _, p := range commonParams {
		paramMap[p.In+":"+p.Name] = p
	}
	for _, p := range op.Parameters {
		paramMap[p.In+":"+p.Name] = p
	}

	var headers []PostmanHeader
	var queries []PostmanQuery
	var variables []PostmanVariable

	headers = append(headers, PostmanHeader{
		Key:   "Accept",
		Value: "application/json",
	})

	// Find path variables from regex
	matches := pathVarRegex.FindAllStringSubmatch(pathStr, -1)
	foundVars := make(map[string]bool)
	for _, match := range matches {
		if len(match) > 1 {
			varName := match[1]
			foundVars[varName] = true
			desc := ""
			if p, ok := paramMap["path:"+varName]; ok {
				desc = p.Description
			}
			variables = append(variables, PostmanVariable{
				Key:         varName,
				Value:       "",
				Description: desc,
			})
		}
	}

	// Sort query and header parameters
	paramKeys := make([]string, 0, len(paramMap))
	for k := range paramMap {
		paramKeys = append(paramKeys, k)
	}
	sort.Strings(paramKeys)

	for _, k := range paramKeys {
		p := paramMap[k]
		switch p.In {
		case "header":
			headers = append(headers, PostmanHeader{
				Key:         p.Name,
				Value:       "",
				Description: p.Description,
			})
		case "query":
			queries = append(queries, PostmanQuery{
				Key:         p.Name,
				Value:       "",
				Description: p.Description,
				Disabled:    !p.Required,
			})
		}
	}

	var body *PostmanBody
	if op.RequestBody != nil {
		headers = append(headers, PostmanHeader{
			Key:   "Content-Type",
			Value: "application/json",
		})
		body = &PostmanBody{
			Mode: "raw",
			Raw:  "{\n  \n}",
			Options: map[string]interface{}{
				"raw": map[string]string{
					"language": "json",
				},
			},
		}
	}

	return PostmanItem{
		Name:        name,
		Description: op.Description,
		Request: &PostmanRequest{
			Method:      method,
			Header:      headers,
			Body:        body,
			Description: op.Description,
			URL: PostmanURL{
				Raw:      fullRawURL,
				Host:     []string{"{{baseUrl}}"},
				Path:     pathSegments,
				Query:    queries,
				Variable: variables,
			},
		},
		Response: make([]interface{}, 0),
	}
}
