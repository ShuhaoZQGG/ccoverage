package scanner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/shuhaozhang/ccoverage/internal/types"
)

func TestScanPlugins_NoSettings(t *testing.T) {
	isolateHomeDir(t)
	dir := t.TempDir()
	items, err := scanPlugins(dir)
	if err != nil {
		t.Fatalf("scanPlugins: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
}

func TestScanPlugins_BasicDetection(t *testing.T) {
	fakeHome := t.TempDir()
	orig := userHomeDirFunc
	userHomeDirFunc = func() (string, error) { return fakeHome, nil }
	t.Cleanup(func() { userHomeDirFunc = orig })

	dir := t.TempDir()

	// Create .claude/settings.json with enabledPlugins.
	claudeDir := filepath.Join(dir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}

	settings := map[string]interface{}{
		"enabledPlugins": map[string]bool{
			"slack@claude-plugins-official":      true,
			"typescript-lsp@claude-plugins-official": true,
			"disabled-plugin@marketplace":        false,
		},
	}
	data, _ := json.Marshal(settings)
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.json"), data, 0644); err != nil {
		t.Fatal(err)
	}

	items, err := scanPlugins(dir)
	if err != nil {
		t.Fatalf("scanPlugins: %v", err)
	}

	// Should find 2 items (disabled-plugin excluded).
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
		for _, item := range items {
			t.Logf("  %s: %s", item.Type, item.Name)
		}
	}

	for _, item := range items {
		if item.Type != types.ConfigPlugin {
			t.Errorf("expected ConfigPlugin, got %s", item.Type)
		}
		// Components should be "unknown" since no cache exists.
		if item.Metadata["components"] != "unknown" {
			t.Errorf("expected components=unknown (no cache), got %q", item.Metadata["components"])
		}
		// Project-level plugins should not have scope metadata.
		if scope, ok := item.Metadata["scope"]; ok {
			t.Errorf("project-level plugin should not have scope, got %q", scope)
		}
	}
}

