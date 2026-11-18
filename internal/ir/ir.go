package ir

import (
	"encoding/json"
	"github.com/miyako/OpenAPI/internal/diagnostics"
)

// CurrentIRVersion defines the supported IR contract version.
const CurrentIRVersion = 1

// Provenance tracks the origin and confidence level of an item.
type Provenance struct {
	File       string                 `json:"file,omitempty"`
	Line       int                    `json:"line,omitempty"`
	Component  string                 `json:"component,omitempty"`
	Confidence diagnostics.Confidence `json:"confidence"`
}

// ProjectMeta describes the 4D project under analysis.
type ProjectMeta struct {
	Name           string `json:"name"`
	Path           string `json:"path"`
	FourDVersion   string `json:"fourDVersion,omitempty"`
	Tool4dBuild    string `json:"tool4dBuild,omitempty"`
	Tool4dVerified bool   `json:"tool4dVerified"`
}

// Component describes an external or local 4D component.
type Component struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
	Source  string `json:"source"` // "local", "github", "environment"
	Path    string `json:"path"`
}

// Attribute represents an attribute / field of a 4D DataClass.
type Attribute struct {
	Name         string     `json:"name"`
	Type         string     `json:"type"`     // OpenAPI type: string, integer, number, boolean, object, array
	Format       string     `json:"format,omitempty"` // int32, int64, float, double, date, date-time
	FourDType    string     `json:"fourDType"` // alpha, text, integer, longint, integer64, real, date, time, boolean, picture, blob, object, collection
	PrimaryKey   bool       `json:"primaryKey,omitempty"`
	ReadOnly     bool       `json:"readOnly,omitempty"`
	Exposed      bool       `json:"exposed"`
	Nullable     bool       `json:"nullable"`
	Autosequence bool       `json:"autosequence,omitempty"`
	Description  string     `json:"description,omitempty"`
	Provenance   Provenance `json:"provenance"`
}

// QueriesSupport defines supported query features for a DataClass.
type QueriesSupport struct {
	Filterable bool `json:"filterable"`
	Sortable   bool `json:"sortable"`
	Paginated  bool `json:"paginated"`
}

// DataClass represents an exposed 4D ORDA DataClass (table).
type DataClass struct {
	Name        string         `json:"name"`
	Exposed     bool           `json:"exposed"`
	Scope       string         `json:"scope,omitempty"` // "public"
	Description string         `json:"description,omitempty"`
	Attributes  []Attribute    `json:"attributes"`
	Queries     QueriesSupport `json:"queries"`
	Provenance  Provenance     `json:"provenance"`
}

// Parameter represents a parameter for a class function or handler.
type Parameter struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Format      string `json:"format,omitempty"`
	FourDType   string `json:"fourDType,omitempty"`
	Required    bool   `json:"required"`
	Description string `json:"description,omitempty"`
}

// ReturnType represents the return type of a function.
type ReturnType struct {
	Type        string `json:"type"`
	Format      string `json:"format,omitempty"`
	FourDType   string `json:"fourDType,omitempty"`
	Description string `json:"description,omitempty"`
}

// FunctionTarget describes the receiver type of an exposed 4D class function.
type FunctionTarget string

const (
	TargetDataStore       FunctionTarget = "datastore"
	TargetDataClass       FunctionTarget = "dataclass"
	TargetEntity          FunctionTarget = "entity"
	TargetEntitySelection FunctionTarget = "entityselection"
	TargetSingleton       FunctionTarget = "singleton"
)

// Function represents an exposed 4D class function.
type Function struct {
	Name        string         `json:"name"`
	Target      FunctionTarget `json:"target"`
	ClassName   string         `json:"className"`
	Exposed     bool           `json:"exposed"`
	OnHTTPGet   bool           `json:"onHTTPGet"`
	Summary     string         `json:"summary,omitempty"`
	Description string         `json:"description,omitempty"`
	Tag         string         `json:"tag,omitempty"`
	Parameters  []Parameter    `json:"parameters,omitempty"`
	ReturnType  *ReturnType    `json:"returnType,omitempty"`
	Provenance  Provenance     `json:"provenance"`
}

// Handler represents a custom HTTP route handler from HTTPHandlers.json.
type Handler struct {
	Class       string     `json:"class"`
	Method      string     `json:"method"`
	Pattern     string     `json:"pattern"`
	IsRegex     bool       `json:"isRegex"`
	Verbs       []string   `json:"verbs"`
	Summary     string     `json:"summary,omitempty"`
	Description string     `json:"description,omitempty"`
	Provenance  Provenance `json:"provenance"`
}

// Privilege represents an access control privilege.
type Privilege struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Roles       []string `json:"roles,omitempty"`
}

// IR represents the complete versioned Intermediate Representation.
type IR struct {
	IRVersion   int                      `json:"irVersion"`
	Project     ProjectMeta              `json:"project"`
	Components  []Component              `json:"components"`
	DataClasses []DataClass              `json:"dataclasses"`
	Functions   []Function               `json:"functions"`
	Handlers    []Handler                `json:"handlers"`
	Privileges  []Privilege              `json:"privileges"`
	Diagnostics []diagnostics.Diagnostic `json:"diagnostics"`
}

// NewIR creates a new initialized IR.
func NewIR(project ProjectMeta) *IR {
	return &IR{
		IRVersion:   CurrentIRVersion,
		Project:     project,
		Components:  make([]Component, 0),
		DataClasses: make([]DataClass, 0),
		Functions:   make([]Function, 0),
		Handlers:    make([]Handler, 0),
		Privileges:  make([]Privilege, 0),
		Diagnostics: make([]diagnostics.Diagnostic, 0),
	}
}

// ToJSON serializes IR to indented deterministic JSON.
func (ir *IR) ToJSON() ([]byte, error) {
	return json.MarshalIndent(ir, "", "  ")
}
