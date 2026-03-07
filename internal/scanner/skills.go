package scanner

import (
	"os"
	"path/filepath"
	"github.com/shuhaozhang/ccoverage/internal/types"
)

// scanSkills scans .claude/skills/ for subdirectories that contain a SKILL.md
// file and returns a ManifestItem for each discovered skill.
func scanSkills(repoPath string) ([]types.ManifestItem, error) {
	skillsDir := filepath.Join(repoPath, ".claude", "skills")

	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var items []types.ManifestItem

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillMDPath := filepath.Join(skillsDir, entry.Name(), "SKILL.md")

		info, statErr := os.Stat(skillMDPath)
		if statErr != nil {
			// No SKILL.md present in this subdirectory; skip.
			continue
		}

		relPath, relErr := filepath.Rel(repoPath, skillMDPath)
		if relErr != nil {
			continue
		}

		absPath, absErr := filepath.Abs(skillMDPath)
		if absErr != nil {
			absPath = skillMDPath
		}

		items = append(items, types.ManifestItem{
			Type:         types.ConfigSkill,
			Name:         entry.Name(),
			Path:         filepath.ToSlash(relPath),
			AbsPath:      absPath,
			LastModified: info.ModTime(),
			Exists:       true,
		})
	}

	return items, nil
}
