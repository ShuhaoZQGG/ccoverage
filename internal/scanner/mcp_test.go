package scanner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shuhaozhang/ccoverage/internal/types"
)

func TestScanDotMCPJSON(t *testing.T) {
	dir := t.TempDir()

	mcpJSON := `{"mcpServers":{"supabase":{"type":"stdio"},"github":{"type":"sse","url":"http://localhost"}}}`
	if err := os.WriteFile(filepath.Join(dir, ".mcp.json"), []byte(mcpJSON), 0644); err != nil {
		t.Fatal(err)
	}

	items, err := scanDotMCPJSON(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	names := map[string]bool{}
	for _, item := range items {
		names[item.Name] = true
		if item.Type != types.ConfigMCP {
			t.Errorf("expected ConfigMCP, got %s", item.Type)
		}
		if item.Path != ".mcp.json" {
			t.Errorf("expected path .mcp.json, got %s", item.Path)
		}
		if !item.Exists {
			t.Error("expected Exists=true")
		}
	}

	if !names["supabase"] || !names["github"] {
		t.Errorf("expected supabase and github, got %v", names)
	}
}

func TestScanDotMCPJSON_Missing(t *testing.T) {
	dir := t.TempDir()
	items, err := scanDotMCPJSON(dir)
	if err != nil {
		t.Fatal(err)
	}
	if items != nil {
		t.Fatalf("expected nil, got %v", items)
	}
}

func TestScanClaudeJSONMCP_ProjectScoped(t *testing.T) {
	fakeHome := t.TempDir()
	repoPath := t.TempDir()

	absRepo, _ := filepath.Abs(repoPath)

	claudeJSON := `{
		"projects": {
			"` + absRepo + `": {
				"mcpServers": {
					"atlassian": {"type": "sse"},
					"github-work": {"type": "http"}
				}
			}
		}
	}`
	if err := os.WriteFile(filepath.Join(fakeHome, ".claude.json"), []byte(claudeJSON), 0644); err != nil {
		t.Fatal(err)
	}

	orig := userHomeDirFunc
	userHomeDirFunc = func() (string, error) { return fakeHome, nil }
	defer func() { userHomeDirFunc = orig }()

	items, err := scanClaudeJSONMCP(repoPath)
	if err != nil {
		t.Fatal(err)
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	names := map[string]bool{}
	for _, item := range items {
		names[item.Name] = true
		if item.Type != types.ConfigMCP {
			t.Errorf("expected ConfigMCP, got %s", item.Type)
		}
		if item.Path != "~/.claude.json" {
			t.Errorf("expected path ~/.claude.json, got %s", item.Path)
		}
	}

	if !names["atlassian"] || !names["github-work"] {
		t.Errorf("expected atlassian and github-work, got %v", names)
	}
}

func TestScanClaudeJSONMCP_Global(t *testing.T) {
	fakeHome := t.TempDir()
	repoPath := t.TempDir()

	claudeJSON := `{
		"mcpServers": {
			"github-personal": {"type": "stdio"}
		}
	}`
	if err := os.WriteFile(filepath.Join(fakeHome, ".claude.json"), []byte(claudeJSON), 0644); err != nil {
		t.Fatal(err)
	}

	orig := userHomeDirFunc
	userHomeDirFunc = func() (string, error) { return fakeHome, nil }
	defer func() { userHomeDirFunc = orig }()

	items, err := scanClaudeJSONMCP(repoPath)
	if err != nil {
		t.Fatal(err)
	}

	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}

	if items[0].Name != "github-personal" {
		t.Errorf("expected github-personal, got %s", items[0].Name)
	}
}

func TestScanClaudeJSONMCP_ProjectOverridesGlobal(t *testing.T) {
	fakeHome := t.TempDir()
	repoPath := t.TempDir()
	absRepo, _ := filepath.Abs(repoPath)

	claudeJSON := `{
		"mcpServers": {
			"shared": {"type": "stdio", "scope": "global"}
		},
		"projects": {
			"` + absRepo + `": {
				"mcpServers": {
					"shared": {"type": "sse", "scope": "project"}
				}
			}
		}
	}`
	if err := os.WriteFile(filepath.Join(fakeHome, ".claude.json"), []byte(claudeJSON), 0644); err != nil {
		t.Fatal(err)
	}

	orig := userHomeDirFunc
	userHomeDirFunc = func() (string, error) { return fakeHome, nil }
	defer func() { userHomeDirFunc = orig }()

	items, err := scanClaudeJSONMCP(repoPath)
	if err != nil {
		t.Fatal(err)
	}

	if len(items) != 1 {
		t.Fatalf("expected 1 item (deduplicated), got %d", len(items))
	}

	if items[0].Name != "shared" {
		t.Errorf("expected shared, got %s", items[0].Name)
	}
}

func TestScanClaudeJSONMCP_Missing(t *testing.T) {
	fakeHome := t.TempDir()

	orig := userHomeDirFunc
	userHomeDirFunc = func() (string, error) { return fakeHome, nil }
	defer func() { userHomeDirFunc = orig }()

	items, err := scanClaudeJSONMCP(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if items != nil {
		t.Fatalf("expected nil, got %v", items)
	}
}

func TestScanMCP_Deduplication(t *testing.T) {
	repoDir := t.TempDir()
	fakeHome := t.TempDir()

	absRepo, _ := filepath.Abs(repoDir)

	// .mcp.json has "shared" and "repo-only"
	mcpJSON := `{"mcpServers":{"shared":{"type":"stdio"},"repo-only":{"type":"stdio"}}}`
	if err := os.WriteFile(filepath.Join(repoDir, ".mcp.json"), []byte(mcpJSON), 0644); err != nil {
		t.Fatal(err)
	}

	// ~/.claude.json has "shared" and "claude-only"
	claudeJSON := `{
		"projects": {
			"` + absRepo + `": {
				"mcpServers": {
					"shared": {"type": "sse"},
					"claude-only": {"type": "sse"}
				}
			}
		}
	}`
	if err := os.WriteFile(filepath.Join(fakeHome, ".claude.json"), []byte(claudeJSON), 0644); err != nil {
		t.Fatal(err)
	}

	orig := userHomeDirFunc
	userHomeDirFunc = func() (string, error) { return fakeHome, nil }
	defer func() { userHomeDirFunc = orig }()

	items, err := scanMCP(repoDir)
	if err != nil {
		t.Fatal(err)
	}

	// Should have 3: shared (from .mcp.json), repo-only, claude-only
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}

	names := map[string]string{}
	for _, item := range items {
		names[item.Name] = item.Path
	}

	// "shared" should come from .mcp.json, not ~/.claude.json
	if names["shared"] != ".mcp.json" {
		t.Errorf("expected shared from .mcp.json, got path %s", names["shared"])
	}
	if names["repo-only"] != ".mcp.json" {
		t.Errorf("expected repo-only from .mcp.json, got path %s", names["repo-only"])
	}
	if names["claude-only"] != "~/.claude.json" {
		t.Errorf("expected claude-only from ~/.claude.json, got path %s", names["claude-only"])
	}
}
