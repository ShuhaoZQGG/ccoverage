// Package output provides renderers that write CoverageReport data in
// different formats: terminal text with ANSI colours, JSON, and Markdown.
package output

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/ShuhaoZQGG/ccoverage/internal/types"
)

// ANSI colour escape codes.
const (
	ansiReset   = "\033[0m"
	ansiGreen   = "\033[32m"
	ansiYellow  = "\033[33m"
	ansiRed = "\033[31m"
)

// displayName returns the item name with a " (global)" suffix when the item
// has scope=root metadata, indicating it comes from a global config file.
func displayName(item types.ManifestItem) string {
	if item.Metadata != nil && item.Metadata["scope"] == "root" {
		return item.Name + " (global)"
	}
	return item.Name
}

// isTTY reports whether os.Stdout is connected to a terminal. ANSI colour
// codes are only appropriate when writing directly to a terminal; they corrupt
// output that is piped to a file or another process.
func isTTY() bool {
	info, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

// colorize wraps s in the given ANSI colour escape sequence when useColor is
// true. When false it returns s unchanged.
func colorize(s, color string, useColor bool) string {
	if !useColor {
		return s
	}
	return color + s + ansiReset
}

// statusColor returns the ANSI colour code associated with a Status value.
func statusColor(s types.Status) string {
	switch s {
	case types.StatusActive:
		return ansiGreen
	case types.StatusUnderused:
		return ansiYellow
	case types.StatusDormant:
		return ansiRed
	}
	return ""
}

// RenderManifestText writes a human-readable tabular manifest to w. Only shows
// TYPE and NAME columns since scan has no usage data.
func RenderManifestText(manifest *types.Manifest, w io.Writer) {
	if len(manifest.Items) == 0 {
		fmt.Fprintln(w, "No configuration items found.")
		return
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "TYPE\tNAME")
	fmt.Fprintln(tw, "----\t----")
	for _, item := range manifest.Items {
		fmt.Fprintf(tw, "%s\t%s\n", item.Type, displayName(item))
	}
	tw.Flush()

	fmt.Fprintf(w, "\nTotal: %d items\n", len(manifest.Items))
}

// RenderText writes a human-readable tabular report to w. Columns are aligned
// with text/tabwriter. When w is os.Stdout and stdout is a terminal, status
// values are highlighted with ANSI colours.
func RenderText(report *types.CoverageReport, w io.Writer) {
	if len(report.Results) == 0 {
		fmt.Fprintln(w, "No configuration items found.")
		return
	}

	useColor := isTTY()

	// Render plain text through tabwriter first so column widths are
	// calculated without ANSI escape sequences. Colours are applied
	// after alignment by replacing status words at the start of each line.
	var buf bytes.Buffer
	tw := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)

	// Header row.
	fmt.Fprintln(tw, "STATUS\tTYPE\tNAME\tACTIVATIONS\tSESSIONS\t% SESSIONS\tLAST SEEN")
	fmt.Fprintln(tw, "------\t----\t----\t-----------\t--------\t----------\t---------")

	for _, r := range report.Results {
		lastSeen := "—"
		if r.Usage.LastSeen != nil {
			lastSeen = r.Usage.LastSeen.Format("2006-01-02")
		}

		pctSessions := "—"
		if report.SessionsAnalyzed > 0 {
			pctSessions = fmt.Sprintf("%.1f%%", float64(r.Usage.UniqueSessions)/float64(report.SessionsAnalyzed)*100)
		}

		fmt.Fprintf(tw, "%s\t%s\t%s\t%d\t%d\t%s\t%s\n",
			r.Status,
			r.Item.Type,
			displayName(r.Item),
			r.Usage.TotalActivations,
			r.Usage.UniqueSessions,
			pctSessions,
			lastSeen,
		)
	}

	tw.Flush()

	// Apply ANSI colours to status words at the start of each line.
	output := buf.String()
	if useColor {
		for _, status := range []types.Status{types.StatusActive, types.StatusUnderused, types.StatusDormant} {
			plain := string(status)
			colored := colorize(plain, statusColor(status), true)
			output = strings.ReplaceAll(output, "\n"+plain+" ", "\n"+colored+" ")
		}
	}
	fmt.Fprint(w, output)

	s := report.Summary
	fmt.Fprintf(w, "\nTotal: %d | Active: %d | Underused: %d | Dormant: %d\n",
		s.TotalItems, s.Active, s.Underused, s.Dormant,
	)
}
