package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/ShuhaoZQGG/ccoverage/internal/types"
)

func sampleReport() *types.CoverageReport {
	return &types.CoverageReport{
		RepoPath:         "/test/repo",
		LookbackDays:     30,
		SessionsAnalyzed: 10,
		Results: []types.CoverageResult{
			{
				Item:   types.ManifestItem{Type: types.ConfigClaudeMD, Name: "CLAUDE.md"},
				Usage:  types.UsageSummary{TotalActivations: 10, UniqueSessions: 3},
				Status: types.StatusActive,
			},
			{
				Item:   types.ManifestItem{Type: types.ConfigSkill, Name: "db-migration"},
				Usage:  types.UsageSummary{TotalActivations: 0},
				Status: types.StatusDormant,
			},
		},
		Summary: types.ReportSummary{TotalItems: 2, Active: 1, Dormant: 1},
	}
}

func TestRenderText(t *testing.T) {
	var buf bytes.Buffer
	RenderText(sampleReport(), &buf)
	out := buf.String()

	if !strings.Contains(out, "CLAUDE.md") {
		t.Error("expected CLAUDE.md in text output")
	}
	if !strings.Contains(out, "% SESSIONS") {
		t.Error("expected % SESSIONS header")
	}
	if !strings.Contains(out, "30.0%") {
		t.Errorf("expected 30.0%% for CLAUDE.md (3/10), got:\n%s", out)
	}
	if !strings.Contains(out, "0.0%") {
		t.Errorf("expected 0.0%% for db-migration (0/10), got:\n%s", out)
	}
	if !strings.Contains(out, "Total: 2") {
		t.Error("expected summary line")
	}
}

func TestRenderText_Empty(t *testing.T) {
	var buf bytes.Buffer
	RenderText(&types.CoverageReport{}, &buf)
	if !strings.Contains(buf.String(), "No configuration items found") {
		t.Error("expected empty message")
	}
}

func TestRenderJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := RenderJSON(sampleReport(), &buf); err != nil {
		t.Fatal(err)
	}

	var parsed types.CoverageReport
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	if len(parsed.Results) != 2 {
		t.Errorf("expected 2 results, got %d", len(parsed.Results))
	}
}

func TestRenderMarkdown(t *testing.T) {
	var buf bytes.Buffer
	RenderMarkdown(sampleReport(), &buf)
	out := buf.String()

	if !strings.Contains(out, "| Active |") {
		t.Error("expected Active status in markdown")
	}
	if !strings.Contains(out, "% SESSIONS") {
		t.Error("expected % SESSIONS header in markdown")
	}
	if !strings.Contains(out, "30.0%") {
		t.Errorf("expected 30.0%% for CLAUDE.md (3/10), got:\n%s", out)
	}
	if !strings.Contains(out, "0.0%") {
		t.Errorf("expected 0.0%% for db-migration (0/10), got:\n%s", out)
	}
	if !strings.Contains(out, "**Total: 2") {
		t.Error("expected bold summary")
	}
}
