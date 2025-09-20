// Package parser provides functionality for parsing and masking Kubernetes manifests.
package parser

import (
	"fmt"
	"io"
	"strings"

	"github.com/toyamagu-2021/k8s-manifest-diff/pkg/masking"
	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// YamlString processes a YAML string and returns a masked version with secrets masked
func YamlString(yamlStr string) (string, error) {
	reader := strings.NewReader(yamlStr)
	return Yaml(reader)
}

// Yaml processes YAML from an io.Reader and returns a masked version with secrets masked
func Yaml(reader io.Reader) (string, error) {
	objects, err := ParseYAML(reader)
	if err != nil {
		return "", fmt.Errorf("failed to parse YAML: %w", err)
	}

	maskedObjects, err := Objects(objects)
	if err != nil {
		return "", fmt.Errorf("failed to mask objects: %w", err)
	}

	// Convert masked objects back to YAML
	var yamlParts []string
	for _, obj := range maskedObjects {
		yamlBytes, err := yaml.Marshal(obj.Object)
		if err != nil {
			return "", fmt.Errorf("failed to marshal object to YAML: %w", err)
		}
		yamlParts = append(yamlParts, string(yamlBytes))
	}

	// Join with YAML document separator
	return strings.Join(yamlParts, "---\n"), nil
}

// Objects processes a slice of Kubernetes objects and returns masked versions
func Objects(objs []*unstructured.Unstructured) ([]*unstructured.Unstructured, error) {
	if objs == nil {
		return nil, nil
	}

	masker := masking.NewMasker()
	maskedObjects := make([]*unstructured.Unstructured, len(objs))

	for i, obj := range objs {
		if masking.IsSecret(obj) {
			maskedObjects[i] = masker.MaskSecretData(obj)
		} else {
			// For non-secret objects, return a copy to avoid modifying the original
			maskedObjects[i] = obj.DeepCopy()
		}
	}

	return maskedObjects, nil
}
