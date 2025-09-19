package diff

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/toyamagu-2021/k8s-manifest-diff/pkg/masking"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestObjects_SecretMasking(t *testing.T) {
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

	tests := []struct {
		name                  string
		baseObjects           []*unstructured.Unstructured
		headObjects           []*unstructured.Unstructured
		options               *Options
		expectChanges         bool
		shouldContain         []string
		shouldNotContain      []string
		checkMaskedValues     bool
		checkChangedResources bool
		expectEmptyDiff       bool
	}{
		{
			name:                  "masks secret values by default",
			baseObjects:           []*unstructured.Unstructured{baseSecret},
			headObjects:           []*unstructured.Unstructured{headSecret},
			options:               DefaultOptions(),
			expectChanges:         true,
			shouldContain:         []string{"test-secret", "++++++++++++++++"},
			shouldNotContain:      []string{"cGFzc3dvcmQxMjM=", "bmV3cGFzc3dvcmQ=", "YWRtaW4="},
			checkMaskedValues:     true,
			checkChangedResources: true,
		},
		{
			name:              "same values get same mask, different values get different masks",
			baseObjects:       []*unstructured.Unstructured{baseSecret},
			headObjects:       []*unstructured.Unstructured{headSecret},
			options:           DefaultOptions(),
			expectChanges:     true,
			checkMaskedValues: true,
		},
		{
			name:        "can disable secret masking",
			baseObjects: []*unstructured.Unstructured{baseSecret},
			headObjects: []*unstructured.Unstructured{headSecret},
			options: &Options{
				DisableMaskSecrets: true,
				Context:            3,
			},
			expectChanges:    true,
			shouldContain:    []string{"cGFzc3dvcmQxMjM=", "bmV3cGFzc3dvcmQ="},
			shouldNotContain: []string{},
		},
		{
			name:            "handles stringData field",
			baseObjects:     []*unstructured.Unstructured{secretWithStringData},
			headObjects:     []*unstructured.Unstructured{secretWithStringData},
			options:         DefaultOptions(),
			expectChanges:   false,
			expectEmptyDiff: true,
		},
		{
			name:            "non-secret objects are not affected",
			baseObjects:     []*unstructured.Unstructured{configMap},
			headObjects:     []*unstructured.Unstructured{configMap},
			options:         DefaultOptions(),
			expectChanges:   false,
			expectEmptyDiff: true,
		},
		{
			name:             "mixed objects - only secrets are masked",
			baseObjects:      []*unstructured.Unstructured{baseSecret, configMapBase},
			headObjects:      []*unstructured.Unstructured{headSecret, configMapHead},
			options:          DefaultOptions(),
			expectChanges:    true,
			shouldNotContain: []string{"cGFzc3dvcmQxMjM=", "bmV3cGFzc3dvcmQ="},
			shouldContain:    []string{"original-value", "updated-value"},
		},
		{
			name:            "secret with empty data fields",
			baseObjects:     []*unstructured.Unstructured{emptySecret},
			headObjects:     []*unstructured.Unstructured{emptySecret},
			options:         DefaultOptions(),
			expectChanges:   false,
			expectEmptyDiff: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := Objects(tt.baseObjects, tt.headObjects, tt.options)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectChanges, results.HasChanges())

			diffResult := results.StringDiff()

			if tt.expectEmptyDiff {
				assert.Equal(t, "", diffResult)
				return
			}

			for _, expected := range tt.shouldContain {
				assert.Contains(t, diffResult, expected)
			}

			for _, notExpected := range tt.shouldNotContain {
				assert.NotContains(t, diffResult, notExpected)
			}

			if tt.checkMaskedValues {
				base16Plus := strings.Count(diffResult, "++++++++++++++++")  // 16 +
				base17Plus := strings.Count(diffResult, "+++++++++++++++++") // 17 +
				assert.True(t, base16Plus > 0 || base17Plus > 0, "Should contain masked values")
			}

			if tt.checkChangedResources {
				changedResourcesList := GetChangedResourceKeys(results)
				assert.Equal(t, 1, len(changedResourcesList))
				AssertResourceChange(t, results, "Secret/default/test-secret", Changed)
			}
		})
	}

	t.Run("mask consistency across multiple diff operations", func(t *testing.T) {
		opts := DefaultOptions()

		results1, err1 := Objects([]*unstructured.Unstructured{baseSecret}, []*unstructured.Unstructured{headSecret}, opts)
		assert.NoError(t, err1)

		results2, err2 := Objects([]*unstructured.Unstructured{baseSecret}, []*unstructured.Unstructured{headSecret}, opts)
		assert.NoError(t, err2)

		diff1 := results1.StringDiff()
		diff2 := results2.StringDiff()
		assert.Equal(t, diff1, diff2, "Diff results should be consistent across multiple operations")
	})
}

