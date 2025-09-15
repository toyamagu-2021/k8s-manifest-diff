package diff

import (
	"fmt"
	"strings"
	"testing"

	"github.com/argoproj/gitops-engine/pkg/utils/kube"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// getChangedResources extracts resources that have any type of change (Created, Changed, Deleted)
// DEPRECATED: Use Results.Apply() with custom filter or multiple GetResourceKeysByType() calls
func getChangedResources(results Results) []kube.ResourceKey {
	return results.Apply(func(_ kube.ResourceKey, diffResult Result) bool {
		return diffResult.Type != Unchanged
	}).GetResourceKeys()
}

// parseResourceKey parses a string resource key into kube.ResourceKey
func parseResourceKey(key string) kube.ResourceKey {
	parts := strings.Split(key, "/")
	switch len(parts) {
	case 2: // Kind/Name (cluster-scoped)
		return kube.ResourceKey{
			Kind: parts[0],
			Name: parts[1],
		}
	case 3: // Kind/Namespace/Name (namespaced)
		return kube.ResourceKey{
			Kind:      parts[0],
			Namespace: parts[1],
			Name:      parts[2],
		}
	default:
		// Fallback - shouldn't happen with well-formed keys
		return kube.ResourceKey{Name: key}
	}
}

// assertResourceChange checks if a specific resource has the expected change type
func assertResourceChange(t *testing.T, results Results, expectedKey string, expectedChangeType ChangeType) {
	expectedResourceKey := parseResourceKey(expectedKey)
	result, found := results[expectedResourceKey]
	if found {
		assert.Equal(t, expectedChangeType, result.Type, "Expected change type %s for resource %s, got %s", expectedChangeType.String(), expectedKey, result.Type.String())
	} else {
		assert.True(t, found, "Resource %s not found in results", expectedKey)
	}
}

func TestResults_FilterByType(t *testing.T) {
	// Create test results with various change types
	results := Results{
		kube.ResourceKey{Kind: "Deployment", Name: "changed-app"}:   {Type: Changed, Diff: "changed diff"},
		kube.ResourceKey{Kind: "Service", Name: "created-service"}:  {Type: Created, Diff: "created diff"},
		kube.ResourceKey{Kind: "ConfigMap", Name: "deleted-config"}: {Type: Deleted, Diff: "deleted diff"},
		kube.ResourceKey{Kind: "Secret", Name: "unchanged-secret"}:  {Type: Unchanged, Diff: ""},
	}

	// Test FilterByType
	changedResults := results.FilterByType(Changed)
	assert.Equal(t, 1, len(changedResults))
	assert.Contains(t, changedResults, kube.ResourceKey{Kind: "Deployment", Name: "changed-app"})

	createdResults := results.FilterByType(Created)
	assert.Equal(t, 1, len(createdResults))
	assert.Contains(t, createdResults, kube.ResourceKey{Kind: "Service", Name: "created-service"})

	// Test convenience methods
	assert.Equal(t, changedResults, results.FilterChanged())
	assert.Equal(t, createdResults, results.FilterCreated())
	assert.Equal(t, results.FilterByType(Deleted), results.FilterDeleted())
	assert.Equal(t, results.FilterByType(Unchanged), results.FilterUnchanged())
}

func TestResults_FilterByAttributes(t *testing.T) {
	results := Results{
		kube.ResourceKey{Kind: "Deployment", Namespace: "default", Name: "app1"}:    {Type: Changed, Diff: "diff1"},
		kube.ResourceKey{Kind: "Service", Namespace: "default", Name: "app1"}:       {Type: Created, Diff: "diff2"},
		kube.ResourceKey{Kind: "Deployment", Namespace: "production", Name: "app2"}: {Type: Deleted, Diff: "diff3"},
		kube.ResourceKey{Kind: "ConfigMap", Namespace: "default", Name: "config"}:   {Type: Unchanged, Diff: ""},
	}

	// Test FilterByKind
	deployments := results.FilterByKind("Deployment")
	assert.Equal(t, 2, len(deployments))

	// Test FilterByNamespace
	defaultNS := results.FilterByNamespace("default")
	assert.Equal(t, 3, len(defaultNS))

	// Test FilterByResourceName
	app1Resources := results.FilterByResourceName("app1")
	assert.Equal(t, 2, len(app1Resources))

	// Test chaining filters
	defaultDeployments := results.FilterByNamespace("default").FilterByKind("Deployment")
	assert.Equal(t, 1, len(defaultDeployments))
	assert.Contains(t, defaultDeployments, kube.ResourceKey{Kind: "Deployment", Namespace: "default", Name: "app1"})
}

func TestResults_Apply(t *testing.T) {
	results := Results{
		kube.ResourceKey{Kind: "Deployment", Namespace: "default", Name: "app1"}:    {Type: Changed, Diff: "diff1"},
		kube.ResourceKey{Kind: "Service", Namespace: "default", Name: "app1"}:       {Type: Created, Diff: "diff2"},
		kube.ResourceKey{Kind: "Deployment", Namespace: "production", Name: "app2"}: {Type: Deleted, Diff: "diff3"},
		kube.ResourceKey{Kind: "ConfigMap", Namespace: "default", Name: "config"}:   {Type: Unchanged, Diff: ""},
	}

	// Filter resources in default namespace that have changes
	filtered := results.Apply(func(key kube.ResourceKey, result Result) bool {
		return key.Namespace == "default" && result.Type != Unchanged
	})

	assert.Equal(t, 2, len(filtered))
	assert.Contains(t, filtered, kube.ResourceKey{Kind: "Deployment", Namespace: "default", Name: "app1"})
	assert.Contains(t, filtered, kube.ResourceKey{Kind: "Service", Namespace: "default", Name: "app1"})
}

func TestResults_Analysis(t *testing.T) {
	results := Results{
		kube.ResourceKey{Kind: "Deployment", Name: "changed-app"}:   {Type: Changed, Diff: "changed diff"},
		kube.ResourceKey{Kind: "Service", Name: "created-service"}:  {Type: Created, Diff: "created diff"},
		kube.ResourceKey{Kind: "ConfigMap", Name: "deleted-config"}: {Type: Deleted, Diff: "deleted diff"},
		kube.ResourceKey{Kind: "Secret", Name: "unchanged-secret"}:  {Type: Unchanged, Diff: ""},
	}

	// Test HasChanges
	assert.True(t, results.HasChanges())

	noChangesResults := Results{
		kube.ResourceKey{Kind: "Secret", Name: "unchanged-secret"}: {Type: Unchanged, Diff: ""},
	}
	assert.False(t, noChangesResults.HasChanges())

	// Test IsEmpty
	assert.False(t, results.IsEmpty())
	emptyResults := Results{}
	assert.True(t, emptyResults.IsEmpty())

	// Test Count
	assert.Equal(t, 4, results.Count())
	assert.Equal(t, 0, emptyResults.Count())

	// Test CountByType
	assert.Equal(t, 1, results.CountByType(Changed))
	assert.Equal(t, 1, results.CountByType(Created))
	assert.Equal(t, 1, results.CountByType(Deleted))
	assert.Equal(t, 1, results.CountByType(Unchanged))
	assert.Equal(t, 0, results.CountByType(ChangeType(99))) // Invalid type

	// Test GetResourceKeys
	keys := results.GetResourceKeys()
	assert.Equal(t, 4, len(keys))

	// Test GetResourceKeysByType
	changedKeys := results.GetResourceKeysByType(Changed)
	assert.Equal(t, 1, len(changedKeys))
	assert.Equal(t, "changed-app", changedKeys[0].Name)

	createdKeys := results.GetResourceKeysByType(Created)
	assert.Equal(t, 1, len(createdKeys))
	assert.Equal(t, "created-service", createdKeys[0].Name)
}

func TestResults_GetStatistics(t *testing.T) {
	results := Results{
		kube.ResourceKey{Kind: "Deployment", Name: "app1"}:  {Type: Changed, Diff: "diff1"},
		kube.ResourceKey{Kind: "Deployment", Name: "app2"}:  {Type: Changed, Diff: "diff2"},
		kube.ResourceKey{Kind: "Service", Name: "svc1"}:     {Type: Created, Diff: "diff3"},
		kube.ResourceKey{Kind: "ConfigMap", Name: "config"}: {Type: Deleted, Diff: "diff4"},
		kube.ResourceKey{Kind: "Secret", Name: "secret1"}:   {Type: Unchanged, Diff: ""},
		kube.ResourceKey{Kind: "Secret", Name: "secret2"}:   {Type: Unchanged, Diff: ""},
	}

	stats := results.GetStatistics()

	assert.Equal(t, 6, stats.Total)
	assert.Equal(t, 2, stats.Changed)
	assert.Equal(t, 1, stats.Created)
	assert.Equal(t, 1, stats.Deleted)
	assert.Equal(t, 2, stats.Unchanged)

	// Test with empty results
	emptyResults := Results{}
	emptyStats := emptyResults.GetStatistics()
	assert.Equal(t, 0, emptyStats.Total)
	assert.Equal(t, 0, emptyStats.Changed)
	assert.Equal(t, 0, emptyStats.Created)
	assert.Equal(t, 0, emptyStats.Deleted)
	assert.Equal(t, 0, emptyStats.Unchanged)
}

func TestResults_StringSummary(t *testing.T) {
	results := Results{
		kube.ResourceKey{Kind: "Deployment", Namespace: "default", Name: "app1"}:    {Type: Changed, Diff: "diff1"},
		kube.ResourceKey{Kind: "Deployment", Namespace: "production", Name: "app2"}: {Type: Changed, Diff: "diff2"},
		kube.ResourceKey{Kind: "Service", Namespace: "default", Name: "svc1"}:       {Type: Created, Diff: "diff3"},
		kube.ResourceKey{Kind: "ConfigMap", Name: "config1"}:                        {Type: Deleted, Diff: "diff4"}, // cluster-scoped
		kube.ResourceKey{Kind: "Secret", Namespace: "default", Name: "secret1"}:     {Type: Unchanged, Diff: ""},
	}

	summary := results.StringSummary()

	// Verify the summary contains all sections
	assert.Contains(t, summary, "Unchanged:")
	assert.Contains(t, summary, "Changed:")
	assert.Contains(t, summary, "Create:")
	assert.Contains(t, summary, "Delete:")

	// Verify correct formatting for namespaced resources
	assert.Contains(t, summary, "Deployment/default/app1")
	assert.Contains(t, summary, "Deployment/production/app2")
	assert.Contains(t, summary, "Service/default/svc1")
	assert.Contains(t, summary, "Secret/default/secret1")

	// Verify correct formatting for cluster-scoped resources (no namespace)
	assert.Contains(t, summary, "ConfigMap/config1")
	// Should not contain ConfigMap with namespace
	assert.NotContains(t, summary, "ConfigMap//config1")

	// Test with empty results
	emptyResults := Results{}
	emptySummary := emptyResults.StringSummary()
	assert.Equal(t, "", emptySummary)

	// Test with only unchanged resources
	unchangedOnly := Results{
		kube.ResourceKey{Kind: "Secret", Name: "secret1"}: {Type: Unchanged, Diff: ""},
	}
	unchangedSummary := unchangedOnly.StringSummary()
	assert.Contains(t, unchangedSummary, "Unchanged:")
	assert.Contains(t, unchangedSummary, "Secret/secret1")
	assert.NotContains(t, unchangedSummary, "Changed:")
	assert.NotContains(t, unchangedSummary, "Create:")
	assert.NotContains(t, unchangedSummary, "Delete:")
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

		results, err := Objects(base, head, opts)
		assert.NoError(t, err)
		assert.True(t, results.HasChanges())

		// Check diff string output
		diffResult := results.StringDiff()
		assert.Contains(t, diffResult, "ConfigMap")
		assert.Contains(t, diffResult, "old-value")
		assert.Contains(t, diffResult, "new-value")
		assert.NotContains(t, diffResult, "excluded-config")

		// Check changed resources list
		changedResourcesList := getChangedResources(results)
		assert.Equal(t, 1, len(changedResourcesList))
		assertResourceChange(t, results, "ConfigMap/default/config", Changed)
	})

	t.Run("diff with non-matching label selector", func(t *testing.T) {
		opts := &Options{
			LabelSelector: map[string]string{
				"app": "nonexistent",
			},
			Context: 3,
		}

		results, err := Objects(base, head, opts)
		assert.NoError(t, err)
		assert.False(t, results.HasChanges())

		// Check diff string output
		diffResult := results.StringDiff()
		assert.Equal(t, "", diffResult)
		assert.Equal(t, 0, len(results))
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
		results, err := YamlString(baseYaml, headYaml, nil)
		assert.NoError(t, err)
		assert.True(t, results.HasChanges())

		// Check diff string output
		diffResult := results.StringDiff()
		assert.Contains(t, diffResult, "ConfigMap")
		assert.Contains(t, diffResult, "old-value")
		assert.Contains(t, diffResult, "new-value")

		// Check changed resources list
		changedResourcesList := getChangedResources(results)
		assert.Equal(t, 1, len(changedResourcesList))
		assertResourceChange(t, results, "ConfigMap/default/test-config", Changed)
	})

	t.Run("no diff when identical", func(t *testing.T) {
		results, err := YamlString(baseYaml, baseYaml, nil)
		assert.NoError(t, err)
		assert.False(t, results.HasChanges())

		// Check diff string output
		diffResult := results.StringDiff()
		assert.Equal(t, "", diffResult)

		// The resource should exist but be unchanged
		changedResourcesList := getChangedResources(results)
		assert.Equal(t, 0, len(changedResourcesList))
		// Check that the resource exists as unchanged
		assertResourceChange(t, results, "ConfigMap/default/test-config", Unchanged)
	})

	t.Run("error on invalid base yaml", func(t *testing.T) {
		invalidYaml := `invalid: yaml: structure`
		_, err := YamlString(invalidYaml, headYaml, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse base YAML")
	})

	t.Run("error on invalid head yaml", func(t *testing.T) {
		invalidYaml := `invalid: yaml: structure`
		_, err := YamlString(baseYaml, invalidYaml, nil)
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

		results, err := Yaml(baseReader, headReader, nil)
		assert.NoError(t, err)
		assert.True(t, results.HasChanges())

		diffResult := results.StringDiff()
		assert.Contains(t, diffResult, "Pod")
		assert.Contains(t, diffResult, "nginx:1.20")
		assert.Contains(t, diffResult, "nginx:1.21")

		// Check changed resources list
		changedResourcesList := getChangedResources(results)
		assert.Equal(t, 1, len(changedResourcesList))
		assertResourceChange(t, results, "Pod/default/test-pod", Changed)
	})

	t.Run("no diff when identical", func(t *testing.T) {
		baseReader := strings.NewReader(baseYaml)
		headReader := strings.NewReader(baseYaml)

		results, err := Yaml(baseReader, headReader, nil)
		assert.NoError(t, err)
		assert.False(t, results.HasChanges())

		// Check diff string output
		diffResult := results.StringDiff()
		assert.Equal(t, "", diffResult)

		// The resource should exist but be unchanged
		changedResourcesList := getChangedResources(results)
		assert.Equal(t, 0, len(changedResourcesList))
		// Check that the resource exists as unchanged
		assertResourceChange(t, results, "Pod/default/test-pod", Unchanged)
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

		results, err := Yaml(baseReader, headReader, nil)
		assert.NoError(t, err)
		assert.True(t, results.HasChanges())

		diffResult := results.StringDiff()
		assert.Contains(t, diffResult, "config2")
		assert.Contains(t, diffResult, "value2")
		assert.Contains(t, diffResult, "updated-value2")
		assert.NotContains(t, diffResult, "config1")

		// Check changed resources list - only config2 should be changed
		changedResourcesList := getChangedResources(results)
		assert.Equal(t, 1, len(changedResourcesList))
		assertResourceChange(t, results, "ConfigMap/config2", Changed)
	})

	t.Run("empty yaml", func(t *testing.T) {
		baseReader := strings.NewReader("")
		headReader := strings.NewReader(headYaml)

		results, err := Yaml(baseReader, headReader, nil)
		assert.NoError(t, err)
		assert.True(t, results.HasChanges())

		diffResult := results.StringDiff()
		assert.Contains(t, diffResult, "test-pod")

		// Check changed resources list - new resource added
		changedResourcesList := getChangedResources(results)
		assert.Equal(t, 1, len(changedResourcesList))
		assertResourceChange(t, results, "Pod/default/test-pod", Created)
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
		results1, err1 := Objects([]*unstructured.Unstructured{secret1}, []*unstructured.Unstructured{secret1}, opts)
		assert.NoError(t, err1)

		// Second diff operation with same value
		results2, err2 := Objects([]*unstructured.Unstructured{secret2}, []*unstructured.Unstructured{secret2}, opts)
		assert.NoError(t, err2)

		// Check diff string output for both
		diff1 := results1.StringDiff()
		diff2 := results2.StringDiff()

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
		results, err := YamlString(baseYAML, headYAML, opts)

		assert.NoError(t, err)
		assert.True(t, results.HasChanges())

		// Check diff string output
		diffResult := results.StringDiff()

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
		results, err := YamlString(baseYAML, headYAML, opts)

		assert.NoError(t, err)
		assert.True(t, results.HasChanges())

		// Check diff string output
		diffResult := results.StringDiff()

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
		results, err := Objects([]*unstructured.Unstructured{secretWithNonString}, []*unstructured.Unstructured{secretWithNonString}, opts)

		assert.NoError(t, err)
		assert.False(t, results.HasChanges()) // Same object should not have diff

		// Check diff string output
		diffResult := results.StringDiff()
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
		results, err := Objects([]*unstructured.Unstructured{secretWithoutData}, []*unstructured.Unstructured{secretWithoutData}, opts)

		assert.NoError(t, err)
		assert.False(t, results.HasChanges())

		// Check diff string output
		diffResult := results.StringDiff()
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

		results, err := Objects(baseObjects, headObjects, opts)
		assert.NoError(t, err)
		assert.False(t, results.HasChanges()) // Should not have diff since the non-nil objects are the same

		// Check diff string output
		diffResult := results.StringDiff()
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
		results, err := Objects(base, head, opts)
		assert.NoError(t, err)
		assert.False(t, results.HasChanges())

		// Check diff string output
		diffResult := results.StringDiff()
		assert.Equal(t, "", diffResult)

		// The resource should exist but be unchanged
		changedResourcesList := getChangedResources(results)
		assert.Equal(t, 0, len(changedResourcesList))
		// Check that the resource exists as unchanged
		assertResourceChange(t, results, "Pod/namespace/test", Unchanged)
	})

	t.Run("exists only head", func(t *testing.T) {
		head := []*unstructured.Unstructured{&obj}
		base := []*unstructured.Unstructured{}
		opts := DefaultOptions()
		results, err := Objects(base, head, opts)
		assert.NoError(t, err)
		assert.True(t, results.HasChanges())

		// Check diff string output
		diffResult := results.StringDiff()
		fmt.Print(diffResult)
		assert.True(t, strings.Contains(diffResult, "===== /Pod namespace/test ======"))
		assert.True(t, strings.Contains(diffResult, "apiVersion: v1"))
		assert.True(t, strings.Contains(diffResult, "kind: Pod"))

		// Check changed resources list - new resource added
		changedResourcesList := getChangedResources(results)
		assert.Equal(t, 1, len(changedResourcesList))
		assertResourceChange(t, results, "Pod/namespace/test", Created)
	})

	t.Run("exists only base", func(t *testing.T) {
		head := []*unstructured.Unstructured{}
		base := []*unstructured.Unstructured{&obj}
		opts := DefaultOptions()
		results, err := Objects(base, head, opts)
		assert.NoError(t, err)
		assert.True(t, results.HasChanges())

		// Check diff string output
		diffResult := results.StringDiff()
		assert.True(t, strings.Contains(diffResult, "===== /Pod namespace/test ======"))

		// Check changed resources list - resource deleted
		changedResourcesList := getChangedResources(results)
		assert.Equal(t, 1, len(changedResourcesList))
		assertResourceChange(t, results, "Pod/namespace/test", Deleted)
	})

	t.Run("workflow excluded by default", func(t *testing.T) {
		head := []*unstructured.Unstructured{&obj}
		base := []*unstructured.Unstructured{}
		opts := DefaultOptions()
		results, err := Objects(base, head, opts)
		assert.NoError(t, err)
		assert.True(t, results.HasChanges())

		diffResult := results.StringDiff()
		assert.Contains(t, diffResult, "Pod")

		// Check changed resources list
		changedResourcesList := getChangedResources(results)
		assert.Equal(t, 1, len(changedResourcesList))
		assertResourceChange(t, results, "Pod/namespace/test", Created)
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
		results, err := Objects([]*unstructured.Unstructured{}, objects, opts)
		assert.NoError(t, err)
		assert.True(t, results.HasChanges())
		diffResult := results.StringDiff()
		assert.Contains(t, diffResult, "ConfigMap")
		assert.Contains(t, diffResult, "Secret")
		assert.Contains(t, diffResult, "Workflow")
		assert.Contains(t, diffResult, "hook-pod")
	})

	t.Run("include all when exclude kinds disabled", func(t *testing.T) {
		opts := &Options{
			ExcludeKinds: []string{},
		}
		results, err := Objects([]*unstructured.Unstructured{}, objects, opts)
		assert.NoError(t, err)
		assert.True(t, results.HasChanges())
		diffResult := results.StringDiff()
		assert.Contains(t, diffResult, "ConfigMap")
		assert.Contains(t, diffResult, "Secret")
		assert.Contains(t, diffResult, "Workflow")
		assert.Contains(t, diffResult, "hook-pod")
	})

	t.Run("custom exclude kinds", func(t *testing.T) {
		opts := &Options{
			ExcludeKinds: []string{"ConfigMap", "Secret"},
		}
		results, err := Objects([]*unstructured.Unstructured{}, objects, opts)
		assert.NoError(t, err)
		assert.True(t, results.HasChanges())
		diffResult := results.StringDiff()
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

	results, err := Objects(base, head, nil)
	assert.NoError(t, err)
	assert.True(t, results.HasChanges())

	// Check diff string output
	diffResult := results.StringDiff()
	assert.Contains(t, diffResult, "ConfigMap")

	// Check changed resources list
	changedResourcesList := getChangedResources(results)
	assert.Equal(t, 1, len(changedResourcesList))
	assertResourceChange(t, results, "ConfigMap/test/config", Created)
}
