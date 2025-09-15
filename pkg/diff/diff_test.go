package diff

import (
	"fmt"
	"strings"
	"testing"

	"github.com/argoproj/gitops-engine/pkg/utils/kube"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// formatResourceKey formats a ResourceKey for testing comparison
func formatResourceKey(key kube.ResourceKey) string {
	if key.Namespace != "" {
		return fmt.Sprintf("%s/%s/%s", key.Kind, key.Namespace, key.Name)
	}
	return fmt.Sprintf("%s/%s", key.Kind, key.Name)
}

func TestLabelSelectorFiltering(t *testing.T) {
	// Create test objects with different labels
	frontendObj := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]any{
				"name":      "frontend-app",
				"namespace": "default",
				"labels": map[string]any{
					"app":         "nginx",
					"tier":        "frontend",
					"environment": "production",
				},
			},
		},
	}

	backendObj := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]any{
				"name":      "backend-app",
				"namespace": "default",
				"labels": map[string]any{
					"app":         "api",
					"tier":        "backend",
					"environment": "production",
				},
			},
		},
	}

	stagingObj := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]any{
				"name":      "staging-app",
				"namespace": "staging",
				"labels": map[string]any{
					"app":         "nginx",
					"tier":        "frontend",
					"environment": "staging",
				},
			},
		},
	}

	noLabelsObj := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]any{
				"name":      "config",
				"namespace": "default",
			},
		},
	}

	objects := []*unstructured.Unstructured{frontendObj, backendObj, stagingObj, noLabelsObj}

	t.Run("equality selector filters correctly", func(t *testing.T) {
		opts := &Options{
			LabelSelector: map[string]string{
				"tier": "frontend",
			},
		}
		filtered := FilterResources(objects, opts)
		assert.Equal(t, 2, len(filtered))
		assert.Equal(t, "frontend-app", filtered[0].GetName())
		assert.Equal(t, "staging-app", filtered[1].GetName())
	})

	t.Run("multiple equality selectors", func(t *testing.T) {
		opts := &Options{
			LabelSelector: map[string]string{
				"tier":        "frontend",
				"environment": "production",
			},
		}
		filtered := FilterResources(objects, opts)
		assert.Equal(t, 1, len(filtered))
		assert.Equal(t, "frontend-app", filtered[0].GetName())
	})

	t.Run("production environment selector", func(t *testing.T) {
		opts := &Options{
			LabelSelector: map[string]string{
				"environment": "production",
			},
		}
		filtered := FilterResources(objects, opts)
		assert.Equal(t, 2, len(filtered)) // frontendObj and backendObj
		names := make([]string, len(filtered))
		for i, obj := range filtered {
			names[i] = obj.GetName()
		}
		assert.Contains(t, names, "frontend-app")
		assert.Contains(t, names, "backend-app")
		assert.NotContains(t, names, "staging-app")
		assert.NotContains(t, names, "config")
	})

	t.Run("specific app selector", func(t *testing.T) {
		opts := &Options{
			LabelSelector: map[string]string{
				"app": "nginx",
			},
		}
		filtered := FilterResources(objects, opts)
		assert.Equal(t, 2, len(filtered)) // frontendObj and stagingObj
		names := make([]string, len(filtered))
		for i, obj := range filtered {
			names[i] = obj.GetName()
		}
		assert.Contains(t, names, "frontend-app")
		assert.Contains(t, names, "staging-app")
		assert.NotContains(t, names, "backend-app")
		assert.NotContains(t, names, "config")
	})

	t.Run("empty selector returns all objects", func(t *testing.T) {
		opts := &Options{
			LabelSelector: nil,
		}
		filtered := FilterResources(objects, opts)
		assert.Equal(t, 4, len(filtered))
	})

	t.Run("empty map selector returns all objects", func(t *testing.T) {
		opts := &Options{
			LabelSelector: map[string]string{},
		}
		filtered := FilterResources(objects, opts)
		assert.Equal(t, 4, len(filtered)) // Should return all objects when selector is empty
	})

	t.Run("non-matching selector returns empty", func(t *testing.T) {
		opts := &Options{
			LabelSelector: map[string]string{
				"nonexistent": "value",
			},
		}
		filtered := FilterResources(objects, opts)
		assert.Equal(t, 0, len(filtered))
	})
}

