package diagnostics

import (
	"fmt"
	"io"
	"sync"
)

// Confidence represents the certainty level of extracted metadata.
type Confidence string

const (
	ConfidenceVerified Confidence = "verified" // Directly confirmed by runtime or authoritative catalog
	ConfidenceInferred Confidence = "inferred" // Extracted from source declaration or pattern
	ConfidenceUnknown  Confidence = "unknown"  // Insufficient information
)

// Severity represents diagnostic severity.
type Severity string

const (
	SeverityInfo    Severity = "info"
	SeverityWarning Severity = "warning"
	SeverityError   Severity = "error"
)

// Diagnostic represents an event, warning, or error during extraction and compilation.
type Diagnostic struct {
	Code       string     `json:"code"`
	Severity   Severity   `json:"severity"`
	Message    string     `json:"message"`
	Source     string     `json:"source,omitempty"`
	Line       int        `json:"line,omitempty"`
	Confidence Confidence `json:"confidence,omitempty"`
}

// Collector collects diagnostics across the entire pipeline.
type Collector struct {
	mu          sync.Mutex
	diagnostics []Diagnostic
}

// NewCollector creates a new diagnostic collector.
func NewCollector() *Collector {
	return &Collector{
		diagnostics: make([]Diagnostic, 0),
	}
}

// Add appends a diagnostic to the collector.
func (c *Collector) Add(d Diagnostic) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.diagnostics = append(c.diagnostics, d)
}

// AddInfo convenience method for informational diagnostics.
func (c *Collector) AddInfo(code, message, source string, confidence Confidence) {
	c.Add(Diagnostic{
		Code:       code,
		Severity:   SeverityInfo,
		Message:    message,
		Source:     source,
		Confidence: confidence,
	})
}

// AddWarning convenience method for warning diagnostics.
func (c *Collector) AddWarning(code, message, source string, line int, confidence Confidence) {
	c.Add(Diagnostic{
		Code:       code,
		Severity:   SeverityWarning,
		Message:    message,
		Source:     source,
		Line:       line,
		Confidence: confidence,
	})
}

// AddError convenience method for error diagnostics.
func (c *Collector) AddError(code, message, source string, line int) {
	c.Add(Diagnostic{
		Code:       code,
		Severity:   SeverityError,
		Message:    message,
		Source:     source,
		Line:       line,
		Confidence: ConfidenceUnknown,
	})
}

// All returns a slice of all collected diagnostics.
func (c *Collector) All() []Diagnostic {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]Diagnostic, len(c.diagnostics))
	copy(out, c.diagnostics)
	return out
}

// HasErrors returns true if any diagnostic has error severity.
func (c *Collector) HasErrors() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, d := range c.diagnostics {
		if d.Severity == SeverityError {
			return true
		}
	}
	return false
}

// HasIncomplete returns true if any item has unknown or unverified confidence.
func (c *Collector) HasIncomplete() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, d := range c.diagnostics {
		if d.Confidence == ConfidenceUnknown || d.Severity == SeverityWarning {
			return true
		}
	}
	return false
}

// PrintFormatted prints diagnostics to an output writer (e.g. stderr).
func (c *Collector) PrintFormatted(w io.Writer) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, d := range c.diagnostics {
		prefix := "INFO"
		if d.Severity == SeverityWarning {
			prefix = "WARN"
		} else if d.Severity == SeverityError {
			prefix = "ERROR"
		}

		location := ""
		if d.Source != "" {
			if d.Line > 0 {
				location = fmt.Sprintf(" [%s:%d]", d.Source, d.Line)
			} else {
				location = fmt.Sprintf(" [%s]", d.Source)
			}
		}

		conf := ""
		if d.Confidence != "" {
			conf = fmt.Sprintf(" (%s)", d.Confidence)
		}

		fmt.Fprintf(w, "[%s] %s%s: %s%s\n", prefix, d.Code, location, d.Message, conf)
	}
}
