package diff

import (
	"fmt"
	"strings"

	"github.com/toyamagu-2021/k8s-manifest-diff/pkg/filter"
)

// ResourceKey uniquely identifies a Kubernetes resource
type ResourceKey struct {
	Name      string
	Namespace string
	Group     string
	Kind      string
}

// String returns a string representation of the ResourceKey
func (k ResourceKey) String() string {
	if k.Namespace != "" {
		return fmt.Sprintf("%s/%s/%s/%s", k.Group, k.Kind, k.Namespace, k.Name)
	}
	return fmt.Sprintf("%s/%s/%s", k.Group, k.Kind, k.Name)
}

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
type Results map[ResourceKey]Result

// Statistics represents statistics about diff results
type Statistics struct {
	Total     int
	Changed   int
	Created   int
	Deleted   int
	Unchanged int
}

// StringDiff returns a concatenated string of all diff results with summary header
func (dr Results) StringDiff() string {
	var result strings.Builder

	// Check if there are any changes that need diff output
	hasDiffContent := false
	for _, diffResult := range dr {
		if diffResult.Diff != "" {
			hasDiffContent = true
			break
		}
	}

	// Add summary content as comment header only if there are changes
	if hasDiffContent {
		summaryComments := dr.StringSummaryAsComments()
		if summaryComments != "" {
			result.WriteString(summaryComments)
			result.WriteString("#\n")
		}
	}

	// Add diff content
	for _, diffResult := range dr {
		if diffResult.Diff != "" {
			result.WriteString(diffResult.Diff)
		}
	}
	return result.String()
}

// StringSummary returns a summary string organized by change types: Unchanged, Changed, Create, Delete
func (dr Results) StringSummary() string {
	var result strings.Builder

	// Helper function to format ResourceKey as string
	formatResourceKey := func(key ResourceKey) string {
		if key.Namespace != "" {
			return fmt.Sprintf("%s/%s/%s", key.Kind, key.Namespace, key.Name)
		}
		return fmt.Sprintf("%s/%s", key.Kind, key.Name)
	}

	// Helper function to write a section with count and header comment
	writeSection := func(title string, keys []ResourceKey) {
		if len(keys) > 0 {
			// Add section header comment
			result.WriteString(fmt.Sprintf("# %s: %d resources\n", title, len(keys)))
			result.WriteString(fmt.Sprintf("%s (%d):\n", title, len(keys)))
			for _, key := range keys {
				result.WriteString(fmt.Sprintf("  %s\n", formatResourceKey(key)))
			}
			result.WriteString("\n")
		}
	}

	// Get sections
	unchangedKeys := dr.FilterUnchanged().GetResourceKeys()
	changedKeys := dr.FilterChanged().GetResourceKeys()
	createdKeys := dr.FilterCreated().GetResourceKeys()
	deletedKeys := dr.FilterDeleted().GetResourceKeys()

	// Only add comment header if there are any resources
	stats := dr.GetStatistics()
	if stats.Total > 0 {
		result.WriteString(fmt.Sprintf("# Summary: %d total, %d changed, %d created, %d deleted, %d unchanged\n",
			stats.Total, stats.Changed, stats.Created, stats.Deleted, stats.Unchanged))
		result.WriteString("#\n")
	}

	// Use filtering methods to organize resources by change type
	writeSection("Unchanged", unchangedKeys)
	writeSection("Changed", changedKeys)
	writeSection("Create", createdKeys)
	writeSection("Delete", deletedKeys)

	return strings.TrimRight(result.String(), "\n")
}

// StringSummaryAsComments returns the summary content formatted as comment lines
func (dr Results) StringSummaryAsComments() string {
	summaryContent := dr.StringSummary()
	if summaryContent == "" {
		return ""
	}

	var result strings.Builder
	lines := strings.Split(summaryContent, "\n")
	if lines == nil {
		return ""
	}

	for _, line := range lines {
		if line != "" {
			result.WriteString(fmt.Sprintf("# %s\n", line))
		} else {
			result.WriteString("#\n")
		}
	}
	return result.String()
}

