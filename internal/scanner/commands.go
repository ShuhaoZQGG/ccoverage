package scanner

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/ShuhaoZQGG/ccoverage/internal/types"
)

// scanCommands scans .claude/commands/ for Markdown files and returns one
// ManifestItem per file. The item Name is "/filename" (no .md extension),
// matching the slash-command convention used by Claude Code.
func scanCommands(repoPath string) ([]types.ManifestItem, error) {
	commandsDir := filepath.Join(repoPath, ".claude", "commands")

	entries, err := os.ReadDir(commandsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var items []types.ManifestItem

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		filePath := filepath.Join(commandsDir, entry.Name())

		info, statErr := entry.Info()
		if statErr != nil {
			continue
		}

		relPath, relErr := filepath.Rel(repoPath, filePath)
		if relErr != nil {
			continue
		}

		absPath, absErr := filepath.Abs(filePath)
		if absErr != nil {
			absPath = filePath
		}

		// Strip the .md extension and prepend "/" to form the command name.
		baseName := strings.TrimSuffix(entry.Name(), ".md")
		name := "/" + baseName

		items = append(items, types.ManifestItem{
			Type:         types.ConfigCommand,
			Name:         name,
			Path:         filepath.ToSlash(relPath),
			AbsPath:      absPath,
			LastModified: info.ModTime(),
		})
	}

	return items, nil
}
