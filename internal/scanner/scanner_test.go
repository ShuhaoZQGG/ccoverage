package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

func isolateHomeDir(t *testing.T) {
	t.Helper()
	fakeHome := t.TempDir()
	orig := userHomeDirFunc
	userHomeDirFunc = func() (string, error) { return fakeHome, nil }
	t.Cleanup(func() { userHomeDirFunc = orig })
}

func TestBuildManifest_EmptyRepo(t *testing.T) {
	isolateHomeDir(t)
	dir := t.TempDir()
	manifest, err := BuildManifest(dir)
	if err != nil {
		t.Fatalf("BuildManifest: %v", err)
	}
	if len(manifest.Items) != 0 {
		t.Errorf("expected 0 items in empty repo, got %d", len(manifest.Items))
	}
}

func TestBuildManifest_WithConfig(t *testing.T) {
	isolateHomeDir(t)
	dir := t.TempDir()

	// Create CLAUDE.md
	if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("# Test"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create .claude/commands/deploy.md
	cmdDir := filepath.Join(dir, ".claude", "commands")
	if err := os.MkdirAll(cmdDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cmdDir, "deploy.md"), []byte("deploy"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create .claude/skills/my-skill/SKILL.md
	skillDir := filepath.Join(dir, ".claude", "skills", "my-skill")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("skill"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create .mcp.json
	mcpJSON := `{"mcpServers": {"supabase": {"command": "npx"}}}`
	if err := os.WriteFile(filepath.Join(dir, ".mcp.json"), []byte(mcpJSON), 0644); err != nil {
		t.Fatal(err)
	}

	// Create .claude/settings.json with hooks
	settingsJSON := `{"hooks": {"PreToolUse": [{"matcher": "Bash", "hooks": [{"type": "command", "command": "echo hi"}]}]}}`
	if err := os.WriteFile(filepath.Join(dir, ".claude", "settings.json"), []byte(settingsJSON), 0644); err != nil {
		t.Fatal(err)
	}

	manifest, err := BuildManifest(dir)
	if err != nil {
		t.Fatalf("BuildManifest: %v", err)
	}

	// Should find: CLAUDE.md, command, skill, MCP server, hook
	if len(manifest.Items) != 5 {
		t.Errorf("expected 5 items, got %d", len(manifest.Items))
		for _, item := range manifest.Items {
			t.Logf("  %s: %s (%s)", item.Type, item.Name, item.Path)
		}
	}

	// Verify types
	typeCount := map[string]int{}
	for _, item := range manifest.Items {
		typeCount[string(item.Type)]++
	}

	for _, expected := range []string{"CLAUDE.md", "Skill", "MCP", "Hook", "Command"} {
		if typeCount[expected] == 0 {
			t.Errorf("expected at least one %s item", expected)
		}
	}
}