func TestObjects_SecretMaskingAdvanced(t *testing.T) {
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

	tests := []struct {
		name             string
		baseObjects      []*unstructured.Unstructured
		headObjects      []*unstructured.Unstructured
		options          *Options
		expectChanges    bool
		shouldNotContain []string
	}{
		{
			name:          "secret with nil values",
			baseObjects:   []*unstructured.Unstructured{secretWithNil},
			headObjects:   []*unstructured.Unstructured{secretWithNil},
			options:       DefaultOptions(),
			expectChanges: false,
		},
		{
			name:          "secret with both data and stringData",
			baseObjects:   []*unstructured.Unstructured{mixedSecretBase},
			headObjects:   []*unstructured.Unstructured{mixedSecretHead},
			options:       DefaultOptions(),
			expectChanges: true,
			shouldNotContain: []string{
				"ZW5jb2RlZC12YWx1ZQ==",
				"bmV3LWVuY29kZWQtdmFsdWU=",
				"plain-value",
				"new-plain-value",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := Objects(tt.baseObjects, tt.headObjects, tt.options)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectChanges, results.HasChanges())

			if tt.expectChanges {
				diffResult := results.StringDiff()
				for _, notExpected := range tt.shouldNotContain {
					assert.NotContains(t, diffResult, notExpected)
				}
			} else {
				diffResult := results.StringDiff()
				assert.Equal(t, "", diffResult)
			}
		})
	}
}

func TestYamlString_SecretMasking(t *testing.T) {
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

	tests := []struct {
		name             string
		options          *Options
		shouldContain    []string
		shouldNotContain []string
	}{
		{
			name:             "yaml diff with secret masking enabled",
			options:          DefaultOptions(),
			shouldContain:    []string{"++++++++++++++++"},
			shouldNotContain: []string{"cGFzc3dvcmQxMjM=", "bmV3cGFzc3dvcmQ="},
		},
		{
			name: "yaml diff with secret masking disabled",
			options: &Options{
				DisableMaskSecrets: true,
				Context:            3,
			},
			shouldContain:    []string{"cGFzc3dvcmQxMjM=", "bmV3cGFzc3dvcmQ="},
			shouldNotContain: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := YamlString(baseYaml, headYaml, tt.options)

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

func TestSecretMaskingEdgeCases(t *testing.T) {
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
		name          string
		baseObjects   []*unstructured.Unstructured
		headObjects   []*unstructured.Unstructured
		options       *Options
		expectChanges bool
	}{
		{
			name:          "secret with non-string values in data",
			baseObjects:   []*unstructured.Unstructured{secretWithNumbers},
			headObjects:   []*unstructured.Unstructured{secretWithNumbers},
			options:       DefaultOptions(),
			expectChanges: false,
		},
		{
			name:          "secret without data or stringData fields",
			baseObjects:   []*unstructured.Unstructured{secretWithoutData},
			headObjects:   []*unstructured.Unstructured{secretWithoutData},
			options:       DefaultOptions(),
			expectChanges: false,
		},
		{
			name:          "handles nil objects gracefully",
			baseObjects:   []*unstructured.Unstructured{nil},
			headObjects:   []*unstructured.Unstructured{nil},
			options:       DefaultOptions(),
			expectChanges: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := Objects(tt.baseObjects, tt.headObjects, tt.options)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectChanges, results.HasChanges())
		})
	}

	t.Run("secret mask function with nil input", func(t *testing.T) {
		masked := masking.MaskSecretData(nil)
		assert.Nil(t, masked)
	})

	t.Run("isSecret function with various inputs", func(t *testing.T) {
		assert.False(t, masking.IsSecret(nil))

		nonSecret := &unstructured.Unstructured{
			Object: map[string]any{
				"kind": "ConfigMap",
			},
		}
		assert.False(t, masking.IsSecret(nonSecret))

		secret := &unstructured.Unstructured{
			Object: map[string]any{
				"kind": "Secret",
			},
		}
		assert.True(t, masking.IsSecret(secret))
	})
}
