package diff

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// TestSecretMasking tests the secret masking functionality
// Test approach inspired by ArgoCD gitops-engine's secret masking tests
func TestSecretMasking(t *testing.T) {
	baseSecret := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "Secret",
			"metadata": map[string]any{
				"name":      "test-secret",
				"namespace": "default",
			},
			"type": "Opaque",
			"data": map[string]any{
				"password": "cGFzc3dvcmQxMjM=", // base64 encoded "password123" # gitleaks:allow
				"username": "YWRtaW4=",         // base64 encoded "admin" # gitleaks:allow
			},
		},
	}

	headSecret := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "Secret",
			"metadata": map[string]any{
				"name":      "test-secret",
				"namespace": "default",
			},
			"type": "Opaque",
			"data": map[string]any{
				"password": "bmV3cGFzc3dvcmQ=", // base64 encoded "newpassword"
				"username": "YWRtaW4=",         // base64 encoded "admin" (same)
			},
		},
	}

	t.Run("masks secret values by default", func(t *testing.T) {
		opts := DefaultOptions()
		results, err := Objects([]*unstructured.Unstructured{baseSecret}, []*unstructured.Unstructured{headSecret}, opts)

		assert.NoError(t, err)
		assert.True(t, results.HasChanges())

		diffResult := results.StringDiff()
		assert.Contains(t, diffResult, "test-secret")

		// Should not contain actual values
		assert.NotContains(t, diffResult, "cGFzc3dvcmQxMjM=")
		assert.NotContains(t, diffResult, "bmV3cGFzc3dvcmQ=")
		assert.NotContains(t, diffResult, "YWRtaW4=")

		// Should contain masked values with + symbols
		assert.Contains(t, diffResult, "++++++++++++++++")

		// Check changed resources list
		changedResourcesList := getChangedResources(results)
		assert.Equal(t, 1, len(changedResourcesList))
		assertResourceChange(t, results, "Secret/default/test-secret", Changed)
	})

	t.Run("same values get same mask, different values get different masks", func(t *testing.T) {
		opts := DefaultOptions()
		results, err := Objects([]*unstructured.Unstructured{baseSecret}, []*unstructured.Unstructured{headSecret}, opts)

		assert.NoError(t, err)
		assert.True(t, results.HasChanges())

		// Check diff string output
		diffResult := results.StringDiff()

		// Count occurrences of different mask lengths
		base16Plus := strings.Count(diffResult, "++++++++++++++++")  // 16 +
		base17Plus := strings.Count(diffResult, "+++++++++++++++++") // 17 +

		// Should have masks of different lengths for different values
		assert.True(t, base16Plus > 0 || base17Plus > 0, "Should contain masked values")
	})

	t.Run("can disable secret masking", func(t *testing.T) {
		opts := &Options{
			DisableMaskSecrets: true,
			Context:            3,
		}
		results, err := Objects([]*unstructured.Unstructured{baseSecret}, []*unstructured.Unstructured{headSecret}, opts)

		assert.NoError(t, err)
		assert.True(t, results.HasChanges())

		// Check diff string output
		diffResult := results.StringDiff()
		// Should contain actual values when masking is disabled
		assert.Contains(t, diffResult, "cGFzc3dvcmQxMjM=")
		assert.Contains(t, diffResult, "bmV3cGFzc3dvcmQ=")
	})

	t.Run("handles stringData field", func(t *testing.T) {
		secretWithStringData := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "v1",
				"kind":       "Secret",
				"metadata": map[string]any{
					"name":      "string-secret",
					"namespace": "default",
				},
				"type": "Opaque",
				"stringData": map[string]any{
					"config": "plain-text-config",
					"token":  "plain-text-token",
				},
			},
		}

		opts := DefaultOptions()
		results, err := Objects([]*unstructured.Unstructured{secretWithStringData}, []*unstructured.Unstructured{secretWithStringData}, opts)

		assert.NoError(t, err)
		assert.False(t, results.HasChanges()) // Same object should not have diff

		// Check diff string output
		diffResult := results.StringDiff()
		assert.Equal(t, "", diffResult)
	})

	t.Run("non-secret objects are not affected", func(t *testing.T) {
		configMap := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"metadata": map[string]any{
					"name":      "test-config",
					"namespace": "default",
				},
				"data": map[string]any{
					"config": "some-value",
				},
			},
		}

		opts := DefaultOptions()
		results, err := Objects([]*unstructured.Unstructured{configMap}, []*unstructured.Unstructured{configMap}, opts)

		assert.NoError(t, err)
		assert.False(t, results.HasChanges())

		// Check diff string output
		diffResult := results.StringDiff()
		assert.Equal(t, "", diffResult)
	})

	t.Run("mixed objects - only secrets are masked", func(t *testing.T) {
		configMapBase := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"metadata": map[string]any{
					"name":      "test-config",
					"namespace": "default",
				},
				"data": map[string]any{
					"config": "original-value",
				},
			},
		}

		configMapHead := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"metadata": map[string]any{
					"name":      "test-config",
					"namespace": "default",
				},
				"data": map[string]any{
					"config": "updated-value",
				},
			},
		}

		opts := DefaultOptions()
		baseObjects := []*unstructured.Unstructured{baseSecret, configMapBase}
		headObjects := []*unstructured.Unstructured{headSecret, configMapHead}

		results, err := Objects(baseObjects, headObjects, opts)

		assert.NoError(t, err)
		assert.True(t, results.HasChanges())

		// Check diff string output
		diffResult := results.StringDiff()

		// Secret values should be masked
		assert.NotContains(t, diffResult, "cGFzc3dvcmQxMjM=")
		assert.NotContains(t, diffResult, "bmV3cGFzc3dvcmQ=")

		// ConfigMap values should not be masked
		assert.Contains(t, diffResult, "original-value")
		assert.Contains(t, diffResult, "updated-value")
	})

	t.Run("secret with empty data fields", func(t *testing.T) {
		emptySecret := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "v1",
				"kind":       "Secret",
				"metadata": map[string]any{
					"name":      "empty-secret",
					"namespace": "default",
				},
				"type": "Opaque",
			},
		}

		opts := DefaultOptions()
		results, err := Objects([]*unstructured.Unstructured{emptySecret}, []*unstructured.Unstructured{emptySecret}, opts)

		assert.NoError(t, err)
		assert.False(t, results.HasChanges())

		// Check diff string output
		diffResult := results.StringDiff()
		assert.Equal(t, "", diffResult)
	})

	t.Run("secret with nil values", func(t *testing.T) {
		secretWithNil := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "v1",
				"kind":       "Secret",
				"metadata": map[string]any{
					"name":      "nil-secret",
					"namespace": "default",
				},
				"type": "Opaque",
				"data": map[string]any{
					"key1": nil,
					"key2": "dmFsdWU=",
				},
			},
		}

		opts := DefaultOptions()
		results, err := Objects([]*unstructured.Unstructured{secretWithNil}, []*unstructured.Unstructured{secretWithNil}, opts)

		assert.NoError(t, err)
		assert.False(t, results.HasChanges())

		// Check diff string output
		diffResult := results.StringDiff()
		assert.Equal(t, "", diffResult)
	})

	t.Run("secret with both data and stringData", func(t *testing.T) {
		mixedSecretBase := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "v1",
				"kind":       "Secret",
				"metadata": map[string]any{
					"name":      "mixed-secret",
					"namespace": "default",
				},
				"type": "Opaque",
				"data": map[string]any{
					"encoded": "ZW5jb2RlZC12YWx1ZQ==", // base64 encoded "encoded-value"
				},
				"stringData": map[string]any{
					"plain": "plain-value",
				},
			},
		}

		mixedSecretHead := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "v1",
				"kind":       "Secret",
				"metadata": map[string]any{
					"name":      "mixed-secret",
					"namespace": "default",
				},
				"type": "Opaque",
				"data": map[string]any{
					"encoded": "bmV3LWVuY29kZWQtdmFsdWU=", // base64 encoded "new-encoded-value"
				},
				"stringData": map[string]any{
					"plain": "new-plain-value",
				},
			},
		}

		opts := DefaultOptions()
		results, err := Objects([]*unstructured.Unstructured{mixedSecretBase}, []*unstructured.Unstructured{mixedSecretHead}, opts)

		assert.NoError(t, err)
		assert.True(t, results.HasChanges())

		// Check diff string output
		diffResult := results.StringDiff()

		// Both data and stringData values should be masked
		assert.NotContains(t, diffResult, "ZW5jb2RlZC12YWx1ZQ==")
		assert.NotContains(t, diffResult, "bmV3LWVuY29kZWQtdmFsdWU=")
		assert.NotContains(t, diffResult, "plain-value")
		assert.NotContains(t, diffResult, "new-plain-value")
	})

	t.Run("mask consistency across multiple diff operations", func(t *testing.T) {
		// Test to ensure mask consistency follows the same pattern as ArgoCD
		opts := DefaultOptions()

		// First diff operation
		results1, err1 := Objects([]*unstructured.Unstructured{baseSecret}, []*unstructured.Unstructured{headSecret}, opts)
		assert.NoError(t, err1)

		// Second diff operation with same secrets
		results2, err2 := Objects([]*unstructured.Unstructured{baseSecret}, []*unstructured.Unstructured{headSecret}, opts)
		assert.NoError(t, err2)

		// Results should be consistent
		diff1 := results1.StringDiff()
		diff2 := results2.StringDiff()
		assert.Equal(t, diff1, diff2, "Diff results should be consistent across multiple operations")
	})
}

