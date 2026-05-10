package main

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "agrep",
	Short: "Code structure recon for AI agents",
	Long: `agrep extracts function signatures, type declarations, and doc comments
from source files so AI agents can understand a codebase's shape without
reading every line.

Use it as a low-cost scout before invoking 'read' on a specific file.`,
	SilenceUsage: true,
}

func main() {
	// cobra writes the error to stderr itself; we only need to set the
	// non-zero exit code.
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
