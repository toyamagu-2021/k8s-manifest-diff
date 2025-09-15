package diff

import (
	"strings"
	"testing"

	"github.com/argoproj/gitops-engine/pkg/utils/kube"
	"github.com/stretchr/testify/assert"
)

// GetChangedResourceKeys extracts resources that have any type of change (Created, Changed, Deleted)
// DEPRECATED: Use Results.Apply() with custom filter or multiple GetResourceKeysByType() calls
func GetChangedResourceKeys(results Results) []kube.ResourceKey {
	return results.Apply(func(_ kube.ResourceKey, diffResult Result) bool {
		return diffResult.Type != Unchanged
	}).GetResourceKeys()
}

// ParseResourceKey parses a string resource key into kube.ResourceKey
func ParseResourceKey(key string) kube.ResourceKey {
	parts := strings.Split(key, "/")
	switch len(parts) {
	case 2: // Kind/Name (cluster-scoped)
		return kube.ResourceKey{
			Kind: parts[0],
			Name: parts[1],
		}
	case 3: // Kind/Namespace/Name (namespaced)
		return kube.ResourceKey{
			Kind:      parts[0],
			Namespace: parts[1],
			Name:      parts[2],
		}
	default:
		// Fallback - shouldn't happen with well-formed keys
		return kube.ResourceKey{Name: key}
	}
}

// AssertResourceChange checks if a specific resource has the expected change type
func AssertResourceChange(t *testing.T, results Results, expectedKey string, expectedChangeType ChangeType) {
	expectedResourceKey := ParseResourceKey(expectedKey)

	// First try exact match
	result, found := results[expectedResourceKey]

	// If not found, try to match by Kind, Namespace, and Name (ignoring Group)
	if !found {
		for key, res := range results {
			if key.Kind == expectedResourceKey.Kind &&
				key.Namespace == expectedResourceKey.Namespace &&
				key.Name == expectedResourceKey.Name {
				result = res
				found = true
				break
			}
		}
	}

	if found {
		assert.Equal(t, expectedChangeType, result.Type, "Expected change type %s for resource %s, got %s", expectedChangeType.String(), expectedKey, result.Type.String())
	} else {
		assert.True(t, found, "Resource %s not found in results", expectedKey)
	}
}
