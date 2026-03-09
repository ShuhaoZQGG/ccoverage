package scanner

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/shuhaozhang/ccoverage/internal/types"
)

// lspExtRe matches backtick-wrapped extensions like `.go` in README files.
var lspExtRe = regexp.MustCompile("`(\\.[a-zA-Z0-9]+)`")

// pluginSettingsFile mirrors the enabledPlugins section of settings files.
type pluginSettingsFile struct {
	EnabledPlugins map[string]bool `json:"enabledPlugins"`
}

// scanPlugins discovers enabled plugins from settings files and inspects the
// plugin cache to determine each plugin's components.
func scanPlugins(repoPath string) ([]types.ManifestItem, error) {
	homeDir, err := userHomeDirFunc()
	if err != nil {
		return nil, nil
	}

	// Settings files in priority order (first wins on duplicates).
	settingsPaths := []string{
		filepath.Join(repoPath, ".claude", "settings.local.json"),
		filepath.Join(repoPath, ".claude", "settings.json"),
		filepath.Join(homeDir, ".claude", "settings.json"),
	}

	seen := make(map[string]bool)
	var items []types.ManifestItem

	for _, settingsPath := range settingsPaths {
		data, readErr := os.ReadFile(settingsPath)
		if readErr != nil {
			continue
		}

		var parsed pluginSettingsFile
		if jsonErr := json.Unmarshal(data, &parsed); jsonErr != nil {
			continue
		}

		info, statErr := os.Stat(settingsPath)
		lastMod := zeroTime()
		if statErr == nil {
			lastMod = info.ModTime()
		}

		relPath, relErr := filepath.Rel(repoPath, settingsPath)
		if relErr != nil {
			relPath = settingsPath
		}
		// Use ~ shorthand for home-dir paths.
		if strings.HasPrefix(settingsPath, homeDir) {
			relPath = "~" + settingsPath[len(homeDir):]
		}

		absPath, absErr := filepath.Abs(settingsPath)
		if absErr != nil {
			absPath = settingsPath
		}

		for key, enabled := range parsed.EnabledPlugins {
			if !enabled || seen[key] {
				continue
			}
			seen[key] = true

			name, marketplace := parsePluginKey(key)
			metadata := map[string]string{
				"plugin":      name,
				"marketplace": marketplace,
			}

			// Inspect the plugin cache for components.
			components, mcpServers, skillNames, cmdNames, lspExts := inspectPluginCache(homeDir, marketplace, name)
			if components != "" {
				metadata["components"] = components
			}
			if mcpServers != "" {
				metadata["mcp_servers"] = mcpServers
			}
			if skillNames != "" {
				metadata["skill_names"] = skillNames
			}
			if cmdNames != "" {
				metadata["command_names"] = cmdNames
			}
			if lspExts != "" {
				metadata["lsp_extensions"] = lspExts
			}
			if strings.HasPrefix(settingsPath, homeDir) {
				metadata["scope"] = "root"
			}

			items = append(items, types.ManifestItem{
				Type:         types.ConfigPlugin,
				Name:         key,
				Path:         filepath.ToSlash(relPath),
				AbsPath:      absPath,
				LastModified: lastMod,
				Metadata:     metadata,
			})
		}
	}

	return items, nil
}

// parsePluginKey splits "name@marketplace" into its components.
// If no @ is present, marketplace defaults to "unknown".
func parsePluginKey(key string) (name, marketplace string) {
	if idx := strings.LastIndex(key, "@"); idx > 0 {
		return key[:idx], key[idx+1:]
	}
	return key, "unknown"
}

