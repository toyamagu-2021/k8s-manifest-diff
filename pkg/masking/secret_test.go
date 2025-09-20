package masking

// gitleaks:ignore-file
import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestIsSecret(t *testing.T) {
	tests := []struct {
		name     string
		obj      *unstructured.Unstructured
		expected bool
	}{
		{
			name:     "nil object",
			obj:      nil,
			expected: false,
		},
		{
			name: "secret object",
			obj: &unstructured.Unstructured{
				Object: map[string]any{
					"kind": "Secret",
				},
			},
			expected: true,
		},
		{
			name: "non-secret object",
			obj: &unstructured.Unstructured{
				Object: map[string]any{
					"kind": "ConfigMap",
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsSecret(tt.obj))
		})
	}
}

func TestMaskSecretData(t *testing.T) {
	// Reset masking state before each test
	ResetMaskingState()

	secret := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "Secret",
			"metadata": map[string]any{
				"name":      "test-secret",
				"namespace": "default",
			},
			"type": "Opaque",
			"data": map[string]any{
				"password": "cGFzc3dvcmQxMjM=", // base64 encoded "password123" // gitleaks:allow
				"username": "YWRtaW4=",         // base64 encoded "admin"
			},
			"stringData": map[string]any{
				"config": "plain-text-config",
				"token":  "plain-text-token",
			},
		},
	}

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

	tests := []struct {
		name              string
		obj               *unstructured.Unstructured
		expectMasked      bool
		checkOriginalData bool
	}{
		{
			name:              "masks secret data",
			obj:               secret,
			expectMasked:      true,
			checkOriginalData: true,
		},
		{
			name:         "non-secret object unchanged",
			obj:          configMap,
			expectMasked: false,
		},
		{
			name:         "nil object returns nil",
			obj:          nil,
			expectMasked: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			masked, err := MaskSecretData(tt.obj)
			assert.NoError(t, err)

			if tt.obj == nil {
				assert.Nil(t, masked)
				return
			}

			if !tt.expectMasked {
				assert.Equal(t, tt.obj, masked)
				return
			}

			// Verify the original object is not modified
			if tt.checkOriginalData {
				originalData, found, _ := unstructured.NestedMap(tt.obj.Object, "data")
				assert.True(t, found)
				assert.Equal(t, "cGFzc3dvcmQxMjM=", originalData["password"])
				assert.Equal(t, "YWRtaW4=", originalData["username"])

				originalStringData, found, _ := unstructured.NestedMap(tt.obj.Object, "stringData")
				assert.True(t, found)
				assert.Equal(t, "plain-text-config", originalStringData["config"])
				assert.Equal(t, "plain-text-token", originalStringData["token"])
			}

			// Verify the masked object has masked values
			maskedData, found, _ := unstructured.NestedMap(masked.Object, "data")
			assert.True(t, found)
			assert.NotEqual(t, "cGFzc3dvcmQxMjM=", maskedData["password"])
			assert.NotEqual(t, "YWRtaW4=", maskedData["username"])
			assert.True(t, strings.Contains(maskedData["password"].(string), "+"))
			assert.True(t, strings.Contains(maskedData["username"].(string), "+"))

			maskedStringData, found, _ := unstructured.NestedMap(masked.Object, "stringData")
			assert.True(t, found)
			assert.NotEqual(t, "plain-text-config", maskedStringData["config"])
			assert.NotEqual(t, "plain-text-token", maskedStringData["token"])
			assert.True(t, strings.Contains(maskedStringData["config"].(string), "+"))
			assert.True(t, strings.Contains(maskedStringData["token"].(string), "+"))
		})
	}
}

