// Package usage locates and parses Claude session JSONL files to extract
// usage events for manifest items.
package usage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// encodeRepoPath converts an absolute repository path into the encoded form
// that Claude uses when naming project directories.
//
// The encoding replaces every "/" separator with "-" and strips the leading
// "-" that would otherwise result from the leading "/" in an absolute path.
//
// Example:
//
//	/Users/foo/project  →  Users-foo-project
func encodeRepoPath(repoPath string) string {
	return strings.ReplaceAll(repoPath, "/", "-")
}

// LocateSessionFiles returns the absolute paths of all *.jsonl session files
// belonging to repoPath that were modified within the past lookbackDays days.
//
// Claude stores session files under:
//
//	~/.claude/projects/<encoded-repo-path>/*.jsonl
//
// Subagent sessions may reside one level deeper:
//
//	~/.claude/projects/<encoded-repo-path>/*/*.jsonl
func LocateSessionFiles(repoPath string, lookbackDays int) ([]string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("usage: resolve home directory: %w", err)
	}

	encoded := encodeRepoPath(repoPath)
	projectDir := filepath.Join(homeDir, ".claude", "projects", encoded)

	cutoff := time.Now().AddDate(0, 0, -lookbackDays)

	var files []string

	// Collect top-level *.jsonl files and subagent *.jsonl files found one
	// directory beneath the project directory.
	patterns := []string{
		filepath.Join(projectDir, "*.jsonl"),
		filepath.Join(projectDir, "*", "*.jsonl"),
	}

	for _, pattern := range patterns {
		matches, globErr := filepath.Glob(pattern)
		if globErr != nil {
			// Glob only returns an error for a malformed pattern; treat as
			// non-fatal and skip this pattern.
			continue
		}

		for _, match := range matches {
			info, statErr := os.Stat(match)
			if statErr != nil {
				continue
			}

			if info.ModTime().After(cutoff) {
				files = append(files, match)
			}
		}
	}

	return files, nil
}