// inspectPluginCache looks inside ~/.claude/plugins/cache/<marketplace>/<pluginName>/
// and returns a comma-separated component list, MCP server names, skill names,
// command names, and LSP supported extensions. Skill and command names include
// both the bare name and a "pluginName:name" prefixed form so that
// namespace-prefixed invocations in session logs can be matched.
func inspectPluginCache(homeDir, marketplace, pluginName string) (components, mcpServers, skillNames, commandNames, lspExtensions string) {
	cacheDir := filepath.Join(homeDir, ".claude", "plugins", "cache", marketplace, pluginName)
	// Find version subdirectory (take first match).
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		return "unknown", "", "", "", ""
	}

	var versionDir string
	for _, e := range entries {
		if e.IsDir() {
			versionDir = filepath.Join(cacheDir, e.Name())
			break
		}
	}
	if versionDir == "" {
		return "unknown", "", "", "", ""
	}

	var comps []string
	var servers []string
	var skills []string
	var cmds []string

	// Check for .mcp.json
	if data, readErr := os.ReadFile(filepath.Join(versionDir, ".mcp.json")); readErr == nil {
		var mcp mcpFile
		if json.Unmarshal(data, &mcp) == nil && len(mcp.MCPServers) > 0 {
			comps = append(comps, "mcp")
			for serverName := range mcp.MCPServers {
				servers = append(servers, serverName)
			}
		}
	}

	// Check for skills/*/SKILL.md
	// Store both bare name and "pluginName:name" prefixed form so that
	// namespace-prefixed invocations in session logs can be matched.
	skillsDir := filepath.Join(versionDir, "skills")
	if skillEntries, readErr := os.ReadDir(skillsDir); readErr == nil {
		for _, se := range skillEntries {
			if !se.IsDir() {
				continue
			}
			skillMD := filepath.Join(skillsDir, se.Name(), "SKILL.md")
			if _, statErr := os.Stat(skillMD); statErr == nil {
				bare := se.Name()
				skills = append(skills, bare, pluginName+":"+bare)
			}
		}
		if len(skills) > 0 {
			comps = append(comps, "skills")
		}
	}

	// Check for .lsp.json or plugin name ending in "-lsp" (LSP-only plugins
	// like gopls-lsp and swift-lsp have no .lsp.json in their cache).
	hasLSP := false
	if _, statErr := os.Stat(filepath.Join(versionDir, ".lsp.json")); statErr == nil {
		hasLSP = true
	} else if strings.HasSuffix(pluginName, "-lsp") {
		hasLSP = true
	}
	if hasLSP {
		comps = append(comps, "lsp")
		// Try to parse supported extensions from README.md.
		lspExtensions = parseLSPExtensions(filepath.Join(versionDir, "README.md"))
	}

	// Check for commands/*.md — extract individual command names with
	// both bare and "pluginName:name" prefixed forms.
	commandsDir := filepath.Join(versionDir, "commands")
	if cmdEntries, readErr := os.ReadDir(commandsDir); readErr == nil {
		for _, ce := range cmdEntries {
			if ce.IsDir() || !strings.HasSuffix(ce.Name(), ".md") {
				continue
			}
			bare := strings.TrimSuffix(ce.Name(), ".md")
			cmds = append(cmds, bare, pluginName+":"+bare)
		}
		if len(cmds) > 0 {
			comps = append(comps, "commands")
		}
	} else if _, statErr := os.Stat(commandsDir); statErr == nil {
		// commands/ directory exists but couldn't be read — still note it.
		comps = append(comps, "commands")
	}

	// Check for hooks/
	if _, statErr := os.Stat(filepath.Join(versionDir, "hooks")); statErr == nil {
		comps = append(comps, "hooks")
	}

	// Check for agents/
	if _, statErr := os.Stat(filepath.Join(versionDir, "agents")); statErr == nil {
		comps = append(comps, "agents")
	}

	if len(comps) == 0 {
		return "unknown", "", "", "", ""
	}

	return strings.Join(comps, ","), strings.Join(servers, ","), strings.Join(skills, ","), strings.Join(cmds, ","), lspExtensions
}

// parseLSPExtensions reads a plugin README.md and extracts supported file
// extensions from the "## Supported Extensions" section. Returns a
// comma-separated string of extensions (e.g. ".go" or ".ts,.tsx,.js,.jsx").
func parseLSPExtensions(readmePath string) string {
	f, err := os.Open(readmePath)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	inSection := false
	var exts []string

	for scanner.Scan() {
		line := scanner.Text()

		if inSection {
			// Stop at the next heading.
			if strings.HasPrefix(line, "## ") || strings.HasPrefix(line, "# ") {
				break
			}
			matches := lspExtRe.FindAllStringSubmatch(line, -1)
			for _, m := range matches {
				exts = append(exts, m[1])
			}
		}

		if strings.TrimSpace(line) == "## Supported Extensions" {
			inSection = true
		}
	}

	return strings.Join(exts, ",")
}
