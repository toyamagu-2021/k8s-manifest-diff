// Package filter provides filtering functionality for Kubernetes resources.
package filter

import (
	"slices"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Option controls the filtering behavior for Kubernetes resources
type Option struct {
	ExcludeKinds       []string          // List of Kinds to exclude from filtering
	LabelSelector      map[string]string // Label selector to filter resources (exact match)
	AnnotationSelector map[string]string // Annotation selector to filter resources (exact match)
}

// DefaultOption returns the default filtering options
func DefaultOption() *Option {
	return &Option{
		ExcludeKinds:       nil,
		LabelSelector:      nil,
		AnnotationSelector: nil,
	}
}

// Resources removes resources based on the provided filter options
func Resources(objs []*unstructured.Unstructured, opts *Option) []*unstructured.Unstructured {
	if opts == nil {
		opts = DefaultOption()
	}

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
			excludeKinds = DefaultOption().ExcludeKinds
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
