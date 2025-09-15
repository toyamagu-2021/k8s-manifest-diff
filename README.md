# k8s-yaml-diff

A CLI tool and Go library for parsing Kubernetes YAML/JSON manifests and computing diffs between sets of Kubernetes objects.

## Features

- CLI tool for comparing Kubernetes YAML manifests
- Flexible filtering options (exclude kinds, label/annotation selectors)
- Go library with simple API for programmatic usage

## Installation

### Install CLI Tool

```bash
go install github.com/toyamagu-2021/k8s-yaml-diff/cmd/k8s-yaml-diff@latest
```

### Install Go Library

```bash
go get github.com/toyamagu-2021/k8s-yaml-diff
```

## CLI Usage

### Basic Usage

Compare two Kubernetes YAML files:

```bash
k8s-yaml-diff diff base.yaml head.yaml
```

### Filtering Options

Exclude specific resource kinds:
```bash
k8s-yaml-diff diff base.yaml head.yaml --exclude-kinds Job,CronJob,Pod
```

Filter by labels:
```bash
k8s-yaml-diff diff base.yaml head.yaml --label app=nginx --label tier=frontend
```

Filter by annotations:
```bash
k8s-yaml-diff diff base.yaml head.yaml --annotation app.kubernetes.io/managed-by=helm
```

Control diff context lines:
```bash
k8s-yaml-diff diff base.yaml head.yaml --context 5
```

### Version Information

```bash
k8s-yaml-diff version
```

### Exit Codes

The tool follows standard Unix diff and ArgoCD conventions:

- `0`: No differences found
- `1`: Differences found
- `2`: Error occurred (e.g., file not found, parsing error)

## Library Usage

### Simple YAML String Comparison

```go
package main

import (
    "fmt"
    "github.com/toyamagu-2021/k8s-yaml-diff/pkg/diff"
)

func main() {
    baseYaml := `
apiVersion: v1
kind: ConfigMap
metadata:
  name: example
data:
  key: value1
`

    headYaml := `
apiVersion: v1
kind: ConfigMap
metadata:
  name: example
data:
  key: value2
`

    diffResult, hasDiff, err := diff.YamlString(baseYaml, headYaml, nil)
    if err != nil {
        panic(err)
    }

    if hasDiff {
        fmt.Printf("Differences found:\n%s\n", diffResult)
    } else {
        fmt.Println("No differences found")
    }
}
```

### Custom Options

```go
opts := &diff.Options{
    ExcludeKinds:       []string{"Job", "CronJob"},
    LabelSelector:      map[string]string{"app": "nginx"},
    AnnotationSelector: map[string]string{"helm.sh/managed-by": "helm"},
    Context:            5,
}

diffResult, hasDiff, err := diff.YamlString(baseYaml, headYaml, opts)
```

## Build from Source

```bash
git clone https://github.com/toyamagu-2021/k8s-yaml-diff
cd k8s-yaml-diff
make build
```

## License

This project is licensed under the MIT License.

## Contributing

Contributions are welcome! Please submit issues and pull requests on GitHub.