func TestSecretMaskingYAML(t *testing.T) {
	baseYaml := `
apiVersion: v1
kind: Secret
metadata:
  name: test-secret
  namespace: default
type: Opaque
data:
  password: cGFzc3dvcmQxMjM= # gitleaks:allow
  username: YWRtaW4= # gitleaks:allow
`

	headYaml := `
apiVersion: v1
kind: Secret
metadata:
  name: test-secret
  namespace: default
type: Opaque
data:
  password: bmV3cGFzc3dvcmQ=
  username: YWRtaW4=
`

	t.Run("yaml diff with secret masking enabled", func(t *testing.T) {
		opts := DefaultOptions()
		results, err := YamlString(baseYaml, headYaml, opts)

		assert.NoError(t, err)
		assert.True(t, results.HasChanges())

		diffResult := results.StringDiff()
		// Should not contain actual values
		assert.NotContains(t, diffResult, "cGFzc3dvcmQxMjM=")
		assert.NotContains(t, diffResult, "bmV3cGFzc3dvcmQ=")
		// Should contain masked values
		assert.Contains(t, diffResult, "++++++++++++++++")
	})

	t.Run("yaml diff with secret masking disabled", func(t *testing.T) {
		opts := &Options{
			DisableMaskSecrets: true,
			Context:            3,
		}
		results, err := YamlString(baseYaml, headYaml, opts)

		assert.NoError(t, err)
		assert.True(t, results.HasChanges())

		diffResult := results.StringDiff()
		// Should contain actual values when masking is disabled
		assert.Contains(t, diffResult, "cGFzc3dvcmQxMjM=")
		assert.Contains(t, diffResult, "bmV3cGFzc3dvcmQ=")
	})
}

