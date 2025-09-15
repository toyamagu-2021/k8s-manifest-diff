package diff

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestObjects_LabelSelector(t *testing.T) {
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

	tests := []struct {
		name                   string
		labelSelector          map[string]string
		expectChanges          bool
		expectedCount          int
		expectedResourceKey    string
		expectedChangeType     ChangeType
		shouldContainInDiff    []string
		shouldNotContainInDiff []string
	}{
		{
			name:                   "diff with matching label selector",
			labelSelector:          map[string]string{"app": "nginx"},
			expectChanges:          true,
			expectedCount:          1,
			expectedResourceKey:    "Pod/default/labeled-pod",
			expectedChangeType:     Created,
			shouldContainInDiff:    []string{"labeled-pod"},
			shouldNotContainInDiff: []string{"unlabeled-pod"},
		},
		{
			name:                   "diff with non-matching label selector",
			labelSelector:          map[string]string{"app": "nonexistent"},
			expectChanges:          false,
			expectedCount:          0,
			expectedResourceKey:    "",
			expectedChangeType:     Unchanged,
			shouldContainInDiff:    []string{},
			shouldNotContainInDiff: []string{"labeled-pod", "unlabeled-pod"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &Options{
				LabelSelector: tt.labelSelector,
			}
			results, err := Objects(baseObjects, headObjects, opts)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectChanges, results.HasChanges())
			assert.Equal(t, tt.expectedCount, results.Count())

			if tt.expectedResourceKey != "" {
				AssertResourceChange(t, results, tt.expectedResourceKey, tt.expectedChangeType)
			}

			diffResult := results.StringDiff()
			for _, expected := range tt.shouldContainInDiff {
				assert.Contains(t, diffResult, expected)
			}
			for _, notExpected := range tt.shouldNotContainInDiff {
				assert.NotContains(t, diffResult, notExpected)
			}
		})
	}
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

	tests := []struct {
		name                   string
		baseYaml               string
		headYaml               string
		options                *Options
		expectChanges          bool
		expectedCount          int
		expectedResourceKey    string
		expectedChangeType     ChangeType
		shouldContainInDiff    []string
		shouldNotContainInDiff []string
		expectError            bool
		expectedErrorMessage   string
		expectEmptyDiff        bool
	}{
		{
			name:                   "diff with changes",
			baseYaml:               baseYaml,
			headYaml:               headYaml,
			options:                nil,
			expectChanges:          true,
			expectedCount:          1,
			expectedResourceKey:    "ConfigMap/default/test-config",
			expectedChangeType:     Changed,
			shouldContainInDiff:    []string{"ConfigMap", "old-value", "new-value"},
			shouldNotContainInDiff: []string{},
		},
		{
			name:                "no diff when identical",
			baseYaml:            baseYaml,
			headYaml:            baseYaml,
			options:             nil,
			expectChanges:       false,
			expectedCount:       1,
			expectedResourceKey: "ConfigMap/default/test-config",
			expectedChangeType:  Unchanged,
			expectEmptyDiff:     true,
		},
		{
			name:                 "error on invalid base yaml",
			baseYaml:             `invalid: yaml: structure`,
			headYaml:             headYaml,
			options:              nil,
			expectError:          true,
			expectedErrorMessage: "failed to parse base YAML",
		},
		{
			name:                 "error on invalid head yaml",
			baseYaml:             baseYaml,
			headYaml:             `invalid: yaml: structure`,
			options:              nil,
			expectError:          true,
			expectedErrorMessage: "failed to parse head YAML",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := YamlString(tt.baseYaml, tt.headYaml, tt.options)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrorMessage)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectChanges, results.HasChanges())

			if tt.expectedResourceKey != "" {
				assert.Equal(t, tt.expectedCount, results.Count())
				AssertResourceChange(t, results, tt.expectedResourceKey, tt.expectedChangeType)
			}

			diffResult := results.StringDiff()

			if tt.expectEmptyDiff {
				assert.Equal(t, "", diffResult)
			} else {
				for _, expected := range tt.shouldContainInDiff {
					assert.Contains(t, diffResult, expected)
				}
				for _, notExpected := range tt.shouldNotContainInDiff {
					assert.NotContains(t, diffResult, notExpected)
				}
			}

			if tt.expectChanges {
				changedResourcesList := GetChangedResourceKeys(results)
				if tt.expectedChangeType != Unchanged {
					assert.Equal(t, 1, len(changedResourcesList))
				}
			}
		})
	}
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

	tests := []struct {
		name                   string
		baseYaml               string
		headYaml               string
		options                *Options
		expectChanges          bool
		expectedCount          int
		expectedResourceKeys   map[string]ChangeType
		shouldContainInDiff    []string
		shouldNotContainInDiff []string
		expectEmptyDiff        bool
	}{
		{
			name:          "diff with io.Reader",
			baseYaml:      baseYaml,
			headYaml:      headYaml,
			options:       nil,
			expectChanges: true,
			expectedCount: 1,
			expectedResourceKeys: map[string]ChangeType{
				"Pod/default/test-pod": Changed,
			},
			shouldContainInDiff: []string{"Pod", "nginx:1.20", "nginx:1.21"},
		},
		{
			name:          "no diff when identical",
			baseYaml:      baseYaml,
			headYaml:      baseYaml,
			options:       nil,
			expectChanges: false,
			expectedCount: 1,
			expectedResourceKeys: map[string]ChangeType{
				"Pod/default/test-pod": Unchanged,
			},
			expectEmptyDiff: true,
		},
		{
			name:          "multiple objects in yaml",
			baseYaml:      multiYamlBase,
			headYaml:      multiYamlHead,
			options:       nil,
			expectChanges: true,
			expectedCount: 2,
			expectedResourceKeys: map[string]ChangeType{
				"ConfigMap//config1": Changed,
				"ConfigMap//config2": Unchanged,
			},
			shouldContainInDiff: []string{"config1", "value1", "modified-value1"},
		},
		{
			name:          "empty yaml",
			baseYaml:      "",
			headYaml:      baseYaml,
			options:       nil,
			expectChanges: true,
			expectedCount: 1,
			expectedResourceKeys: map[string]ChangeType{
				"Pod/default/test-pod": Created,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseReader := strings.NewReader(tt.baseYaml)
			headReader := strings.NewReader(tt.headYaml)

			results, err := Yaml(baseReader, headReader, tt.options)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectChanges, results.HasChanges())
			assert.Equal(t, tt.expectedCount, results.Count())

			for resourceKey, changeType := range tt.expectedResourceKeys {
				AssertResourceChange(t, results, resourceKey, changeType)
			}

			diffResult := results.StringDiff()

			if tt.expectEmptyDiff {
				assert.Equal(t, "", diffResult)
			} else {
				for _, expected := range tt.shouldContainInDiff {
					assert.Contains(t, diffResult, expected)
				}
				for _, notExpected := range tt.shouldNotContainInDiff {
					assert.NotContains(t, diffResult, notExpected)
				}
			}

			if tt.expectChanges {
				changedResourcesList := GetChangedResourceKeys(results)
				changedCount := 0
				for _, changeType := range tt.expectedResourceKeys {
					if changeType != Unchanged {
						changedCount++
					}
				}
				assert.Equal(t, changedCount, len(changedResourcesList))
			}
		})
	}
}