func TestScanPlugins_CacheInspection(t *testing.T) {
	fakeHome := t.TempDir()
	orig := userHomeDirFunc
	userHomeDirFunc = func() (string, error) { return fakeHome, nil }
	t.Cleanup(func() { userHomeDirFunc = orig })

	dir := t.TempDir()

	// Create settings with one plugin.
	claudeDir := filepath.Join(dir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}

	settings := map[string]interface{}{
		"enabledPlugins": map[string]bool{
			"slack@claude-plugins-official": true,
		},
	}
	data, _ := json.Marshal(settings)
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.json"), data, 0644); err != nil {
		t.Fatal(err)
	}

	// Create plugin cache with components.
	cacheDir := filepath.Join(fakeHome, ".claude", "plugins", "cache", "claude-plugins-official", "slack", "1.0.0")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Add .mcp.json
	mcpJSON := `{"mcpServers": {"slack-mcp": {"command": "npx"}}}`
	if err := os.WriteFile(filepath.Join(cacheDir, ".mcp.json"), []byte(mcpJSON), 0644); err != nil {
		t.Fatal(err)
	}

	// Add skills
	skillDir := filepath.Join(cacheDir, "skills", "slack-notify")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("skill"), 0644); err != nil {
		t.Fatal(err)
	}

	// Add .lsp.json
	if err := os.WriteFile(filepath.Join(cacheDir, ".lsp.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	items, err := scanPlugins(dir)
	if err != nil {
		t.Fatalf("scanPlugins: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}

	item := items[0]
	if item.Metadata["mcp_servers"] != "slack-mcp" {
		t.Errorf("expected mcp_servers=slack-mcp, got %q", item.Metadata["mcp_servers"])
	}
	if item.Metadata["skill_names"] != "slack-notify,slack:slack-notify" {
		t.Errorf("expected skill_names=slack-notify,slack:slack-notify, got %q", item.Metadata["skill_names"])
	}
	if item.Metadata["components"] != "mcp,skills,lsp" {
		t.Errorf("expected components=mcp,skills,lsp, got %q", item.Metadata["components"])
	}
}

func TestScanPlugins_DedupAcrossScopes(t *testing.T) {
	fakeHome := t.TempDir()
	orig := userHomeDirFunc
	userHomeDirFunc = func() (string, error) { return fakeHome, nil }
	t.Cleanup(func() { userHomeDirFunc = orig })

	dir := t.TempDir()

	// Enable same plugin in both repo and global settings.
	for _, path := range []string{
		filepath.Join(dir, ".claude", "settings.json"),
		filepath.Join(fakeHome, ".claude", "settings.json"),
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		settings := map[string]interface{}{
			"enabledPlugins": map[string]bool{
				"slack@claude-plugins-official": true,
			},
		}
		data, _ := json.Marshal(settings)
		if err := os.WriteFile(path, data, 0644); err != nil {
			t.Fatal(err)
		}
	}

	items, err := scanPlugins(dir)
	if err != nil {
		t.Fatalf("scanPlugins: %v", err)
	}

	// Should find only 1 item (dedup: first settings file wins).
	if len(items) != 1 {
		t.Errorf("expected 1 item (dedup), got %d", len(items))
	}
}

func TestParsePluginKey(t *testing.T) {
	tests := []struct {
		key         string
		wantName    string
		wantMarket  string
	}{
		{"slack@claude-plugins-official", "slack", "claude-plugins-official"},
		{"my-plugin@custom", "my-plugin", "custom"},
		{"no-at-sign", "no-at-sign", "unknown"},
		{"multi@at@signs", "multi@at", "signs"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			name, market := parsePluginKey(tt.key)
			if name != tt.wantName {
				t.Errorf("name: got %q, want %q", name, tt.wantName)
			}
			if market != tt.wantMarket {
				t.Errorf("marketplace: got %q, want %q", market, tt.wantMarket)
			}
		})
	}
}

func TestBuildManifest_WithPlugins(t *testing.T) {
	fakeHome := t.TempDir()
	orig := userHomeDirFunc
	userHomeDirFunc = func() (string, error) { return fakeHome, nil }
	t.Cleanup(func() { userHomeDirFunc = orig })

	dir := t.TempDir()

	// Create .claude/settings.json with a plugin.
	claudeDir := filepath.Join(dir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	settings := map[string]interface{}{
		"enabledPlugins": map[string]bool{
			"test-plugin@marketplace": true,
		},
	}
	data, _ := json.Marshal(settings)
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.json"), data, 0644); err != nil {
		t.Fatal(err)
	}

	manifest, err := BuildManifest(dir)
	if err != nil {
		t.Fatalf("BuildManifest: %v", err)
	}

	var pluginItems []types.ManifestItem
	for _, item := range manifest.Items {
		if item.Type == types.ConfigPlugin {
			pluginItems = append(pluginItems, item)
		}
	}

	if len(pluginItems) != 1 {
		t.Errorf("expected 1 plugin item, got %d", len(pluginItems))
	}
	if len(pluginItems) > 0 && pluginItems[0].Name != "test-plugin@marketplace" {
		t.Errorf("expected name test-plugin@marketplace, got %q", pluginItems[0].Name)
	}
}

func TestScanPlugins_SettingsLocalPrecedence(t *testing.T) {
	fakeHome := t.TempDir()
	orig := userHomeDirFunc
	userHomeDirFunc = func() (string, error) { return fakeHome, nil }
	t.Cleanup(func() { userHomeDirFunc = orig })

	dir := t.TempDir()
	claudeDir := filepath.Join(dir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}

	// settings.local.json has plugin A, settings.json has A and B.
	localSettings := map[string]interface{}{
		"enabledPlugins": map[string]bool{
			"pluginA@market": true,
		},
	}
	data, _ := json.Marshal(localSettings)
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.local.json"), data, 0644); err != nil {
		t.Fatal(err)
	}

	repoSettings := map[string]interface{}{
		"enabledPlugins": map[string]bool{
			"pluginA@market": true,
			"pluginB@market": true,
		},
	}
	data, _ = json.Marshal(repoSettings)
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.json"), data, 0644); err != nil {
		t.Fatal(err)
	}

	items, err := scanPlugins(dir)
	if err != nil {
		t.Fatalf("scanPlugins: %v", err)
	}

	// Should find 2 items: A from local, B from repo settings.
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	names := []string{items[0].Name, items[1].Name}
	sort.Strings(names)
	if names[0] != "pluginA@market" || names[1] != "pluginB@market" {
		t.Errorf("unexpected names: %v", names)
	}

	// pluginA should come from settings.local.json (first wins).
	for _, item := range items {
		if item.Name == "pluginA@market" {
			if item.Path != ".claude/settings.local.json" {
				t.Errorf("pluginA should come from settings.local.json, got %q", item.Path)
			}
		}
		// Both plugins are project-level — no scope metadata.
		if scope, ok := item.Metadata["scope"]; ok {
			t.Errorf("project-level plugin %s should not have scope, got %q", item.Name, scope)
		}
	}
}

func TestScanPlugins_GlobalScope(t *testing.T) {
	fakeHome := t.TempDir()
	orig := userHomeDirFunc
	userHomeDirFunc = func() (string, error) { return fakeHome, nil }
	t.Cleanup(func() { userHomeDirFunc = orig })

	dir := t.TempDir()

	// Create global settings with a plugin.
	globalClaudeDir := filepath.Join(fakeHome, ".claude")
	if err := os.MkdirAll(globalClaudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	settings := map[string]interface{}{
		"enabledPlugins": map[string]bool{
			"global-plugin@marketplace": true,
		},
	}
	data, _ := json.Marshal(settings)
	if err := os.WriteFile(filepath.Join(globalClaudeDir, "settings.json"), data, 0644); err != nil {
		t.Fatal(err)
	}

	items, err := scanPlugins(dir)
	if err != nil {
		t.Fatalf("scanPlugins: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}

	if items[0].Metadata["scope"] != "root" {
		t.Errorf("global plugin should have scope=root, got %q", items[0].Metadata["scope"])
	}
}