func TestMaskValue(t *testing.T) {
	// Reset masking state before test
	ResetMaskingState()

	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{
			name:     "first value gets base mask",
			value:    "value1",
			expected: "++++++++++++++++",
		},
		{
			name:     "second value gets extended mask",
			value:    "value2",
			expected: "+++++++++++++++++",
		},
		{
			name:     "same value gets same mask",
			value:    "value1",
			expected: "++++++++++++++++",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskValue(tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMaskValueConsistency(t *testing.T) {
	// Reset masking state before test
	ResetMaskingState()

	value1 := "test-value-1"
	value2 := "test-value-2"

	// First calls
	mask1a := MaskValue(value1)
	mask2a := MaskValue(value2)

	// Second calls with same values
	mask1b := MaskValue(value1)
	mask2b := MaskValue(value2)

	// Same values should get same masks
	assert.Equal(t, mask1a, mask1b)
	assert.Equal(t, mask2a, mask2b)

	// Different values should get different masks
	assert.NotEqual(t, mask1a, mask2a)
}

func TestMaskSecretDataEdgeCases(t *testing.T) {
	// Reset masking state before test
	ResetMaskingState()

	secretWithNilValues := &unstructured.Unstructured{
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

	secretWithNumberValues := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "Secret",
			"metadata": map[string]any{
				"name":      "number-secret",
				"namespace": "default",
			},
			"type": "Opaque",
			"data": map[string]any{
				"number": "123",
				"string": "dmFsdWU=",
			},
		},
	}

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

	tests := []struct {
		name         string
		obj          *unstructured.Unstructured
		expectMasked bool
		expectNil    bool
	}{
		{
			name:         "secret with nil values",
			obj:          secretWithNilValues,
			expectMasked: false,
			expectNil:    true, // Should be rejected due to nil value
		},
		{
			name:         "secret with string number values",
			obj:          secretWithNumberValues,
			expectMasked: false,
			expectNil:    true, // Should be rejected due to invalid base64
		},
		{
			name:         "secret without data fields",
			obj:          secretWithoutData,
			expectMasked: true,
			expectNil:    false, // Should be valid (no data/stringData fields)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			masked, err := MaskSecretData(tt.obj)

			if tt.expectNil {
				assert.Error(t, err, "Expected masking to return error for invalid Secret")
				assert.Nil(t, masked, "Expected masking to return nil for invalid Secret")
			} else {
				assert.NoError(t, err, "Expected no error for valid Secret")
				assert.NotNil(t, masked, "Expected masking to return a result")
				if masked != nil {
					assert.Equal(t, "Secret", masked.GetKind(), "Expected result to be a Secret")
				}
			}
		})
	}
}

func TestResetMaskingState(t *testing.T) {
	// Add some values to the default masker state
	MaskValue("test1")
	MaskValue("test2")

	// Verify state has values by checking that we get consistent masks
	mask1a := MaskValue("test1")
	mask2a := MaskValue("test2")
	assert.NotEqual(t, mask1a, mask2a) // Different values should have different masks

	// Reset state
	ResetMaskingState()

	// Verify masking works as expected after reset (should get base mask again)
	mask1b := MaskValue("test1")
	assert.Equal(t, "++++++++++++++++", mask1b)
}

func TestMaskerInstance(t *testing.T) {
	// Test that different masker instances have independent state
	masker1 := NewMasker()
	masker2 := NewMasker()

	// Add values to first masker
	mask1a := masker1.MaskValue("value1")
	mask1b := masker1.MaskValue("value2")

	// Add same values to second masker
	mask2a := masker2.MaskValue("value1")
	mask2b := masker2.MaskValue("value2")

	// Both should start with base mask
	assert.Equal(t, "++++++++++++++++", mask1a)
	assert.Equal(t, "++++++++++++++++", mask2a)

	// Second values should be extended
	assert.Equal(t, "+++++++++++++++++", mask1b)
	assert.Equal(t, "+++++++++++++++++", mask2b)

	// Reset first masker only
	masker1.Reset()
	mask1c := masker1.MaskValue("value1")
	mask2c := masker2.MaskValue("value1")

	// First masker should reset to base, second should keep existing mapping
	assert.Equal(t, "++++++++++++++++", mask1c)
	assert.Equal(t, "++++++++++++++++", mask2c) // Should return existing mapping
}

func TestMaskSecretDataComplexStructures(t *testing.T) {
	// Reset masking state before test
	ResetMaskingState()

	tests := []struct {
		name           string
		secret         *unstructured.Unstructured
		expectedMasked bool
		checkFunc      func(t *testing.T, masked *unstructured.Unstructured)
	}{
		{
			name: "secret with nested map structures in data",
			secret: &unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "v1",
					"kind":       "Secret",
					"metadata": map[string]any{
						"name":      "nested-secret",
						"namespace": "default",
					},
					"type": "Opaque",
					"data": map[string]any{
						"config.yaml": "YXBwOgogIG5hbWU6IG15YXBwCiAgZGI6CiAgICBob3N0OiBsb2NhbGhvc3QKICAgIHBhc3N3b3JkOiBzZWNyZXQ=", // base64 encoded yaml with nested structure
						"simple":      "c2ltcGxldmFsdWU=",                                                                         // base64 encoded "simplevalue"
					},
				},
			},
			expectedMasked: true,
			checkFunc: func(t *testing.T, masked *unstructured.Unstructured) {
				maskedData, found, _ := unstructured.NestedMap(masked.Object, "data")
				assert.True(t, found)

				// Check that both keys are present and masked
				assert.Contains(t, maskedData, "config.yaml")
				assert.Contains(t, maskedData, "simple")

				// Check that values are masked (contain +)
				configValue, ok := maskedData["config.yaml"].(string)
				assert.True(t, ok)
				assert.True(t, strings.Contains(configValue, "+"))
				assert.NotEqual(t, "YXBwOgogIG5hbWU6IG15YXBwCiAgZGI6CiAgICBob3N0OiBsb2NhbGhvc3QKICAgIHBhc3N3b3JkOiBzZWNyZXQ=", configValue)

				simpleValue, ok := maskedData["simple"].(string)
				assert.True(t, ok)
				assert.True(t, strings.Contains(simpleValue, "+"))
				assert.NotEqual(t, "c2ltcGxldmFsdWU=", simpleValue)
			},
		},
		{
			name: "secret with nested map structures in stringData",
			secret: &unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "v1",
					"kind":       "Secret",
					"metadata": map[string]any{
						"name":      "nested-string-secret",
						"namespace": "default",
					},
					"type": "Opaque",
					"stringData": map[string]any{
						"config.json": `{"app":{"name":"myapp","db":{"host":"localhost","password":"secret"}}}`,
						"simple":      "simplevalue",
					},
				},
			},
			expectedMasked: true,
			checkFunc: func(t *testing.T, masked *unstructured.Unstructured) {
				maskedStringData, found, _ := unstructured.NestedMap(masked.Object, "stringData")
				assert.True(t, found)

				// Check that both keys are present and masked
				assert.Contains(t, maskedStringData, "config.json")
				assert.Contains(t, maskedStringData, "simple")

				// Check that values are masked (contain +)
				configValue, ok := maskedStringData["config.json"].(string)
				assert.True(t, ok)
				assert.True(t, strings.Contains(configValue, "+"))
				assert.NotEqual(t, `{"app":{"name":"myapp","db":{"host":"localhost","password":"secret"}}}`, configValue)

				simpleValue, ok := maskedStringData["simple"].(string)
				assert.True(t, ok)
				assert.True(t, strings.Contains(simpleValue, "+"))
				assert.NotEqual(t, "simplevalue", simpleValue)
			},
		},
		{
			name: "secret with array/list-like structured data",
			secret: &unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "v1",
					"kind":       "Secret",
					"metadata": map[string]any{
						"name":      "array-secret",
						"namespace": "default",
					},
					"type": "Opaque",
					"data": map[string]any{
						"users.yaml": "LSBuYW1lOiBhZG1pbgogIHBhc3N3b3JkOiBzZWNyZXQxCi0gbmFtZTogdXNlcgogIHBhc3N3b3JkOiBzZWNyZXQy", // base64 encoded yaml array
						"keys":       "a2V5MQprZXkyCmtleTM=",                                                                     // base64 encoded "key1\nkey2\nkey3"
					},
				},
			},
			expectedMasked: true,
			checkFunc: func(t *testing.T, masked *unstructured.Unstructured) {
				maskedData, found, _ := unstructured.NestedMap(masked.Object, "data")
				assert.True(t, found)

				// Check that both keys are present and masked
				assert.Contains(t, maskedData, "users.yaml")
				assert.Contains(t, maskedData, "keys")

				// Check that values are masked (contain +)
				usersValue, ok := maskedData["users.yaml"].(string)
				assert.True(t, ok)
				assert.True(t, strings.Contains(usersValue, "+"))
				assert.NotEqual(t, "LSBuYW1lOiBhZG1pbgogIHBhc3N3b3JkOiBzZWNyZXQxCi0gbmFtZTogdXNlcgogIHBhc3N3b3JkOiBzZWNyZXQy", usersValue)

				keysValue, ok := maskedData["keys"].(string)
				assert.True(t, ok)
				assert.True(t, strings.Contains(keysValue, "+"))
				assert.NotEqual(t, "a2V5MQprZXkyCmtleTM=", keysValue)
			},
		},
		{
			name: "secret with deeply nested object structure in stringData",
			secret: &unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "v1",
					"kind":       "Secret",
					"metadata": map[string]any{
						"name":      "deep-nested-secret",
						"namespace": "default",
					},
					"type": "Opaque",
					"stringData": map[string]any{
						"hofe.fuge": "test",
						"deep.config": `{
							"level1": {
								"level2": {
									"level3": {
										"secret": "very-secret-value",
										"password": "another-secret"
									}
								}
							}
						}`,
					},
				},
			},
			expectedMasked: true,
			checkFunc: func(t *testing.T, masked *unstructured.Unstructured) {
				maskedStringData, found, _ := unstructured.NestedMap(masked.Object, "stringData")
				assert.True(t, found)

				// Check that both keys are present and masked
				assert.Contains(t, maskedStringData, "hofe.fuge")
				assert.Contains(t, maskedStringData, "deep.config")

				// Check that values are masked (contain +)
				hofeValue, ok := maskedStringData["hofe.fuge"].(string)
				assert.True(t, ok)
				assert.True(t, strings.Contains(hofeValue, "+"))
				assert.NotEqual(t, "test", hofeValue)

				deepValue, ok := maskedStringData["deep.config"].(string)
				assert.True(t, ok)
				assert.True(t, strings.Contains(deepValue, "+"))
				// The entire JSON string should be masked as one unit
				assert.NotContains(t, deepValue, "very-secret-value")
				assert.NotContains(t, deepValue, "another-secret")
			},
		},
		{
			name: "secret with mixed complex structures",
			secret: &unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "v1",
					"kind":       "Secret",
					"metadata": map[string]any{
						"name":      "mixed-secret",
						"namespace": "default",
					},
					"type": "Opaque",
					"data": map[string]any{
						"certificates": "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCmNlcnQgZGF0YSBoZXJlCi0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0=", // base64 cert
					},
					"stringData": map[string]any{
						"config.yaml": `
apiVersion: v1
data:
  database:
    host: db.example.com
    port: 5432
    credentials:
      username: dbuser
      password: dbpass123
  cache:
    redis:
      host: redis.example.com
      password: redispass456
`,
						"env.list": "DB_HOST=localhost\nDB_PASS=secret123\nREDIS_URL=redis://user:pass@localhost:6379",
					},
				},
			},
			expectedMasked: true,
			checkFunc: func(t *testing.T, masked *unstructured.Unstructured) {
				// Check data field
				maskedData, found, _ := unstructured.NestedMap(masked.Object, "data")
				assert.True(t, found)
				assert.Contains(t, maskedData, "certificates")

				certValue, ok := maskedData["certificates"].(string)
				assert.True(t, ok)
				assert.True(t, strings.Contains(certValue, "+"))
				assert.NotEqual(t, "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCmNlcnQgZGF0YSBoZXJlCi0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0=", certValue)

				// Check stringData field
				maskedStringData, found, _ := unstructured.NestedMap(masked.Object, "stringData")
				assert.True(t, found)
				assert.Contains(t, maskedStringData, "config.yaml")
				assert.Contains(t, maskedStringData, "env.list")

				configValue, ok := maskedStringData["config.yaml"].(string)
				assert.True(t, ok)
				assert.True(t, strings.Contains(configValue, "+"))
				assert.NotContains(t, configValue, "dbpass123")
				assert.NotContains(t, configValue, "redispass456")

				envValue, ok := maskedStringData["env.list"].(string)
				assert.True(t, ok)
				assert.True(t, strings.Contains(envValue, "+"))
				assert.NotContains(t, envValue, "secret123")
				assert.NotContains(t, envValue, "pass@localhost")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset masking state for each test to ensure consistent behavior
			ResetMaskingState()

			masked, err := MaskSecretData(tt.secret)
			assert.NoError(t, err)
			assert.NotNil(t, masked)
			assert.Equal(t, "Secret", masked.GetKind())

			if tt.expectedMasked {
				// Ensure original is not modified
				assert.NotEqual(t, tt.secret.Object, masked.Object, "Original secret should not be modified")

				// Run custom checks
				if tt.checkFunc != nil {
					tt.checkFunc(t, masked)
				}
			}
		})
	}
}

