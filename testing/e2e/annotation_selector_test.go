package e2e

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnnotationSelectorE2E(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectDiff     bool
		expectedOutput []string
		notExpected    []string
	}{
		{
			name:       "no annotation selector shows all diffs",
			args:       []string{"diff", "fixtures/selectors/annotation-test-base.yaml", "fixtures/selectors/annotation-test-head.yaml"},
			expectDiff: true,
			expectedOutput: []string{
				"frontend-app",
				"backend-app",
				"app-config",
				"db-secret",
			},
		},
		{
			name:       "managed-by helm selector",
			args:       []string{"diff", "fixtures/selectors/annotation-test-base.yaml", "fixtures/selectors/annotation-test-head.yaml", "--annotation=app.kubernetes.io/managed-by=helm"},
			expectDiff: true,
			expectedOutput: []string{
				"frontend-app",
				"app-config",
			},
			notExpected: []string{
				"backend-app",
				"db-secret",
			},
		},
		{
			name:       "managed-by kubectl selector",
			args:       []string{"diff", "fixtures/selectors/annotation-test-base.yaml", "fixtures/selectors/annotation-test-head.yaml", "--annotation=app.kubernetes.io/managed-by=kubectl"},
			expectDiff: true,
			expectedOutput: []string{
				"backend-app",
				"db-secret",
			},
			notExpected: []string{
				"frontend-app",
				"app-config",
			},
		},
		{
			name:       "multiple annotation selectors (AND logic)",
			args:       []string{"diff", "fixtures/selectors/annotation-test-base.yaml", "fixtures/selectors/annotation-test-head.yaml", "--annotation=app.kubernetes.io/managed-by=helm", "--annotation=deployment.category=web"},
			expectDiff: true,
			expectedOutput: []string{
				"frontend-app",
			},
			notExpected: []string{
				"backend-app",
				"app-config",
				"db-secret",
			},
		},
		{
			name:       "deployment category web selector",
			args:       []string{"diff", "fixtures/selectors/annotation-test-base.yaml", "fixtures/selectors/annotation-test-head.yaml", "--annotation=deployment.category=web"},
			expectDiff: true,
			expectedOutput: []string{
				"frontend-app",
			},
			notExpected: []string{
				"backend-app",
				"app-config",
				"db-secret",
			},
		},
		{
			name:       "deployment category api selector",
			args:       []string{"diff", "fixtures/selectors/annotation-test-base.yaml", "fixtures/selectors/annotation-test-head.yaml", "--annotation=deployment.category=api"},
			expectDiff: true,
			expectedOutput: []string{
				"backend-app",
			},
			notExpected: []string{
				"frontend-app",
				"app-config",
				"db-secret",
			},
		},
		{
			name:       "config category web selector",
			args:       []string{"diff", "fixtures/selectors/annotation-test-base.yaml", "fixtures/selectors/annotation-test-head.yaml", "--annotation=config.category=web"},
			expectDiff: true,
			expectedOutput: []string{
				"app-config",
			},
			notExpected: []string{
				"frontend-app",
				"backend-app",
				"db-secret",
			},
		},
		{
			name:       "secret category database selector",
			args:       []string{"diff", "fixtures/selectors/annotation-test-base.yaml", "fixtures/selectors/annotation-test-head.yaml", "--annotation=secret.category=database"},
			expectDiff: true,
			expectedOutput: []string{
				"db-secret",
			},
			notExpected: []string{
				"frontend-app",
				"backend-app",
				"app-config",
			},
		},
		{
			name:       "non-matching annotation selector",
			args:       []string{"diff", "fixtures/selectors/annotation-test-base.yaml", "fixtures/selectors/annotation-test-head.yaml", "--annotation=nonexistent=value"},
			expectDiff: false,
			expectedOutput: []string{
				"No differences found",
			},
		},
		{
			name:       "mixed label and annotation selectors",
			args:       []string{"diff", "fixtures/selectors/annotation-test-base.yaml", "fixtures/selectors/annotation-test-head.yaml", "--label=tier=frontend", "--annotation=app.kubernetes.io/managed-by=helm"},
			expectDiff: true,
			expectedOutput: []string{
				"frontend-app",
				"app-config",
			},
			notExpected: []string{
				"backend-app",
				"db-secret",
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

func TestAnnotationSelectorWithExcludeKinds(t *testing.T) {
	t.Run("annotation selector with exclude kinds", func(t *testing.T) {
		result := runDiffCommand("diff", "fixtures/selectors/annotation-test-base.yaml", "fixtures/selectors/annotation-test-head.yaml",
			"--annotation=app.kubernetes.io/managed-by=helm", "--exclude-kinds=Deployment")

		// Should find differences (ConfigMap only)
		assertHasDiff(t, result)

		// Should contain ConfigMap but not Deployments
		assertDiffOutput(t, result, []string{"app-config"})
		assertNotInOutput(t, result, []string{"frontend-app", "backend-app", "db-secret"})
	})
}

func TestAnnotationSelectorHelpMessage(t *testing.T) {
	t.Run("help message shows annotation flag", func(t *testing.T) {
		result := runDiffCommand("diff", "--help")

		assert.Equal(t, 0, result.ExitCode, "Help should return exit code 0")
		assert.Contains(t, result.Output, "--annotation strings")
		assert.Contains(t, result.Output, "Annotation selector to filter resources")
		assert.Contains(t, result.Output, "Can be specified multiple times")
	})
}

func TestAnnotationSelectorValidation(t *testing.T) {
	tests := []struct {
		name           string
		annotationArgs []string
		expectError    bool
	}{
		{
			name:           "valid single annotation",
			annotationArgs: []string{"--annotation=app.kubernetes.io/managed-by=helm"},
			expectError:    false,
		},
		{
			name:           "valid multiple annotations",
			annotationArgs: []string{"--annotation=app.kubernetes.io/managed-by=helm", "--annotation=deployment.category=web"},
			expectError:    false,
		},
		{
			name:           "annotation without equals sign is ignored",
			annotationArgs: []string{"--annotation=invalidannotation"},
			expectError:    false, // Should not error, just ignore invalid format
		},
		{
			name:           "empty annotation value",
			annotationArgs: []string{"--annotation=app.kubernetes.io/managed-by="},
			expectError:    false, // Should handle empty values gracefully
		},
		{
			name:           "annotation with special characters",
			annotationArgs: []string{"--annotation=deployment.kubernetes.io/revision=1"},
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := append([]string{"diff", "fixtures/selectors/annotation-test-base.yaml", "fixtures/selectors/annotation-test-head.yaml"}, tt.annotationArgs...)
			result := runDiffCommand(args...)

			if tt.expectError {
				assertError(t, result)
			} else {
				// Command may exit with 1 (diff found) or 0 (no diff), both are valid for parsing
				assert.Contains(t, []int{0, 1}, result.ExitCode,
					"Expected exit code 0 or 1 for valid annotation selector, got %d", result.ExitCode)
			}
		})
	}
}

// Test with different annotation structures
func TestAnnotationSelectorWithVariousYAMLStructures(t *testing.T) {
	// Create a temporary YAML file with no annotations
	noAnnotationsYAML := `apiVersion: v1
kind: ConfigMap
metadata:
  name: no-annotations-config
  namespace: default
  labels:
    app: test
data:
  key: value
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: no-annotations-deployment
  namespace: default
  labels:
    app: test
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
	tmpFile := "fixtures/no-annotations.yaml"
	err := os.WriteFile(tmpFile, []byte(noAnnotationsYAML), 0644)
	require.NoError(t, err)
	defer func() {
		if err := os.Remove(tmpFile); err != nil {
			t.Logf("Warning: failed to remove temp file %s: %v", tmpFile, err)
		}
	}()

	t.Run("no annotations in objects with annotation selector", func(t *testing.T) {
		result := runDiffCommand("diff", tmpFile, tmpFile, "--annotation=app.kubernetes.io/managed-by=helm")

		// Should find no differences
		assertNoDiff(t, result)
	})
}

// Benchmark test for performance with many objects
func BenchmarkAnnotationSelectorPerformance(b *testing.B) {
	// Create a large YAML file with many objects
	var yamlContent strings.Builder
	for i := 0; i < 100; i++ {
		yamlContent.WriteString("---\n")
		yamlContent.WriteString("apiVersion: v1\n")
		yamlContent.WriteString("kind: ConfigMap\n")
		yamlContent.WriteString("metadata:\n")
		yamlContent.WriteString("  name: config-" + string(rune(i)) + "\n")
		yamlContent.WriteString("  annotations:\n")
		if i%2 == 0 {
			yamlContent.WriteString("    app.kubernetes.io/managed-by: helm\n")
		} else {
			yamlContent.WriteString("    app.kubernetes.io/managed-by: kubectl\n")
		}
		yamlContent.WriteString("data:\n")
		yamlContent.WriteString("  key: value\n")
	}

	tmpFile := "fixtures/large-annotations.yaml"
	err := os.WriteFile(tmpFile, []byte(yamlContent.String()), 0644)
	require.NoError(b, err)
	defer func() {
		if err := os.Remove(tmpFile); err != nil {
			b.Logf("Warning: failed to remove temp file %s: %v", tmpFile, err)
		}
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := runDiffCommand("diff", tmpFile, tmpFile, "--annotation=app.kubernetes.io/managed-by=helm")
		if result.ExitCode != 0 && result.ExitCode != 1 {
			b.Fatalf("Unexpected exit code: %d", result.ExitCode)
		}
	}
}
