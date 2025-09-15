package diff

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

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