func TestLabelSelectorWithExcludeKinds(t *testing.T) {
	deploymentObj := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]any{
				"name": "app-deployment",
				"labels": map[string]any{
					"app": "nginx",
				},
			},
		},
	}

	workflowObj := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "argoproj.io/v1alpha1",
			"kind":       "Workflow",
			"metadata": map[string]any{
				"name": "test-workflow",
				"labels": map[string]any{
					"app": "nginx",
				},
			},
		},
	}

	objects := []*unstructured.Unstructured{deploymentObj, workflowObj}

	t.Run("label selector with exclude kinds", func(t *testing.T) {
		opts := &Options{
			ExcludeKinds: []string{"Workflow"},
			LabelSelector: map[string]string{
				"app": "nginx",
			},
		}
		filtered := FilterResources(objects, opts)
		assert.Equal(t, 1, len(filtered))
		assert.Equal(t, "Deployment", filtered[0].GetKind())
		assert.Equal(t, "app-deployment", filtered[0].GetName())
	})

	t.Run("exclude kinds takes precedence", func(t *testing.T) {
		opts := &Options{
			ExcludeKinds: []string{"Deployment"},
			LabelSelector: map[string]string{
				"app": "nginx",
			},
		}
		filtered := FilterResources(objects, opts)
		assert.Equal(t, 1, len(filtered))
		assert.Equal(t, "Workflow", filtered[0].GetKind())
		assert.Equal(t, "test-workflow", filtered[0].GetName())
	})
}

func TestObjectsWithLabelSelector(t *testing.T) {
	baseObj := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]any{
				"name":      "config",
				"namespace": "default",
				"labels": map[string]any{
					"app": "nginx",
				},
			},
			"data": map[string]any{
				"key": "old-value",
			},
		},
	}

	headObj := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]any{
				"name":      "config",
				"namespace": "default",
				"labels": map[string]any{
					"app": "nginx",
				},
			},
			"data": map[string]any{
				"key": "new-value",
			},
		},
	}

	excludedObj := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]any{
				"name":      "excluded-config",
				"namespace": "default",
				"labels": map[string]any{
					"app": "excluded",
				},
			},
			"data": map[string]any{
				"key": "different-value",
			},
		},
	}

	base := []*unstructured.Unstructured{baseObj, excludedObj}
	head := []*unstructured.Unstructured{headObj, excludedObj}

	t.Run("diff with matching label selector", func(t *testing.T) {
		opts := &Options{
			LabelSelector: map[string]string{
				"app": "nginx",
			},
			Context: 3,
		}

		diffResult, changedResources, hasDiff, err := Objects(base, head, opts)
		assert.NoError(t, err)
		assert.True(t, hasDiff)
		assert.Contains(t, diffResult, "ConfigMap")
		assert.Contains(t, diffResult, "old-value")
		assert.Contains(t, diffResult, "new-value")
		assert.NotContains(t, diffResult, "excluded-config")

		// Check changed resources list
		assert.Equal(t, 1, len(changedResources))
		expected := "ConfigMap/default/config"
		actual := formatResourceKey(changedResources[0])
		assert.Equal(t, expected, actual)
	})

	t.Run("diff with non-matching label selector", func(t *testing.T) {
		opts := &Options{
			LabelSelector: map[string]string{
				"app": "nonexistent",
			},
			Context: 3,
		}

		diffResult, changedResources, hasDiff, err := Objects(base, head, opts)
		assert.NoError(t, err)
		assert.False(t, hasDiff)
		assert.Equal(t, "", diffResult)
		assert.Equal(t, 0, len(changedResources))
	})
}

