package diff

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/pmezard/go-difflib/difflib"
	"github.com/toyamagu-2021/k8s-manifest-diff/pkg/masking"
	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type objBaseHead struct {
	base *unstructured.Unstructured
	head *unstructured.Unstructured
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
	if !opts.DisableMaskSecrets && (masking.IsSecret(live) || masking.IsSecret(target)) {
		preparedLive = masking.MaskSecretData(live)
		preparedTarget = masking.MaskSecretData(target)
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
func parseObjsToMap(base, head []*unstructured.Unstructured) map[ResourceKey]objBaseHead {
	objMap := map[ResourceKey]objBaseHead{}
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
func getResourceKeyFromObj(obj *unstructured.Unstructured) ResourceKey {
	name := obj.GetName()
	if name == "" {
		name = obj.GetGenerateName()
	}
	return ResourceKey{
		Name:      name,
		Namespace: obj.GetNamespace(),
		Group:     obj.GroupVersionKind().Group,
		Kind:      obj.GroupVersionKind().Kind,
	}
}