// StringSummaryMarkdown returns a summary string in Markdown format
func (dr Results) StringSummaryMarkdown() string {
	var result strings.Builder

	// Helper function to format ResourceKey as string
	formatResourceKey := func(key ResourceKey) string {
		if key.Namespace != "" {
			return fmt.Sprintf("`%s/%s/%s`", key.Kind, key.Namespace, key.Name)
		}
		return fmt.Sprintf("`%s/%s`", key.Kind, key.Name)
	}

	// Helper function to write a section with count and header
	writeSection := func(title string, keys []ResourceKey) {
		if len(keys) > 0 {
			result.WriteString(fmt.Sprintf("## %s (%d)\n", title, len(keys)))
			for _, key := range keys {
				result.WriteString(fmt.Sprintf("- %s\n", formatResourceKey(key)))
			}
			result.WriteString("\n")
		}
	}

	// Get sections
	unchangedKeys := dr.FilterUnchanged().GetResourceKeys()
	changedKeys := dr.FilterChanged().GetResourceKeys()
	createdKeys := dr.FilterCreated().GetResourceKeys()
	deletedKeys := dr.FilterDeleted().GetResourceKeys()

	// Only add header if there are any resources
	stats := dr.GetStatistics()
	if stats.Total > 0 {
		result.WriteString("# Kubernetes Manifest Diff\n\n")
		result.WriteString("## Summary\n")
		result.WriteString(fmt.Sprintf("**Total Resources**: %d  \n", stats.Total))
		result.WriteString(fmt.Sprintf("**Changed**: %d | **Created**: %d | **Deleted**: %d | **Unchanged**: %d\n\n",
			stats.Changed, stats.Created, stats.Deleted, stats.Unchanged))
	}

	// Use filtering methods to organize resources by change type
	writeSection("Created Resources", createdKeys)
	writeSection("Changed Resources", changedKeys)
	writeSection("Deleted Resources", deletedKeys)
	writeSection("Unchanged Resources", unchangedKeys)

	return strings.TrimRight(result.String(), "\n")
}

// StringDiffMarkdown returns a concatenated string of all diff results with markdown formatting
func (dr Results) StringDiffMarkdown() string {
	var result strings.Builder

	// Check if there are any changes that need diff output
	hasDiffContent := false
	for _, diffResult := range dr {
		if diffResult.Diff != "" {
			hasDiffContent = true
			break
		}
	}

	// Add summary content as markdown header only if there are changes
	if hasDiffContent {
		summaryMarkdown := dr.StringSummaryMarkdown()
		if summaryMarkdown != "" {
			result.WriteString(summaryMarkdown)
			result.WriteString("\n\n---\n\n")
			result.WriteString("## Resource Changes\n\n")
		}
	}

	// Add diff content with markdown formatting
	for key, diffResult := range dr {
		if diffResult.Diff != "" {
			// Extract the original diff content without the header
			lines := strings.Split(diffResult.Diff, "\n")
			var diffLines []string
			headerFound := false
			for _, line := range lines {
				if strings.HasPrefix(line, "===== ") && strings.HasSuffix(line, " ======") {
					headerFound = true
					continue
				}
				if headerFound {
					diffLines = append(diffLines, line)
				}
			}

			// Format resource header in markdown
			if key.Namespace != "" {
				result.WriteString(fmt.Sprintf("### %s/%s %s/%s\n", key.Group, key.Kind, key.Namespace, key.Name))
			} else {
				result.WriteString(fmt.Sprintf("### %s/%s %s\n", key.Group, key.Kind, key.Name))
			}

			// Add diff content in code block
			result.WriteString("```diff\n")
			result.WriteString(strings.Join(diffLines, "\n"))
			result.WriteString("\n```\n\n")
		}
	}
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
func (dr Results) Apply(filter func(ResourceKey, Result) bool) Results {
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
func (dr Results) GetResourceKeys() []ResourceKey {
	keys := make([]ResourceKey, 0, len(dr))
	for key := range dr {
		keys = append(keys, key)
	}
	return keys
}

// GetResourceKeysByType returns a slice of resource keys with the specified change type
func (dr Results) GetResourceKeysByType(changeType ChangeType) []ResourceKey {
	keys := make([]ResourceKey, 0)
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

// Options controls the diff behavior with filtering and masking options
type Options struct {
	FilterOption          *filter.Option // Filtering options
	Context               int            // Number of context lines in diff output
	DisableMaskingSecrets bool           // Disable masking of secret values (default: false)
}

// DefaultOptions returns the default diff options
func DefaultOptions() *Options {
	return &Options{
		FilterOption:          filter.DefaultOption(),
		Context:               3,
		DisableMaskingSecrets: false,
	}
}
