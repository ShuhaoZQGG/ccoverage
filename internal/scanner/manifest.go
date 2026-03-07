package scanner

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/shuhaozhang/ccoverage/internal/types"
)

// zeroTime returns the zero value of time.Time and is used as a fallback when
// os.Stat fails for a file we already know exists.
func zeroTime() time.Time {
	return time.Time{}
}

// BuildManifest resolves repoPath to an absolute path, runs all sub-scanners,
// aggregates their results, and returns a populated Manifest.
func BuildManifest(repoPath string) (*types.Manifest, error) {
	absRepo, err := filepath.Abs(repoPath)
	if err != nil {
		return nil, fmt.Errorf("scanner: resolve repo path: %w", err)
	}

	type scanFn func(string) ([]types.ManifestItem, error)

	scanners := []struct {
		name string
		fn   scanFn
	}{
		{"claude_md", scanClaudeMD},
		{"skills", scanSkills},
		{"mcp", scanMCP},
		{"hooks", scanHooks},
		{"commands", scanCommands},
	}

	manifest := &types.Manifest{
		RepoPath:  absRepo,
		ScannedAt: time.Now(),
	}

	for _, s := range scanners {
		items, scanErr := s.fn(absRepo)
		if scanErr != nil {
			return nil, fmt.Errorf("scanner: %s: %w", s.name, scanErr)
		}
		manifest.Items = append(manifest.Items, items...)
	}

	return manifest, nil
}
