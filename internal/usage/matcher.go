package usage

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/shuhaozhang/ccoverage/internal/types"
)

// manifestKey returns the canonical lookup key for a manifest item.
func manifestKey(configType types.ConfigType, name string) string {
	return fmt.Sprintf("%s:%s", configType, name)
}

// MatchUsage parses every file listed in sessionFiles, then correlates the
// extracted usage events with the items declared in manifest.
//
// It returns:
//   - a map keyed by "Type:Name" → *UsageSummary for every manifest item that
//     was observed at least once (items with no usage are absent from the map),
//   - the deduplicated list of all cwd values seen across every session file,
//   - any fatal error encountered (individual parse failures are logged and
//     skipped rather than surfaced here).
//
// CLAUDE.md matching uses directory containment: a CLAUDE.md item is
// considered "used" in any session whose cwd is equal to, or a subdirectory
// of, the directory that contains the CLAUDE.md file.  Synthetic UsageEvents
// are generated for these implicit activations.
func MatchUsage(
	manifest *types.Manifest,
	sessionFiles []string,
) (map[string]*types.UsageSummary, []string, error) {
	// -----------------------------------------------------------------------
	// 1. Parse all session files.
	// -----------------------------------------------------------------------
	var allEvents []types.UsageEvent
	allCwdSet := make(map[string]struct{})
	allTouchedDirSet := make(map[string]struct{})
	sessionIDSet := make(map[string]struct{})
	// Maps each touched dir to the session IDs that produced it.
	touchedDirToSessions := make(map[string]map[string]struct{})

	for _, path := range sessionFiles {
		events, cwds, touchedDirs, err := ParseSessionFile(path)
		if err != nil {
			log.Printf("usage: skipping session file %q: %v", path, err)
			continue
		}

		allEvents = append(allEvents, events...)

		for _, cwd := range cwds {
			allCwdSet[cwd] = struct{}{}
		}

		sessionID := sessionIDFromPath(path)

		for _, dir := range touchedDirs {
			allTouchedDirSet[dir] = struct{}{}
			if touchedDirToSessions[dir] == nil {
				touchedDirToSessions[dir] = make(map[string]struct{})
			}
			if sessionID != "" {
				touchedDirToSessions[dir][sessionID] = struct{}{}
			}
		}

		// Track unique session IDs derived from the events themselves.
		for _, e := range events {
			if e.SessionID != "" {
				sessionIDSet[e.SessionID] = struct{}{}
			}
		}

		// Also register the session derived from the filename even when it
		// produced no events, so that the session count is accurate.
		if sessionID != "" {
			sessionIDSet[sessionID] = struct{}{}
		}
	}

	allCwds := cwdsFromSet(allCwdSet)

	// -----------------------------------------------------------------------
	// 2. Build the summary map from explicit events.
	// -----------------------------------------------------------------------
	summaries := make(map[string]*types.UsageSummary)

	for _, evt := range allEvents {
		key := manifestKey(evt.ConfigType, evt.Name)
		s := getOrCreateSummary(summaries, key)
		s.TotalActivations++
		updateTimeRange(s, evt.Timestamp)
	}

	// Unique session counts per manifest item require re-aggregating by key.
	// Build a per-key session set.
	sessionsByKey := make(map[string]map[string]struct{})
	for _, evt := range allEvents {
		key := manifestKey(evt.ConfigType, evt.Name)
		if sessionsByKey[key] == nil {
			sessionsByKey[key] = make(map[string]struct{})
		}
		if evt.SessionID != "" {
			sessionsByKey[key][evt.SessionID] = struct{}{}
		}
	}
	for key, sessions := range sessionsByKey {
		if s, ok := summaries[key]; ok {
			s.UniqueSessions = len(sessions)
		}
	}

	// -----------------------------------------------------------------------
	// 3. CLAUDE.md directory-containment matching.
	// -----------------------------------------------------------------------
	// For each CLAUDE.md manifest item, check whether any observed cwd is
	// at or beneath the directory containing the CLAUDE.md file.  When a
	// match is found we generate synthetic events to credit the item.

	for _, item := range manifest.Items {
		if item.Type != types.ConfigClaudeMD {
			continue
		}

		// The directory that must be an ancestor of (or equal to) the cwd.
		claudeDir := filepath.Dir(item.AbsPath)

		key := manifestKey(item.Type, item.Name)

		// Collect sessions that fired this CLAUDE.md via cwd matching.
		matchSessions := make(map[string]struct{})
		var firstMatch, lastMatch time.Time

		for _, cwd := range allCwds {
			if !isDirOrDescendant(cwd, claudeDir) {
				continue
			}

			// Find all session IDs that were active in this cwd.
			for _, evt := range allEvents {
				if evt.Cwd == cwd {
					if evt.SessionID != "" {
						matchSessions[evt.SessionID] = struct{}{}
					}
					updateTimeRangeInline(&firstMatch, &lastMatch, evt.Timestamp)
				}
			}

			// If no events carry this cwd (possible when only the envelope
			// cwd field was set but no events were emitted), we still want to
			// count the activation.  Use a sentinel session key derived from
			// the cwd so deduplication still works.
			if len(matchSessions) == 0 {
				matchSessions["cwd:"+cwd] = struct{}{}
			}
		}

		// Also check directories touched by file-oriented tools. A session
		// that reads/edits a file under a CLAUDE.md's directory triggers
		// loading of that CLAUDE.md even if the session cwd is elsewhere.
		for dir, sessions := range touchedDirToSessions {
			if !isDirOrDescendant(dir, claudeDir) {
				continue
			}
			for sid := range sessions {
				matchSessions[sid] = struct{}{}
			}
			// Use event timestamps from matching sessions for time range.
			for _, evt := range allEvents {
				if _, ok := sessions[evt.SessionID]; ok {
					updateTimeRangeInline(&firstMatch, &lastMatch, evt.Timestamp)
				}
			}
		}

		if len(matchSessions) == 0 {
			continue
		}

		s := getOrCreateSummary(summaries, key)
		// Each matched session counts as one activation for CLAUDE.md.
		s.TotalActivations += len(matchSessions)
		s.UniqueSessions += len(matchSessions)
		if !firstMatch.IsZero() {
			if s.FirstSeen.IsZero() || firstMatch.Before(s.FirstSeen) {
				s.FirstSeen = firstMatch
			}
			if lastMatch.After(s.LastSeen) {
				s.LastSeen = lastMatch
			}
		}
	}

	return summaries, allCwds, nil
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

func getOrCreateSummary(m map[string]*types.UsageSummary, key string) *types.UsageSummary {
	if s, ok := m[key]; ok {
		return s
	}
	s := &types.UsageSummary{}
	m[key] = s
	return s
}

func updateTimeRange(s *types.UsageSummary, t time.Time) {
	if t.IsZero() {
		return
	}
	if s.FirstSeen.IsZero() || t.Before(s.FirstSeen) {
		s.FirstSeen = t
	}
	if t.After(s.LastSeen) {
		s.LastSeen = t
	}
}

func updateTimeRangeInline(first, last *time.Time, t time.Time) {
	if t.IsZero() {
		return
	}
	if first.IsZero() || t.Before(*first) {
		*first = t
	}
	if t.After(*last) {
		*last = t
	}
}

// isDirOrDescendant reports whether candidate is equal to parent or is a
// directory beneath parent.
func isDirOrDescendant(candidate, parent string) bool {
	// Normalise both paths so we compare clean, absolute paths.
	candidate = filepath.Clean(candidate)
	parent = filepath.Clean(parent)

	if candidate == parent {
		return true
	}

	// A descendant path must start with parent + the OS path separator to
	// avoid false positives like /foo/barbaz matching /foo/bar.
	return strings.HasPrefix(candidate, parent+string(filepath.Separator))
}

// sessionIDFromPath derives a session ID from the filename using the same
// convention as ParseSessionFile.
func sessionIDFromPath(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, ".jsonl")
}