func TestMaskSecretDataActualNestedStructures(t *testing.T) {
	// Reset masking state before test
	ResetMaskingState()

	tests := []struct {
		name           string
		secret         *unstructured.Unstructured
		expectedMasked bool
		checkFunc      func(t *testing.T, original, masked *unstructured.Unstructured)
	}{
		{
			name: "secret with only string values in data (realistic K8s Secret)",
			secret: &unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "v1",
					"kind":       "Secret",
					"metadata": map[string]any{
						"name":      "realistic-secret",
						"namespace": "default",
					},
					"type": "Opaque",
					"data": map[string]any{
						"stringValue":    "c2VjcmV0VmFsdWU=",                                                             // base64 "secretValue"
						"config.json":    "eyJhcHAiOnsibmFtZSI6Im15YXBwIiwiZGIiOnsiaG9zdCI6ImRiLmV4YW1wbGUuY29tIn19fQ==", // base64 JSON
						"users.yaml":     "LSBuYW1lOiBhZG1pbgogIC0gbmFtZTogdXNlcg==",                                     // base64 YAML array
						"keys":           "a2V5MQprZXkyCmtleTM=",                                                         // base64 multiline
						"empty":          "",                                                                             // empty string
						"hofe.fuge.test": "dGVzdFZhbHVl",                                                                 // base64 "testValue" with dotted key
					},
				},
			},
			expectedMasked: true,
			checkFunc: func(t *testing.T, _, masked *unstructured.Unstructured) {
				maskedData, found, _ := unstructured.NestedMap(masked.Object, "data")
				assert.True(t, found)

				// All non-empty string values should be masked
				testCases := []struct {
					key            string
					originalVal    string
					shouldBeMasked bool
				}{
					{"stringValue", "c2VjcmV0VmFsdWU=", true},
					{"config.json", "eyJhcHAiOnsibmFtZSI6Im15YXBwIiwiZGIiOnsiaG9zdCI6ImRiLmV4YW1wbGUuY29tIn19fQ==", true},
					{"users.yaml", "LSBuYW1lOiBhZG1pbgogIC0gbmFtZTogdXNlcg==", true},
					{"keys", "a2V5MQprZXkyCmtleTM=", true}, // gitleaks:allow
					{"hofe.fuge.test", "dGVzdFZhbHVl", true},
				}

				for _, tc := range testCases {
					assert.Contains(t, maskedData, tc.key, "Key %s should exist in masked data", tc.key)
					maskedValue, ok := maskedData[tc.key].(string)
					assert.True(t, ok, "Value for key %s should be string", tc.key)

					if tc.shouldBeMasked {
						assert.True(t, strings.Contains(maskedValue, "+"), "Value for key %s should be masked", tc.key)
						assert.NotEqual(t, tc.originalVal, maskedValue, "Value for key %s should be different from original", tc.key)
					}
				}

				// Empty string handling - this might be masked or preserved depending on implementation
				if emptyValue, exists := maskedData["empty"]; exists {
					t.Logf("Empty string value in masked data: %v (type: %T)", emptyValue, emptyValue)
				}
			},
		},
		{
			name: "secret with complex string data structures in stringData",
			secret: &unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "v1",
					"kind":       "Secret",
					"metadata": map[string]any{
						"name":      "complex-stringdata-secret",
						"namespace": "default",
					},
					"type": "Opaque",
					"stringData": map[string]any{
						"stringValue": "plainTextSecret",
						// JSON structure as string
						"config.json": `{"database":{"host":"db.example.com","password":"super-secret-password"},"api":{"token":"api-token-value","key":"api-key-value"}}`,
						// YAML structure as string
						"config.yaml": `
database:
  host: db.example.com
  password: super-secret-password
api:
  token: api-token-value
  key: api-key-value`,
						// Environment variables as string
						"env": "DB_HOST=localhost\nDB_PASS=secret123\nAPI_TOKEN=token456",
						// Certificate as string
						"tls.crt": "-----BEGIN CERTIFICATE-----\nMIICertificateDataHere\n-----END CERTIFICATE-----",
						// Private key as string
						"tls.key": "-----BEGIN PRIVATE KEY-----\nMIIPrivateKeyDataHere\n-----END PRIVATE KEY-----",
						"empty":   "",
					},
				},
			},
			expectedMasked: true,
			checkFunc: func(t *testing.T, original, masked *unstructured.Unstructured) {
				originalStringData, found, _ := unstructured.NestedMap(original.Object, "stringData")
				assert.True(t, found)
				maskedStringData, found, _ := unstructured.NestedMap(masked.Object, "stringData")
				assert.True(t, found)

				// All string values should be masked
				testCases := []struct {
					key        string
					shouldMask bool
				}{
					{"stringValue", true},
					{"config.json", true},
					{"config.yaml", true},
					{"env", true},
					{"tls.crt", true},
					{"tls.key", true},
				}

				for _, tc := range testCases {
					assert.Contains(t, maskedStringData, tc.key, "Key %s should exist in masked data", tc.key)
					maskedValue, ok := maskedStringData[tc.key].(string)
					assert.True(t, ok, "Value for key %s should be string", tc.key)

					if tc.shouldMask {
						originalValue := originalStringData[tc.key].(string)
						assert.True(t, strings.Contains(maskedValue, "+"), "Value for key %s should be masked", tc.key)
						assert.NotEqual(t, originalValue, maskedValue, "Value for key %s should be different from original", tc.key)

						// Ensure sensitive data is not leaked
						if tc.key == "config.json" {
							assert.NotContains(t, maskedValue, "super-secret-password")
							assert.NotContains(t, maskedValue, "api-token-value")
						}
						if tc.key == "config.yaml" {
							assert.NotContains(t, maskedValue, "super-secret-password")
							assert.NotContains(t, maskedValue, "api-token-value")
						}
						if tc.key == "env" {
							assert.NotContains(t, maskedValue, "secret123")
							assert.NotContains(t, maskedValue, "token456")
						}
					}
				}

				// Empty string handling
				if emptyValue, exists := maskedStringData["empty"]; exists {
					t.Logf("Empty string value in masked data: %v (type: %T)", emptyValue, emptyValue)
				}
			},
		},
		{
			name: "secret with edge cases - empty strings and special characters",
			secret: &unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "v1",
					"kind":       "Secret",
					"metadata": map[string]any{
						"name":      "edge-case-secret",
						"namespace": "default",
					},
					"type": "Opaque",
					"data": map[string]any{
						"validString":    "dmFsaWRTdHJpbmc=", // base64 "validString"
						"emptyString":    "",
						"whitespaceOnly": "ICAgIA==",                                                                                                             // base64 "    " (spaces)
						"newlines":       "bGluZTEKbGluZTIKbGluZTM=",                                                                                             // base64 "line1\nline2\nline3"
						"specialChars":   "ISQlJiooKQ==",                                                                                                         // base64 "!$%&*()"
						"unicode":        "8J+YgPCfkZE=",                                                                                                         // base64 "😀👑" (emoji)
						"veryLong":       "YWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWE=", // base64 long string
					},
					"stringData": map[string]any{
						"plainString":    "plainSecretValue",
						"emptyPlain":     "",
						"jsonString":     `{"nested":{"deeply":{"nested":{"value":"secret123"}}}}`,
						"xmlString":      `<config><database><password>secret456</password></database></config>`,
						"base64InString": "dGhpcyBpcyBub3QgYmFzZTY0IGVuY29kZWQ=", // This is actually base64 but in stringData
						"sqlQuery":       "SELECT * FROM users WHERE password = 'secret789' AND api_key = 'key123'",
					},
				},
			},
			expectedMasked: true,
			checkFunc: func(t *testing.T, original, masked *unstructured.Unstructured) {
				// Check data field
				originalData, found, _ := unstructured.NestedMap(original.Object, "data")
				assert.True(t, found)
				maskedData, found, _ := unstructured.NestedMap(masked.Object, "data")
				assert.True(t, found)

				// Test cases for data field
				dataTests := []struct {
					key        string
					shouldMask bool
				}{
					{"validString", true},
					{"whitespaceOnly", true},
					{"newlines", true},
					{"specialChars", true},
					{"unicode", true},
					{"veryLong", true},
				}

				for _, tc := range dataTests {
					if originalVal, exists := originalData[tc.key]; exists {
						assert.Contains(t, maskedData, tc.key, "Key %s should exist in masked data", tc.key)

						if maskedVal, ok := maskedData[tc.key].(string); ok && tc.shouldMask {
							if originalValStr, ok := originalVal.(string); ok && originalValStr != "" {
								assert.True(t, strings.Contains(maskedVal, "+"), "Value for key %s should be masked", tc.key)
								assert.NotEqual(t, originalValStr, maskedVal, "Value for key %s should be different from original", tc.key)
							}
						}
					}
				}

				// Empty string handling
				if emptyString, exists := maskedData["emptyString"]; exists {
					t.Logf("Empty string in data field: %v (type: %T)", emptyString, emptyString)
				}

				// Check stringData field
				originalStringData, found, _ := unstructured.NestedMap(original.Object, "stringData")
				assert.True(t, found)
				maskedStringData, found, _ := unstructured.NestedMap(masked.Object, "stringData")
				assert.True(t, found)

				// Test cases for stringData field
				stringDataTests := []struct {
					key         string
					shouldMask  bool
					secretWords []string // Words that should not appear in masked value
				}{
					{"plainString", true, []string{"plainSecretValue"}},
					{"jsonString", true, []string{"secret123"}},
					{"xmlString", true, []string{"secret456"}},
					{"base64InString", true, []string{"dGhpcyBpcyBub3QgYmFzZTY0IGVuY29kZWQ="}},
					{"sqlQuery", true, []string{"secret789", "key123"}},
				}

				for _, tc := range stringDataTests {
					if originalVal, exists := originalStringData[tc.key]; exists {
						assert.Contains(t, maskedStringData, tc.key, "Key %s should exist in masked stringData", tc.key)

						if maskedVal, ok := maskedStringData[tc.key].(string); ok && tc.shouldMask {
							if originalValStr, ok := originalVal.(string); ok && originalValStr != "" {
								assert.True(t, strings.Contains(maskedVal, "+"), "Value for key %s should be masked", tc.key)
								assert.NotEqual(t, originalValStr, maskedVal, "Value for key %s should be different from original", tc.key)

								// Check that secrets don't leak
								for _, secretWord := range tc.secretWords {
									assert.NotContains(t, maskedVal, secretWord, "Masked value for key %s should not contain secret word %s", tc.key, secretWord)
								}
							}
						}
					}
				}

				// Empty string handling in stringData
				if emptyPlain, exists := maskedStringData["emptyPlain"]; exists {
					t.Logf("Empty string in stringData field: %v (type: %T)", emptyPlain, emptyPlain)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset masking state for each test
			ResetMaskingState()

			// Keep reference to original for comparison
			originalCopy := tt.secret.DeepCopy()

			masked, err := MaskSecretData(tt.secret)
			assert.NoError(t, err)
			assert.NotNil(t, masked)
			assert.Equal(t, "Secret", masked.GetKind())

			if tt.expectedMasked {
				// Ensure original is not modified
				assert.Equal(t, tt.secret.Object, originalCopy.Object, "Original secret should not be modified")

				// Run custom checks
				if tt.checkFunc != nil {
					tt.checkFunc(t, tt.secret, masked)
				}
			}
		})
	}
}

