// Package main provides the k8s-yaml-diff CLI tool for comparing Kubernetes YAML manifests.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/toyamagu-2021/k8s-yaml-diff/pkg/diff"
	"github.com/toyamagu-2021/k8s-yaml-diff/pkg/parser"
)

var (
	version = "1.0.0"
	commit  = "none"
	date    = "unknown"
)

var (
	excludeKinds         []string
	labelSelectors       []string
	annotationSelectors  []string
	context              int
	disableMaskingSecret bool
	summary              bool
)

var rootCmd = &cobra.Command{
	Use:   "k8s-yaml-diff",
	Short: "Compare Kubernetes YAML manifests",
	Long: `k8s-yaml-diff is a tool for comparing Kubernetes YAML manifests.
It can filter out specific resources like hooks, secrets, or custom kinds,
and use custom diff commands for comparison.`,
}

var diffCmd = &cobra.Command{
	Use:   "diff [base-file] [head-file]",
	Short: "Compare two Kubernetes YAML files",
	Long: `Compare two Kubernetes YAML manifest files and show the differences.
Supports filtering options to exclude specific resource types.`,
	Args: cobra.ExactArgs(2),
	RunE: func(_ *cobra.Command, args []string) error {
		baseFile := args[0]
		headFile := args[1]

		// Sanitize file paths to prevent path traversal
		baseFile = filepath.Clean(baseFile)
		headFile = filepath.Clean(headFile)

		// Read base file
		baseReader, err := os.Open(baseFile) // #nosec G304 - file paths are CLI arguments and cleaned
		if err != nil {
			return fmt.Errorf("failed to open base file: %w", err)
		}
		defer func() {
			if err := baseReader.Close(); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to close base file: %v\n", err)
			}
		}()

		baseObjs, err := parser.ParseYAML(baseReader)
		if err != nil {
			return fmt.Errorf("failed to parse base file: %w", err)
		}

		// Read head file
		headReader, err := os.Open(headFile) // #nosec G304 - file paths are CLI arguments and cleaned
		if err != nil {
			return fmt.Errorf("failed to open head file: %w", err)
		}
		defer func() {
			if err := headReader.Close(); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to close head file: %v\n", err)
			}
		}()

		headObjs, err := parser.ParseYAML(headReader)
		if err != nil {
			return fmt.Errorf("failed to parse head file: %w", err)
		}

		// Parse label selectors into map
		labelSelectorMap := make(map[string]string)
		for _, selector := range labelSelectors {
			if strings.Contains(selector, "=") {
				parts := strings.SplitN(selector, "=", 2)
				if len(parts) == 2 {
					labelSelectorMap[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
				}
			}
		}

		// Parse annotation selectors into map
		annotationSelectorMap := make(map[string]string)
		for _, selector := range annotationSelectors {
			if strings.Contains(selector, "=") {
				parts := strings.SplitN(selector, "=", 2)
				if len(parts) == 2 {
					annotationSelectorMap[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
				}
			}
		}

		// Create diff options
		opts := &diff.Options{
			ExcludeKinds:       excludeKinds,
			LabelSelector:      labelSelectorMap,
			AnnotationSelector: annotationSelectorMap,
			Context:            context,
			DisableMaskSecrets: disableMaskingSecret,
		}

		// Perform diff
		diffResult, resourceChanges, hasDiff, err := diff.Objects(baseObjs, headObjs, opts)
		if err != nil {
			return fmt.Errorf("failed to diff objects: %w", err)
		}

		if hasDiff {
			if summary {
				// Output only the list of changed resources with their change types
				for resource, changeType := range resourceChanges {
					// Skip unchanged resources in summary mode
					if changeType == diff.Unchanged {
						continue
					}

					// Format ResourceKey as string for output
					resourceID := fmt.Sprintf("%s/%s", resource.Kind, resource.Name)
					if resource.Namespace != "" {
						resourceID = fmt.Sprintf("%s/%s/%s", resource.Kind, resource.Namespace, resource.Name)
					}
					fmt.Printf("%s (%s)\n", resourceID, changeType.String())
				}
			} else {
				// Output the full diff
				fmt.Print(diffResult)
			}
			os.Exit(1)
		}
		fmt.Println("No differences found")

		return nil
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Printf("k8s-yaml-diff version %s\n", version)
		fmt.Printf("commit: %s\n", commit)
		fmt.Printf("date: %s\n", date)
	},
}

func init() {
	diffCmd.Flags().StringSliceVar(&excludeKinds, "exclude-kinds", []string{}, "List of Kinds to exclude from diff")
	diffCmd.Flags().StringSliceVar(&labelSelectors, "label", []string{}, "Label selector to filter resources (e.g., 'app=nginx', 'tier=frontend'). Can be specified multiple times.")
	diffCmd.Flags().StringSliceVar(&annotationSelectors, "annotation", []string{}, "Annotation selector to filter resources (e.g., 'app.kubernetes.io/managed-by=helm', 'deployment.category=web'). Can be specified multiple times.")
	diffCmd.Flags().IntVar(&context, "context", 3, "Number of context lines in diff output")
	diffCmd.Flags().BoolVar(&disableMaskingSecret, "disable-masking-secret", false, "Disable masking of Secret data values in diff output")
	diffCmd.Flags().BoolVar(&summary, "summary", false, "Output only the list of changed resources instead of full diff")

	rootCmd.AddCommand(diffCmd)
	rootCmd.AddCommand(versionCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(2)
	}
}
