// Package masking provides functionality for masking Kubernetes manifests.
package masking

import (
	"fmt"
	"os"
	"sync"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// Masker manages secret masking state and provides consistent value masking
type Masker struct {
	mu                 sync.RWMutex
	valueToReplacement map[string]string
	currentReplacement string
}

// NewMasker creates a new Masker instance with fresh state
func NewMasker() *Masker {
	return &Masker{
		valueToReplacement: make(map[string]string),
		currentReplacement: "++++++++++++++++",
	}
}

// Global default masker for backward compatibility
var defaultMasker = NewMasker()

// IsSecret checks if the unstructured object is a Secret
func IsSecret(obj *unstructured.Unstructured) bool {
	return obj != nil && obj.GetKind() == "Secret"
}

// ValidateSecret validates that the Secret object conforms to Kubernetes Secret specification
// It ensures that both 'data' and 'stringData' fields contain only string values as required by K8s API
func ValidateSecret(obj *unstructured.Unstructured) (err error) {
	if obj == nil {
		return fmt.Errorf("secret object is nil")
	}

	if !IsSecret(obj) {
		return fmt.Errorf("object is not a Secret, got kind: %s", obj.GetKind())
	}

	// Get Secret name and namespace for better error messages
	secretName := obj.GetName()
	secretNamespace := obj.GetNamespace()
	secretIdentifier := fmt.Sprintf("%s/%s", secretNamespace, secretName)
	if secretNamespace == "" {
		secretIdentifier = secretName
	}
	if secretIdentifier == "" {
		secretIdentifier = "unnamed"
	}

	// Recover from panics that may occur when processing invalid structures
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("invalid Secret structure for %s: %v", secretIdentifier, r)
		}
	}()

	// Validate that data field contains only string values
	if dataMap, found, err := unstructured.NestedMap(obj.Object, "data"); err != nil {
		return fmt.Errorf("invalid data field structure for Secret %s: %w", secretIdentifier, err)
	} else if found {
		for key, value := range dataMap {
			if _, ok := value.(string); !ok {
				return fmt.Errorf("invalid data field for Secret %s: key '%s' has non-string value of type %T", secretIdentifier, key, value)
			}
		}
	}

	// Validate that stringData field contains only string values
	if stringDataMap, found, err := unstructured.NestedMap(obj.Object, "stringData"); err != nil {
		return fmt.Errorf("invalid stringData field structure for Secret %s: %w", secretIdentifier, err)
	} else if found {
		for key, value := range stringDataMap {
			if _, ok := value.(string); !ok {
				return fmt.Errorf("invalid stringData field for Secret %s: key '%s' has non-string value of type %T", secretIdentifier, key, value)
			}
		}
	}

	// Additional validation: try to convert to structured Secret to catch other issues
	// This uses a simpler approach that doesn't rely on encoding/decoding
	secret := &corev1.Secret{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, secret); err != nil {
		return fmt.Errorf("failed to convert Secret %s to structured format: %w", secretIdentifier, err)
	}

	return nil
}

// MaskSecretData creates a masked copy of the Secret object using the Masker instance
func (m *Masker) MaskSecretData(obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	if obj == nil || !IsSecret(obj) {
		return obj, nil
	}

	// Validate the Secret structure before processing to prevent masking leakage
	if err := ValidateSecret(obj); err != nil {
		return nil, fmt.Errorf("secret validation failed: %w", err)
	}

	// Create a deep copy to avoid modifying the original
	masked := obj.DeepCopy()

	// Process data field (base64 encoded values)
	if dataMap, found, _ := unstructured.NestedMap(masked.Object, "data"); found {
		for key, value := range dataMap {
			if strValue, ok := value.(string); ok {
				// Mask each value uniquely but consistently
				maskedValue := m.MaskValue(strValue)
				dataMap[key] = maskedValue
			}
		}
		if err := unstructured.SetNestedMap(masked.Object, dataMap, "data"); err != nil {
			// Log error but continue processing
			fmt.Fprintf(os.Stderr, "Warning: failed to set nested map for data field: %v\n", err)
		}
	}

	// Process stringData field (plain text values)
	if stringDataMap, found, _ := unstructured.NestedMap(masked.Object, "stringData"); found {
		for key, value := range stringDataMap {
			if strValue, ok := value.(string); ok {
				// Mask plain text values directly
				maskedValue := m.MaskValue(strValue)
				stringDataMap[key] = maskedValue
			}
		}
		if err := unstructured.SetNestedMap(masked.Object, stringDataMap, "stringData"); err != nil {
			// Log error but continue processing
			fmt.Fprintf(os.Stderr, "Warning: failed to set nested map for stringData field: %v\n", err)
		}
	}

	return masked, nil
}

// MaskSecretData creates a masked copy of the Secret object using the default masker
// Implementation based on ArgoCD gitops-engine's secret masking approach:
// https://github.com/argoproj/gitops-engine/blob/v0.6.2/pkg/diff/diff.go
func MaskSecretData(obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	return defaultMasker.MaskSecretData(obj)
}

// MaskValue returns a consistent mask for the same input value using the Masker instance
// Same values get identical masks, different values get different length masks
func (m *Masker) MaskValue(value string) string {
	m.mu.RLock()
	if replacement, exists := m.valueToReplacement[value]; exists {
		m.mu.RUnlock()
		return replacement
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if replacement, exists := m.valueToReplacement[value]; exists {
		return replacement
	}

	// Create new replacement for this value
	currentReplacement := m.currentReplacement
	m.valueToReplacement[value] = currentReplacement
	m.currentReplacement = m.currentReplacement + "+"

	return currentReplacement
}

// Reset resets the masking state for this Masker instance
func (m *Masker) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.valueToReplacement = make(map[string]string)
	m.currentReplacement = "++++++++++++++++"
}

// MaskValue returns a consistent mask for the same input value using the default masker
// Same values get identical masks, different values get different length masks
func MaskValue(value string) string {
	return defaultMasker.MaskValue(value)
}

// ResetMaskingState resets the default masker's state.
// This is useful for testing or when you want to start fresh with masking.
func ResetMaskingState() {
	defaultMasker.Reset()
}
