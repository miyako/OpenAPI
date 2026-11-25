package diagnostics

import (
	"bytes"
	"strings"
	"sync"
	"testing"
)

func TestCollectorBasic(t *testing.T) {
	col := NewCollector()

	if col.HasErrors() {
		t.Errorf("Expected HasErrors to be false on empty collector")
	}

	col.AddInfo("INFO_TEST", "Test info message", "test.4DProject", ConfidenceVerified)
	col.AddWarning("WARN_TEST", "Test warning message", "Class.4dm", 42, ConfidenceInferred)

	if col.HasErrors() {
		t.Errorf("Expected HasErrors to still be false without error diagnostics")
	}

	col.AddError("ERR_TEST", "Fatal failure", "Syntax.4dm", 10)

	if !col.HasErrors() {
		t.Errorf("Expected HasErrors to be true after AddError")
	}

	all := col.All()
	if len(all) != 3 {
		t.Errorf("Expected 3 diagnostics, got %d", len(all))
	}

	var buf bytes.Buffer
	col.PrintFormatted(&buf)
	out := buf.String()

	if !strings.Contains(out, "ERR_TEST") {
		t.Errorf("Expected formatted output to contain ERR_TEST, got:\n%s", out)
	}
	if !strings.Contains(out, "WARN_TEST") {
		t.Errorf("Expected formatted output to contain WARN_TEST, got:\n%s", out)
	}
}

func TestCollectorConcurrency(t *testing.T) {
	col := NewCollector()
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			col.AddInfo("INFO_CONC", "Concurrent info", "file.4dm", ConfidenceInferred)
			col.AddWarning("WARN_CONC", "Concurrent warning", "file.4dm", idx, ConfidenceInferred)
		}(i)
	}

	wg.Wait()

	all := col.All()
	if len(all) != 100 {
		t.Errorf("Expected 100 diagnostics from concurrent writes, got %d", len(all))
	}
}
