# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

k8s-yaml-diff is a Go library and CLI tool for parsing Kubernetes YAML/JSON manifests and computing diffs between sets of Kubernetes objects. It supports flexible filtering options and is compatible with ArgoCD workflows.

## Build Commands

```bash
# Build the project
make build

# Clean build artifacts
make clean

# Download and verify dependencies
make deps

# Tidy go modules
make tidy
```

## Development Commands

```bash
# Format code and imports
make fmt

# Check formatting without modifying files
make fmt-check

# Run go vet
make vet

# Run all linting tools (fmt-check, vet, staticcheck, golint, ineffassign, errcheck, misspell)
make lint

# Run tests
go test ./... -v

# Run tests for specific package
go test ./pkg/diff -v
go test ./pkg/parser -v

# Run root level tests
go test . -v
```

## Code Architecture

### Package Structure

- **`cmd/k8s-yaml-diff/`**: CLI application entry point with cobra-based command handling
- **`pkg/parser/`**: YAML/JSON parsing logic using k8s.io/apimachinery
- **`pkg/diff/`**: Core diffing logic with filtering and comparison capabilities

### Core Components

1. **Parser (`pkg/parser/parser.go`)**:
   - `ParseYAML()`: Converts YAML/JSON streams into unstructured Kubernetes objects
   - Uses `k8s.io/apimachinery/pkg/util/yaml.NewYAMLOrJSONDecoder`

2. **Diff Engine (`pkg/diff/diff.go`)**:
   - `DiffObjects()`: Main diffing function that compares two sets of K8s objects
   - `DiffYaml()` and `DiffYamlString()`: Convenience wrappers for YAML input
   - `FilterResources()`: Applies label selectors, annotation selectors, and kind exclusions
   - Uses `github.com/pmezard/go-difflib/difflib` for unified diff output

3. **CLI (`cmd/k8s-yaml-diff/main.go`)**:
   - Cobra-based CLI with `diff` and `version` subcommands
   - Supports flags: `--exclude-kinds`, `--label`, `--annotation`, `--context`
   - Returns exit code 1 when differences found (standard diff behavior)

### Key Data Flow

1. Parse YAML files into `[]*unstructured.Unstructured` using parser package
2. Apply filtering based on `DiffOptions` (exclude kinds, label/annotation selectors)
3. Convert objects to `map[kube.ResourceKey]objBaseHead` for comparison
4. Generate unified diff using go-difflib for each changed resource
5. Return formatted diff string with resource headers

### Default Filtering Behavior

- Excludes `Workflow` resources by default
- Supports label selector filtering (exact match)
- Supports annotation selector filtering (exact match)
- Context lines in diff output default to 3

### Testing

- Tests are located in `diff_test.go` at project root
- Uses `github.com/stretchr/testify/assert` for assertions
- Covers filtering options, diff scenarios, and edge cases
- Test data uses unstructured objects with various K8s resource types

## Dependencies

Key dependencies:
- `k8s.io/apimachinery`: Kubernetes API machinery for unstructured objects
- `github.com/argoproj/gitops-engine`: GitOps engine utilities for ResourceKey
- `github.com/pmezard/go-difflib`: Unified diff generation
- `github.com/spf13/cobra`: CLI framework
- `gopkg.in/yaml.v2`: YAML marshaling for diff output