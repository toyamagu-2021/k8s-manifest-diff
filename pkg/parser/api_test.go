package parser

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/toyamagu-2021/k8s-manifest-diff/pkg/filter"
	"github.com/toyamagu-2021/k8s-manifest-diff/pkg/masking"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestYamlString(t *testing.T) {
	// Reset masking state before test
	masking.ResetMaskingState()

	tests := []struct {
		name         string
		input        string
		expectError  bool
		expectMasked bool
		checkSecrets []string // Secret names that should be masked
	}{
		{
			name: "single secret",
			input: `apiVersion: v1
kind: Secret
metadata:
  name: test-secret
  namespace: default
type: Opaque
data:
  password: cGFzc3dvcmQxMjM= # gitleaks:allow
  username: YWRtaW4=`,
			expectError:  false,
			expectMasked: true,
			checkSecrets: []string{"test-secret"},
		},
		{
			name: "secret with stringData",
			input: `apiVersion: v1
kind: Secret
metadata:
  name: string-secret
  namespace: default
type: Opaque
stringData:
  config: plain-text-config
  token: plain-text-token`,
			expectError:  false,
			expectMasked: true,
			checkSecrets: []string{"string-secret"},
		},
		{
			name: "mixed resources with secret", // gitleaks:allow
			input: `apiVersion: v1
kind: ConfigMap
metadata:
  name: test-config
  namespace: default
data:
  config: some-value
---
apiVersion: v1
kind: Secret
metadata:
  name: test-secret
  namespace: default
type: Opaque
data:
  password: cGFzc3dvcmQxMjM= # gitleaks:allow`, // gitleaks:allow
			expectError:  false,
			expectMasked: true,
			checkSecrets: []string{"test-secret"},
		},
		{
			name: "no secrets",
			input: `apiVersion: v1
kind: ConfigMap
metadata:
  name: test-config
  namespace: default
data:
  config: some-value`,
			expectError:  false,
			expectMasked: false,
			checkSecrets: []string{},
		},
		{
			name:        "invalid YAML",
			input:       "invalid: yaml: content:\n  - unclosed",
			expectError: true,
		},
		{
			name:         "empty input",
			input:        "",
			expectError:  false,
			expectMasked: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := YamlString(tt.input, nil)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)

			if tt.input == "" {
				// Empty input should return empty result
				assert.Empty(t, result)
				return
			}

			assert.NotEmpty(t, result)

			// Convert Results to YAML string for verification
			resultYAML := result.String()

			if tt.expectMasked {
				// Check that secret data is masked
				for _, secretName := range tt.checkSecrets {
					assert.Contains(t, resultYAML, secretName, "Secret name should be present")
					// Verify original values are not present
					assert.NotContains(t, resultYAML, "cGFzc3dvcmQxMjM=", "Original base64 password should not be present")
					assert.NotContains(t, resultYAML, "YWRtaW4=", "Original base64 username should not be present")
					assert.NotContains(t, resultYAML, "plain-text-config", "Original plain text config should not be present")
					assert.NotContains(t, resultYAML, "plain-text-token", "Original plain text token should not be present")
					// Verify masks are present
					assert.Contains(t, resultYAML, "+", "Masked values should contain + characters")
				}
			}

			// Verify YAML structure is maintained
			assert.Contains(t, resultYAML, "apiVersion", "YAML structure should be maintained")
			assert.Contains(t, resultYAML, "kind", "YAML structure should be maintained")
		})
	}
}

