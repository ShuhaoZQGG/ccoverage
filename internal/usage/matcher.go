package usage

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ShuhaoZQGG/ccoverage/internal/types"
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
			if s.FirstSeen == nil || firstMatch.Before(*s.FirstSeen) {
				s.FirstSeen = &firstMatch
			}
			if s.LastSeen == nil || lastMatch.After(*s.LastSeen) {
				s.LastSeen = &lastMatch
			}
		}
	}

	// -----------------------------------------------------------------------
	// 4. Plugin component matching.
	// -----------------------------------------------------------------------
	// For each Plugin manifest item, check whether any of its components
	// (MCP servers, skills, LSP) were observed in session events. Credit
	// the plugin with the aggregated activations from its components.

	for _, item := range manifest.Items {
		if item.Type != types.ConfigPlugin {
			continue
		}

		key := manifestKey(item.Type, item.Name)
		pluginSummary := matchPluginComponents(item, allEvents)
		if pluginSummary != nil {
			s := getOrCreateSummary(summaries, key)
			s.TotalActivations += pluginSummary.TotalActivations
			s.UniqueSessions += pluginSummary.UniqueSessions
			if pluginSummary.FirstSeen != nil {
				if s.FirstSeen == nil || pluginSummary.FirstSeen.Before(*s.FirstSeen) {
					s.FirstSeen = pluginSummary.FirstSeen
				}
			}
			if pluginSummary.LastSeen != nil {
				if s.LastSeen == nil || pluginSummary.LastSeen.After(*s.LastSeen) {
					s.LastSeen = pluginSummary.LastSeen
				}
			}
		}
	}

	return summaries, allCwds, nil
}

// MatchSingleSession parses sessionFile and reports, for each item in
// manifest, whether it was active in that session.
//
// For Skill/MCP/Hook/Command items a match requires at least one UsageEvent
// with the same ConfigType and Name.  For CLAUDE.md items the same
// directory-containment logic used by MatchUsage applies: the item is active
// when any cwd or touched directory in the session is at or beneath the
// directory that contains the CLAUDE.md file.
//
// Returns (nil, nil) when sessionFile is empty.
func MatchSingleSession(manifest *types.Manifest, sessionFile string) (*types.LastSessionReport, error) {
	if sessionFile == "" {
		return nil, nil
	}

	events, cwds, touchedDirs, err := ParseSessionFile(sessionFile)
	if err != nil {
		return nil, fmt.Errorf("usage: parse session file: %w", err)
	}

	sessionID := sessionIDFromPath(sessionFile)

	// Determine the session timestamp from the file's mtime.
	var sessionTime time.Time
	if info, statErr := os.Stat(sessionFile); statErr == nil {
		sessionTime = info.ModTime()
	}

	// Build sets for O(1) lookups.
	type eventKey struct {
		ct   types.ConfigType
		name string
	}
	activeKeys := make(map[eventKey]int, len(events))
	for _, e := range events {
		activeKeys[eventKey{e.ConfigType, e.Name}]++
	}

	cwdSet := make(map[string]struct{}, len(cwds))
	for _, c := range cwds {
		cwdSet[c] = struct{}{}
	}

	touchedDirSet := make(map[string]struct{}, len(touchedDirs))
	for _, d := range touchedDirs {
		touchedDirSet[d] = struct{}{}
	}

	items := make([]types.LastSessionItem, 0, len(manifest.Items))
	for _, item := range manifest.Items {
		var active bool
		var count int

		if item.Type == types.ConfigClaudeMD {
			claudeDir := filepath.Dir(item.AbsPath)
			for cwd := range cwdSet {
				if isDirOrDescendant(cwd, claudeDir) {
					active = true
					break
				}
			}
			if !active {
				for dir := range touchedDirSet {
					if isDirOrDescendant(dir, claudeDir) {
						active = true
						break
					}
				}
			}
			if active {
				count = 1
			}
		} else if item.Type == types.ConfigPlugin {
			// Plugin matching: check if any of its components were used.
			mcpServers := splitMetadata(item.Metadata["mcp_servers"])
			skillNames := splitMetadata(item.Metadata["skill_names"])
			commandNames := splitMetadata(item.Metadata["command_names"])
			hasLSP := strings.Contains(item.Metadata["components"], "lsp")
			lspExts := splitMetadata(item.Metadata["lsp_extensions"])
			for _, s := range mcpServers {
				count += activeKeys[eventKey{types.ConfigMCP, s}]
			}
			for _, s := range skillNames {
				count += activeKeys[eventKey{types.ConfigSkill, s}]
			}
			for _, s := range commandNames {
				count += activeKeys[eventKey{types.ConfigCommand, s}]
				// parseUserLine also emits ConfigSkill for slash commands.
				count += activeKeys[eventKey{types.ConfigSkill, s}]
			}
			// Skills may also appear as commands (parseUserLine emits both).
			for _, s := range skillNames {
				count += activeKeys[eventKey{types.ConfigCommand, s}]
			}
			if hasLSP {
				// Match extension-qualified LSP events against this plugin's extensions.
				for ek, c := range activeKeys {
					if ek.ct == types.ConfigPlugin && matchLSPEvent(ek.name, lspExts) {
						count += c
					}
				}
			}
			active = count > 0
		} else {
			count = activeKeys[eventKey{item.Type, item.Name}]
			active = count > 0
		}

		items = append(items, types.LastSessionItem{
			Type:   item.Type,
			Name:   item.Name,
			Active: active,
			Count:  count,
		})
	}

	return &types.LastSessionReport{
		SessionID: sessionID,
		Timestamp: sessionTime,
		Items:     items,
	}, nil
}

