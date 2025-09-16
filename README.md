# k8s-manifest-diff

A CLI tool and Go library for parsing Kubernetes YAML/JSON manifests and computing diffs between sets of Kubernetes objects.

## Features

- CLI tool for comparing Kubernetes YAML manifests
- Flexible filtering options (exclude kinds, label/annotation selectors)
- Secret data masking for security (with option to disable)
- Summary mode for high-level diff overview
- Go library with simple API for programmatic usage
- Unified diff output with customizable context lines

## Installation

### Install CLI Tool

```bash
go install github.com/toyamagu-2021/k8s-manifest-diff/cmd/k8s-manifest-diff@latest
```

### Install Go Library

```bash
go get github.com/toyamagu-2021/k8s-manifest-diff
```

## CLI Usage

### Basic Usage

Compare two Kubernetes YAML files:

```bash
k8s-manifest-diff diff base.yaml head.yaml
```

### Filtering Options

Exclude specific resource kinds:
```bash
k8s-manifest-diff diff base.yaml head.yaml --exclude-kinds Job,CronJob,Pod
```

Filter by labels:
```bash
k8s-manifest-diff diff base.yaml head.yaml --label app=nginx --label tier=frontend
```

Filter by annotations:
```bash
k8s-manifest-diff diff base.yaml head.yaml --annotation app.kubernetes.io/managed-by=helm
```

Control diff context lines:
```bash
k8s-manifest-diff diff base.yaml head.yaml --context 5
```

Disable secret masking:
```bash
k8s-manifest-diff diff base.yaml head.yaml --disable-masking-secret
```

Show only summary of changes:
```bash
k8s-manifest-diff diff base.yaml head.yaml --summary
```

### Version Information

```bash
k8s-manifest-diff version
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
    "github.com/toyamagu-2021/k8s-manifest-diff/pkg/diff"
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

    results, err := diff.YamlString(baseYaml, headYaml, nil)
    if err != nil {
        panic(err)
    }

    if results.HasChanges() {
        fmt.Printf("Changed resources: %v\n", results.FilterChanged().GetResourceKeys())
        fmt.Printf("Differences found:\n%s\n", results.StringDiff())
    } else {
        fmt.Println("No differences found")
    }
}
```

### Custom Options

```go
opts := &diff.Options{
    ExcludeKinds:           []string{"Job", "CronJob"},
    LabelSelector:          map[string]string{"app": "nginx"},
    AnnotationSelector:     map[string]string{"helm.sh/managed-by": "helm"},
    Context:                5,
    DisableMaskingSecret:   false, // Enable secret masking by default
}

results, err := diff.YamlString(baseYaml, headYaml, opts)
if err != nil {
    panic(err)
}

fmt.Printf("Changed resources: %v\n", results.FilterChanged().GetResourceKeys())
if results.HasChanges() {
    fmt.Printf("Diff:\n%s\n", results.StringDiff())
}
```

## Build from Source

```bash
git clone https://github.com/toyamagu-2021/k8s-manifest-diff
cd k8s-manifest-diff
make build
```

## Development

### Prerequisites

- Go 1.25.0 or later
- Make

### Build and Development Commands

```bash
# Build the project
make build

# Clean build artifacts
make clean

# Download and verify dependencies
make deps

# Tidy go modules
make tidy

# Format code and imports
make fmt

# Check formatting without modifying files
make fmt-check

# Run go vet
make vet

# Run all linting tools
make lint

# Run unit tests
make test

# Run E2E tests
make test-e2e

# Run all tests (unit + E2E)
make test-all
```

### Package Architecture

The project is organized into the following packages:

- **`cmd/k8s-manifest-diff/`**: CLI application entry point with cobra-based commands
- **`pkg/parser/`**: YAML/JSON parsing using k8s.io/apimachinery
- **`pkg/diff/`**: Core diffing logic with filtering and secret masking
- **`testing/e2e/`**: End-to-end test scenarios

## License

This project is licensed under the MIT License.

## Contributing

Contributions are welcome! Please submit issues and pull requests on GitHub.
