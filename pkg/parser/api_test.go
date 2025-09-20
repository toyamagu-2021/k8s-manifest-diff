package parser

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
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

			if tt.expectMasked {
				// Check that secret data is masked
				for _, secretName := range tt.checkSecrets {
					assert.Contains(t, result, secretName, "Secret name should be present")
					// Verify original values are not present
					assert.NotContains(t, result, "cGFzc3dvcmQxMjM=", "Original base64 password should not be present")
					assert.NotContains(t, result, "YWRtaW4=", "Original base64 username should not be present")
					assert.NotContains(t, result, "plain-text-config", "Original plain text config should not be present")
					assert.NotContains(t, result, "plain-text-token", "Original plain text token should not be present")
					// Verify masks are present
					assert.Contains(t, result, "+", "Masked values should contain + characters")
				}
			}

			// Verify YAML structure is maintained
			assert.Contains(t, result, "apiVersion", "YAML structure should be maintained")
			assert.Contains(t, result, "kind", "YAML structure should be maintained")
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

	// Verify secret is masked
	assert.Contains(t, result, "test-secret", "Secret name should be present")
	assert.NotContains(t, result, "cGFzc3dvcmQxMjM=", "Original base64 password should not be present")
	assert.NotContains(t, result, "YWRtaW4=", "Original base64 username should not be present")
	assert.NotContains(t, result, "plain-text-config", "Original plain text config should not be present")
	assert.Contains(t, result, "+", "Masked values should contain + characters")

	// Verify YAML structure is maintained
	assert.Contains(t, result, "apiVersion: v1", "YAML structure should be maintained")
	assert.Contains(t, result, "kind: Secret", "YAML structure should be maintained")
	assert.Contains(t, result, "metadata:", "YAML structure should be maintained")
	assert.Contains(t, result, "data:", "YAML structure should be maintained")
	assert.Contains(t, result, "stringData:", "YAML structure should be maintained")
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
				assert.Nil(t, result)
				return
			}

			assert.Len(t, result, len(tt.objects), "Result should have same length as input")

			maskedCount := 0
			for i, obj := range result {
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
					assert.NotEqual(t, tt.objects[i], obj, "Original object should not be modified")
				} else {
					// For non-secrets, should be a deep copy
					assert.Equal(t, tt.objects[i].GetKind(), obj.GetKind(), "Non-secret object kind should be preserved")
					assert.NotSame(t, tt.objects[i], obj, "Non-secret object should be a copy")
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

	// Get masked data from both secrets
	data1, found, _ := unstructured.NestedMap(result[0].Object, "data")
	assert.True(t, found)
	data2, found, _ := unstructured.NestedMap(result[1].Object, "data")
	assert.True(t, found)

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

			// Verify secrets are masked
			assert.Contains(t, result, "+", "Masked values should contain + characters")
			// Verify YAML structure is maintained
			assert.Contains(t, result, "apiVersion", "YAML structure should be maintained")
			assert.Contains(t, result, "kind", "YAML structure should be maintained")
			// Verify document separators are maintained for multiple documents
			if strings.Contains(tt.input, "---") {
				assert.Contains(t, result, "---", "Document separators should be maintained")
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
