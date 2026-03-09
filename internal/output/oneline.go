package output

import (
	"fmt"
	"io"

	"github.com/shuhaozhang/ccoverage/internal/types"
)

// RenderOneLine writes a single-line coverage summary to w. If the manifest
// is empty (no config items found), nothing is written.
func RenderOneLine(report *types.CoverageReport, w io.Writer) {
	s := report.Summary
	if s.TotalItems == 0 {
		return
	}

	needsAttention := s.Underused + s.Dormant
	if needsAttention == 0 {
		fmt.Fprintln(w, "ccoverage: 100% active | all config items healthy")
		return
	}

	pct := s.Active * 100 / s.TotalItems
	fmt.Fprintf(w, "ccoverage: %d%% active | %d items need attention | run \"ccoverage report\" for details\n", pct, needsAttention)
}
