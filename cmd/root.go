package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	repoPath     string
	lookbackDays int
	outputFormat string
)

var rootCmd = &cobra.Command{
	Use:   "ccoverage",
	Short: "Coverage analysis for Claude Code project configuration",
	Long:  "ccoverage scans a repo's Claude Code config (skills, MCP servers, hooks, CLAUDE.md files, commands) and joins it against session data to produce a coverage report.",
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&repoPath, "target", "t", ".", "Path to the repository to analyze")
	rootCmd.PersistentFlags().IntVarP(&lookbackDays, "lookback-days", "d", 30, "Number of days to look back for usage data")
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "format", "f", "text", "Output format: text, json, md")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