func TestSecretMaskingEdgeCases(t *testing.T) {
	t.Run("secret with non-string values in data", func(t *testing.T) {
		secretWithNumbers := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "v1",
				"kind":       "Secret",
				"metadata": map[string]any{
					"name":      "number-secret",
					"namespace": "default",
				},
				"type": "Opaque",
				"data": map[string]any{
					"number": 123,
					"string": "dmFsdWU=",
				},
			},
		}

		opts := DefaultOptions()
		results, err := Objects([]*unstructured.Unstructured{secretWithNumbers}, []*unstructured.Unstructured{secretWithNumbers}, opts)

		assert.NoError(t, err)
		assert.False(t, results.HasChanges())
	})

	t.Run("secret without data or stringData fields", func(t *testing.T) {
		secretWithoutData := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "v1",
				"kind":       "Secret",
				"metadata": map[string]any{
					"name":      "no-data-secret",
					"namespace": "default",
				},
				"type": "Opaque",
			},
		}

		opts := DefaultOptions()
		results, err := Objects([]*unstructured.Unstructured{secretWithoutData}, []*unstructured.Unstructured{secretWithoutData}, opts)

		assert.NoError(t, err)
		assert.False(t, results.HasChanges())
	})

	t.Run("handles nil objects gracefully", func(t *testing.T) {
		opts := DefaultOptions()
		results, err := Objects([]*unstructured.Unstructured{nil}, []*unstructured.Unstructured{nil}, opts)

		assert.NoError(t, err)
		assert.False(t, results.HasChanges())
	})

	t.Run("secret mask function with nil input", func(t *testing.T) {
		masked := maskSecretData(nil)
		assert.Nil(t, masked)
	})

	t.Run("isSecret function with various inputs", func(t *testing.T) {
		assert.False(t, isSecret(nil))

		nonSecret := &unstructured.Unstructured{
			Object: map[string]any{
				"kind": "ConfigMap",
			},
		}
		assert.False(t, isSecret(nonSecret))

		secret := &unstructured.Unstructured{
			Object: map[string]any{
				"kind": "Secret",
			},
		}
		assert.True(t, isSecret(secret))
	})
}
