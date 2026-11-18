package tool4d

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/miyako/OpenAPI/internal/diagnostics"
)

// Runner manages locating and running tool4d in headless mode.
type Runner struct {
	Path      string
	Timeout   time.Duration
	Collector *diagnostics.Collector
}

// NewRunner creates a tool4d runner instance.
func NewRunner(customPath string, timeout time.Duration, col *diagnostics.Collector) *Runner {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &Runner{
		Path:      customPath,
		Timeout:   timeout,
		Collector: col,
	}
}

// Locate searches for tool4d executable.
func (r *Runner) Locate() (string, error) {
	if r.Path != "" {
		if _, err := os.Stat(r.Path); err == nil {
			return r.Path, nil
		}
		return "", fmt.Errorf("specified tool4d path not found: %s", r.Path)
	}

	if envPath := os.Getenv("TOOL4D_PATH"); envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			return envPath, nil
		}
	}

	var candidates []string
	switch runtime.GOOS {
	case "darwin":
		candidates = []string{
			"/Applications/tool4d.app/Contents/MacOS/tool4d",
			"/Applications/4D/tool4d.app/Contents/MacOS/tool4d",
		}
	case "windows":
		candidates = []string{
			`C:\Program Files\4D\tool4d\tool4d.exe`,
			`C:\Program Files (x86)\4D\tool4d\tool4d.exe`,
		}
	case "linux":
		candidates = []string{
			"/opt/4D/tool4d",
			"/usr/local/bin/tool4d",
		}
	}

	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}

	// Check if in PATH
	if p, err := exec.LookPath("tool4d"); err == nil {
		return p, nil
	}

	return "", fmt.Errorf("tool4d not found on system")
}

// Run executes tool4d against a project file with required flags.
func (r *Runner) Run(projectFile string, startupMethod string, args []string) (string, error) {
	binPath, err := r.Locate()
	if err != nil {
		return "", err
	}

	cmdArgs := []string{
		"--project", projectFile,
		"--dataless",
		"--skip-onstartup",
	}

	if startupMethod != "" {
		cmdArgs = append(cmdArgs, fmt.Sprintf("--startup-method=%s", startupMethod))
	}

	cmdArgs = append(cmdArgs, args...)

	ctx, cancel := context.WithTimeout(context.Background(), r.Timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, binPath, cmdArgs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		r.Collector.AddError("ERR_TOOL4D_TIMEOUT", fmt.Sprintf("tool4d timed out after %v", r.Timeout), projectFile, 0)
		return "", fmt.Errorf("tool4d execution timed out")
	}

	if err != nil {
		r.Collector.AddWarning("WARN_TOOL4D_EXIT", fmt.Sprintf("tool4d exited with error: %v, stderr: %s", err, strings.TrimSpace(stderr.String())), projectFile, 0, diagnostics.ConfidenceInferred)
		return stdout.String(), err
	}

	return stdout.String(), nil
}