func TestMaskSecretDataEdgeCasesAndConsistency(t *testing.T) {
	// Reset masking state before test
	ResetMaskingState()

	// Test that same values across different secrets get same masks
	secret1 := &unstructured.Unstructured{
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
	}

	secret2 := &unstructured.Unstructured{
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
	}

	// Mask both secrets
	masked1, err1 := MaskSecretData(secret1)
	assert.NoError(t, err1)
	masked2, err2 := MaskSecretData(secret2)
	assert.NoError(t, err2)

	// Get masked data
	maskedData1, found, _ := unstructured.NestedMap(masked1.Object, "data")
	assert.True(t, found)
	maskedData2, found, _ := unstructured.NestedMap(masked2.Object, "data")
	assert.True(t, found)

	// Same values should get same masks
	assert.Equal(t, maskedData1["shared"], maskedData2["shared"], "Same values should get identical masks")

	// Different values should get different masks
	assert.NotEqual(t, maskedData1["unique"], maskedData2["unique"], "Different values should get different masks")

	// All values should be masked (contain +)
	for key, value := range maskedData1 {
		if strValue, ok := value.(string); ok {
			assert.True(t, strings.Contains(strValue, "+"), "Value for key %s should be masked", key)
		}
	}

	for key, value := range maskedData2 {
		if strValue, ok := value.(string); ok {
			assert.True(t, strings.Contains(strValue, "+"), "Value for key %s should be masked", key)
		}
	}
}

