// Package parser provides utilities for parsing Kubernetes YAML and JSON manifests.
package parser

import (
	"fmt"
	"io"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	kubeyaml "k8s.io/apimachinery/pkg/util/yaml"
)

// ParseYAML reads a YAML or JSON stream and returns unstructured objects.
// If the unmarshaller encounters an error, objects read up until the error are returned.
func ParseYAML(reader io.Reader) ([]*unstructured.Unstructured, error) {
	d := kubeyaml.NewYAMLOrJSONDecoder(reader, 4096)
	var objs []*unstructured.Unstructured
	for {
		u := &unstructured.Unstructured{}
		if err := d.Decode(&u); err != nil {
			if err == io.EOF {
				break
			}
			return objs, fmt.Errorf("failed to unmarshal manifest: %v", err)
		}
		if u == nil {
			continue
		}
		objs = append(objs, u)
	}
	return objs, nil
}
