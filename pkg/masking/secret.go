// Package masking provides functionality for masking sensitive data in Kubernetes secrets.
package masking

import (
	"fmt"
	"os"
	"sync"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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

// MaskSecretData creates a masked copy of the Secret object using the Masker instance
func (m *Masker) MaskSecretData(obj *unstructured.Unstructured) *unstructured.Unstructured {
	if obj == nil || !IsSecret(obj) {
		return obj
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

	return masked
}

// MaskSecretData creates a masked copy of the Secret object using the default masker
// Implementation based on ArgoCD gitops-engine's secret masking approach:
// https://github.com/argoproj/gitops-engine/blob/v0.6.2/pkg/diff/diff.go
func MaskSecretData(obj *unstructured.Unstructured) *unstructured.Unstructured {
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
