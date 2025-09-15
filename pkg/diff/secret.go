package diff

import (
	"fmt"
	"os"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Global value mapping for consistent masking across different secrets
// Uses the same approach as gitops-engine with "+" symbols of varying lengths
var globalValueToReplacement = make(map[string]string)
var globalReplacement = "++++++++++++++++"

// isSecret checks if the unstructured object is a Secret
func isSecret(obj *unstructured.Unstructured) bool {
	return obj != nil && obj.GetKind() == "Secret"
}

// maskSecretData creates a masked copy of the Secret object with shared value mapping
// Implementation based on ArgoCD gitops-engine's secret masking approach:
// https://github.com/argoproj/gitops-engine/blob/v0.6.2/pkg/diff/diff.go
func maskSecretData(obj *unstructured.Unstructured) *unstructured.Unstructured {
	if obj == nil || !isSecret(obj) {
		return obj
	}

	// Create a deep copy to avoid modifying the original
	masked := obj.DeepCopy()

	// Process data field (base64 encoded values)
	if dataMap, found, _ := unstructured.NestedMap(masked.Object, "data"); found {
		for key, value := range dataMap {
			if strValue, ok := value.(string); ok {
				// Mask each value uniquely but consistently
				maskedValue := maskValue(strValue)
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
				maskedValue := maskValue(strValue)
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

// maskValue returns a consistent mask for the same input value
// Same values get identical masks, different values get different length masks
func maskValue(value string) string {
	if replacement, exists := globalValueToReplacement[value]; exists {
		return replacement
	}

	// Create new replacement for this value
	currentReplacement := globalReplacement
	globalValueToReplacement[value] = currentReplacement
	globalReplacement = globalReplacement + "+"

	return currentReplacement
}
