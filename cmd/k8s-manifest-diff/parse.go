package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/toyamagu-2021/k8s-manifest-diff/pkg/parser"
)

var parseCmd = &cobra.Command{
	Use:   "parse [file1] [file2] ...",
	Short: "Mask secrets in Kubernetes YAML manifests",
	Long: `Mask secrets in Kubernetes YAML manifest files and output the masked versions.
This command processes one or more YAML files and outputs the manifests with
secret data values masked for security purposes.`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		for i, file := range args {
			// Sanitize file path to prevent path traversal
			file = filepath.Clean(file)

			// Open and read the file
			reader, err := os.Open(file) // #nosec G304 - file paths are CLI arguments and cleaned
			if err != nil {
				return fmt.Errorf("failed to open file %s: %w", file, err)
			}

			// Process the file
			maskedYaml, err := parser.Yaml(reader)
			if err != nil {
				if closeErr := reader.Close(); closeErr != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to close file %s: %v\n", file, closeErr)
				}
				return fmt.Errorf("failed to mask file %s: %w", file, err)
			}

			// Close the file
			if err := reader.Close(); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to close file %s: %v\n", file, err)
			}

			// Output the masked YAML
			// If multiple files, add a comment header
			if len(args) > 1 {
				fmt.Printf("# File: %s\n", file)
			}
			fmt.Print(maskedYaml)

			// Add separator between files (except for the last one)
			if i < len(args)-1 {
				fmt.Printf("\n---\n\n")
			}
		}
		return nil
	},
}
