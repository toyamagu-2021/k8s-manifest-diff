// Package parser provides functionality for parsing and masking Kubernetes manifests.
package parser

import (
	"fmt"
	"io"
	"strings"

	"github.com/toyamagu-2021/k8s-manifest-diff/pkg/filter"
	"github.com/toyamagu-2021/k8s-manifest-diff/pkg/masking"
	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Options controls the parsing and masking behavior
type Options struct {
	FilterOption          *filter.Option // Filtering options
	DisableMaskingSecrets bool           // Disable masking of secret values (default: false)
}

// DefaultOptions returns the default parsing options
func DefaultOptions() *Options {
	return &Options{
		FilterOption:          filter.DefaultOption(),
		DisableMaskingSecrets: false,
	}
}

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

// Results represents a collection of resources
type Results map[ResourceKey]*unstructured.Unstructured

// String converts Results to YAML string representation
func (r Results) String() string {
	if len(r) == 0 {
		return ""
	}

	// Create header with resource list as YAML comments
	var resourceList []string
	for key := range r {
		if key.Namespace != "" {
			resourceList = append(resourceList, fmt.Sprintf("# %s/%s %s/%s", key.Group, key.Kind, key.Namespace, key.Name))
		} else {
			resourceList = append(resourceList, fmt.Sprintf("# %s/%s %s", key.Group, key.Kind, key.Name))
		}
	}
	header := fmt.Sprintf("# Resources (%d)\n%s\n\n", len(r), strings.Join(resourceList, "\n"))

	var yamlParts []string
	for _, obj := range r {
		yamlBytes, err := yaml.Marshal(obj.Object)
		if err != nil {
			// Return error information if marshaling fails
			return fmt.Sprintf("Error marshaling object to YAML: %v", err)
		}
		yamlParts = append(yamlParts, strings.TrimSpace(string(yamlBytes)))
	}
	return header + strings.Join(yamlParts, "\n---\n")
}

// YamlString processes a YAML string and returns Results with optional masking
func YamlString(yamlStr string, opts *Options) (Results, error) {
	reader := strings.NewReader(yamlStr)
	return Yaml(reader, opts)
}

// Yaml processes YAML from an io.Reader and returns Results with optional masking
func Yaml(reader io.Reader, opts *Options) (Results, error) {
	if opts == nil {
		opts = DefaultOptions()
	}

	objects, err := ParseYAML(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return Objects(objects, opts)
}

// Objects processes a slice of Kubernetes objects and returns Results with optional masking and filtering
func Objects(objs []*unstructured.Unstructured, opts *Options) (Results, error) {
	if opts == nil {
		opts = DefaultOptions()
	}

	if objs == nil {
		return make(Results), nil
	}

	// Apply filtering first
	filteredObjs := filter.Resources(objs, opts.FilterOption)

	masker := masking.NewMasker()
	results := make(Results)

	for _, obj := range filteredObjs {
		// Create resource key
		key := ResourceKey{
			Name:      obj.GetName(),
			Namespace: obj.GetNamespace(),
			Group:     obj.GetObjectKind().GroupVersionKind().Group,
			Kind:      obj.GetKind(),
		}

		var processedObj *unstructured.Unstructured
		if masking.IsSecret(obj) && !opts.DisableMaskingSecrets {
			maskedObj, err := masker.MaskSecretData(obj)
			if err != nil {
				return nil, fmt.Errorf("failed to mask secret: %w", err)
			}
			processedObj = maskedObj
		} else {
			// For non-secret objects or when masking is disabled, return a copy to avoid modifying the original
			processedObj = obj.DeepCopy()
		}

		results[key] = processedObj
	}

	return results, nil
}
