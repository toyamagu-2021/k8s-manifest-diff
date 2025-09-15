package diff

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestFilterResources_LabelSelector(t *testing.T) {
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

	tests := []struct {
		name             string
		labelSelector    map[string]string
		expectedCount    int
		expectedNames    []string
		notExpectedNames []string
	}{
		{
			name:             "equality selector filters correctly",
			labelSelector:    map[string]string{"tier": "frontend"},
			expectedCount:    2,
			expectedNames:    []string{"frontend-app", "staging-app"},
			notExpectedNames: []string{"backend-app", "config"},
		},
		{
			name:             "multiple equality selectors",
			labelSelector:    map[string]string{"tier": "frontend", "environment": "production"},
			expectedCount:    1,
			expectedNames:    []string{"frontend-app"},
			notExpectedNames: []string{"backend-app", "staging-app", "config"},
		},
		{
			name:             "production environment selector",
			labelSelector:    map[string]string{"environment": "production"},
			expectedCount:    2,
			expectedNames:    []string{"frontend-app", "backend-app"},
			notExpectedNames: []string{"staging-app", "config"},
		},
		{
			name:             "specific app selector",
			labelSelector:    map[string]string{"app": "nginx"},
			expectedCount:    2,
			expectedNames:    []string{"frontend-app", "staging-app"},
			notExpectedNames: []string{"backend-app", "config"},
		},
		{
			name:             "empty selector returns all objects",
			labelSelector:    nil,
			expectedCount:    4,
			expectedNames:    []string{"frontend-app", "backend-app", "staging-app", "config"},
			notExpectedNames: []string{},
		},
		{
			name:             "non-matching selector returns empty",
			labelSelector:    map[string]string{"nonexistent": "value"},
			expectedCount:    0,
			expectedNames:    []string{},
			notExpectedNames: []string{"frontend-app", "backend-app", "staging-app", "config"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &Options{
				LabelSelector: tt.labelSelector,
			}
			filtered := FilterResources(objects, opts)
			assert.Equal(t, tt.expectedCount, len(filtered))

			if tt.expectedCount > 0 {
				names := make([]string, len(filtered))
				for i, obj := range filtered {
					names[i] = obj.GetName()
				}

				for _, expectedName := range tt.expectedNames {
					assert.Contains(t, names, expectedName)
				}

				for _, notExpectedName := range tt.notExpectedNames {
					assert.NotContains(t, names, notExpectedName)
				}
			}
		})
	}
}

func TestFilterResources_LabelSelectorWithExcludeKinds(t *testing.T) {
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

	tests := []struct {
		name          string
		excludeKinds  []string
		labelSelector map[string]string
		expectedCount int
		expectedKind  string
		expectedName  string
	}{
		{
			name:          "label selector with exclude kinds",
			excludeKinds:  []string{"Workflow"},
			labelSelector: map[string]string{"app": "nginx"},
			expectedCount: 1,
			expectedKind:  "Deployment",
			expectedName:  "app-deployment",
		},
		{
			name:          "exclude kinds takes precedence",
			excludeKinds:  []string{"Deployment"},
			labelSelector: map[string]string{"app": "nginx"},
			expectedCount: 1,
			expectedKind:  "Workflow",
			expectedName:  "test-workflow",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &Options{
				ExcludeKinds:  tt.excludeKinds,
				LabelSelector: tt.labelSelector,
			}
			filtered := FilterResources(objects, opts)
			assert.Equal(t, tt.expectedCount, len(filtered))
			assert.Equal(t, tt.expectedKind, filtered[0].GetKind())
			assert.Equal(t, tt.expectedName, filtered[0].GetName())
		})
	}
}

