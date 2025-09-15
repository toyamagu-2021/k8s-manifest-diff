package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	binaryName = "k8s-yaml-diff-e2e-test"
	timeout    = 30 * time.Second
)

var binaryPath string

// TestMain builds the binary before running tests and cleans up after
func TestMain(m *testing.M) {
	// Build the binary
	binaryPath = filepath.Join(".", binaryName)
	cmd := exec.Command("go", "build", "-o", binaryPath, "../../cmd/k8s-yaml-diff")
	if err := cmd.Run(); err != nil {
		fmt.Printf("Failed to build binary: %v\n", err)
		os.Exit(1)
	}

	// Make sure the path is absolute for reliable execution
	absPath, err := filepath.Abs(binaryPath)
	if err != nil {
		fmt.Printf("Failed to get absolute path: %v\n", err)
		os.Exit(1)
	}
	binaryPath = absPath

	// Run tests
	code := m.Run()

	// Cleanup
	if err := os.Remove(binaryPath); err != nil {
		fmt.Printf("Warning: failed to remove binary path %s: %v\n", binaryPath, err)
	}
	os.Exit(code)
}

// CommandResult holds the result of running a command
type CommandResult struct {
	Output   string
	ExitCode int
	Error    error
}

// runDiffCommand executes the k8s-yaml-diff command with given args
func runDiffCommand(args ...string) CommandResult {
	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = "."

	output, err := cmd.CombinedOutput()
	exitCode := 0

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			// Command failed to start
			return CommandResult{
				Output:   string(output),
				ExitCode: -1,
				Error:    err,
			}
		}
	}

	return CommandResult{
		Output:   string(output),
		ExitCode: exitCode,
		Error:    nil,
	}
}

// assertDiffOutput checks that the output contains all expected strings
func assertDiffOutput(t *testing.T, result CommandResult, expectedStrings []string) {
	t.Helper()
	for _, expected := range expectedStrings {
		assert.Contains(t, result.Output, expected,
			"Expected output to contain '%s', but got:\n%s", expected, result.Output)
	}
}

// assertNotInOutput checks that the output does not contain any of the given strings
func assertNotInOutput(t *testing.T, result CommandResult, unexpectedStrings []string) {
	t.Helper()
	for _, unexpected := range unexpectedStrings {
		assert.NotContains(t, result.Output, unexpected,
			"Expected output to NOT contain '%s', but got:\n%s", unexpected, result.Output)
	}
}

// assertNoDiff checks that there's no diff (exit code 0 and specific output pattern)
func assertNoDiff(t *testing.T, result CommandResult) {
	t.Helper()
	assert.Equal(t, 0, result.ExitCode, "Expected exit code 0 for no diff")
	// When no diff is found, the tool may output "No differences found" or be empty
	output := strings.TrimSpace(result.Output)
	if output != "" && output != "No differences found" {
		t.Errorf("Expected no output or 'No differences found' for identical files, got: %s", output)
	}
}

// assertHasDiff checks that there is a diff (exit code 1)
func assertHasDiff(t *testing.T, result CommandResult) {
	t.Helper()
	assert.Equal(t, 1, result.ExitCode, "Expected exit code 1 for diff found")
	assert.NotEmpty(t, strings.TrimSpace(result.Output), "Expected diff output")
}

// assertError checks that the command failed with non-zero exit code
func assertError(t *testing.T, result CommandResult) {
	t.Helper()
	assert.NotEqual(t, 0, result.ExitCode, "Expected non-zero exit code for error")
}

// getFixturePath returns the full path to a fixture file
func getFixturePath(category, filename string) string {
	return filepath.Join("fixtures", category, filename)
}