// TestMaskSecretDataWithInvalidFormats tests that invalid Secret formats are properly handled
// This test demonstrates the vulnerability where non-string values are silently skipped
func TestMaskSecretDataWithInvalidFormats(t *testing.T) {
	// Reset masking state before test
	ResetMaskingState()

	tests := []struct {
		name        string
		secret      *unstructured.Unstructured
		expectError bool
		description string
	}{
		{
			name: "secret with nested map in data field",
			secret: &unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "v1",
					"kind":       "Secret",
					"metadata": map[string]any{
						"name":      "invalid-nested-secret",
						"namespace": "default",
					},
					"type": "Opaque",
					"data": map[string]any{
						"config": map[string]any{
							"username": "admin",
							"password": "secret123",
						},
						"validKey": "dmFsaWRWYWx1ZQ==", // gitleaks:allow
					},
				},
			},
			expectError: true,
			description: "Nested maps in data field should be rejected",
		},
		{
			name: "secret with integer values",
			secret: &unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "v1",
					"kind":       "Secret",
					"metadata": map[string]any{
						"name":      "invalid-int-secret",
						"namespace": "default",
					},
					"type": "Opaque",
					"data": map[string]any{
						"port":     1234,
						"validKey": "dmFsaWRWYWx1ZQ==", // gitleaks:allow
					},
					"stringData": map[string]any{
						"timeout":  300,
						"validKey": "validValue",
					},
				},
			},
			expectError: true,
			description: "Integer values should be rejected",
		},
		{
			name: "valid secret should pass",
			secret: &unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "v1",
					"kind":       "Secret",
					"metadata": map[string]any{
						"name":      "valid-secret",
						"namespace": "default",
					},
					"type": "Opaque",
					"data": map[string]any{
						"username": "YWRtaW4=",
						"password": "cGFzc3dvcmQ=",
						"config":   "Y29uZmlnZGF0YQ==",
					},
					"stringData": map[string]any{
						"token":  "plainTextToken",
						"apiKey": "plainTextApiKey",
					},
				},
			},
			expectError: false,
			description: "Valid Secret with only string values should pass",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// With validation implemented, invalid formats should now be rejected
			masked, err := MaskSecretData(tt.secret)

			if tt.expectError {
				assert.Error(t, err, tt.description)
				assert.Nil(t, masked)
				t.Logf("Validation correctly rejected: %s", tt.description)
			} else {
				assert.NoError(t, err, tt.description)
				assert.NotNil(t, masked, "Valid Secret should be masked")
				assert.Equal(t, "Secret", masked.GetKind(), "Masked object should still be a Secret")
				t.Logf("Validation correctly accepted: %s", tt.description)
			}
		})
	}
}

