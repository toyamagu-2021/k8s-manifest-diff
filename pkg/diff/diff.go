// Package diff provides functionality for comparing Kubernetes objects and generating diffs.
package diff

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"slices"
	"strings"

	"github.com/argoproj/gitops-engine/pkg/utils/kube"
	"github.com/pmezard/go-difflib/difflib"
	"github.com/toyamagu-2021/k8s-yaml-diff/pkg/parser"
	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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

type objBaseHead struct {
	base *unstructured.Unstructured
	head *unstructured.Unstructured
}

// YamlString compares two YAML strings and returns the diff
// Returns: Results, bool (has differences), error
func YamlString(baseYaml, headYaml string, opts *Options) (Results, bool, error) {
	baseReader := strings.NewReader(baseYaml)
	headReader := strings.NewReader(headYaml)
	return Yaml(baseReader, headReader, opts)
}

// Yaml compares YAML from two io.Reader sources and returns the diff
// Returns: Results, bool (has differences), error
func Yaml(baseReader, headReader io.Reader, opts *Options) (Results, bool, error) {
	baseObjects, err := parser.ParseYAML(baseReader)
	if err != nil {
		return nil, false, fmt.Errorf("failed to parse base YAML: %w", err)
	}

	headObjects, err := parser.ParseYAML(headReader)
	if err != nil {
		return nil, false, fmt.Errorf("failed to parse head YAML: %w", err)
	}

	return Objects(baseObjects, headObjects, opts)
}

// Objects compares two sets of Kubernetes objects and returns the diff
// Returns: Results, bool (has differences), error
func Objects(base, head []*unstructured.Unstructured, opts *Options) (Results, bool, error) {
	if opts == nil {
		opts = DefaultOptions()
	}

	base = FilterResources(base, opts)
	head = FilterResources(head, opts)
	objMap := parseObjsToMap(base, head)
	results := make(Results)
	foundDiff := false

	for k, v := range objMap {
		changeType := determineChangeType(v.base, v.head)

		var diffStr string
		// Generate diff output only for resources that need it
		if needsDiff := requiresDiffOutput(changeType); needsDiff {
			foundDiff = true
			diffOutput, code, err := getDiffStr(k.Name, v.head, v.base, opts)
			if code > 1 {
				return nil, false, err
			}
			header := fmt.Sprintf("===== %s/%s %s/%s ======\n", k.Group, k.Kind, k.Namespace, k.Name)
			diffStr = header + diffOutput
		}

		results[k] = Result{
			Type: changeType,
			Diff: diffStr,
		}
	}
	return results, foundDiff, nil
}

// determineChangeType determines the type of change between base and head objects
func determineChangeType(base, head *unstructured.Unstructured) ChangeType {
	switch {
	case base == nil && head != nil:
		// Resource exists only in head (newly created)
		return Created
	case base != nil && head == nil:
		// Resource exists only in base (deleted)
		return Deleted
	case reflect.DeepEqual(base, head):
		// Resource exists in both with no changes
		return Unchanged
	default:
		// Resource exists in both with changes
		return Changed
	}
}

// requiresDiffOutput determines if a change type requires diff output generation
func requiresDiffOutput(changeType ChangeType) bool {
	return changeType != Unchanged
}

// FilterResources removes resources based on the provided options
func FilterResources(objs []*unstructured.Unstructured, opts *Options) []*unstructured.Unstructured {
	filtered := make([]*unstructured.Unstructured, 0, len(objs))

	// Check if label selector is provided
	hasLabelSelector := len(opts.LabelSelector) > 0
	// Check if annotation selector is provided
	hasAnnotationSelector := len(opts.AnnotationSelector) > 0

	for _, obj := range objs {
		if obj == nil {
			continue
		}

		kind := obj.GetObjectKind().GroupVersionKind().Kind

		// Skip kinds in exclude list
		var excludeKinds []string
		if opts.ExcludeKinds == nil {
			// Use default exclude kinds when none specified
			excludeKinds = DefaultOptions().ExcludeKinds
		} else {
			// Use provided exclude kinds (empty slice means exclude nothing)
			excludeKinds = opts.ExcludeKinds
		}

		if slices.Contains(excludeKinds, kind) {
			continue
		}

		// Apply label selector filter if provided
		if hasLabelSelector {
			objLabels := obj.GetLabels()
			match := true
			for key, value := range opts.LabelSelector {
				if objValue, exists := objLabels[key]; !exists || objValue != value {
					match = false
					break
				}
			}
			if !match {
				continue
			}
		}

		// Apply annotation selector filter if provided
		if hasAnnotationSelector {
			objAnnotations := obj.GetAnnotations()
			match := true
			for key, value := range opts.AnnotationSelector {
				if objValue, exists := objAnnotations[key]; !exists || objValue != value {
					match = false
					break
				}
			}
			if !match {
				continue
			}
		}

		filtered = append(filtered, obj)
	}
	return filtered
}

