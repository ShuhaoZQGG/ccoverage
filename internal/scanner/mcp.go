package scanner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"github.com/shuhaozhang/ccoverage/internal/types"
)

// mcpFile mirrors the top-level structure of .mcp.json.
// The values inside mcpServers are arbitrary objects; we only need the keys.
type mcpFile struct {
	MCPServers map[string]json.RawMessage `json:"mcpServers"`
}

// scanMCP parses .mcp.json at the repository root and returns one ManifestItem
// per entry in the mcpServers map.
func scanMCP(repoPath string) ([]types.ManifestItem, error) {
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
		return nil, err
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
			Exists:       true,
		})
	}

	return items, nil
}