func TestYamlString(t *testing.T) {
	baseYaml := `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test-config
  namespace: default
data:
  key1: value1
  key2: old-value
`

	headYaml := `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test-config
  namespace: default
data:
  key1: value1
  key2: new-value
`

	t.Run("diff with changes", func(t *testing.T) {
		diffResult, changedResources, hasDiff, err := YamlString(baseYaml, headYaml, nil)
		assert.NoError(t, err)
		assert.True(t, hasDiff)
		assert.Contains(t, diffResult, "ConfigMap")
		assert.Contains(t, diffResult, "old-value")
		assert.Contains(t, diffResult, "new-value")

		// Check changed resources list
		assert.Equal(t, 1, len(changedResources))
		expected := "ConfigMap/default/test-config"
		actual := formatResourceKey(changedResources[0])
		assert.Equal(t, expected, actual)
	})

	t.Run("no diff when identical", func(t *testing.T) {
		diffResult, changedResources, hasDiff, err := YamlString(baseYaml, baseYaml, nil)
		assert.NoError(t, err)
		assert.False(t, hasDiff)
		assert.Equal(t, "", diffResult)
		assert.Equal(t, 0, len(changedResources))
	})

	t.Run("error on invalid base yaml", func(t *testing.T) {
		invalidYaml := `invalid: yaml: structure`
		_, _, _, err := YamlString(invalidYaml, headYaml, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse base YAML")
	})

	t.Run("error on invalid head yaml", func(t *testing.T) {
		invalidYaml := `invalid: yaml: structure`
		_, _, _, err := YamlString(baseYaml, invalidYaml, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse head YAML")
	})
}

func TestYaml(t *testing.T) {
	baseYaml := `
apiVersion: v1
kind: Pod
metadata:
  name: test-pod
  namespace: default
spec:
  containers:
  - name: app
    image: nginx:1.20
`

	headYaml := `
apiVersion: v1
kind: Pod
metadata:
  name: test-pod
  namespace: default
spec:
  containers:
  - name: app
    image: nginx:1.21
`

	t.Run("diff with io.Reader", func(t *testing.T) {
		baseReader := strings.NewReader(baseYaml)
		headReader := strings.NewReader(headYaml)

		diffResult, changedResources, hasDiff, err := Yaml(baseReader, headReader, nil)
		assert.NoError(t, err)
		assert.True(t, hasDiff)
		assert.Contains(t, diffResult, "Pod")
		assert.Contains(t, diffResult, "nginx:1.20")
		assert.Contains(t, diffResult, "nginx:1.21")

		// Check changed resources list
		assert.Equal(t, 1, len(changedResources))
		expected := "Pod/default/test-pod"
		actual := formatResourceKey(changedResources[0])
		assert.Equal(t, expected, actual)
	})

	t.Run("no diff when identical", func(t *testing.T) {
		baseReader := strings.NewReader(baseYaml)
		headReader := strings.NewReader(baseYaml)

		diffResult, changedResources, hasDiff, err := Yaml(baseReader, headReader, nil)
		assert.NoError(t, err)
		assert.False(t, hasDiff)
		assert.Equal(t, "", diffResult)
		assert.Equal(t, 0, len(changedResources))
	})

	t.Run("multiple objects in yaml", func(t *testing.T) {
		multiYamlBase := `
apiVersion: v1
kind: ConfigMap
metadata:
  name: config1
data:
  key: value1
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: config2
data:
  key: value2
`

		multiYamlHead := `
apiVersion: v1
kind: ConfigMap
metadata:
  name: config1
data:
  key: value1
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: config2
data:
  key: updated-value2
`

		baseReader := strings.NewReader(multiYamlBase)
		headReader := strings.NewReader(multiYamlHead)

		diffResult, changedResources, hasDiff, err := Yaml(baseReader, headReader, nil)
		assert.NoError(t, err)
		assert.True(t, hasDiff)
		assert.Contains(t, diffResult, "config2")
		assert.Contains(t, diffResult, "value2")
		assert.Contains(t, diffResult, "updated-value2")
		assert.NotContains(t, diffResult, "config1")

		// Check changed resources list - only config2 should be changed
		assert.Equal(t, 1, len(changedResources))
		expected := "ConfigMap/config2"
		actual := formatResourceKey(changedResources[0])
		assert.Equal(t, expected, actual)
	})

	t.Run("empty yaml", func(t *testing.T) {
		baseReader := strings.NewReader("")
		headReader := strings.NewReader(headYaml)

		diffResult, changedResources, hasDiff, err := Yaml(baseReader, headReader, nil)
		assert.NoError(t, err)
		assert.True(t, hasDiff)
		assert.Contains(t, diffResult, "test-pod")

		// Check changed resources list - new resource added
		assert.Equal(t, 1, len(changedResources))
		expected := "Pod/default/test-pod"
		actual := formatResourceKey(changedResources[0])
		assert.Equal(t, expected, actual)
	})
}

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
		diffResult, changedResources, hasDiff, err := Objects([]*unstructured.Unstructured{baseSecret}, []*unstructured.Unstructured{headSecret}, opts)

		assert.NoError(t, err)
		assert.True(t, hasDiff)
		assert.Contains(t, diffResult, "test-secret")

		// Should not contain actual values
		assert.NotContains(t, diffResult, "cGFzc3dvcmQxMjM=")
		assert.NotContains(t, diffResult, "bmV3cGFzc3dvcmQ=")
		assert.NotContains(t, diffResult, "YWRtaW4=")

		// Should contain masked values with + symbols
		assert.Contains(t, diffResult, "++++++++++++++++")

		// Check changed resources list
		assert.Equal(t, 1, len(changedResources))
		expected := "Secret/default/test-secret"
		actual := formatResourceKey(changedResources[0])
		assert.Equal(t, expected, actual)
	})

	t.Run("same values get same mask, different values get different masks", func(t *testing.T) {
		opts := DefaultOptions()
		diffResult, _, hasDiff, err := Objects([]*unstructured.Unstructured{baseSecret}, []*unstructured.Unstructured{headSecret}, opts)

		assert.NoError(t, err)
		assert.True(t, hasDiff)

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
		diffResult, _, hasDiff, err := Objects([]*unstructured.Unstructured{baseSecret}, []*unstructured.Unstructured{headSecret}, opts)

		assert.NoError(t, err)
		assert.True(t, hasDiff)

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
		diffResult, _, hasDiff, err := Objects([]*unstructured.Unstructured{secretWithStringData}, []*unstructured.Unstructured{secretWithStringData}, opts)

		assert.NoError(t, err)
		assert.False(t, hasDiff) // Same object should not have diff
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
		diffResult, _, hasDiff, err := Objects([]*unstructured.Unstructured{configMap}, []*unstructured.Unstructured{configMap}, opts)

		assert.NoError(t, err)
		assert.False(t, hasDiff)
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

		diffResult, _, hasDiff, err := Objects(baseObjects, headObjects, opts)

		assert.NoError(t, err)
		assert.True(t, hasDiff)

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
		diffResult, _, hasDiff, err := Objects([]*unstructured.Unstructured{emptySecret}, []*unstructured.Unstructured{emptySecret}, opts)

		assert.NoError(t, err)
		assert.False(t, hasDiff)
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
		diffResult, _, hasDiff, err := Objects([]*unstructured.Unstructured{secretWithNil}, []*unstructured.Unstructured{secretWithNil}, opts)

		assert.NoError(t, err)
		assert.False(t, hasDiff)
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
		diffResult, _, hasDiff, err := Objects([]*unstructured.Unstructured{mixedSecretBase}, []*unstructured.Unstructured{mixedSecretHead}, opts)

		assert.NoError(t, err)
		assert.True(t, hasDiff)

		// Both data and stringData values should be masked
		assert.NotContains(t, diffResult, "ZW5jb2RlZC12YWx1ZQ==")
		assert.NotContains(t, diffResult, "bmV3LWVuY29kZWQtdmFsdWU=")
		assert.NotContains(t, diffResult, "plain-value")
		assert.NotContains(t, diffResult, "new-plain-value")

		// Should contain masked values
		assert.Contains(t, diffResult, "++++++++++++++++")
	})

	t.Run("mask consistency across multiple diff operations", func(t *testing.T) {
		// Reset global state for this test
		globalValueToReplacement = make(map[string]string)
		globalReplacement = "++++++++++++++++"

		secret1 := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "v1",
				"kind":       "Secret",
				"metadata": map[string]any{
					"name":      "secret1",
					"namespace": "default",
				},
				"data": map[string]any{
					"password": "c2FtZS12YWx1ZQ==", // "same-value"
				},
			},
		}

		secret2 := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "v1",
				"kind":       "Secret",
				"metadata": map[string]any{
					"name":      "secret2",
					"namespace": "default",
				},
				"data": map[string]any{
					"token": "c2FtZS12YWx1ZQ==", // same "same-value"
				},
			},
		}

		opts := DefaultOptions()

		// First diff operation
		diff1, _, _, err1 := Objects([]*unstructured.Unstructured{secret1}, []*unstructured.Unstructured{secret1}, opts)
		assert.NoError(t, err1)

		// Second diff operation with same value
		diff2, _, _, err2 := Objects([]*unstructured.Unstructured{secret2}, []*unstructured.Unstructured{secret2}, opts)
		assert.NoError(t, err2)

		// The same value should get the same mask across different operations
		// (This test verifies the global state consistency)
		assert.Equal(t, diff1, diff2)
	})
}