// getDiffStr generates diff string between live and target objects
func getDiffStr(name string, live, target *unstructured.Unstructured, opts *Options) (string, int, error) {
	preparedLive, preparedTarget := prepareObjectsForDiff(live, target, opts)

	liveData, err := convertObjectToYAML(preparedLive)
	if err != nil {
		return "", 99, err
	}

	targetData, err := convertObjectToYAML(preparedTarget)
	if err != nil {
		return "", 99, err
	}

	diffText, err := generateUnifiedDiff(name, liveData, targetData, opts.Context)
	if err != nil {
		return "", 99, err
	}

	exitCode := determineDiffExitCode(diffText)
	return diffText, exitCode, nil
}

// prepareObjectsForDiff handles secret masking and returns prepared objects for diff
func prepareObjectsForDiff(live, target *unstructured.Unstructured, opts *Options) (*unstructured.Unstructured, *unstructured.Unstructured) {
	preparedLive := live
	preparedTarget := target

	// Mask secrets if enabled
	if !opts.DisableMaskSecrets && (isSecret(live) || isSecret(target)) {
		preparedLive = maskSecretData(live)
		preparedTarget = maskSecretData(target)
	}

	return preparedLive, preparedTarget
}

// convertObjectToYAML converts an unstructured object to YAML string
func convertObjectToYAML(obj *unstructured.Unstructured) (string, error) {
	if obj == nil {
		return "", nil
	}

	bytes, err := yaml.Marshal(obj)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

// generateUnifiedDiff creates a unified diff between two YAML strings
func generateUnifiedDiff(name, liveData, targetData string, context int) (string, error) {
	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(liveData),
		B:        difflib.SplitLines(targetData),
		FromFile: fmt.Sprintf("%s-live.yaml", name),
		ToFile:   fmt.Sprintf("%s.yaml", name),
		Context:  context,
	}

	return difflib.GetUnifiedDiffString(diff)
}

// determineDiffExitCode returns exit code based on diff presence
func determineDiffExitCode(diffText string) int {
	if strings.TrimSpace(diffText) != "" {
		return 1
	}
	return 0
}

// parseObjsToMap converts base and head unstructured arrays to a map
// Key is Kubernetes identifier, values can be nil if only present in one side
func parseObjsToMap(base, head []*unstructured.Unstructured) map[kube.ResourceKey]objBaseHead {
	objMap := map[kube.ResourceKey]objBaseHead{}
	for _, obj := range base {
		key := getResourceKeyFromObj(obj)
		objMap[key] = objBaseHead{base: obj, head: nil}
	}

	for _, obj := range head {
		key := getResourceKeyFromObj(obj)

		if baseObj, ok := objMap[key]; ok {
			baseObj.head = obj
			objMap[key] = baseObj
			continue
		}
		objMap[key] = objBaseHead{base: nil, head: obj}
	}
	return objMap
}

// getResourceKeyFromObj extracts ResourceKey from unstructured object
func getResourceKeyFromObj(obj *unstructured.Unstructured) kube.ResourceKey {
	name := obj.GetName()
	if name == "" {
		name = obj.GetGenerateName()
	}
	return kube.ResourceKey{
		Name:      name,
		Namespace: obj.GetNamespace(),
		Group:     obj.GroupVersionKind().Group,
		Kind:      obj.GroupVersionKind().Kind,
	}
}

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

// Global value mapping for consistent masking across different secrets
// Uses the same approach as gitops-engine with "+" symbols of varying lengths
var globalValueToReplacement = make(map[string]string)
var globalReplacement = "++++++++++++++++"

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
