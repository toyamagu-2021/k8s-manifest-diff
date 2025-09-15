package diff

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestObjectsWithLabelSelector(t *testing.T) {
	podWithLabel := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]any{
				"name":      "labeled-pod",
				"namespace": "default",
				"labels": map[string]any{
					"app": "nginx",
				},
			},
		},
	}

	podWithoutLabel := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]any{
				"name":      "unlabeled-pod",
				"namespace": "default",
			},
		},
	}

	headObjects := []*unstructured.Unstructured{podWithLabel, podWithoutLabel}
	baseObjects := []*unstructured.Unstructured{}

	t.Run("diff with matching label selector", func(t *testing.T) {
		opts := &Options{
			LabelSelector: map[string]string{
				"app": "nginx",
			},
		}
		results, err := Objects(baseObjects, headObjects, opts)
		assert.NoError(t, err)
		assert.True(t, results.HasChanges())

		// Only the labeled pod should be in results
		assert.Equal(t, 1, results.Count())
		assertResourceChange(t, results, "Pod/default/labeled-pod", Created)

		diffResult := results.StringDiff()
		assert.Contains(t, diffResult, "labeled-pod")
		assert.NotContains(t, diffResult, "unlabeled-pod")
	})

	t.Run("diff with non-matching label selector", func(t *testing.T) {
		opts := &Options{
			LabelSelector: map[string]string{
				"app": "nonexistent",
			},
		}
		results, err := Objects(baseObjects, headObjects, opts)
		assert.NoError(t, err)
		assert.False(t, results.HasChanges())
		assert.Equal(t, 0, results.Count())
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
  key: modified-value1
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: config2
data:
  key: value2
`

		baseReader := strings.NewReader(multiYamlBase)
		headReader := strings.NewReader(multiYamlHead)

		results, err := Yaml(baseReader, headReader, nil)
		assert.NoError(t, err)
		assert.True(t, results.HasChanges())

		// Should have 2 resources total (config1 changed, config2 unchanged)
		assert.Equal(t, 2, results.Count())
		assertResourceChange(t, results, "ConfigMap//config1", Changed)
		assertResourceChange(t, results, "ConfigMap//config2", Unchanged)

		diffResult := results.StringDiff()
		assert.Contains(t, diffResult, "config1")
		assert.Contains(t, diffResult, "value1")
		assert.Contains(t, diffResult, "modified-value1")
	})

	t.Run("empty yaml", func(t *testing.T) {
		emptyReader := strings.NewReader("")
		baseReader := strings.NewReader(baseYaml)

		results, err := Yaml(emptyReader, baseReader, nil)
		assert.NoError(t, err)
		assert.True(t, results.HasChanges())

		// Should show one created resource
		assert.Equal(t, 1, results.Count())
		assertResourceChange(t, results, "Pod/default/test-pod", Created)
	})
}

func TestObjects(t *testing.T) {
	// Hook pod with annotation
	hookPod := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]any{
				"name":      "test",
				"namespace": "namespace",
				"annotations": map[string]any{
					"argocd.argoproj.io/hook": "PreSync",
				},
				"labels": map[string]any{
					"duck": "all-species",
				},
			},
		},
	}

	t.Run("nodiff", func(t *testing.T) {
		results, err := Objects([]*unstructured.Unstructured{hookPod}, []*unstructured.Unstructured{hookPod}, nil)
		assert.NoError(t, err)
		assert.False(t, results.HasChanges())
		assert.Equal(t, 1, results.Count())

		// Check diff string output - should be empty for identical objects
		diffResult := results.StringDiff()
		assert.Equal(t, "", diffResult)

		assertResourceChange(t, results, "Pod/namespace/test", Unchanged)
	})

	t.Run("exists only head", func(t *testing.T) {
		results, err := Objects([]*unstructured.Unstructured{}, []*unstructured.Unstructured{hookPod}, nil)
		assert.NoError(t, err)
		assert.True(t, results.HasChanges())
		assert.Equal(t, 1, results.Count())

		// Check diff string output
		diffResult := results.StringDiff()
		assert.Contains(t, diffResult, "/Pod namespace/test")

		assertResourceChange(t, results, "Pod/namespace/test", Created)
	})

	t.Run("exists only base", func(t *testing.T) {
		results, err := Objects([]*unstructured.Unstructured{hookPod}, []*unstructured.Unstructured{}, nil)
		assert.NoError(t, err)
		assert.True(t, results.HasChanges())
		assert.Equal(t, 1, results.Count())

		// Check diff string output
		diffResult := results.StringDiff()
		assert.Contains(t, diffResult, "/Pod namespace/test")

		assertResourceChange(t, results, "Pod/namespace/test", Deleted)
	})

	t.Run("workflow excluded by default", func(t *testing.T) {
		workflow := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "argoproj.io/v1alpha1",
				"kind":       "Workflow",
				"metadata": map[string]any{
					"name": "test-workflow",
				},
			},
		}

		results, err := Objects([]*unstructured.Unstructured{}, []*unstructured.Unstructured{workflow}, DefaultOptions())
		assert.NoError(t, err)
		assert.True(t, results.HasChanges()) // Should include workflow by default (no exclusion in default options)
		assert.Equal(t, 1, results.Count())

		// Check that the workflow resource is found
		keys := results.GetResourceKeys()
		assert.Equal(t, 1, len(keys))
		assert.Equal(t, "Workflow", keys[0].Kind)
		assert.Equal(t, "test-workflow", keys[0].Name)
		assert.Equal(t, "", keys[0].Namespace) // cluster-scoped
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
