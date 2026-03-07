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
	rootCmd.PersistentFlags().StringVar(&repoPath, "repo-path", ".", "Path to the repository to analyze")
	rootCmd.PersistentFlags().IntVar(&lookbackDays, "lookback-days", 30, "Number of days to look back for usage data")
	rootCmd.PersistentFlags().StringVar(&outputFormat, "format", "text", "Output format: text, json, md")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