func TestYaml(t *testing.T) {
	// Reset masking state before test
	masking.ResetMaskingState()

	secretYaml := `apiVersion: v1
kind: Secret
metadata:
  name: test-secret
  namespace: default
type: Opaque
data:
  password: cGFzc3dvcmQxMjM= # gitleaks:allow
  username: YWRtaW4=
stringData:
  config: plain-text-config`

	reader := strings.NewReader(secretYaml)
	result, err := Yaml(reader, nil)

	assert.NoError(t, err)
	assert.NotEmpty(t, result)

	// Convert Results to YAML string for verification
	resultYAML := result.String()

	// Verify secret is masked
	assert.Contains(t, resultYAML, "test-secret", "Secret name should be present")
	assert.NotContains(t, resultYAML, "cGFzc3dvcmQxMjM=", "Original base64 password should not be present")
	assert.NotContains(t, resultYAML, "YWRtaW4=", "Original base64 username should not be present")
	assert.NotContains(t, resultYAML, "plain-text-config", "Original plain text config should not be present")
	assert.Contains(t, resultYAML, "+", "Masked values should contain + characters")

	// Verify YAML structure is maintained
	assert.Contains(t, resultYAML, "apiVersion: v1", "YAML structure should be maintained")
	assert.Contains(t, resultYAML, "kind: Secret", "YAML structure should be maintained")
	assert.Contains(t, resultYAML, "metadata:", "YAML structure should be maintained")
	assert.Contains(t, resultYAML, "data:", "YAML structure should be maintained")
	assert.Contains(t, resultYAML, "stringData:", "YAML structure should be maintained")
}

func TestObjects(t *testing.T) {
	// Reset masking state before test
	masking.ResetMaskingState()

	tests := []struct {
		name         string
		objects      []*unstructured.Unstructured
		expectError  bool
		expectMasked int // Number of objects that should be masked
	}{
		{
			name:         "nil input",
			objects:      nil,
			expectError:  false,
			expectMasked: 0,
		},
		{
			name:         "empty slice",
			objects:      []*unstructured.Unstructured{},
			expectError:  false,
			expectMasked: 0,
		},
		{
			name: "single secret",
			objects: []*unstructured.Unstructured{
				{
					Object: map[string]any{
						"apiVersion": "v1",
						"kind":       "Secret",
						"metadata": map[string]any{
							"name":      "test-secret",
							"namespace": "default",
						},
						"type": "Opaque",
						"data": map[string]any{
							"password": "cGFzc3dvcmQxMjM=", // gitleaks:allow
							"username": "YWRtaW4=",
						},
					},
				},
			},
			expectError:  false,
			expectMasked: 1,
		},
		{
			name: "mixed objects",
			objects: []*unstructured.Unstructured{
				{
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
				},
				{
					Object: map[string]any{
						"apiVersion": "v1",
						"kind":       "Secret",
						"metadata": map[string]any{
							"name":      "test-secret",
							"namespace": "default",
						},
						"type": "Opaque",
						"data": map[string]any{
							"password": "cGFzc3dvcmQxMjM=", // gitleaks:allow
						},
					},
				},
			},
			expectError:  false,
			expectMasked: 1,
		},
		{
			name: "multiple secrets",
			objects: []*unstructured.Unstructured{
				{
					Object: map[string]any{
						"apiVersion": "v1",
						"kind":       "Secret",
						"metadata": map[string]any{
							"name":      "secret1",
							"namespace": "default",
						},
						"type": "Opaque",
						"data": map[string]any{
							"password": "cGFzc3dvcmQxMjM=", // gitleaks:allow
						},
					},
				},
				{
					Object: map[string]any{
						"apiVersion": "v1",
						"kind":       "Secret",
						"metadata": map[string]any{
							"name":      "secret2",
							"namespace": "default",
						},
						"type": "Opaque",
						"stringData": map[string]any{
							"token": "plain-text-token",
						},
					},
				},
			},
			expectError:  false,
			expectMasked: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Objects(tt.objects, nil)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)

			if tt.objects == nil {
				assert.Empty(t, result)
				return
			}

			assert.Len(t, result, len(tt.objects), "Result should have same length as input")

			maskedCount := 0
			originalObjectsByKey := make(map[ResourceKey]*unstructured.Unstructured)
			for _, origObj := range tt.objects {
				key := ResourceKey{
					Name:      origObj.GetName(),
					Namespace: origObj.GetNamespace(),
					Group:     origObj.GetObjectKind().GroupVersionKind().Group,
					Kind:      origObj.GetKind(),
				}
				originalObjectsByKey[key] = origObj
			}

			for key, obj := range result {
				assert.NotNil(t, obj, "Result object should not be nil")

				if masking.IsSecret(obj) {
					maskedCount++
					// Verify secret is masked
					if dataMap, found, _ := unstructured.NestedMap(obj.Object, "data"); found {
						for _, value := range dataMap {
							if strValue, ok := value.(string); ok && strValue != "" {
								assert.True(t, strings.Contains(strValue, "+"), "Secret data should be masked")
							}
						}
					}
					if stringDataMap, found, _ := unstructured.NestedMap(obj.Object, "stringData"); found {
						for _, value := range stringDataMap {
							if strValue, ok := value.(string); ok && strValue != "" {
								assert.True(t, strings.Contains(strValue, "+"), "Secret stringData should be masked")
							}
						}
					}
					// Verify original is not modified
					if origObj, exists := originalObjectsByKey[key]; exists {
						assert.NotEqual(t, origObj, obj, "Original object should not be modified")
					}
				} else {
					// For non-secrets, should be a deep copy
					if origObj, exists := originalObjectsByKey[key]; exists {
						assert.Equal(t, origObj.GetKind(), obj.GetKind(), "Non-secret object kind should be preserved")
						assert.NotSame(t, origObj, obj, "Non-secret object should be a copy")
					}
				}
			}

			assert.Equal(t, tt.expectMasked, maskedCount, "Number of masked objects should match expectation")
		})
	}
}

