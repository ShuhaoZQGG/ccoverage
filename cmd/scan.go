package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/shuhaozhang/ccoverage/internal/scanner"
	"github.com/shuhaozhang/ccoverage/internal/types"
	"github.com/spf13/cobra"
)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan repository for Claude Code configuration items",
	Long:  "Build a manifest of all Claude Code configuration (CLAUDE.md, skills, MCP servers, hooks, commands) found in the repository.",
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

	report := &types.CoverageReport{
		RepoPath:     manifest.RepoPath,
		LookbackDays: 0,
		Results:      make([]types.CoverageResult, len(manifest.Items)),
		Summary: types.ReportSummary{
			TotalItems: len(manifest.Items),
		},
	}

	for i, item := range manifest.Items {
		status := types.StatusDormant
		if !item.Exists {
			status = types.StatusOrphaned
			report.Summary.Orphaned++
		} else {
			report.Summary.Dormant++
		}
		report.Results[i] = types.CoverageResult{
			Item:   item,
			Status: status,
		}
	}

	return renderReport(report)
}