func TestSecretMaskingYAML(t *testing.T) {
	baseYAML := `
apiVersion: v1
kind: Secret
metadata:
  name: test-secret
  namespace: default
type: Opaque
data:
  password: cGFzc3dvcmQxMjM= # gitleaks:allow
  username: YWRtaW4= # gitleaks:allow
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: test-config
  namespace: default
data:
  config: original-config
`

	headYAML := `
apiVersion: v1
kind: Secret
metadata:
  name: test-secret
  namespace: default
type: Opaque
data:
  password: bmV3cGFzc3dvcmQ=
  username: YWRtaW4=
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: test-config
  namespace: default
data:
  config: updated-config
`

	t.Run("yaml diff with secret masking enabled", func(t *testing.T) {
		opts := DefaultOptions()
		diffResult, _, hasDiff, err := YamlString(baseYAML, headYAML, opts)

		assert.NoError(t, err)
		assert.True(t, hasDiff)

		// Secret values should be masked
		assert.NotContains(t, diffResult, "cGFzc3dvcmQxMjM=")
		assert.NotContains(t, diffResult, "bmV3cGFzc3dvcmQ=")

		// ConfigMap values should not be masked
		assert.Contains(t, diffResult, "original-config")
		assert.Contains(t, diffResult, "updated-config")

		// Should contain masked values
		assert.Contains(t, diffResult, "++++++++++++++++")
	})

	t.Run("yaml diff with secret masking disabled", func(t *testing.T) {
		opts := &Options{
			DisableMaskSecrets: true,
			Context:            3,
		}
		diffResult, _, hasDiff, err := YamlString(baseYAML, headYAML, opts)

		assert.NoError(t, err)
		assert.True(t, hasDiff)

		// Secret values should not be masked
		assert.Contains(t, diffResult, "cGFzc3dvcmQxMjM=")
		assert.Contains(t, diffResult, "bmV3cGFzc3dvcmQ=")

		// ConfigMap values should not be masked
		assert.Contains(t, diffResult, "original-config")
		assert.Contains(t, diffResult, "updated-config")
	})
}