func TestObjectsConsistency(t *testing.T) {
	// Reset masking state before test
	masking.ResetMaskingState()

	// Create two secrets with same and different values
	objects := []*unstructured.Unstructured{
		{
			Object: map[string]any{
				"apiVersion": "v1",
				"kind":       "Secret",
				"metadata": map[string]any{
					"name":      "secret1",
					"namespace": "default",
				},
				"type": "Opaque",
				"data": map[string]any{
					"shared": "c2hhcmVkVmFsdWU=", // base64 "sharedValue"
					"unique": "dW5pcXVlMQ==",     // base64 "unique1"
				},
			},
		},
		{
			Object: map[string]any{
				"apiVersion": "v1",
				"kind":       "Secret",
				"metadata": map[string]any{
					"name":      "secret2",
					"namespace": "default",
				},
				"type": "Opaque",
				"data": map[string]any{
					"shared": "c2hhcmVkVmFsdWU=", // base64 "sharedValue" (same as secret1)
					"unique": "dW5pcXVlMg==",     // base64 "unique2" (different from secret1)
				},
			},
		},
	}

	result, err := Objects(objects, nil)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	// Get masked data from both secrets - need to iterate through results
	var data1, data2 map[string]interface{}
	resultSlice := make([]*unstructured.Unstructured, 0, len(result))
	for _, obj := range result {
		resultSlice = append(resultSlice, obj)
	}

	data1Map, found, _ := unstructured.NestedMap(resultSlice[0].Object, "data")
	assert.True(t, found)
	data1 = data1Map
	data2Map, found, _ := unstructured.NestedMap(resultSlice[1].Object, "data")
	assert.True(t, found)
	data2 = data2Map

	// Same values should get same masks
	assert.Equal(t, data1["shared"], data2["shared"], "Same values should get identical masks")

	// Different values should get different masks
	assert.NotEqual(t, data1["unique"], data2["unique"], "Different values should get different masks")

	// All values should be masked
	for key, value := range data1 {
		if strValue, ok := value.(string); ok {
			assert.True(t, strings.Contains(strValue, "+"), "Value for key %s should be masked", key)
		}
	}
	for key, value := range data2 {
		if strValue, ok := value.(string); ok {
			assert.True(t, strings.Contains(strValue, "+"), "Value for key %s should be masked", key)
		}
	}
}

