package e2e

import (
	"strings"
	"testing"
)

func TestExcludeKindsE2E(t *testing.T) {
	tests := []struct {
		name           string
		baseFile       string
		headFile       string
		excludeKinds   []string
		expectDiff     bool
		expectedOutput []string
		notExpected    []string
	}{
		{
			name:       "no exclusion shows all diffs",
			baseFile:   "kinds/mixed-base.yaml",
			headFile:   "kinds/mixed-head.yaml",
			expectDiff: true,
			expectedOutput: []string{
				"test-app",      // Deployment changes
				"test-service",  // Service changes
				"test-workflow", // Workflow changes should be shown
			},
		},
		{
			name:         "exclude Deployment kind",
			baseFile:     "kinds/mixed-base.yaml",
			headFile:     "kinds/mixed-head.yaml",
			excludeKinds: []string{"Deployment"},
			expectDiff:   true,
			expectedOutput: []string{
				"test-service",  // Service changes should be shown
				"test-workflow", // Workflow changes should be shown
			},
			notExpected: []string{
				"apps/Deployment", // Deployment kind should not be present
			},
		},
		{
			name:         "exclude Service kind",
			baseFile:     "kinds/mixed-base.yaml",
			headFile:     "kinds/mixed-head.yaml",
			excludeKinds: []string{"Service"},
			expectDiff:   true,
			expectedOutput: []string{
				"apps/Deployment", // Deployment changes should be present
				"test-workflow",   // Workflow changes should be shown
			},
			notExpected: []string{
				"/Service", // Service kind should not be present
			},
		},
		{
			name:         "exclude multiple kinds",
			baseFile:     "kinds/mixed-base.yaml",
			headFile:     "kinds/mixed-head.yaml",
			excludeKinds: []string{"Deployment", "Service"},
			expectDiff:   true, // Workflow should still show differences
			expectedOutput: []string{
				"test-workflow", // Only Workflow changes should be shown
			},
			notExpected: []string{
				"apps/Deployment", // Deployment excluded
				"/Service",        // Service excluded
			},
		},
		{
			name:         "exclude all kinds including Workflow",
			baseFile:     "kinds/mixed-base.yaml",
			headFile:     "kinds/mixed-head.yaml",
			excludeKinds: []string{"Deployment", "Service", "Workflow"},
			expectDiff:   false, // No resources left
		},
		{
			name:         "exclude only Deployment and Service",
			baseFile:     "kinds/mixed-base.yaml",
			headFile:     "kinds/mixed-head.yaml",
			excludeKinds: []string{"Deployment", "Service"},
			expectDiff:   true,
			expectedOutput: []string{
				"test-workflow", // Workflow should be included
			},
			notExpected: []string{
				"apps/Deployment", // Deployment excluded
				"/Service",        // Service excluded
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

			args := []string{"diff", baseFile, headFile}

			// Add exclude-kinds flags if specified
			for _, kind := range tt.excludeKinds {
				args = append(args, "--exclude-kinds", kind)
			}

			result := runDiffCommand(args...)

			if tt.expectDiff {
				assertHasDiff(t, result)
				if len(tt.expectedOutput) > 0 {
					assertDiffOutput(t, result, tt.expectedOutput)
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

func TestExcludeKindsMultipleFlags(t *testing.T) {
	// Test that multiple --exclude-kinds flags work correctly
	baseFile := getFixturePath("kinds", "mixed-base.yaml")
	headFile := getFixturePath("kinds", "mixed-head.yaml")

	result := runDiffCommand("diff", baseFile, headFile,
		"--exclude-kinds", "Deployment",
		"--exclude-kinds", "Service")

	// Should exclude both Deployment and Service, but Workflow should still show differences
	assertHasDiff(t, result)
	assertDiffOutput(t, result, []string{"test-workflow"})
	assertNotInOutput(t, result, []string{"apps/Deployment", "/Service"})
}