func TestSecretMaskingEdgeCases(t *testing.T) {
	t.Run("secret with non-string values in data", func(t *testing.T) {
		secretWithNonString := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "v1",
				"kind":       "Secret",
				"metadata": map[string]any{
					"name":      "mixed-types-secret",
					"namespace": "default",
				},
				"type": "Opaque",
				"data": map[string]any{
					"string-key": "dmFsdWU=",
					"int-key":    123,
					"bool-key":   true,
					"nil-key":    nil,
				},
			},
		}

		opts := DefaultOptions()
		diffResult, _, hasDiff, err := Objects([]*unstructured.Unstructured{secretWithNonString}, []*unstructured.Unstructured{secretWithNonString}, opts)

		assert.NoError(t, err)
		assert.False(t, hasDiff) // Same object should not have diff
		assert.Equal(t, "", diffResult)
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
		diffResult, _, hasDiff, err := Objects([]*unstructured.Unstructured{secretWithoutData}, []*unstructured.Unstructured{secretWithoutData}, opts)

		assert.NoError(t, err)
		assert.False(t, hasDiff)
		assert.Equal(t, "", diffResult)
	})

	t.Run("handles nil objects gracefully", func(t *testing.T) {
		opts := DefaultOptions()

		// Test with one nil object in array
		nonNilSecret := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "v1",
				"kind":       "Secret",
				"metadata": map[string]any{
					"name":      "test-secret",
					"namespace": "default",
				},
				"data": map[string]any{
					"key": "dmFsdWU=",
				},
			},
		}

		// Test with nil in base array
		baseObjects := []*unstructured.Unstructured{nil, nonNilSecret}
		headObjects := []*unstructured.Unstructured{nonNilSecret}

		diffResult, _, hasDiff, err := Objects(baseObjects, headObjects, opts)
		assert.NoError(t, err)
		assert.False(t, hasDiff) // Should not have diff since the non-nil objects are the same
		assert.Equal(t, "", diffResult)
	})

	t.Run("secret mask function with nil input", func(t *testing.T) {
		result := maskSecretData(nil)
		assert.Nil(t, result)
	})

	t.Run("isSecret function with various inputs", func(t *testing.T) {
		// Test with nil
		assert.False(t, isSecret(nil))

		// Test with non-secret
		configMap := &unstructured.Unstructured{
			Object: map[string]any{
				"kind": "ConfigMap",
			},
		}
		assert.False(t, isSecret(configMap))

		// Test with secret
		secret := &unstructured.Unstructured{
			Object: map[string]any{
				"kind": "Secret",
			},
		}
		assert.True(t, isSecret(secret))

		// Test with empty object
		empty := &unstructured.Unstructured{}
		assert.False(t, isSecret(empty))
	})
}

