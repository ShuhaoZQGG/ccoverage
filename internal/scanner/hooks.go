package scanner

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/shuhaozhang/ccoverage/internal/types"
)

// hookEntry represents a single hook definition inside an event block.
type hookEntry struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

// hookMatcher represents one element of the per-event array: a matcher string
// paired with a list of hook definitions.
type hookMatcher struct {
	Matcher string      `json:"matcher"`
	Hooks   []hookEntry `json:"hooks"`
}

// settingsFile mirrors the relevant parts of .claude/settings.json.
type settingsFile struct {
	Hooks map[string][]hookMatcher `json:"hooks"`
}

// scanHooks parses .claude/settings.json and returns one ManifestItem per
// event+matcher combination found in the hooks section.
func scanHooks(repoPath string) ([]types.ManifestItem, error) {
	settingsPath := filepath.Join(repoPath, ".claude", "settings.json")

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var parsed settingsFile
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("invalid JSON in %s: %w", settingsPath, err)
	}

	if len(parsed.Hooks) == 0 {
		return nil, nil
	}

	info, statErr := os.Stat(settingsPath)
	lastMod := zeroTime()
	if statErr == nil {
		lastMod = info.ModTime()
	}

	relPath, relErr := filepath.Rel(repoPath, settingsPath)
	if relErr != nil {
		relPath = filepath.Join(".claude", "settings.json")
	}

	absPath, absErr := filepath.Abs(settingsPath)
	if absErr != nil {
		absPath = settingsPath
	}

	var items []types.ManifestItem

	for event, matchers := range parsed.Hooks {
		for _, m := range matchers {
			// Collect the command from the first command-type hook, if present.
			command := ""
			for _, h := range m.Hooks {
				if strings.EqualFold(h.Type, "command") && h.Command != "" {
					command = h.Command
					break
				}
			}

			metadata := map[string]string{
				"command": command,
			}

			items = append(items, types.ManifestItem{
				Type:         types.ConfigHook,
				Name:         fmt.Sprintf("%s:%s", event, m.Matcher),
				Path:         filepath.ToSlash(relPath),
				AbsPath:      absPath,
				LastModified: lastMod,
				Metadata:     metadata,
			})
		}
	}

	return items, nil
}
