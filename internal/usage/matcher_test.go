package usage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ShuhaoZQGG/ccoverage/internal/types"
)

func TestMatchSingleSession(t *testing.T) {
	sessionPath, err := filepath.Abs("../../testdata/sample_session.jsonl")
	if err != nil {
		t.Fatal(err)
	}

	t.Run("empty session file returns nil", func(t *testing.T) {
		manifest := &types.Manifest{}
		got, err := MatchSingleSession(manifest, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != nil {
			t.Errorf("expected nil report, got %+v", got)
		}
	})

	t.Run("skill item active", func(t *testing.T) {
		manifest := &types.Manifest{
			Items: []types.ManifestItem{
				{Type: types.ConfigSkill, Name: "db-migration"},
			},
		}
		report, err := MatchSingleSession(manifest, sessionPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if report == nil {
			t.Fatal("expected non-nil report")
		}
		if len(report.Items) != 1 {
			t.Fatalf("expected 1 item, got %d", len(report.Items))
		}
		if !report.Items[0].Active {
			t.Errorf("expected db-migration to be active")
		}
		if report.Items[0].Count < 1 {
			t.Errorf("expected count >= 1, got %d", report.Items[0].Count)
		}
	})

	t.Run("skill item inactive", func(t *testing.T) {
		manifest := &types.Manifest{
			Items: []types.ManifestItem{
				{Type: types.ConfigSkill, Name: "nonexistent-skill"},
			},
		}
		report, err := MatchSingleSession(manifest, sessionPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if report.Items[0].Active {
			t.Errorf("expected nonexistent-skill to be inactive")
		}
		if report.Items[0].Count != 0 {
			t.Errorf("expected count 0 for inactive item, got %d", report.Items[0].Count)
		}
	})

	t.Run("mcp item active", func(t *testing.T) {
		manifest := &types.Manifest{
			Items: []types.ManifestItem{
				{Type: types.ConfigMCP, Name: "supabase"},
			},
		}
		report, err := MatchSingleSession(manifest, sessionPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !report.Items[0].Active {
			t.Errorf("expected supabase MCP to be active")
		}
	})

	t.Run("hook item active", func(t *testing.T) {
		manifest := &types.Manifest{
			Items: []types.ManifestItem{
				{Type: types.ConfigHook, Name: "PreToolUse:Bash"},
			},
		}
		report, err := MatchSingleSession(manifest, sessionPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !report.Items[0].Active {
			t.Errorf("expected hook PreToolUse:Bash to be active")
		}
	})

	t.Run("command item active", func(t *testing.T) {
		manifest := &types.Manifest{
			Items: []types.ManifestItem{
				{Type: types.ConfigCommand, Name: "/deploy"},
			},
		}
		report, err := MatchSingleSession(manifest, sessionPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !report.Items[0].Active {
			t.Errorf("expected command /deploy to be active")
		}
	})

	t.Run("skill detected via slash command", func(t *testing.T) {
		// The sample session contains <command-name>/deploy</command-name>.
		// A skill named "deploy" (no slash) should be matched via the
		// dual-emit logic in parseUserLine.
		manifest := &types.Manifest{
			Items: []types.ManifestItem{
				{Type: types.ConfigSkill, Name: "deploy"},
			},
		}
		report, err := MatchSingleSession(manifest, sessionPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if report == nil {
			t.Fatal("expected non-nil report")
		}
		if len(report.Items) != 1 {
			t.Fatalf("expected 1 item, got %d", len(report.Items))
		}
		if !report.Items[0].Active {
			t.Errorf("expected skill 'deploy' to be active via slash command /deploy")
		}
	})

	t.Run("claude.md active via cwd match", func(t *testing.T) {
		// sample_session cwd is /Users/test/myproject; a CLAUDE.md in that
		// directory should be considered active.
		manifest := &types.Manifest{
			Items: []types.ManifestItem{
				{
					Type:    types.ConfigClaudeMD,
					Name:    "CLAUDE.md",
					AbsPath: "/Users/test/myproject/CLAUDE.md",
				},
			},
		}
		report, err := MatchSingleSession(manifest, sessionPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !report.Items[0].Active {
			t.Errorf("expected CLAUDE.md to be active via cwd match")
		}
		if report.Items[0].Count != 1 {
			t.Errorf("expected CLAUDE.md count 1, got %d", report.Items[0].Count)
		}
	})

	t.Run("claude.md active via touched dir", func(t *testing.T) {
		// sample_session touches /Users/test/myproject/src/api; a CLAUDE.md
		// living in src/api should match.
		manifest := &types.Manifest{
			Items: []types.ManifestItem{
				{
					Type:    types.ConfigClaudeMD,
					Name:    "CLAUDE.md",
					AbsPath: "/Users/test/myproject/src/api/CLAUDE.md",
				},
			},
		}
		report, err := MatchSingleSession(manifest, sessionPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !report.Items[0].Active {
			t.Errorf("expected CLAUDE.md to be active via touched dir")
		}
	})

	t.Run("claude.md inactive when not under cwd or touched", func(t *testing.T) {
		manifest := &types.Manifest{
			Items: []types.ManifestItem{
				{
					Type:    types.ConfigClaudeMD,
					Name:    "CLAUDE.md",
					AbsPath: "/Users/other/project/CLAUDE.md",
				},
			},
		}
		report, err := MatchSingleSession(manifest, sessionPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if report.Items[0].Active {
			t.Errorf("expected CLAUDE.md to be inactive")
		}
	})

	t.Run("session id derived from filename", func(t *testing.T) {
		manifest := &types.Manifest{}
		report, err := MatchSingleSession(manifest, sessionPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		wantID := "sample_session"
		if report.SessionID != wantID {
			t.Errorf("got session ID %q, want %q", report.SessionID, wantID)
		}
	})

	t.Run("timestamp from file mtime", func(t *testing.T) {
		manifest := &types.Manifest{}
		report, err := MatchSingleSession(manifest, sessionPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		info, _ := os.Stat(sessionPath)
		if !report.Timestamp.Equal(info.ModTime()) {
			t.Errorf("got timestamp %v, want %v", report.Timestamp, info.ModTime())
		}
	})
}