func TestFilterResources_AnnotationSelector(t *testing.T) {
	frontendObj := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]any{
				"name":      "frontend-app",
				"namespace": "default",
				"annotations": map[string]any{
					"app.kubernetes.io/managed-by": "helm",
					"deployment.category":          "web",
					"environment":                  "production",
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
				"annotations": map[string]any{
					"app.kubernetes.io/managed-by": "kubectl",
					"deployment.category":          "api",
					"environment":                  "production",
				},
			},
		},
	}

	stagingObj := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]any{
				"name":      "staging-config",
				"namespace": "staging",
				"annotations": map[string]any{
					"app.kubernetes.io/managed-by": "helm",
					"config.category":              "staging",
					"environment":                  "staging",
				},
			},
		},
	}

	noAnnotationsObj := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "Secret",
			"metadata": map[string]any{
				"name":      "secret",
				"namespace": "default",
			},
		},
	}

	objects := []*unstructured.Unstructured{frontendObj, backendObj, stagingObj, noAnnotationsObj}

	tests := []struct {
		name                string
		annotationSelector  map[string]string
		expectedCount       int
		expectedNames       []string
		notExpectedNames    []string
	}{
		{
			name:                "managed-by helm selector",
			annotationSelector:  map[string]string{"app.kubernetes.io/managed-by": "helm"},
			expectedCount:       2,
			expectedNames:       []string{"frontend-app", "staging-config"},
			notExpectedNames:    []string{"backend-app", "secret"},
		},
		{
			name:                "multiple annotation selectors (AND logic)",
			annotationSelector:  map[string]string{"app.kubernetes.io/managed-by": "helm", "environment": "production"},
			expectedCount:       1,
			expectedNames:       []string{"frontend-app"},
			notExpectedNames:    []string{"backend-app", "staging-config", "secret"},
		},
		{
			name:                "environment production selector",
			annotationSelector:  map[string]string{"environment": "production"},
			expectedCount:       2,
			expectedNames:       []string{"frontend-app", "backend-app"},
			notExpectedNames:    []string{"staging-config", "secret"},
		},
		{
			name:                "deployment category web selector",
			annotationSelector:  map[string]string{"deployment.category": "web"},
			expectedCount:       1,
			expectedNames:       []string{"frontend-app"},
			notExpectedNames:    []string{"backend-app", "staging-config", "secret"},
		},
		{
			name:                "empty selector returns all objects",
			annotationSelector:  nil,
			expectedCount:       4,
			expectedNames:       []string{"frontend-app", "backend-app", "staging-config", "secret"},
			notExpectedNames:    []string{},
		},
		{
			name:                "non-matching selector returns empty",
			annotationSelector:  map[string]string{"nonexistent": "value"},
			expectedCount:       0,
			expectedNames:       []string{},
			notExpectedNames:    []string{"frontend-app", "backend-app", "staging-config", "secret"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &Options{
				AnnotationSelector: tt.annotationSelector,
			}
			filtered := FilterResources(objects, opts)
			assert.Equal(t, tt.expectedCount, len(filtered))

			if tt.expectedCount > 0 {
				names := make([]string, len(filtered))
				for i, obj := range filtered {
					names[i] = obj.GetName()
				}

				for _, expectedName := range tt.expectedNames {
					assert.Contains(t, names, expectedName)
				}

				for _, notExpectedName := range tt.notExpectedNames {
					assert.NotContains(t, names, notExpectedName)
				}
			}
		})
	}
}

