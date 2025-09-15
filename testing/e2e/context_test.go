package e2e

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContextLinesE2E(t *testing.T) {
	baseFile := getFixturePath("basic", "test-base.yaml")
	headFile := getFixturePath("basic", "test-head.yaml")

	tests := []struct {
		name             string
		contextLines     int
		expectDiff       bool
		minExpectedLines int // Minimum number of lines expected in diff output
		maxExpectedLines int // Maximum number of lines expected in diff output
	}{
		{
			name:             "default context (3 lines)",
			contextLines:     -1, // Don't specify --context flag
			expectDiff:       true,
			minExpectedLines: 5,   // At least some diff output
			maxExpectedLines: 100, // But not too much
		},
		{
			name:             "zero context lines",
			contextLines:     0,
			expectDiff:       true,
			minExpectedLines: 3,  // At least the changed lines + headers
			maxExpectedLines: 30, // Should be minimal but allow for actual output
		},
		{
			name:             "one context line",
			contextLines:     1,
			expectDiff:       true,
			minExpectedLines: 5,
			maxExpectedLines: 50,
		},
		{
			name:             "three context lines",
			contextLines:     3,
			expectDiff:       true,
			minExpectedLines: 10,
			maxExpectedLines: 80,
		},
		{
			name:             "large context lines",
			contextLines:     10,
			expectDiff:       true,
			minExpectedLines: 15,
			maxExpectedLines: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []string{"diff", baseFile, headFile}

			// Add context flag if specified
			if tt.contextLines >= 0 {
				args = append(args, "--context", fmt.Sprintf("%d", tt.contextLines))
			}

			result := runDiffCommand(args...)

			if tt.expectDiff {
				assertHasDiff(t, result)

				// Count lines in output
				lines := strings.Split(strings.TrimSpace(result.Output), "\n")
				lineCount := len(lines)

				assert.GreaterOrEqual(t, lineCount, tt.minExpectedLines,
					"Expected at least %d lines of output, got %d", tt.minExpectedLines, lineCount)
				assert.LessOrEqual(t, lineCount, tt.maxExpectedLines,
					"Expected at most %d lines of output, got %d", tt.maxExpectedLines, lineCount)
			} else {
				assertNoDiff(t, result)
			}
		})
	}
}

func TestContextLinesValidation(t *testing.T) {
	baseFile := getFixturePath("basic", "test-base.yaml")
	headFile := getFixturePath("basic", "test-head.yaml")

	tests := []struct {
		name         string
		contextValue string
		expectError  bool
	}{
		{
			name:         "valid positive number",
			contextValue: "5",
			expectError:  false,
		},
		{
			name:         "zero is valid",
			contextValue: "0",
			expectError:  false,
		},
		{
			name:         "negative number should be rejected",
			contextValue: "-1",
			expectError:  true,
		},
		{
			name:         "non-numeric value should be rejected",
			contextValue: "abc",
			expectError:  true,
		},
		{
			name:         "empty value should be rejected",
			contextValue: "",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runDiffCommand("diff", baseFile, headFile, "--context", tt.contextValue)

			if tt.expectError {
				assertError(t, result)
			} else {
				// Should either show diff (exit code 1) or no diff (exit code 0)
				// but not an error (exit code 2 or higher)
				assert.Contains(t, []int{0, 1}, result.ExitCode,
					"Expected exit code 0 or 1 for valid context value, got %d", result.ExitCode)
			}
		})
	}
}