func TestYamlStringComplexScenarios(t *testing.T) {
	// Reset masking state before test
	masking.ResetMaskingState()

	tests := []struct {
		name        string
		input       string
		expectError bool
	}{
		{
			name: "multiple documents with secrets",
			input: `apiVersion: v1
kind: Secret
metadata:
  name: secret1
  namespace: default
type: Opaque
data:
  password: cGFzc3dvcmQxMjM= # gitleaks:allow
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: config1
  namespace: default
data:
  config: value
---
apiVersion: v1
kind: Secret
metadata:
  name: secret2
  namespace: default
type: Opaque
stringData:
  token: plain-text-token`,
			expectError: false,
		},
		{
			name: "secret with complex data structures",
			input: `apiVersion: v1
kind: Secret
metadata:
  name: complex-secret
  namespace: default
type: Opaque
data:
  config.json: eyJhcHAiOnsibmFtZSI6Im15YXBwIiwiZGIiOnsiaG9zdCI6ImRiLmV4YW1wbGUuY29tIn19fQ==
  users.yaml: LSBuYW1lOiBhZG1pbgogIHBhc3N3b3JkOiBzZWNyZXQx
stringData:
  env: |
    DB_HOST=localhost
    DB_PASS=secret123
    API_TOKEN=token456`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := YamlString(tt.input, nil)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotEmpty(t, result)

			// Convert Results to YAML string for verification
			resultYAML := result.String()

			// Verify secrets are masked
			assert.Contains(t, resultYAML, "+", "Masked values should contain + characters")
			// Verify YAML structure is maintained
			assert.Contains(t, resultYAML, "apiVersion", "YAML structure should be maintained")
			assert.Contains(t, resultYAML, "kind", "YAML structure should be maintained")
			// Verify document separators are maintained for multiple documents
			if strings.Contains(tt.input, "---") {
				assert.Contains(t, resultYAML, "---", "Document separators should be maintained")
			}
		})
	}
}

func TestYamlStringEdgeCases(t *testing.T) {
	// Reset masking state before test
	masking.ResetMaskingState()

	tests := []struct {
		name        string
		input       string
		expectError bool
		expectEmpty bool
	}{
		{
			name:        "empty string",
			input:       "",
			expectError: false,
			expectEmpty: false, // Should return empty YAML, not error
		},
		{
			name:        "whitespace only",
			input:       "   \n  \t  \n   ",
			expectError: true,
			expectEmpty: false,
		},
		{
			name: "yaml comments only",
			input: `# This is a comment
# Another comment`,
			expectError: false,
			expectEmpty: false,
		},
		{
			name: "yaml with only document separator",
			input: `---
---`,
			expectError: false,
			expectEmpty: false,
		},
		{
			name:        "malformed yaml",
			input:       "invalid: yaml: content:\n  - unclosed\n    nested:",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := YamlString(tt.input, nil)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			if tt.expectEmpty {
				assert.Empty(t, result)
			}
		})
	}
}

