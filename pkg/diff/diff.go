// Package diff provides functionality for comparing Kubernetes objects and generating diffs.
package diff

import (
	"fmt"
	"io"
	"strings"

	"github.com/toyamagu-2021/k8s-manifest-diff/pkg/filter"
	"github.com/toyamagu-2021/k8s-manifest-diff/pkg/parser"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// YamlString compares two YAML strings and returns the diff
func YamlString(baseYaml, headYaml string, opts *Options) (Results, error) {
	baseReader := strings.NewReader(baseYaml)
	headReader := strings.NewReader(headYaml)
	return Yaml(baseReader, headReader, opts)
}

// Yaml compares YAML from two io.Reader sources and returns the diff
func Yaml(baseReader, headReader io.Reader, opts *Options) (Results, error) {
	baseObjects, err := parser.ParseYAML(baseReader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse base YAML: %w", err)
	}

	headObjects, err := parser.ParseYAML(headReader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse head YAML: %w", err)
	}

	return Objects(baseObjects, headObjects, opts)
}

// Objects compares two sets of Kubernetes objects and returns the diff
func Objects(base, head []*unstructured.Unstructured, opts *Options) (Results, error) {
	if opts == nil {
		opts = DefaultOptions()
	}

	base = filter.Resources(base, opts.FilterOption)
	head = filter.Resources(head, opts.FilterOption)
	objMap := parseObjsToMap(base, head)
	results := make(Results)

	for k, v := range objMap {
		changeType := determineChangeType(v.base, v.head)

		var diffStr string
		// Generate diff output only for resources that need it
		if needsDiff := requiresDiffOutput(changeType); needsDiff {
			diffOutput, code, err := getDiffStr(k.Name, v.head, v.base, opts)
			if code > 1 {
				return nil, err
			}
			header := fmt.Sprintf("===== %s/%s %s/%s ======\n", k.Group, k.Kind, k.Namespace, k.Name)
			diffStr = header + diffOutput
		}

		results[k] = Result{
			Type: changeType,
			Diff: diffStr,
		}
	}
	return results, nil
}
