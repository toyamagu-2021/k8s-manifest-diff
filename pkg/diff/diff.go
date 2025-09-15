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
// Returns: (diff string, changed resources []kube.ResourceKey, has differences bool, error)
func YamlString(baseYaml, headYaml string, opts *Options) (string, []kube.ResourceKey, bool, error) {
	baseReader := strings.NewReader(baseYaml)
	headReader := strings.NewReader(headYaml)
	return Yaml(baseReader, headReader, opts)
}

// Yaml compares YAML from two io.Reader sources and returns the diff
// Returns: (diff string, changed resources []kube.ResourceKey, has differences bool, error)
func Yaml(baseReader, headReader io.Reader, opts *Options) (string, []kube.ResourceKey, bool, error) {
	baseObjects, err := parser.ParseYAML(baseReader)
	if err != nil {
		return "", nil, false, fmt.Errorf("failed to parse base YAML: %w", err)
	}

	headObjects, err := parser.ParseYAML(headReader)
	if err != nil {
		return "", nil, false, fmt.Errorf("failed to parse head YAML: %w", err)
	}

	return Objects(baseObjects, headObjects, opts)
}

// Objects compares two sets of Kubernetes objects and returns the diff
// Returns: (diff string, changed resources []kube.ResourceKey, has differences bool, error)
func Objects(base, head []*unstructured.Unstructured, opts *Options) (string, []kube.ResourceKey, bool, error) {
	if opts == nil {
		opts = DefaultOptions()
	}

	base = FilterResources(base, opts)
	head = FilterResources(head, opts)
	objMap := parseObjsToMap(base, head)
	foundDiff := false
	diff := ""
	var changedResources []kube.ResourceKey

	for k, v := range objMap {
		if reflect.DeepEqual(v.base, v.head) {
			continue
		}

		foundDiff = true
		diffStr, code, err := getDiffStr(k.Name, v.head, v.base, opts)
		if code > 1 {
			return "", nil, false, err
		}
		header := fmt.Sprintf("===== %s/%s %s/%s ======\n", k.Group, k.Kind, k.Namespace, k.Name)
		diff += header + diffStr

		// Add ResourceKey to changed resources list
		changedResources = append(changedResources, k)
	}
	return diff, changedResources, foundDiff, nil
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