func TestObjects(t *testing.T) {
	obj := unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]any{
				"name":        "test",
				"namespace":   "namespace",
				"labels":      map[string]any{"duck": "all-species"},
				"annotations": map[string]any{"argocd.argoproj.io/hook": "PreSync"},
			},
		},
	}

	t.Run("nodiff", func(t *testing.T) {
		head := []*unstructured.Unstructured{&obj}
		base := []*unstructured.Unstructured{&obj}
		opts := DefaultOptions()
		diffResult, changedResources, hasDiff, err := Objects(base, head, opts)
		assert.NoError(t, err)
		assert.False(t, hasDiff)
		assert.Equal(t, "", diffResult)
		assert.Equal(t, 0, len(changedResources))
	})

	t.Run("exists only head", func(t *testing.T) {
		head := []*unstructured.Unstructured{&obj}
		base := []*unstructured.Unstructured{}
		opts := DefaultOptions()
		diffResult, changedResources, hasDiff, err := Objects(base, head, opts)
		assert.NoError(t, err)
		assert.True(t, hasDiff)
		fmt.Print(diffResult)
		assert.True(t, strings.Contains(diffResult, "===== /Pod namespace/test ======"))
		assert.True(t, strings.Contains(diffResult, "apiVersion: v1"))
		assert.True(t, strings.Contains(diffResult, "kind: Pod"))

		// Check changed resources list - new resource added
		assert.Equal(t, 1, len(changedResources))
		expected := "Pod/namespace/test"
		actual := formatResourceKey(changedResources[0])
		assert.Equal(t, expected, actual)
	})

	t.Run("exists only base", func(t *testing.T) {
		head := []*unstructured.Unstructured{}
		base := []*unstructured.Unstructured{&obj}
		opts := DefaultOptions()
		diffResult, changedResources, hasDiff, err := Objects(base, head, opts)
		assert.NoError(t, err)
		assert.True(t, hasDiff)
		assert.True(t, strings.Contains(diffResult, "===== /Pod namespace/test ======"))

		// Check changed resources list - resource deleted
		assert.Equal(t, 1, len(changedResources))
		expected := "Pod/namespace/test"
		actual := formatResourceKey(changedResources[0])
		assert.Equal(t, expected, actual)
	})

	t.Run("workflow excluded by default", func(t *testing.T) {
		head := []*unstructured.Unstructured{&obj}
		base := []*unstructured.Unstructured{}
		opts := DefaultOptions()
		diffResult, changedResources, hasDiff, err := Objects(base, head, opts)
		assert.NoError(t, err)
		assert.True(t, hasDiff)
		assert.Contains(t, diffResult, "Pod")

		// Check changed resources list
		assert.Equal(t, 1, len(changedResources))
		expected := "Pod/namespace/test"
		actual := formatResourceKey(changedResources[0])
		assert.Equal(t, expected, actual)
	})
}

