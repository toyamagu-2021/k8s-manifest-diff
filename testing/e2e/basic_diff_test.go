package e2e

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBasicDiffE2E(t *testing.T) {
	tests := []struct {
		name        string
		baseFile    string
		headFile    string
		expectDiff  bool
		expected    []string
		notExpected []string
	}{
		{
			name:       "identical files show no diff",
			baseFile:   "basic/identical.yaml",
			headFile:   "basic/identical.yaml",
			expectDiff: false,
		},
		{
			name:       "different files show diff",
			baseFile:   "basic/test-base.yaml",
			headFile:   "basic/test-head.yaml",
			expectDiff: true,
			expected: []string{
				"frontend-app",
				"backend-app",
			},
		},
		{
			name:       "missing base file",
			baseFile:   "basic/nonexistent.yaml",
			headFile:   "basic/identical.yaml",
			expectDiff: false, // This will be an error case, handled by assertError
		},
		{
			name:       "missing head file",
			baseFile:   "basic/identical.yaml",
			headFile:   "basic/nonexistent.yaml",
			expectDiff: false, // This will be an error case, handled by assertError
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Split category/filename for getFixturePath
			baseParts := strings.Split(tt.baseFile, "/")
			baseFile := getFixturePath(baseParts[0], baseParts[1])
			headParts := strings.Split(tt.headFile, "/")
			headFile := getFixturePath(headParts[0], headParts[1])

			result := runDiffCommand("diff", baseFile, headFile)

			// Check for file existence errors first
			if tt.name == "missing base file" || tt.name == "missing head file" {
				assertError(t, result)
				return
			}

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

func TestDiffExitCodes(t *testing.T) {
	tests := []struct {
		name         string
		baseFile     string
		headFile     string
		expectedCode int
		description  string
	}{
		{
			name:         "identical files return 0",
			baseFile:     "basic/identical.yaml",
			headFile:     "basic/identical.yaml",
			expectedCode: 0,
			description:  "No differences found",
		},
		{
			name:         "different files return 1",
			baseFile:     "basic/test-base.yaml",
			headFile:     "basic/test-head.yaml",
			expectedCode: 1,
			description:  "Differences found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Split category/filename for getFixturePath
			baseParts := strings.Split(tt.baseFile, "/")
			baseFile := getFixturePath(baseParts[0], baseParts[1])
			headParts := strings.Split(tt.headFile, "/")
			headFile := getFixturePath(headParts[0], headParts[1])

			result := runDiffCommand("diff", baseFile, headFile)

			assert.Equal(t, tt.expectedCode, result.ExitCode,
				"Expected exit code %d for %s, got %d",
				tt.expectedCode, tt.description, result.ExitCode)
		})
	}
}
