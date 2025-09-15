package diff

import (
	"fmt"
	"strings"

	"github.com/argoproj/gitops-engine/pkg/utils/kube"
)

// ChangeType represents the type of change for a resource
type ChangeType int

const (
	// Unchanged indicates that a resource exists in both base and head with no changes
	Unchanged ChangeType = iota
	// Changed indicates that a resource exists in both base and head with changes
	Changed
	// Created indicates that a resource exists only in head (newly created)
	Created
	// Deleted indicates that a resource exists only in base (deleted)
	Deleted
)

// String returns the string representation of ChangeType
func (ct ChangeType) String() string {
	switch ct {
	case Unchanged:
		return "unchanged"
	case Changed:
		return "changed"
	case Created:
		return "created"
	case Deleted:
		return "deleted"
	default:
		return "unknown"
	}
}

// Result represents the result of a diff operation for a resource
type Result struct {
	Type ChangeType // Type of change (Created, Changed, Deleted, Unchanged)
	Diff string     // Diff string representation
}

// String returns the string representation of Result
func (dr Result) String() string {
	return dr.Diff
}

// Results represents a collection of diff results for multiple resources
type Results map[kube.ResourceKey]Result

// Statistics represents statistics about diff results
type Statistics struct {
	Total     int
	Changed   int
	Created   int
	Deleted   int
	Unchanged int
}

// StringDiff returns a concatenated string of all diff results
func (dr Results) StringDiff() string {
	var result string
	for _, diffResult := range dr {
		if diffResult.Diff != "" {
			result += diffResult.Diff
		}
	}
	return result
}

// StringSummary returns a summary string organized by change types: Unchanged, Changed, Create, Delete
func (dr Results) StringSummary() string {
	var result strings.Builder

	// Helper function to format ResourceKey as string
	formatResourceKey := func(key kube.ResourceKey) string {
		if key.Namespace != "" {
			return fmt.Sprintf("%s/%s/%s", key.Kind, key.Namespace, key.Name)
		}
		return fmt.Sprintf("%s/%s", key.Kind, key.Name)
	}

	// Helper function to write a section
	writeSection := func(title string, keys []kube.ResourceKey) {
		if len(keys) > 0 {
			result.WriteString(fmt.Sprintf("%s:\n", title))
			for _, key := range keys {
				result.WriteString(fmt.Sprintf("  %s\n", formatResourceKey(key)))
			}
			result.WriteString("\n")
		}
	}

	// Use filtering methods to organize resources by change type
	writeSection("Unchanged", dr.FilterUnchanged().GetResourceKeys())
	writeSection("Changed", dr.FilterChanged().GetResourceKeys())
	writeSection("Create", dr.FilterCreated().GetResourceKeys())
	writeSection("Delete", dr.FilterDeleted().GetResourceKeys())

	return strings.TrimRight(result.String(), "\n")
}

// FilterByType returns a new Results containing only resources with the specified change type
func (dr Results) FilterByType(changeType ChangeType) Results {
	result := make(Results)
	for key, diffResult := range dr {
		if diffResult.Type == changeType {
			result[key] = diffResult
		}
	}
	return result
}

// FilterChanged returns a new Results containing only changed resources
func (dr Results) FilterChanged() Results {
	return dr.FilterByType(Changed)
}

// FilterCreated returns a new Results containing only created resources
func (dr Results) FilterCreated() Results {
	return dr.FilterByType(Created)
}

// FilterDeleted returns a new Results containing only deleted resources
func (dr Results) FilterDeleted() Results {
	return dr.FilterByType(Deleted)
}

// FilterUnchanged returns a new Results containing only unchanged resources
func (dr Results) FilterUnchanged() Results {
	return dr.FilterByType(Unchanged)
}

// FilterByKind returns a new Results containing only resources with the specified kind
func (dr Results) FilterByKind(kind string) Results {
	result := make(Results)
	for key, diffResult := range dr {
		if key.Kind == kind {
			result[key] = diffResult
		}
	}
	return result
}

// FilterByNamespace returns a new Results containing only resources with the specified namespace
func (dr Results) FilterByNamespace(namespace string) Results {
	result := make(Results)
	for key, diffResult := range dr {
		if key.Namespace == namespace {
			result[key] = diffResult
		}
	}
	return result
}

// FilterByResourceName returns a new Results containing only resources with the specified name
func (dr Results) FilterByResourceName(name string) Results {
	result := make(Results)
	for key, diffResult := range dr {
		if key.Name == name {
			result[key] = diffResult
		}
	}
	return result
}

// Apply returns a new Results containing only resources that match the filter function
func (dr Results) Apply(filter func(kube.ResourceKey, Result) bool) Results {
	result := make(Results)
	for key, diffResult := range dr {
		if filter(key, diffResult) {
			result[key] = diffResult
		}
	}
	return result
}

// HasChanges returns true if there are any changes (Created, Changed, or Deleted resources)
func (dr Results) HasChanges() bool {
	for _, diffResult := range dr {
		if diffResult.Type != Unchanged {
			return true
		}
	}
	return false
}

// IsEmpty returns true if the Results contains no resources
func (dr Results) IsEmpty() bool {
	return len(dr) == 0
}

// Count returns the total number of resources in the Results
func (dr Results) Count() int {
	return len(dr)
}

// CountByType returns the number of resources with the specified change type
func (dr Results) CountByType(changeType ChangeType) int {
	count := 0
	for _, diffResult := range dr {
		if diffResult.Type == changeType {
			count++
		}
	}
	return count
}

// GetResourceKeys returns a slice of all resource keys in the Results
func (dr Results) GetResourceKeys() []kube.ResourceKey {
	keys := make([]kube.ResourceKey, 0, len(dr))
	for key := range dr {
		keys = append(keys, key)
	}
	return keys
}

// GetResourceKeysByType returns a slice of resource keys with the specified change type
func (dr Results) GetResourceKeysByType(changeType ChangeType) []kube.ResourceKey {
	keys := make([]kube.ResourceKey, 0)
	for key, diffResult := range dr {
		if diffResult.Type == changeType {
			keys = append(keys, key)
		}
	}
	return keys
}

// GetStatistics returns statistics about the diff results
func (dr Results) GetStatistics() Statistics {
	stats := Statistics{
		Total: len(dr),
	}

	for _, diffResult := range dr {
		switch diffResult.Type {
		case Changed:
			stats.Changed++
		case Created:
			stats.Created++
		case Deleted:
			stats.Deleted++
		case Unchanged:
			stats.Unchanged++
		}
	}

	return stats
}

// Options controls the diff behavior
type Options struct {
	ExcludeKinds       []string          // List of Kinds to exclude from diff
	LabelSelector      map[string]string // Label selector to filter resources (exact match)
	AnnotationSelector map[string]string // Annotation selector to filter resources (exact match)
	Context            int               // Number of context lines in diff output
	DisableMaskSecrets bool              // Disable masking of secret values in diff output (default: false)
}

// DefaultOptions returns the default diff options
func DefaultOptions() *Options {
	return &Options{
		ExcludeKinds:       nil,
		LabelSelector:      nil,
		AnnotationSelector: nil,
		Context:            3,
		DisableMaskSecrets: false,
	}
}
