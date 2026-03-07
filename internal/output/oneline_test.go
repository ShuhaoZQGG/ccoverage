package output

import (
	"bytes"
	"testing"

	"github.com/shuhaozhang/ccoverage/internal/types"
)

func TestRenderOneLine_AllHealthy(t *testing.T) {
	report := &types.CoverageReport{
		Summary: types.ReportSummary{TotalItems: 5, Active: 5},
	}
	var buf bytes.Buffer
	RenderOneLine(report, &buf)
	want := "ccoverage: 100% active | all config items healthy\n"
	if buf.String() != want {
		t.Errorf("got %q, want %q", buf.String(), want)
	}
}

func TestRenderOneLine_NeedsAttention(t *testing.T) {
	report := &types.CoverageReport{
		Summary: types.ReportSummary{TotalItems: 10, Active: 7, Underused: 1, Dormant: 1, Orphaned: 1},
	}
	var buf bytes.Buffer
	RenderOneLine(report, &buf)
	want := "ccoverage: 70% active | 3 items need attention | run \"ccoverage report\" for details\n"
	if buf.String() != want {
		t.Errorf("got %q, want %q", buf.String(), want)
	}
}

func TestRenderOneLine_Empty(t *testing.T) {
	report := &types.CoverageReport{}
	var buf bytes.Buffer
	RenderOneLine(report, &buf)
	if buf.Len() != 0 {
		t.Errorf("expected no output for empty manifest, got %q", buf.String())
	}
}
