package e2e

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLabelSelectorE2E(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectDiff     bool
		expectedOutput []string
		notExpected    []string
	}{
		{
			name:       "no label selector shows all diffs",
			args:       []string{"diff", "fixtures/basic/test-base.yaml", "fixtures/basic/test-head.yaml"},
			expectDiff: true,
			expectedOutput: []string{
				"frontend-app",
				"backend-app",
				"app-config",
			},
		},
		{
			name:       "frontend tier selector",
			args:       []string{"diff", "fixtures/basic/test-base.yaml", "fixtures/basic/test-head.yaml", "--label=tier=frontend"},
			expectDiff: true,
			expectedOutput: []string{
				"frontend-app",
				"app-config",
			},
			notExpected: []string{
				"backend-app",
			},
		},
		{
			name:       "backend tier selector",
			args:       []string{"diff", "fixtures/basic/test-base.yaml", "fixtures/basic/test-head.yaml", "--label=tier=backend"},
			expectDiff: true,
			expectedOutput: []string{
				"backend-app",
			},
			notExpected: []string{
				"frontend-app",
				"app-config",
			},
		},
		{
			name:       "multiple label selectors (AND logic)",
			args:       []string{"diff", "fixtures/basic/test-base.yaml", "fixtures/basic/test-head.yaml", "--label=app=nginx", "--label=tier=frontend"},
			expectDiff: true,
			expectedOutput: []string{
				"frontend-app",
				"app-config",
			},
			notExpected: []string{
				"backend-app",
			},
		},
		{
			name:       "production environment selector",
			args:       []string{"diff", "fixtures/basic/test-base.yaml", "fixtures/basic/test-head.yaml", "--label=environment=production"},
			expectDiff: true,
			expectedOutput: []string{
				"frontend-app",
				"backend-app",
				"app-config",
			},
		},
		{
			name:       "non-matching label selector",
			args:       []string{"diff", "fixtures/basic/test-base.yaml", "fixtures/basic/test-head.yaml", "--label=nonexistent=value"},
			expectDiff: false,
			expectedOutput: []string{
				"No differences found",
			},
		},
		{
			name:       "specific app selector",
			args:       []string{"diff", "fixtures/basic/test-base.yaml", "fixtures/basic/test-head.yaml", "--label=app=nginx"},
			expectDiff: true,
			expectedOutput: []string{
				"frontend-app",
				"app-config",
			},
			notExpected: []string{
				"backend-app",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runDiffCommand(tt.args...)

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
				if len(tt.expectedOutput) > 0 {
					assertDiffOutput(t, result, tt.expectedOutput)
				}
			}
		})
	}
}

func TestLabelSelectorWithExcludeKinds(t *testing.T) {
	t.Run("label selector with exclude kinds", func(t *testing.T) {
		result := runDiffCommand("diff", "fixtures/basic/test-base.yaml", "fixtures/basic/test-head.yaml",
			"--label=environment=production", "--exclude-kinds=Deployment")

		// Should find differences (ConfigMap only)
		assertHasDiff(t, result)

		// Should contain ConfigMap but not Deployments
		assertDiffOutput(t, result, []string{"app-config"})
		assertNotInOutput(t, result, []string{"frontend-app", "backend-app"})
	})
}

func TestLabelSelectorHelpMessage(t *testing.T) {
	t.Run("help message shows label flag", func(t *testing.T) {
		result := runDiffCommand("diff", "--help")

		assert.Equal(t, 0, result.ExitCode, "Help should return exit code 0")
		assert.Contains(t, result.Output, "--label strings")
		assert.Contains(t, result.Output, "Label selector to filter resources")
		assert.Contains(t, result.Output, "Can be specified multiple times")
	})
}

func TestLabelSelectorValidation(t *testing.T) {
	tests := []struct {
		name        string
		labelArgs   []string
		expectError bool
	}{
		{
			name:        "valid single label",
			labelArgs:   []string{"--label=app=nginx"},
			expectError: false,
		},
		{
			name:        "valid multiple labels",
			labelArgs:   []string{"--label=app=nginx", "--label=tier=frontend"},
			expectError: false,
		},
		{
			name:        "label without equals sign is ignored",
			labelArgs:   []string{"--label=invalidlabel"},
			expectError: false, // Should not error, just ignore invalid format
		},
		{
			name:        "empty label value",
			labelArgs:   []string{"--label=app="},
			expectError: false, // Should handle empty values gracefully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := append([]string{"diff", "fixtures/basic/test-base.yaml", "fixtures/basic/test-head.yaml"}, tt.labelArgs...)
			result := runDiffCommand(args...)

			if tt.expectError {
				assertError(t, result)
			} else {
				// Command may exit with 1 (diff found) or 0 (no diff), both are valid for parsing
				assert.Contains(t, []int{0, 1}, result.ExitCode,
					"Expected exit code 0 or 1 for valid label selector, got %d", result.ExitCode)
			}
		})
	}
}

// Test with different file structures
func TestLabelSelectorWithVariousYAMLStructures(t *testing.T) {
	// Create a temporary YAML file with no labels
	noLabelsYAML := `apiVersion: v1
kind: ConfigMap
metadata:
  name: no-labels-config
  namespace: default
data:
  key: value
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: no-labels-deployment
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app: test
  template:
    metadata:
      labels:
        app: test
    spec:
      containers:
      - name: test
        image: nginx:latest`

	// Write temporary file
	tmpFile := "fixtures/no-labels.yaml"
	err := os.WriteFile(tmpFile, []byte(noLabelsYAML), 0644)
	require.NoError(t, err)
	defer func() {
		if err := os.Remove(tmpFile); err != nil {
			t.Logf("Warning: failed to remove temp file %s: %v", tmpFile, err)
		}
	}()

	t.Run("no labels in objects with label selector", func(t *testing.T) {
		result := runDiffCommand("diff", tmpFile, tmpFile, "--label=app=nginx")

		// Should find no differences
		assertNoDiff(t, result)
	})
}

// Benchmark test for performance with many objects
func BenchmarkLabelSelectorPerformance(b *testing.B) {
	// Create a large YAML file with many objects
	var yamlContent strings.Builder
	for i := 0; i < 100; i++ {
		yamlContent.WriteString("---\n")
		yamlContent.WriteString("apiVersion: v1\n")
		yamlContent.WriteString("kind: ConfigMap\n")
		yamlContent.WriteString("metadata:\n")
		yamlContent.WriteString("  name: config-" + string(rune(i)) + "\n")
		yamlContent.WriteString("  labels:\n")
		if i%2 == 0 {
			yamlContent.WriteString("    app: nginx\n")
		} else {
			yamlContent.WriteString("    app: apache\n")
		}
		yamlContent.WriteString("data:\n")
		yamlContent.WriteString("  key: value\n")
	}

	tmpFile := "fixtures/large.yaml"
	err := os.WriteFile(tmpFile, []byte(yamlContent.String()), 0644)
	require.NoError(b, err)
	defer func() {
		if err := os.Remove(tmpFile); err != nil {
			b.Logf("Warning: failed to remove temp file %s: %v", tmpFile, err)
		}
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := runDiffCommand("diff", tmpFile, tmpFile, "--label=app=nginx")
		if result.ExitCode != 0 && result.ExitCode != 1 {
			b.Fatalf("Unexpected exit code: %d", result.ExitCode)
		}
	}
}
