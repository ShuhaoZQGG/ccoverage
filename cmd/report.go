package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ShuhaoZQGG/ccoverage/internal/coverage"
	"github.com/ShuhaoZQGG/ccoverage/internal/output"
	"github.com/ShuhaoZQGG/ccoverage/internal/scanner"
	"github.com/ShuhaoZQGG/ccoverage/internal/types"
	"github.com/ShuhaoZQGG/ccoverage/internal/usage"
	"github.com/spf13/cobra"
)

var (
	underuseThreshold int
	statusFilter      string
	typeFilter        string
	errorOnMatch      bool
	lastSession       bool
)

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generate a coverage report for Claude Code configuration",
	Long:  "Scan repository config, locate session history, analyze usage, and produce a coverage report showing Active, Underused, and Dormant items.",
	RunE:  runReport,
}

func init() {
	reportCmd.Flags().IntVar(&underuseThreshold, "threshold", 2, "Activations at or below this value are classified as Underused")
	reportCmd.Flags().StringVar(&statusFilter, "status", "", "Comma-separated statuses to include (Active,Underused,Dormant)")
	reportCmd.Flags().StringVar(&typeFilter, "type", "", "Comma-separated config types to include (CLAUDE.md,Skill,MCP,Hook,Command,Plugin)")
	reportCmd.Flags().BoolVar(&errorOnMatch, "error-on-match", false, "Exit with code 1 if any results remain after filtering")
	reportCmd.Flags().BoolVar(&lastSession, "last-session", false, "Include per-item hit/miss for the most recent session")
	rootCmd.AddCommand(reportCmd)
}

func runReport(cmd *cobra.Command, args []string) error {
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
		fmt.Fprintln(os.Stderr, "  Looked for: CLAUDE.md files, .claude/skills/, .mcp.json, .claude/settings.json hooks, .claude/commands/, plugins")
		return nil
	}

	sessionFiles, err := usage.LocateSessionFiles(absPath, lookbackDays)
	if err != nil {
		return fmt.Errorf("locate sessions: %w", err)
	}

	report, err := coverage.Analyze(manifest, sessionFiles, lookbackDays, underuseThreshold)
	if err != nil {
		return fmt.Errorf("analyze: %w", err)
	}

	if lastSession {
		latestFile, lsErr := usage.LatestSessionFile(absPath)
		if lsErr != nil {
			return fmt.Errorf("locate latest session: %w", lsErr)
		}
		if latestFile != "" {
			lsReport, lsErr := usage.MatchSingleSession(manifest, latestFile)
			if lsErr != nil {
				return fmt.Errorf("match last session: %w", lsErr)
			}
			report.LastSession = lsReport
		}
	}

	if len(sessionFiles) == 0 {
		fmt.Fprintln(os.Stderr, "No session data found. All items shown as Dormant.")
		fmt.Fprintf(os.Stderr, "  Looked in: ~/.claude/projects/%s/\n", usage.EncodeRepoPath(absPath))
		fmt.Fprintf(os.Stderr, "  (No Claude Code sessions found for this repo in the last %d days.)\n", lookbackDays)
	}

	report = filterReport(report, statusFilter, typeFilter)

	if err := renderReport(report); err != nil {
		return err
	}

	if errorOnMatch && len(report.Results) > 0 {
		os.Exit(1)
	}
	return nil
}

func parseStatusFilter(raw string) map[types.Status]bool {
	if raw == "" {
		return nil
	}
	m := make(map[types.Status]bool)
	for _, s := range strings.Split(raw, ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		m[types.Status(strings.ToUpper(s[:1])+strings.ToLower(s[1:]))] = true
	}
	return m
}

func parseTypeFilter(raw string) map[types.ConfigType]bool {
	if raw == "" {
		return nil
	}
	lookup := map[string]types.ConfigType{
		"claude.md": types.ConfigClaudeMD,
		"skill":     types.ConfigSkill,
		"mcp":       types.ConfigMCP,
		"hook":      types.ConfigHook,
		"command":   types.ConfigCommand,
		"plugin":    types.ConfigPlugin,
	}
	m := make(map[types.ConfigType]bool)
	for _, s := range strings.Split(raw, ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if ct, ok := lookup[strings.ToLower(s)]; ok {
			m[ct] = true
		}
	}
	return m
}

func filterReport(report *types.CoverageReport, statusRaw, typeRaw string) *types.CoverageReport {
	statuses := parseStatusFilter(statusRaw)
	configTypes := parseTypeFilter(typeRaw)
	if statuses == nil && configTypes == nil {
		return report
	}

	var filtered []types.CoverageResult
	for _, r := range report.Results {
		if statuses != nil && !statuses[r.Status] {
			continue
		}
		if configTypes != nil && !configTypes[r.Item.Type] {
			continue
		}
		filtered = append(filtered, r)
	}

	summary := types.ReportSummary{TotalItems: len(filtered)}
	for _, r := range filtered {
		switch r.Status {
		case types.StatusActive:
			summary.Active++
		case types.StatusUnderused:
			summary.Underused++
		case types.StatusDormant:
			summary.Dormant++
		}
	}

	return &types.CoverageReport{
		RepoPath:         report.RepoPath,
		LookbackDays:     report.LookbackDays,
		SessionsAnalyzed: report.SessionsAnalyzed,
		Results:          filtered,
		Summary:          summary,
	}
}

func renderReport(report *types.CoverageReport) error {
	switch outputFormat {
	case "json":
		return output.RenderJSON(report, os.Stdout)
	case "md", "markdown":
		output.RenderMarkdown(report, os.Stdout)
	default:
		output.RenderText(report, os.Stdout)
	}
	return nil
}
