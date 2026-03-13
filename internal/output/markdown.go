package output

import (
	"fmt"
	"io"

	"github.com/ShuhaoZQGG/ccoverage/internal/types"
)

// RenderManifestMarkdown writes a Markdown table of manifest items to w.
func RenderManifestMarkdown(manifest *types.Manifest, w io.Writer) {
	if len(manifest.Items) == 0 {
		fmt.Fprintln(w, "No configuration items found.")
		return
	}

	fmt.Fprintln(w, "| TYPE | NAME |")
	fmt.Fprintln(w, "|------|------|")
	for _, item := range manifest.Items {
		fmt.Fprintf(w, "| %s | %s |\n", item.Type, displayName(item))
	}
	fmt.Fprintf(w, "\n**Total: %d items**\n", len(manifest.Items))
}

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
		if r.Usage.LastSeen != nil {
			lastSeen = r.Usage.LastSeen.Format("2006-01-02")
		}

		pctSessions := "—"
		if report.SessionsAnalyzed > 0 {
			pctSessions = fmt.Sprintf("%.1f%%", float64(r.Usage.UniqueSessions)/float64(report.SessionsAnalyzed)*100)
		}

		fmt.Fprintf(w, "| %s | %s | %s | %d | %d | %s | %s |\n",
			r.Status,
			r.Item.Type,
			displayName(r.Item),
			r.Usage.TotalActivations,
			r.Usage.UniqueSessions,
			pctSessions,
			lastSeen,
		)
	}

	s := report.Summary
	fmt.Fprintf(w, "\n**Total: %d | Active: %d | Underused: %d | Dormant: %d**\n",
		s.TotalItems, s.Active, s.Underused, s.Dormant,
	)
}
