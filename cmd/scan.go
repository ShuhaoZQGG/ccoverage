package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ShuhaoZQGG/ccoverage/internal/output"
	"github.com/ShuhaoZQGG/ccoverage/internal/scanner"
	"github.com/ShuhaoZQGG/ccoverage/internal/types"
	"github.com/spf13/cobra"
)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan repository for Claude Code configuration items",
	Long:  "Build a manifest of all Claude Code configuration (CLAUDE.md, skills, MCP servers, hooks, commands, plugins) found in the repository.",
	RunE:  runScan,
}

func init() {
	rootCmd.AddCommand(scanCmd)
}

func runScan(cmd *cobra.Command, args []string) error {
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return fmt.Errorf("resolve repo path: %w", err)
	}

	manifest, err := scanner.BuildManifest(absPath)
	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	if len(manifest.Items) == 0 {
		fmt.Fprintln(os.Stderr, "No Claude Code configuration found in this repository.")
		return nil
	}

	return renderManifest(manifest)
}

func renderManifest(manifest *types.Manifest) error {
	switch outputFormat {
	case "json":
		return output.RenderManifestJSON(manifest, os.Stdout)
	case "md", "markdown":
		output.RenderManifestMarkdown(manifest, os.Stdout)
	default:
		output.RenderManifestText(manifest, os.Stdout)
	}
	return nil
}