// ---------------------------------------------------------------------------
// Plugin component matching
// ---------------------------------------------------------------------------

// matchPluginComponents checks whether any component of a plugin (MCP servers,
// skills, LSP) was observed in the event stream. Returns a summary of matching
// activations, or nil if no matches were found.
func matchPluginComponents(item types.ManifestItem, events []types.UsageEvent) *types.UsageSummary {
	mcpServers := splitMetadata(item.Metadata["mcp_servers"])
	skillNames := splitMetadata(item.Metadata["skill_names"])
	commandNames := splitMetadata(item.Metadata["command_names"])
	hasLSP := strings.Contains(item.Metadata["components"], "lsp")

	var totalActivations int
	sessionSet := make(map[string]struct{})
	var first, last time.Time

	lspExts := splitMetadata(item.Metadata["lsp_extensions"])

	for _, evt := range events {
		matched := false
		switch evt.ConfigType {
		case types.ConfigMCP:
			for _, s := range mcpServers {
				if evt.Name == s {
					matched = true
					break
				}
			}
		case types.ConfigSkill:
			for _, s := range skillNames {
				if evt.Name == s {
					matched = true
					break
				}
			}
			if !matched {
				for _, s := range commandNames {
					if evt.Name == s {
						matched = true
						break
					}
				}
			}
		case types.ConfigCommand:
			for _, s := range commandNames {
				if evt.Name == s {
					matched = true
					break
				}
			}
			if !matched {
				for _, s := range skillNames {
					if evt.Name == s {
						matched = true
						break
					}
				}
			}
		case types.ConfigPlugin:
			if hasLSP {
				matched = matchLSPEvent(evt.Name, lspExts)
			}
		}

		if matched {
			totalActivations++
			if evt.SessionID != "" {
				sessionSet[evt.SessionID] = struct{}{}
			}
			updateTimeRangeInline(&first, &last, evt.Timestamp)
		}
	}

	if totalActivations == 0 {
		return nil
	}

	s := &types.UsageSummary{
		TotalActivations: totalActivations,
		UniqueSessions:   len(sessionSet),
	}
	if !first.IsZero() {
		s.FirstSeen = &first
		s.LastSeen = &last
	}
	return s
}

// matchLSPEvent checks whether an LSP event name matches a plugin's supported
// extensions. Event names are either "LSP" (legacy, no extension info) or
// "LSP:.ext" (extension-qualified). When lspExts is non-empty, only
// extension-qualified events matching one of those extensions are accepted.
// When lspExts is empty, both "LSP" and "LSP:*" match (backwards compat for
// plugins without parsed extensions).
func matchLSPEvent(eventName string, lspExts []string) bool {
	if eventName == "LSP" {
		// Legacy unqualified event — match if plugin has no extension info.
		return len(lspExts) == 0
	}
	if !strings.HasPrefix(eventName, "LSP:") {
		return false
	}
	ext := eventName[4:] // e.g. ".go"
	if len(lspExts) == 0 {
		// Plugin has no extension info — accept any LSP event.
		return true
	}
	for _, e := range lspExts {
		if e == ext {
			return true
		}
	}
	return false
}

// splitMetadata splits a comma-separated metadata value into a slice.
// Returns nil for empty strings.
func splitMetadata(val string) []string {
	if val == "" {
		return nil
	}
	parts := strings.Split(val, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
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
	if s.FirstSeen == nil || t.Before(*s.FirstSeen) {
		s.FirstSeen = &t
	}
	if s.LastSeen == nil || t.After(*s.LastSeen) {
		s.LastSeen = &t
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
