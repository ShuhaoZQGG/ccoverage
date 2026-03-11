package scanner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ShuhaoZQGG/ccoverage/internal/types"
)

func TestScanSkillsDir_ProjectLevel(t *testing.T) {
	repo := t.TempDir()
	skillDir := filepath.Join(repo, ".claude", "skills", "my-skill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# My Skill"), 0o644); err != nil {
		t.Fatal(err)
	}

	items, err := scanSkillsDir(filepath.Join(repo, ".claude", "skills"), repo, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Name != "my-skill" {
		t.Errorf("expected name my-skill, got %s", items[0].Name)
	}
	if items[0].Path != ".claude/skills/my-skill/SKILL.md" {
		t.Errorf("unexpected path: %s", items[0].Path)
	}
	if items[0].Metadata != nil {
		t.Errorf("project-level skill should have nil metadata, got %v", items[0].Metadata)
	}
}

func TestScanSkillsDir_RootLevel(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "my-root-skill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# Root Skill"), 0o644); err != nil {
		t.Fatal(err)
	}

	items, err := scanSkillsDir(dir, "/fake/repo", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Name != "my-root-skill" {
		t.Errorf("expected name my-root-skill, got %s", items[0].Name)
	}
	if items[0].Path != "~/.claude/skills/my-root-skill/SKILL.md" {
		t.Errorf("unexpected path: %s", items[0].Path)
	}
	if items[0].Metadata == nil || items[0].Metadata["scope"] != "root" {
		t.Errorf("root-level skill should have scope=root metadata, got %v", items[0].Metadata)
	}
}

func TestScanSkillsDir_MissingDir(t *testing.T) {
	items, err := scanSkillsDir("/nonexistent/path", "/fake", false)
	if err != nil {
		t.Fatal(err)
	}
	if items != nil {
		t.Errorf("expected nil for missing dir, got %v", items)
	}
}

func TestScanSkillsDir_SkipNonDirs(t *testing.T) {
	dir := t.TempDir()
	// Create a regular file (not a directory) in the skills dir.
	if err := os.WriteFile(filepath.Join(dir, "not-a-skill.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}

	items, err := scanSkillsDir(dir, "/fake", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
}

func TestScanSkills_ProjectShadowsRoot(t *testing.T) {
	// We can't easily mock os.UserHomeDir, so test the dedup logic via
	// scanSkillsDir directly and verify the shadowing behavior.
	repo := t.TempDir()
	rootDir := t.TempDir()

	// Create same-named skill in both locations.
	for _, base := range []string{
		filepath.Join(repo, ".claude", "skills", "shared-skill"),
		filepath.Join(rootDir, "shared-skill"),
	} {
		if err := os.MkdirAll(base, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(base, "SKILL.md"), []byte("# Shared"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// Also create a root-only skill.
	rootOnly := filepath.Join(rootDir, "root-only")
	if err := os.MkdirAll(rootOnly, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rootOnly, "SKILL.md"), []byte("# Root Only"), 0o644); err != nil {
		t.Fatal(err)
	}

	projectItems, err := scanSkillsDir(filepath.Join(repo, ".claude", "skills"), repo, false)
	if err != nil {
		t.Fatal(err)
	}

	seen := make(map[string]bool, len(projectItems))
	for _, item := range projectItems {
		seen[item.Name] = true
	}

	rootItems, err := scanSkillsDir(rootDir, repo, true)
	if err != nil {
		t.Fatal(err)
	}

	// Apply same dedup logic as scanSkills.
	combined := append([]types.ManifestItem{}, projectItems...)
	for _, item := range rootItems {
		if !seen[item.Name] {
			combined = append(combined, item)
		}
	}

	if len(combined) != 2 {
		t.Fatalf("expected 2 items (shared-skill from project + root-only), got %d", len(combined))
	}

	// shared-skill should be project-level (no metadata).
	for _, item := range combined {
		if item.Name == "shared-skill" {
			if item.Metadata != nil {
				t.Errorf("shared-skill should be project-level (shadowed), got metadata %v", item.Metadata)
			}
		}
		if item.Name == "root-only" {
			if item.Metadata == nil || item.Metadata["scope"] != "root" {
				t.Errorf("root-only should have scope=root metadata")
			}
		}
	}
}