func TestObjectsWithFiltering(t *testing.T) {
	// Reset masking state before test
	masking.ResetMaskingState()

	// Create test objects
	configMap := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]any{
				"name":      "test-config",
				"namespace": "default",
				"labels": map[string]any{
					"app":  "myapp",
					"tier": "frontend",
				},
				"annotations": map[string]any{
					"version": "1.0",
				},
			},
			"data": map[string]any{
				"config": "some-value",
			},
		},
	}

	deployment := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]any{
				"name":      "test-deployment",
				"namespace": "default",
				"labels": map[string]any{
					"app":  "myapp",
					"tier": "backend",
				},
				"annotations": map[string]any{
					"version": "2.0",
				},
			},
			"spec": map[string]any{
				"replicas": int64(3),
			},
		},
	}

	secret := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "Secret",
			"metadata": map[string]any{
				"name":      "test-secret",
				"namespace": "default",
				"labels": map[string]any{
					"app":  "myapp",
					"tier": "data",
				},
			},
			"type": "Opaque",
			"data": map[string]any{
				"password": "cGFzc3dvcmQxMjM=", // gitleaks:allow
			},
		},
	}

	tests := []struct {
		name          string
		objects       []*unstructured.Unstructured
		opts          *Options
		expectedCount int
		expectedKinds []string
		expectError   bool
		expectMasked  int
	}{
		{
			name:          "no filtering",
			objects:       []*unstructured.Unstructured{configMap, deployment, secret},
			opts:          nil,
			expectedCount: 3,
			expectedKinds: []string{"ConfigMap", "Deployment", "Secret"},
			expectMasked:  1,
		},
		{
			name:    "exclude deployments",
			objects: []*unstructured.Unstructured{configMap, deployment, secret},
			opts: &Options{
				FilterOption: &filter.Option{
					ExcludeKinds: []string{"Deployment"},
				},
			},
			expectedCount: 2,
			expectedKinds: []string{"ConfigMap", "Secret"},
			expectMasked:  1,
		},
		{
			name:    "exclude multiple kinds",
			objects: []*unstructured.Unstructured{configMap, deployment, secret},
			opts: &Options{
				FilterOption: &filter.Option{
					ExcludeKinds: []string{"Deployment", "Secret"},
				},
			},
			expectedCount: 1,
			expectedKinds: []string{"ConfigMap"},
			expectMasked:  0,
		},
		{
			name:    "label selector filtering",
			objects: []*unstructured.Unstructured{configMap, deployment, secret},
			opts: &Options{
				FilterOption: &filter.Option{
					LabelSelector: map[string]string{
						"tier": "frontend",
					},
				},
			},
			expectedCount: 1,
			expectedKinds: []string{"ConfigMap"},
			expectMasked:  0,
		},
		{
			name:    "annotation selector filtering",
			objects: []*unstructured.Unstructured{configMap, deployment, secret},
			opts: &Options{
				FilterOption: &filter.Option{
					AnnotationSelector: map[string]string{
						"version": "2.0",
					},
				},
			},
			expectedCount: 1,
			expectedKinds: []string{"Deployment"},
			expectMasked:  0,
		},
		{
			name:    "combined filtering",
			objects: []*unstructured.Unstructured{configMap, deployment, secret},
			opts: &Options{
				FilterOption: &filter.Option{
					ExcludeKinds: []string{"Secret"},
					LabelSelector: map[string]string{
						"app": "myapp",
					},
				},
			},
			expectedCount: 2,
			expectedKinds: []string{"ConfigMap", "Deployment"},
			expectMasked:  0,
		},
		{
			name:    "filter out all objects",
			objects: []*unstructured.Unstructured{configMap, deployment, secret},
			opts: &Options{
				FilterOption: &filter.Option{
					LabelSelector: map[string]string{
						"nonexistent": "label",
					},
				},
			},
			expectedCount: 0,
			expectedKinds: []string{},
			expectMasked:  0,
		},
		{
			name:    "disable masking with filtering",
			objects: []*unstructured.Unstructured{configMap, deployment, secret},
			opts: &Options{
				FilterOption: &filter.Option{
					LabelSelector: map[string]string{
						"app": "myapp",
					},
				},
				DisableMaskingSecrets: true,
			},
			expectedCount: 3,
			expectedKinds: []string{"ConfigMap", "Deployment", "Secret"},
			expectMasked:  0, // Masking disabled
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Objects(tt.objects, tt.opts)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Len(t, result, tt.expectedCount, "Result count should match expected")

			// Check that the expected kinds are present
			actualKinds := make([]string, 0, len(result))
			maskedCount := 0
			for _, obj := range result {
				assert.NotNil(t, obj, "Result object should not be nil")
				actualKinds = append(actualKinds, obj.GetKind())

				if masking.IsSecret(obj) && (tt.opts == nil || !tt.opts.DisableMaskingSecrets) {
					maskedCount++
					// Verify secret is masked
					if dataMap, found, _ := unstructured.NestedMap(obj.Object, "data"); found {
						for _, value := range dataMap {
							if strValue, ok := value.(string); ok && strValue != "" {
								assert.True(t, strings.Contains(strValue, "+"), "Secret data should be masked")
							}
						}
					}
				}
			}

			// Check if all expected kinds are present (order doesn't matter)
			for _, expectedKind := range tt.expectedKinds {
				assert.Contains(t, actualKinds, expectedKind, "Expected kind %s should be present", expectedKind)
			}

			assert.Equal(t, tt.expectMasked, maskedCount, "Number of masked objects should match expectation")
		})
	}
}

