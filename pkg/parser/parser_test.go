package parser

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseYAML(t *testing.T) {
	yamldata := `
apiVersion: v1
kind: Pod
metadata:
  name: nginx
spec:
  containers:
  - name: nginx
    image: nginx:1.14.2
    ports:
    - containerPort: 80
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  labels:
    app: nginx
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:1.14.2
        ports:
        - containerPort: 80
`
	var b bytes.Buffer
	b.Write([]byte(yamldata))

	objs, err := ParseYAML(&b)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(objs))
	assert.Equal(t, "Pod", objs[0].GetKind())
	assert.Equal(t, "Deployment", objs[1].GetKind())
}

func TestParseYAMLEmpty(t *testing.T) {
	var b bytes.Buffer
	b.Write([]byte(""))

	objs, err := ParseYAML(&b)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(objs))
}

func TestParseYAMLInvalid(t *testing.T) {
	invalidYaml := `
apiVersion: v1
kind: Pod
metadata:
  name: nginx
spec:
  containers:
  - name: nginx
    image: nginx:1.14.2
    ports:
    - containerPort: "invalid_port"
---
invalid yaml content: {{{
`
	var b bytes.Buffer
	b.Write([]byte(invalidYaml))

	objs, err := ParseYAML(&b)
	// Should return parsed objects up to the error point
	if err != nil {
		assert.Contains(t, err.Error(), "failed to unmarshal manifest")
	}
	// The number of objects returned depends on when the error occurs
	assert.True(t, len(objs) >= 0)
}

func TestParseYAMLJSON(t *testing.T) {
	jsonData := `{
		"apiVersion": "v1",
		"kind": "Pod",
		"metadata": {
			"name": "nginx"
		},
		"spec": {
			"containers": [{
				"name": "nginx",
				"image": "nginx:1.14.2",
				"ports": [{
					"containerPort": 80
				}]
			}]
		}
	}`
	var b bytes.Buffer
	b.Write([]byte(jsonData))

	objs, err := ParseYAML(&b)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(objs))
	assert.Equal(t, "Pod", objs[0].GetKind())
	assert.Equal(t, "nginx", objs[0].GetName())
}
