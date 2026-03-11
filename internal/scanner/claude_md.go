package scanner

import (
	"os"
	"path/filepath"
	"github.com/ShuhaoZQGG/ccoverage/internal/types"
)

// skipDirs contains directory names that should never be descended into.
var skipDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	".claude":      true,
}

// scanClaudeMD walks the repository tree and returns a ManifestItem for every
// CLAUDE.md file found, skipping .git/, node_modules/, and .claude/.
func scanClaudeMD(repoPath string) ([]types.ManifestItem, error) {
	var items []types.ManifestItem

	err := filepath.WalkDir(repoPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			// Skip paths we cannot access rather than aborting the whole walk.
			return nil
		}

		if d.IsDir() {
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		if d.Name() != "CLAUDE.md" {
			return nil
		}

		info, statErr := d.Info()
		if statErr != nil {
			// Skip files whose metadata we cannot read.
			return nil
		}

		relPath, relErr := filepath.Rel(repoPath, path)
		if relErr != nil {
			return nil
		}

		absPath, absErr := filepath.Abs(path)
		if absErr != nil {
			absPath = path
		}

		items = append(items, types.ManifestItem{
			Type:         types.ConfigClaudeMD,
			Name:         filepath.ToSlash(relPath),
			Path:         filepath.ToSlash(relPath),
			AbsPath:      absPath,
			LastModified: info.ModTime(),
		})

		return nil
	})

	if err != nil {
		return nil, err
	}

	return items, nil
}
