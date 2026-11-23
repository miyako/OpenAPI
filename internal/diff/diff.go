package diff

import (
	"fmt"
	"sort"
	"strings"

	"github.com/miyako/OpenAPI/internal/openapi"
)

// ChangeType defines the semantic nature of a change.
type ChangeType string

const (
	ChangeBreaking    ChangeType = "BREAKING"
	ChangeNonBreaking ChangeType = "NON_BREAKING"
	ChangeInfo        ChangeType = "INFO"
)

// Change represents a single semantic difference between two specs.
type Change struct {
	Type        ChangeType `json:"type"`
	Code        string     `json:"code"`
	Path        string     `json:"path,omitempty"`
	Method      string     `json:"method,omitempty"`
	Description string     `json:"description"`
}

// DiffResult holds all changes found between two OpenAPI specifications.
type DiffResult struct {
	Changes []Change `json:"changes"`
}

// HasBreaking returns true if any detected change is breaking.
func (dr *DiffResult) HasBreaking() bool {
	for _, c := range dr.Changes {
		if c.Type == ChangeBreaking {
			return true
		}
	}
	return false
}

// Compare computes the semantic differences between oldSpec and newSpec.
func Compare(oldSpec, newSpec *openapi.OpenAPISpec) *DiffResult {
	res := &DiffResult{
		Changes: make([]Change, 0),
	}

	// 1. Check Paths
	for path, oldItem := range oldSpec.Paths {
		newItem, exists := newSpec.Paths[path]
		if !exists {
			res.Changes = append(res.Changes, Change{
				Type:        ChangeBreaking,
				Code:        "PATH_REMOVED",
				Path:        path,
				Description: fmt.Sprintf("Path '%s' was removed", path),
			})
			continue
		}

		compareOperations(path, "GET", oldItem.Get, newItem.Get, res)
		compareOperations(path, "POST", oldItem.Post, newItem.Post, res)
		compareOperations(path, "PUT", oldItem.Put, newItem.Put, res)
		compareOperations(path, "DELETE", oldItem.Delete, newItem.Delete, res)
		compareOperations(path, "PATCH", oldItem.Patch, newItem.Patch, res)
	}

	// Check added paths
	for path := range newSpec.Paths {
		if _, exists := oldSpec.Paths[path]; !exists {
			res.Changes = append(res.Changes, Change{
				Type:        ChangeNonBreaking,
				Code:        "PATH_ADDED",
				Path:        path,
				Description: fmt.Sprintf("Path '%s' was added", path),
			})
		}
	}

	// 2. Check Schemas
	for schName, oldSch := range oldSpec.Components.Schemas {
		newSch, exists := newSpec.Components.Schemas[schName]
		if !exists {
			res.Changes = append(res.Changes, Change{
				Type:        ChangeBreaking,
				Code:        "SCHEMA_REMOVED",
				Description: fmt.Sprintf("Component schema '%s' was removed", schName),
			})
			continue
		}

		// Check removed properties
		for propName := range oldSch.Properties {
			if _, propExists := newSch.Properties[propName]; !propExists {
				res.Changes = append(res.Changes, Change{
					Type:        ChangeBreaking,
					Code:        "PROPERTY_REMOVED",
					Description: fmt.Sprintf("Property '%s' was removed from schema '%s'", propName, schName),
				})
			}
		}

		// Check added required properties (breaking change for clients sending requests)
		oldReq := make(map[string]bool)
		for _, r := range oldSch.Required {
			oldReq[r] = true
		}
		for _, r := range newSch.Required {
			if !oldReq[r] {
				res.Changes = append(res.Changes, Change{
					Type:        ChangeBreaking,
					Code:        "REQUIRED_PROPERTY_ADDED",
					Description: fmt.Sprintf("New required property '%s' added to schema '%s'", r, schName),
				})
			}
		}
	}

	// Stable sort changes
	sort.Slice(res.Changes, func(i, j int) bool {
		if res.Changes[i].Type != res.Changes[j].Type {
			return res.Changes[i].Type == ChangeBreaking
		}
		return res.Changes[i].Description < res.Changes[j].Description
	})

	return res
}

func compareOperations(path, method string, oldOp, newOp *openapi.Operation, res *DiffResult) {
	if oldOp != nil && newOp == nil {
		res.Changes = append(res.Changes, Change{
			Type:        ChangeBreaking,
			Code:        "OPERATION_REMOVED",
			Path:        path,
			Method:      method,
			Description: fmt.Sprintf("%s operation on '%s' was removed", method, path),
		})
		return
	}
	if oldOp == nil && newOp != nil {
		res.Changes = append(res.Changes, Change{
			Type:        ChangeNonBreaking,
			Code:        "OPERATION_ADDED",
			Path:        path,
			Method:      method,
			Description: fmt.Sprintf("%s operation on '%s' was added", method, path),
		})
		return
	}
	if oldOp == nil && newOp == nil {
		return
	}

	// Compare parameters: check if a new required parameter was introduced
	oldParamMap := make(map[string]openapi.Parameter)
	for _, p := range oldOp.Parameters {
		oldParamMap[strings.ToLower(p.Name)+":"+strings.ToLower(p.In)] = p
	}

	for _, np := range newOp.Parameters {
		key := strings.ToLower(np.Name) + ":" + strings.ToLower(np.In)
		op, exists := oldParamMap[key]
		if !exists {
			if np.Required {
				res.Changes = append(res.Changes, Change{
					Type:        ChangeBreaking,
					Code:        "REQUIRED_PARAM_ADDED",
					Path:        path,
					Method:      method,
					Description: fmt.Sprintf("New required %s parameter '%s' added to %s %s", np.In, np.Name, method, path),
				})
			} else {
				res.Changes = append(res.Changes, Change{
					Type:        ChangeNonBreaking,
					Code:        "OPTIONAL_PARAM_ADDED",
					Path:        path,
					Method:      method,
					Description: fmt.Sprintf("Optional %s parameter '%s' added to %s %s", np.In, np.Name, method, path),
				})
			}
		} else if !op.Required && np.Required {
			res.Changes = append(res.Changes, Change{
				Type:        ChangeBreaking,
				Code:        "PARAM_NOW_REQUIRED",
				Path:        path,
				Method:      method,
				Description: fmt.Sprintf("Parameter '%s' on %s %s is now required", np.Name, method, path),
			})
		}
	}
}
