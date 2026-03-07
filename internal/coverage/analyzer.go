package coverage

import (
	"fmt"

	"github.com/shuhaozhang/ccoverage/internal/types"
	"github.com/shuhaozhang/ccoverage/internal/usage"
)

// Analyze produces a CoverageReport by joining the manifest against usage data
// extracted from sessionFiles.
//
// Parameters:
//   - manifest:     the scanned repository manifest
//   - sessionFiles: absolute paths to Claude JSONL session files
//   - lookbackDays: the lookback window used to locate those files (stored in
//     the report for informational purposes)
//   - threshold:    activations at or below this value are classified as
//     StatusUnderused rather than StatusActive
func Analyze(manifest *types.Manifest, sessionFiles []string, lookbackDays int, threshold int) (*types.CoverageReport, error) {
	if manifest == nil {
		return nil, fmt.Errorf("coverage: manifest must not be nil")
	}

	summaries, _, err := usage.MatchUsage(manifest, sessionFiles)
	if err != nil {
		return nil, fmt.Errorf("coverage: match usage: %w", err)
	}

	results := make([]types.CoverageResult, 0, len(manifest.Items))

	var summary types.ReportSummary
	summary.TotalItems = len(manifest.Items)

	for _, item := range manifest.Items {
		key := fmt.Sprintf("%s:%s", item.Type, item.Name)
		var usageSummary types.UsageSummary
		if s := summaries[key]; s != nil {
			usageSummary = *s
		}

		status := Classify(item, usageSummary, threshold)

		results = append(results, types.CoverageResult{
			Item:   item,
			Usage:  usageSummary,
			Status: status,
		})

		switch status {
		case types.StatusActive:
			summary.Active++
		case types.StatusUnderused:
			summary.Underused++
		case types.StatusDormant:
			summary.Dormant++
		case types.StatusOrphaned:
			summary.Orphaned++
		}
	}

	report := &types.CoverageReport{
		RepoPath:         manifest.RepoPath,
		LookbackDays:     lookbackDays,
		SessionsAnalyzed: len(sessionFiles),
		Results:          results,
		Summary:          summary,
	}

	return report, nil
}