func TestFilterResources_CombinedLabelAndAnnotationSelector(t *testing.T) {
	frontendObj := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]any{
				"name":      "frontend-app",
				"namespace": "default",
				"labels": map[string]any{
					"app":  "nginx",
					"tier": "frontend",
				},
				"annotations": map[string]any{
					"app.kubernetes.io/managed-by": "helm",
					"deployment.category":          "web",
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
					"app":  "api",
					"tier": "backend",
				},
				"annotations": map[string]any{
					"app.kubernetes.io/managed-by": "kubectl",
					"deployment.category":          "api",
				},
			},
		},
	}

	configObj := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]any{
				"name":      "app-config",
				"namespace": "default",
				"labels": map[string]any{
					"app":  "nginx",
					"tier": "config",
				},
				"annotations": map[string]any{
					"app.kubernetes.io/managed-by": "helm",
					"config.category":              "web",
				},
			},
		},
	}

	objects := []*unstructured.Unstructured{frontendObj, backendObj, configObj}

	tests := []struct {
		name               string
		labelSelector      map[string]string
		annotationSelector map[string]string
		expectedCount      int
		expectedNames      []string
		notExpectedNames   []string
	}{
		{
			name:               "both label and annotation match (AND logic)",
			labelSelector:      map[string]string{"app": "nginx"},
			annotationSelector: map[string]string{"app.kubernetes.io/managed-by": "helm"},
			expectedCount:      2,
			expectedNames:      []string{"frontend-app", "app-config"},
			notExpectedNames:   []string{"backend-app"},
		},
		{
			name:               "label matches but annotation doesn't",
			labelSelector:      map[string]string{"app": "nginx"},
			annotationSelector: map[string]string{"app.kubernetes.io/managed-by": "kubectl"},
			expectedCount:      0,
			expectedNames:      []string{},
			notExpectedNames:   []string{"frontend-app", "backend-app", "app-config"},
		},
		{
			name:               "annotation matches but label doesn't",
			labelSelector:      map[string]string{"app": "api"},
			annotationSelector: map[string]string{"app.kubernetes.io/managed-by": "helm"},
			expectedCount:      0,
			expectedNames:      []string{},
			notExpectedNames:   []string{"frontend-app", "backend-app", "app-config"},
		},
		{
			name:               "multiple label and annotation selectors",
			labelSelector:      map[string]string{"app": "nginx", "tier": "frontend"},
			annotationSelector: map[string]string{"app.kubernetes.io/managed-by": "helm", "deployment.category": "web"},
			expectedCount:      1,
			expectedNames:      []string{"frontend-app"},
			notExpectedNames:   []string{"backend-app", "app-config"},
		},
		{
			name:               "only label selector",
			labelSelector:      map[string]string{"tier": "backend"},
			annotationSelector: nil,
			expectedCount:      1,
			expectedNames:      []string{"backend-app"},
			notExpectedNames:   []string{"frontend-app", "app-config"},
		},
		{
			name:               "only annotation selector",
			labelSelector:      nil,
			annotationSelector: map[string]string{"deployment.category": "api"},
			expectedCount:      1,
			expectedNames:      []string{"backend-app"},
			notExpectedNames:   []string{"frontend-app", "app-config"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &Options{
				LabelSelector:      tt.labelSelector,
				AnnotationSelector: tt.annotationSelector,
			}
			filtered := FilterResources(objects, opts)
			assert.Equal(t, tt.expectedCount, len(filtered))

			if tt.expectedCount > 0 {
				names := make([]string, len(filtered))
				for i, obj := range filtered {
					names[i] = obj.GetName()
				}

				for _, expectedName := range tt.expectedNames {
					assert.Contains(t, names, expectedName)
				}

				for _, notExpectedName := range tt.notExpectedNames {
					assert.NotContains(t, names, notExpectedName)
				}
			}
		})
	}
}

func TestObjects_DiffOptionsFiltering(t *testing.T) {
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

	tests := []struct {
		name             string
		options          *Options
		shouldContain    []string
		shouldNotContain []string
	}{
		{
			name:             "default options include all objects",
			options:          DefaultOptions(),
			shouldContain:    []string{"ConfigMap", "Secret", "Workflow", "hook-pod"},
			shouldNotContain: []string{},
		},
		{
			name:             "include all when exclude kinds disabled",
			options:          &Options{ExcludeKinds: []string{}},
			shouldContain:    []string{"ConfigMap", "Secret", "Workflow", "hook-pod"},
			shouldNotContain: []string{},
		},
		{
			name:             "custom exclude kinds",
			options:          &Options{ExcludeKinds: []string{"ConfigMap", "Secret"}},
			shouldContain:    []string{"Workflow", "hook-pod"},
			shouldNotContain: []string{"ConfigMap", "Secret"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := Objects([]*unstructured.Unstructured{}, objects, tt.options)
			assert.NoError(t, err)
			assert.True(t, results.HasChanges())

			diffResult := results.StringDiff()
			for _, expected := range tt.shouldContain {
				assert.Contains(t, diffResult, expected)
			}
			for _, notExpected := range tt.shouldNotContain {
				assert.NotContains(t, diffResult, notExpected)
			}
		})
	}
}