// TestSecretValidation tests the ValidateSecret function directly
func TestSecretValidation(t *testing.T) {
	tests := []struct {
		name        string
		secret      *unstructured.Unstructured
		expectError bool
		errorText   string
	}{
		{
			name:        "nil secret",
			secret:      nil,
			expectError: true,
			errorText:   "secret object is nil",
		},
		{
			name: "non-secret object",
			secret: &unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "v1",
					"kind":       "ConfigMap",
					"metadata": map[string]any{
						"name":      "test-config",
						"namespace": "default",
					},
				},
			},
			expectError: true,
			errorText:   "object is not a Secret",
		},
		{
			name: "valid secret",
			secret: &unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "v1",
					"kind":       "Secret",
					"metadata": map[string]any{
						"name":      "valid-secret",
						"namespace": "default",
					},
					"type": "Opaque",
					"data": map[string]any{
						"username": "YWRtaW4=",
						"password": "cGFzc3dvcmQ=",
					},
					"stringData": map[string]any{
						"token": "plainTextToken",
					},
				},
			},
			expectError: false,
		},
		{
			name: "secret with nested map in data",
			secret: &unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "v1",
					"kind":       "Secret",
					"metadata": map[string]any{
						"name":      "invalid-secret",
						"namespace": "default",
					},
					"type": "Opaque",
					"data": map[string]any{
						"config": map[string]any{
							"username": "admin",
							"password": "secret",
						},
					},
				},
			},
			expectError: true,
			errorText:   "non-string value of type map[string]interface {}",
		},
		{
			name: "secret with integer value",
			secret: &unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "v1",
					"kind":       "Secret",
					"metadata": map[string]any{
						"name":      "invalid-secret",
						"namespace": "default",
					},
					"type": "Opaque",
					"data": map[string]any{
						"port": 8080,
					},
				},
			},
			expectError: true,
			errorText:   "invalid Secret structure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSecret(tt.secret)

			if tt.expectError {
				assert.Error(t, err, "Expected validation to fail")
				if tt.errorText != "" {
					assert.Contains(t, err.Error(), tt.errorText, "Error message should contain expected text")
				}
			} else {
				assert.NoError(t, err, "Expected validation to pass")
			}
		})
	}
}
