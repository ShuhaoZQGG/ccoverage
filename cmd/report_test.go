package cmd

import (
	"testing"

	"github.com/shuhaozhang/ccoverage/internal/types"
)

func TestParseStatusFilter(t *testing.T) {
	tests := []struct {
		input    string
		expected map[types.Status]bool
	}{
		{"", nil},
		{"Active", map[types.Status]bool{types.StatusActive: true}},
		{"dormant,orphaned", map[types.Status]bool{types.StatusDormant: true, types.StatusOrphaned: true}},
		{"UNDERUSED", map[types.Status]bool{types.StatusUnderused: true}},
		{" Active , Dormant ", map[types.Status]bool{types.StatusActive: true, types.StatusDormant: true}},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseStatusFilter(tt.input)
			if tt.expected == nil {
				if got != nil {
					t.Fatalf("expected nil, got %v", got)
				}
				return
			}
			if len(got) != len(tt.expected) {
				t.Fatalf("expected %d entries, got %d: %v", len(tt.expected), len(got), got)
			}
			for k := range tt.expected {
				if !got[k] {
					t.Errorf("missing key %q", k)
				}
			}
		})
	}
}

func TestParseTypeFilter(t *testing.T) {
	tests := []struct {
		input    string
		expected map[types.ConfigType]bool
	}{
		{"", nil},
		{"MCP", map[types.ConfigType]bool{types.ConfigMCP: true}},
		{"skill,hook", map[types.ConfigType]bool{types.ConfigSkill: true, types.ConfigHook: true}},
		{"CLAUDE.md", map[types.ConfigType]bool{types.ConfigClaudeMD: true}},
		{"command,mcp", map[types.ConfigType]bool{types.ConfigCommand: true, types.ConfigMCP: true}},
		{"invalid", map[types.ConfigType]bool{}},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseTypeFilter(tt.input)
			if tt.expected == nil {
				if got != nil {
					t.Fatalf("expected nil, got %v", got)
				}
				return
			}
			if len(got) != len(tt.expected) {
				t.Fatalf("expected %d entries, got %d: %v", len(tt.expected), len(got), got)
			}
			for k := range tt.expected {
				if !got[k] {
					t.Errorf("missing key %q", k)
				}
			}
		})
	}
}

func TestFilterReport(t *testing.T) {
	report := &types.CoverageReport{
		RepoPath:         "/test",
		LookbackDays:     30,
		SessionsAnalyzed: 5,
		Results: []types.CoverageResult{
			{Item: types.ManifestItem{Type: types.ConfigMCP, Name: "supabase"}, Status: types.StatusActive},
			{Item: types.ManifestItem{Type: types.ConfigSkill, Name: "deploy"}, Status: types.StatusDormant},
			{Item: types.ManifestItem{Type: types.ConfigHook, Name: "lint"}, Status: types.StatusOrphaned},
			{Item: types.ManifestItem{Type: types.ConfigClaudeMD, Name: "CLAUDE.md"}, Status: types.StatusActive},
		},
		Summary: types.ReportSummary{TotalItems: 4, Active: 2, Dormant: 1, Orphaned: 1},
	}

	t.Run("no filters returns original", func(t *testing.T) {
		got := filterReport(report, "", "")
		if got != report {
			t.Fatal("expected same pointer when no filters")
		}
	})

	t.Run("status filter", func(t *testing.T) {
		got := filterReport(report, "Active", "")
		if len(got.Results) != 2 {
			t.Fatalf("expected 2 results, got %d", len(got.Results))
		}
		if got.Summary.Active != 2 || got.Summary.TotalItems != 2 {
			t.Errorf("summary mismatch: %+v", got.Summary)
		}
	})

	t.Run("type filter", func(t *testing.T) {
		got := filterReport(report, "", "MCP")
		if len(got.Results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(got.Results))
		}
		if got.Results[0].Item.Name != "supabase" {
			t.Errorf("expected supabase, got %s", got.Results[0].Item.Name)
		}
	})

	t.Run("combined filters", func(t *testing.T) {
		got := filterReport(report, "Active", "MCP")
		if len(got.Results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(got.Results))
		}
		if got.Results[0].Item.Name != "supabase" {
			t.Errorf("expected supabase, got %s", got.Results[0].Item.Name)
		}
	})

	t.Run("filter with no matches", func(t *testing.T) {
		got := filterReport(report, "Underused", "")
		if len(got.Results) != 0 {
			t.Fatalf("expected 0 results, got %d", len(got.Results))
		}
		if got.Summary.TotalItems != 0 {
			t.Errorf("expected 0 total items, got %d", got.Summary.TotalItems)
		}
	})

	t.Run("preserves report metadata", func(t *testing.T) {
		got := filterReport(report, "Dormant", "")
		if got.RepoPath != "/test" || got.LookbackDays != 30 || got.SessionsAnalyzed != 5 {
			t.Errorf("metadata not preserved: %+v", got)
		}
	})
}
