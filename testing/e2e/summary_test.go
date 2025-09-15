package e2e

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSummaryFlagE2E(t *testing.T) {
	tests := []struct {
		name        string
		baseFile    string
		headFile    string
		expectDiff  bool
		expected    []string
		notExpected []string
	}{
		{
			name:       "summary flag shows only resource names",
			baseFile:   "basic/test-base.yaml",
			headFile:   "basic/test-head.yaml",
			expectDiff: true,
			expected: []string{
				"Changed:",
				"Deployment/default/frontend-app",
				"Deployment/default/backend-app",
				"ConfigMap/default/app-config",
			},
			notExpected: []string{
				"--- frontend-app-live.yaml",
				"--- backend-app-live.yaml",
				"--- app-config-live.yaml",
				"+++ frontend-app.yaml",
				"+++ backend-app.yaml",
				"+++ app-config.yaml",
				"replicas: 4",
				"replicas: 2",
				"debug: true",
				"debug: false",
			},
		},
		{
			name:       "summary flag with identical files shows no output",
			baseFile:   "basic/identical.yaml",
			headFile:   "basic/identical.yaml",
			expectDiff: false,
		},
		{
			name:       "summary flag with secret masking",
			baseFile:   "basic/secret-with-data-base.yaml",
			headFile:   "basic/secret-with-data-head.yaml",
			expectDiff: true,
			expected: []string{
				"Changed:",
				"Secret/default/test-secret",
			},
			notExpected: []string{
				"--- test-secret-live.yaml",
				"+++ test-secret.yaml",
				"password:",
				"api-key:",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Split category/filename for getFixturePath
			baseParts := strings.Split(tt.baseFile, "/")
			baseFile := getFixturePath(baseParts[0], baseParts[1])
			headParts := strings.Split(tt.headFile, "/")
			headFile := getFixturePath(headParts[0], headParts[1])

			result := runDiffCommand("diff", "--summary", baseFile, headFile)

			if tt.expectDiff {
				assertHasDiff(t, result)
				if len(tt.expected) > 0 {
					assertDiffOutput(t, result, tt.expected)
				}
				if len(tt.notExpected) > 0 {
					assertNotInOutput(t, result, tt.notExpected)
				}
			} else {
				assertNoDiff(t, result)
			}
		})
	}
}

func TestSummaryFlagWithFiltersE2E(t *testing.T) {
	tests := []struct {
		name        string
		baseFile    string
		headFile    string
		args        []string
		expectDiff  bool
		expected    []string
		notExpected []string
	}{
		{
			name:       "summary with exclude kinds",
			baseFile:   "kinds/mixed-base.yaml",
			headFile:   "kinds/mixed-head.yaml",
			args:       []string{"--summary", "--exclude-kinds", "Deployment"},
			expectDiff: true,
			expected: []string{
				"Changed:",
				"Service/test-service",
			},
			notExpected: []string{
				"Deployment/test-app",
			},
		},
		{
			name:       "summary with label selector",
			baseFile:   "basic/test-base.yaml",
			headFile:   "basic/test-head.yaml",
			args:       []string{"--summary", "--label", "app=nginx"},
			expectDiff: true,
			expected: []string{
				"Changed:",
				"Deployment/default/frontend-app",
				"ConfigMap/default/app-config",
			},
			notExpected: []string{
				"Deployment/default/backend-app",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Split category/filename for getFixturePath
			baseParts := strings.Split(tt.baseFile, "/")
			baseFile := getFixturePath(baseParts[0], baseParts[1])
			headParts := strings.Split(tt.headFile, "/")
			headFile := getFixturePath(headParts[0], headParts[1])

			args := append([]string{"diff"}, tt.args...)
			args = append(args, baseFile, headFile)
			result := runDiffCommand(args...)

			if tt.expectDiff {
				assertHasDiff(t, result)
				if len(tt.expected) > 0 {
					assertDiffOutput(t, result, tt.expected)
				}
				if len(tt.notExpected) > 0 {
					assertNotInOutput(t, result, tt.notExpected)
				}
			} else {
				assertNoDiff(t, result)
			}
		})
	}
}

func TestSummaryFlagOutputFormat(t *testing.T) {
	baseFile := getFixturePath("basic", "test-base.yaml")
	headFile := getFixturePath("basic", "test-head.yaml")

	// Test without summary flag (full diff)
	fullDiffResult := runDiffCommand("diff", baseFile, headFile)

	// Test with summary flag
	summaryResult := runDiffCommand("diff", "--summary", baseFile, headFile)

	// Both should have differences
	assertHasDiff(t, fullDiffResult)
	assertHasDiff(t, summaryResult)

	// Full diff should contain diff markers and content
	assertDiffOutput(t, fullDiffResult, []string{
		"--- frontend-app-live.yaml",
		"+++ frontend-app.yaml",
		"replicas: 4",
		"replicas: 2",
	})

	// Summary should only contain resource names with section header
	assertDiffOutput(t, summaryResult, []string{
		"Changed:",
		"Deployment/default/frontend-app",
		"Deployment/default/backend-app",
		"ConfigMap/default/app-config",
	})

	// Summary should not contain diff content
	assertNotInOutput(t, summaryResult, []string{
		"--- frontend-app-live.yaml",
		"+++ frontend-app.yaml",
		"replicas: 4",
		"replicas: 2",
		"debug: true",
		"debug: false",
	})

	// Verify line count - summary should be much shorter
	fullLines := len(strings.Split(strings.TrimSpace(fullDiffResult.Output), "\n"))
	summaryLines := len(strings.Split(strings.TrimSpace(summaryResult.Output), "\n"))

	assert.Greater(t, fullLines, 10, "Full diff should have many lines")
	assert.Equal(t, 4, summaryLines, "Summary should have exactly 4 lines (1 header + 3 changed resources)")
}
