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

// Options controls the parsing and masking behavior
type Options struct {
	DisableMaskingSecrets bool // Disable masking of secret values (default: false)
}

// DefaultOptions returns the default parsing options
func DefaultOptions() *Options {
	return &Options{
		DisableMaskingSecrets: false,
	}
}

// YamlString processes a YAML string and returns a version with optional masking
func YamlString(yamlStr string, opts *Options) (string, error) {
	reader := strings.NewReader(yamlStr)
	return Yaml(reader, opts)
}

// Yaml processes YAML from an io.Reader and returns a version with optional masking
func Yaml(reader io.Reader, opts *Options) (string, error) {
	if opts == nil {
		opts = DefaultOptions()
	}

	objects, err := ParseYAML(reader)
	if err != nil {
		return "", fmt.Errorf("failed to parse YAML: %w", err)
	}

	maskedObjects, err := Objects(objects, opts)
	if err != nil {
		return "", fmt.Errorf("failed to process objects: %w", err)
	}

	// Convert processed objects back to YAML
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

// Objects processes a slice of Kubernetes objects and returns versions with optional masking
func Objects(objs []*unstructured.Unstructured, opts *Options) ([]*unstructured.Unstructured, error) {
	if opts == nil {
		opts = DefaultOptions()
	}

	if objs == nil {
		return nil, nil
	}

	masker := masking.NewMasker()
	processedObjects := make([]*unstructured.Unstructured, len(objs))

	for i, obj := range objs {
		if masking.IsSecret(obj) && !opts.DisableMaskingSecrets {
			maskedObj, err := masker.MaskSecretData(obj)
			if err != nil {
				return nil, fmt.Errorf("failed to mask secret: %w", err)
			}
			processedObjects[i] = maskedObj
		} else {
			// For non-secret objects or when masking is disabled, return a copy to avoid modifying the original
			processedObjects[i] = obj.DeepCopy()
		}
	}

	return processedObjects, nil
}
