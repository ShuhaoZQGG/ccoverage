package output

import (
	"fmt"
	"io"

	"github.com/shuhaozhang/ccoverage/internal/types"
)

// RenderMarkdown writes a GitHub-flavoured Markdown table to w. The output
// uses the same columns as RenderText but without ANSI colour codes, making it
// suitable for embedding in Markdown documents and pull-request comments.
func RenderMarkdown(report *types.CoverageReport, w io.Writer) {
	if len(report.Results) == 0 {
		fmt.Fprintln(w, "No configuration items found.")
		return
	}

	// Table header.
	fmt.Fprintln(w, "| STATUS | TYPE | NAME | ACTIVATIONS | SESSIONS | % SESSIONS | LAST SEEN |")
	fmt.Fprintln(w, "|--------|------|------|-------------|----------|------------|-----------|")

	for _, r := range report.Results {
		lastSeen := "—"
		if !r.Usage.LastSeen.IsZero() {
			lastSeen = r.Usage.LastSeen.Format("2006-01-02")
		}

		pctSessions := "—"
		if report.SessionsAnalyzed > 0 {
			pctSessions = fmt.Sprintf("%.1f%%", float64(r.Usage.UniqueSessions)/float64(report.SessionsAnalyzed)*100)
		}

		fmt.Fprintf(w, "| %s | %s | %s | %d | %d | %s | %s |\n",
			r.Status,
			r.Item.Type,
			r.Item.Name,
			r.Usage.TotalActivations,
			r.Usage.UniqueSessions,
			pctSessions,
			lastSeen,
		)
	}

	s := report.Summary
	fmt.Fprintf(w, "\n**Total: %d | Active: %d | Underused: %d | Dormant: %d | Orphaned: %d**\n",
		s.TotalItems, s.Active, s.Underused, s.Dormant, s.Orphaned,
	)
}