func TestYamlStringWithFiltering(t *testing.T) {
	// Reset masking state before test
	masking.ResetMaskingState()

	yamlInput := `apiVersion: v1
kind: ConfigMap
metadata:
  name: test-config
  namespace: default
  labels:
    app: myapp
    tier: frontend
data:
  config: some-value
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-deployment
  namespace: default
  labels:
    app: myapp
    tier: backend
spec:
  replicas: 3
---
apiVersion: v1
kind: Secret
metadata:
  name: test-secret
  namespace: default
  labels:
    app: myapp
    tier: data
type: Opaque
data:
  password: cGFzc3dvcmQxMjM= # gitleaks:allow`

	tests := []struct {
		name                string
		opts                *Options
		expectError         bool
		expectedResources   int
		shouldContainSecret bool
		shouldContainDeploy bool
	}{
		{
			name:                "no filtering",
			opts:                nil,
			expectedResources:   3,
			shouldContainSecret: true,
			shouldContainDeploy: true,
		},
		{
			name: "exclude deployments",
			opts: &Options{
				FilterOption: &filter.Option{
					ExcludeKinds: []string{"Deployment"},
				},
			},
			expectedResources:   2,
			shouldContainSecret: true,
			shouldContainDeploy: false,
		},
		{
			name: "filter by label",
			opts: &Options{
				FilterOption: &filter.Option{
					LabelSelector: map[string]string{
						"tier": "frontend",
					},
				},
			},
			expectedResources:   1,
			shouldContainSecret: false,
			shouldContainDeploy: false,
		},
		{
			name: "exclude secrets",
			opts: &Options{
				FilterOption: &filter.Option{
					ExcludeKinds: []string{"Secret"},
				},
			},
			expectedResources:   2,
			shouldContainSecret: false,
			shouldContainDeploy: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := YamlString(yamlInput, tt.opts)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)

			if tt.expectedResources == 0 {
				assert.Empty(t, result)
				return
			}

			assert.NotEmpty(t, result)

			// Check the actual number of resources in Results
			assert.Len(t, result, tt.expectedResources, "Number of resources should match expected")

			// Convert Results to YAML string for verification
			resultYAML := result.String()

			// Check presence of specific resources
			if tt.shouldContainSecret {
				assert.Contains(t, resultYAML, "kind: Secret", "Should contain Secret")
				assert.Contains(t, resultYAML, "test-secret", "Should contain secret name")
			} else {
				assert.NotContains(t, resultYAML, "kind: Secret", "Should not contain Secret")
			}

			if tt.shouldContainDeploy {
				assert.Contains(t, resultYAML, "kind: Deployment", "Should contain Deployment")
				assert.Contains(t, resultYAML, "test-deployment", "Should contain deployment name")
			} else {
				assert.NotContains(t, resultYAML, "kind: Deployment", "Should not contain Deployment")
			}
		})
	}
}