func TestObjects_Integration(t *testing.T) {
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

	workflow := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "argoproj.io/v1alpha1",
			"kind":       "Workflow",
			"metadata": map[string]any{
				"name": "test-workflow",
			},
		},
	}

	tests := []struct {
		name                string
		baseObjects         []*unstructured.Unstructured
		headObjects         []*unstructured.Unstructured
		options             *Options
		expectChanges       bool
		expectedCount       int
		expectedResourceKey string
		expectedChangeType  ChangeType
		shouldContainInDiff []string
		expectEmptyDiff     bool
		validateAdditional  func(t *testing.T, results Results)
	}{
		{
			name:                "nodiff",
			baseObjects:         []*unstructured.Unstructured{hookPod},
			headObjects:         []*unstructured.Unstructured{hookPod},
			options:             nil,
			expectChanges:       false,
			expectedCount:       1,
			expectedResourceKey: "Pod/namespace/test",
			expectedChangeType:  Unchanged,
			expectEmptyDiff:     true,
		},
		{
			name:                "exists only head",
			baseObjects:         []*unstructured.Unstructured{},
			headObjects:         []*unstructured.Unstructured{hookPod},
			options:             nil,
			expectChanges:       true,
			expectedCount:       1,
			expectedResourceKey: "Pod/namespace/test",
			expectedChangeType:  Created,
			shouldContainInDiff: []string{"/Pod namespace/test"},
		},
		{
			name:                "exists only base",
			baseObjects:         []*unstructured.Unstructured{hookPod},
			headObjects:         []*unstructured.Unstructured{},
			options:             nil,
			expectChanges:       true,
			expectedCount:       1,
			expectedResourceKey: "Pod/namespace/test",
			expectedChangeType:  Deleted,
			shouldContainInDiff: []string{"/Pod namespace/test"},
		},
		{
			name:                "workflow included by default",
			baseObjects:         []*unstructured.Unstructured{},
			headObjects:         []*unstructured.Unstructured{workflow},
			options:             DefaultOptions(),
			expectChanges:       true,
			expectedCount:       1,
			expectedResourceKey: "Workflow//test-workflow",
			expectedChangeType:  Created,
			validateAdditional: func(t *testing.T, results Results) {
				keys := results.GetResourceKeys()
				assert.Equal(t, 1, len(keys))
				assert.Equal(t, "Workflow", keys[0].Kind)
				assert.Equal(t, "test-workflow", keys[0].Name)
				assert.Equal(t, "", keys[0].Namespace)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := Objects(tt.baseObjects, tt.headObjects, tt.options)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectChanges, results.HasChanges())
			assert.Equal(t, tt.expectedCount, results.Count())

			if tt.expectedResourceKey != "" {
				AssertResourceChange(t, results, tt.expectedResourceKey, tt.expectedChangeType)
			}

			diffResult := results.StringDiff()

			if tt.expectEmptyDiff {
				assert.Equal(t, "", diffResult)
			} else {
				for _, expected := range tt.shouldContainInDiff {
					assert.Contains(t, diffResult, expected)
				}
			}

			if tt.validateAdditional != nil {
				tt.validateAdditional(t, results)
			}
		})
	}
}

func TestObjects_WithNilOptions(t *testing.T) {
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

	diffResult := results.StringDiff()
	assert.Contains(t, diffResult, "ConfigMap")

	changedResourcesList := GetChangedResourceKeys(results)
	assert.Equal(t, 1, len(changedResourcesList))
	AssertResourceChange(t, results, "ConfigMap/test/config", Created)
}
