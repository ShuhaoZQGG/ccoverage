package scanner

import (
	"os"
	"path/filepath"

	"github.com/ShuhaoZQGG/ccoverage/internal/types"
)

// scanSkills scans both project-level (.claude/skills/) and root-level
// (~/.claude/skills/) for subdirectories that contain a SKILL.md file.
// Project-level skills shadow root-level skills with the same name.
func scanSkills(repoPath string) ([]types.ManifestItem, error) {
	projectDir := filepath.Join(repoPath, ".claude", "skills")
	projectItems, err := scanSkillsDir(projectDir, repoPath, false)
	if err != nil {
		return nil, err
	}

	// Build set of project-level skill names for dedup.
	seen := make(map[string]bool, len(projectItems))
	for _, item := range projectItems {
		seen[item.Name] = true
	}

	homeDir, err := userHomeDirFunc()
	if err != nil {
		// Can't resolve home dir; return project-level skills only.
		return projectItems, nil
	}

	rootDir := filepath.Join(homeDir, ".claude", "skills")
	rootItems, err := scanSkillsDir(rootDir, repoPath, true)
	if err != nil {
		return projectItems, nil
	}

	for _, item := range rootItems {
		if !seen[item.Name] {
			projectItems = append(projectItems, item)
		}
	}

	return projectItems, nil
}

// scanSkillsDir scans a single skills directory for subdirectories containing
// SKILL.md. If isRoot is true, paths use ~/.claude/skills/ prefix and metadata
// includes scope=root.
func scanSkillsDir(skillsDir, repoPath string, isRoot bool) ([]types.ManifestItem, error) {
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
			continue
		}

		absPath, absErr := filepath.Abs(skillMDPath)
		if absErr != nil {
			absPath = skillMDPath
		}

		var displayPath string
		var metadata map[string]string

		if isRoot {
			displayPath = filepath.ToSlash(filepath.Join("~/.claude/skills", entry.Name(), "SKILL.md"))
			metadata = map[string]string{"scope": "root"}
		} else {
			relPath, relErr := filepath.Rel(repoPath, skillMDPath)
			if relErr != nil {
				continue
			}
			displayPath = filepath.ToSlash(relPath)
		}

		items = append(items, types.ManifestItem{
			Type:         types.ConfigSkill,
			Name:         entry.Name(),
			Path:         displayPath,
			AbsPath:      absPath,
			LastModified: info.ModTime(),
			Metadata:     metadata,
		})
	}

	return items, nil
}
