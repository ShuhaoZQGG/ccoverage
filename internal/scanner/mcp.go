package scanner

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ShuhaoZQGG/ccoverage/internal/types"
)

// mcpFile mirrors the top-level structure of .mcp.json.
// The values inside mcpServers are arbitrary objects; we only need the keys.
type mcpFile struct {
	MCPServers map[string]json.RawMessage `json:"mcpServers"`
}

// claudeJSONFile mirrors the relevant structure of ~/.claude.json.
type claudeJSONFile struct {
	MCPServers map[string]json.RawMessage            `json:"mcpServers"`
	Projects   map[string]claudeJSONProjectConfig     `json:"projects"`
}

type claudeJSONProjectConfig struct {
	MCPServers map[string]json.RawMessage `json:"mcpServers"`
}

// scanMCP discovers MCP servers from .mcp.json at the repo root, plus
// project-scoped and global servers from ~/.claude.json. Repo-local
// .mcp.json entries take precedence on name collisions.
func scanMCP(repoPath string) ([]types.ManifestItem, error) {
	repoItems, err := scanDotMCPJSON(repoPath)
	if err != nil {
		return nil, err
	}

	claudeItems, err := scanClaudeJSONMCP(repoPath)
	if err != nil {
		return nil, err
	}

	// Deduplicate: .mcp.json takes precedence over ~/.claude.json.
	seen := make(map[string]bool, len(repoItems))
	for _, item := range repoItems {
		seen[item.Name] = true
	}

	items := repoItems
	for _, item := range claudeItems {
		if !seen[item.Name] {
			seen[item.Name] = true
			items = append(items, item)
		}
	}

	return items, nil
}

// scanDotMCPJSON parses .mcp.json at the repository root.
func scanDotMCPJSON(repoPath string) ([]types.ManifestItem, error) {
	mcpPath := filepath.Join(repoPath, ".mcp.json")

	data, err := os.ReadFile(mcpPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var parsed mcpFile
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("invalid JSON in %s: %w", mcpPath, err)
	}

	if len(parsed.MCPServers) == 0 {
		return nil, nil
	}

	info, statErr := os.Stat(mcpPath)
	var lastMod = zeroTime()
	if statErr == nil {
		lastMod = info.ModTime()
	}

	relPath, relErr := filepath.Rel(repoPath, mcpPath)
	if relErr != nil {
		relPath = ".mcp.json"
	}

	absPath, absErr := filepath.Abs(mcpPath)
	if absErr != nil {
		absPath = mcpPath
	}

	items := make([]types.ManifestItem, 0, len(parsed.MCPServers))

	for serverName := range parsed.MCPServers {
		items = append(items, types.ManifestItem{
			Type:         types.ConfigMCP,
			Name:         serverName,
			Path:         filepath.ToSlash(relPath),
			AbsPath:      absPath,
			LastModified: lastMod,
		})
	}

	return items, nil
}

// userHomeDirFunc is overridable for testing.
var userHomeDirFunc = os.UserHomeDir

// scanClaudeJSONMCP reads ~/.claude.json and returns MCP servers from both
// the project-scoped config (projects[absRepoPath].mcpServers) and the
// global config (top-level mcpServers). Project-scoped entries take
// precedence over global entries on name collisions.
func scanClaudeJSONMCP(repoPath string) ([]types.ManifestItem, error) {
	homeDir, err := userHomeDirFunc()
	if err != nil {
		return nil, nil
	}

	claudePath := filepath.Join(homeDir, ".claude.json")

	data, err := os.ReadFile(claudePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var parsed claudeJSONFile
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("invalid JSON in %s: %w", claudePath, err)
	}

	absRepo, err := filepath.Abs(repoPath)
	if err != nil {
		absRepo = repoPath
	}

	info, statErr := os.Stat(claudePath)
	var lastMod = zeroTime()
	if statErr == nil {
		lastMod = info.ModTime()
	}

	// Collect project-scoped servers first, then global, deduplicating.
	seen := make(map[string]bool)
	var items []types.ManifestItem

	if proj, ok := parsed.Projects[absRepo]; ok {
		for serverName := range proj.MCPServers {
			seen[serverName] = true
			items = append(items, types.ManifestItem{
				Type:         types.ConfigMCP,
				Name:         serverName,
				Path:         "~/.claude.json",
				AbsPath:      claudePath,
				LastModified: lastMod,
			})
		}
	}

	for serverName := range parsed.MCPServers {
		if !seen[serverName] {
			seen[serverName] = true
			items = append(items, types.ManifestItem{
				Type:         types.ConfigMCP,
				Name:         serverName,
				Path:         "~/.claude.json",
				AbsPath:      claudePath,
				LastModified: lastMod,
			})
		}
	}

	return items, nil
}