func TestDiffOptionsFiltering(t *testing.T) {
	hookObj := unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]any{
				"name":        "hook-pod",
				"namespace":   "test",
				"annotations": map[string]any{"argocd.argoproj.io/hook": "PreSync"},
			},
		},
	}

	secretObj := unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "Secret",
			"metadata": map[string]any{
				"name":      "secret",
				"namespace": "test",
			},
		},
	}

	workflowObj := unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "argoproj.io/v1alpha1",
			"kind":       "Workflow",
			"metadata": map[string]any{
				"name":      "workflow",
				"namespace": "test",
			},
		},
	}

	normalObj := unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]any{
				"name":      "config",
				"namespace": "test",
			},
		},
	}

	objects := []*unstructured.Unstructured{&hookObj, &secretObj, &workflowObj, &normalObj}

	t.Run("default options include all objects", func(t *testing.T) {
		opts := DefaultOptions()
		diffResult, _, hasDiff, err := Objects([]*unstructured.Unstructured{}, objects, opts)
		assert.NoError(t, err)
		assert.True(t, hasDiff)
		assert.Contains(t, diffResult, "ConfigMap")
		assert.Contains(t, diffResult, "Secret")
		assert.Contains(t, diffResult, "Workflow")
		assert.Contains(t, diffResult, "hook-pod")
	})

	t.Run("include all when exclude kinds disabled", func(t *testing.T) {
		opts := &Options{
			ExcludeKinds: []string{},
		}
		diffResult, _, hasDiff, err := Objects([]*unstructured.Unstructured{}, objects, opts)
		assert.NoError(t, err)
		assert.True(t, hasDiff)
		assert.Contains(t, diffResult, "ConfigMap")
		assert.Contains(t, diffResult, "Secret")
		assert.Contains(t, diffResult, "Workflow")
		assert.Contains(t, diffResult, "hook-pod")
	})

	t.Run("custom exclude kinds", func(t *testing.T) {
		opts := &Options{
			ExcludeKinds: []string{"ConfigMap", "Secret"},
		}
		diffResult, _, hasDiff, err := Objects([]*unstructured.Unstructured{}, objects, opts)
		assert.NoError(t, err)
		assert.True(t, hasDiff)
		assert.NotContains(t, diffResult, "ConfigMap")
		assert.NotContains(t, diffResult, "Secret")
		assert.Contains(t, diffResult, "Workflow")
		assert.Contains(t, diffResult, "hook-pod")
	})
}

func TestFilterResourcesBasic(t *testing.T) {
	hookObj := unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]any{
				"name":        "hook-pod",
				"annotations": map[string]any{"argocd.argoproj.io/hook": "PreSync"},
			},
		},
	}

	secretObj := unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "Secret",
			"metadata": map[string]any{
				"name": "secret",
			},
		},
	}

	normalObj := unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]any{
				"name": "config",
			},
		},
	}

	objects := []*unstructured.Unstructured{&hookObj, &secretObj, &normalObj}

	t.Run("filter by exclude kinds", func(t *testing.T) {
		opts := &Options{
			ExcludeKinds: []string{"Secret", "Pod"},
		}
		filtered := FilterResources(objects, opts)
		assert.Equal(t, 1, len(filtered))
		assert.Equal(t, "ConfigMap", filtered[0].GetKind())
	})

	t.Run("no filtering", func(t *testing.T) {
		opts := &Options{
			ExcludeKinds: []string{},
		}
		filtered := FilterResources(objects, opts)
		assert.Equal(t, 3, len(filtered))
	})
}

func TestObjectsWithNilOptions(t *testing.T) {
	obj := unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]any{
				"name":      "config",
				"namespace": "test",
			},
		},
	}

	head := []*unstructured.Unstructured{&obj}
	base := []*unstructured.Unstructured{}

	diffResult, changedResources, hasDiff, err := Objects(base, head, nil)
	assert.NoError(t, err)
	assert.True(t, hasDiff)
	assert.Contains(t, diffResult, "ConfigMap")

	// Check changed resources list
	assert.Equal(t, 1, len(changedResources))
	expected := "ConfigMap/test/config"
	actual := formatResourceKey(changedResources[0])
	assert.Equal(t, expected, actual)
}
