package usage

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/ShuhaoZQGG/ccoverage/internal/types"
)

// scannerBufferSize is the maximum line size accepted by the JSONL scanner.
// Claude session lines can be large when they contain tool outputs.
const scannerBufferSize = 4 * 1024 * 1024 // 4 MB

// commandNameRe matches <command-name>…</command-name> tags embedded in user
// messages that represent slash-command invocations.
var commandNameRe = regexp.MustCompile(`<command-name>(.+?)</command-name>`)

// ---------------------------------------------------------------------------
// Raw wire types – minimal structs that mirror the JSONL schema just enough
// for the fields we need.
// ---------------------------------------------------------------------------

// rawLine is the top-level envelope of every JSONL record.
type rawLine struct {
	Type      string          `json:"type"`
	Message   json.RawMessage `json:"message"`
	Cwd       string          `json:"cwd"`
	Timestamp string          `json:"timestamp"`
	Data      json.RawMessage `json:"data"`
}

// rawMessage represents the "message" field found on assistant / user lines.
type rawMessage struct {
	// Content can be a plain string or an array of content blocks.
	Content json.RawMessage `json:"content"`
}

// rawContentBlock represents a single item in a content array.
type rawContentBlock struct {
	Type  string          `json:"type"`
	Text  string          `json:"text"`
	Name  string          `json:"name"`  // tool_use: tool name
	Input json.RawMessage `json:"input"` // tool_use: arbitrary input object
}

// rawSkillInput is the input shape for the "Skill" tool.
type rawSkillInput struct {
	Skill string `json:"skill"`
}

// rawAgentInput is the input shape for the "Agent" tool.
type rawAgentInput struct {
	SubagentType string `json:"subagent_type"`
}

// rawLSPInput captures the filePath from LSP tool invocations.
type rawLSPInput struct {
	FilePath string `json:"filePath"`
}

// rawFileInput captures path fields from file-oriented tools (Read, Edit,
// Write, Glob, Grep).
type rawFileInput struct {
	FilePath string `json:"file_path"` // Read, Edit, Write
	Path     string `json:"path"`      // Glob, Grep (directory scope)
}

// rawProgressData is the payload carried by "progress" type lines.
type rawProgressData struct {
	Type      string `json:"type"`
	HookEvent string `json:"hookEvent"`
	HookName  string `json:"hookName"`
}

// ---------------------------------------------------------------------------
// ParseSessionFile
// ---------------------------------------------------------------------------

// ParseSessionFile reads the JSONL file at path and returns all usage events
// found within it, a deduplicated list of cwd values, and a deduplicated list
// of directories touched by file-oriented tools (Read, Edit, Write, Glob, Grep).
//
// Lines that cannot be parsed are skipped with a warning; the function never
// returns a hard error due to individual malformed lines.
func ParseSessionFile(path string) ([]types.UsageEvent, []string, []string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("usage: open session file %q: %w", path, err)
	}
	defer f.Close()

	sessionID := strings.TrimSuffix(filepath.Base(path), ".jsonl")

	scanner := bufio.NewScanner(f)
	buf := make([]byte, scannerBufferSize)
	scanner.Buffer(buf, scannerBufferSize)

	var events []types.UsageEvent
	cwdSet := make(map[string]struct{})
	touchedDirSet := make(map[string]struct{})

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var raw rawLine
		if err := json.Unmarshal(line, &raw); err != nil {
			log.Printf("usage: skipping unparseable line in %s: %v", path, err)
			continue
		}

		// Collect cwd for every line that carries one.
		if raw.Cwd != "" {
			cwdSet[raw.Cwd] = struct{}{}
		}

		ts := parseTimestamp(raw.Timestamp)

		switch raw.Type {
		case "assistant":
			evts, blocks := parseAssistantLineWithBlocks(raw, sessionID, ts)
			events = append(events, evts...)
			for _, dir := range extractTouchedDirs(blocks) {
				touchedDirSet[dir] = struct{}{}
			}

		case "user":
			evts := parseUserLine(raw, sessionID, ts)
			events = append(events, evts...)

		case "progress":
			if evt, ok := parseProgressLine(raw, sessionID, ts); ok {
				events = append(events, evt)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return events, cwdsFromSet(cwdSet), cwdsFromSet(touchedDirSet), fmt.Errorf("usage: scanning %q: %w", path, err)
	}

	return events, cwdsFromSet(cwdSet), cwdsFromSet(touchedDirSet), nil
}

// ---------------------------------------------------------------------------
// Per-type parsers
// ---------------------------------------------------------------------------

// parseAssistantLineWithBlocks is like parseAssistantLine but also returns
// the decoded content blocks so the caller can extract additional information
// (e.g. file paths from tool_use blocks).
func parseAssistantLineWithBlocks(raw rawLine, sessionID string, ts time.Time) ([]types.UsageEvent, []rawContentBlock) {
	if len(raw.Message) == 0 {
		return nil, nil
	}

	var msg rawMessage
	if err := json.Unmarshal(raw.Message, &msg); err != nil {
		return nil, nil
	}

	blocks := decodeContentBlocks(msg.Content)
	if blocks == nil {
		return nil, nil
	}

	var events []types.UsageEvent

	for _, block := range blocks {
		if block.Type != "tool_use" {
			continue
		}

		switch {
		case block.Name == "Skill":
			var inp rawSkillInput
			if err := json.Unmarshal(block.Input, &inp); err != nil || inp.Skill == "" {
				continue
			}
			events = append(events, types.UsageEvent{
				ConfigType: types.ConfigSkill,
				Name:       inp.Skill,
				SessionID:  sessionID,
				Timestamp:  ts,
				Cwd:        raw.Cwd,
			})

		case strings.HasPrefix(block.Name, "mcp__"):
			server := mcpServerName(block.Name)
			if server == "" {
				continue
			}
			events = append(events, types.UsageEvent{
				ConfigType: types.ConfigMCP,
				Name:       server,
				SessionID:  sessionID,
				Timestamp:  ts,
				Cwd:        raw.Cwd,
			})

		case block.Name == "LSP":
			// Extract file extension from the filePath input to attribute
			// the LSP call to the correct language plugin.
			lspName := "LSP"
			if len(block.Input) > 0 {
				var inp rawLSPInput
				if json.Unmarshal(block.Input, &inp) == nil && inp.FilePath != "" {
					ext := filepath.Ext(inp.FilePath)
					if ext != "" {
						lspName = "LSP:" + ext
					}
				}
			}
			events = append(events, types.UsageEvent{
				ConfigType: types.ConfigPlugin,
				Name:       lspName,
				SessionID:  sessionID,
				Timestamp:  ts,
				Cwd:        raw.Cwd,
			})

		case block.Name == "Agent":
			var inp rawAgentInput
			if err := json.Unmarshal(block.Input, &inp); err != nil || inp.SubagentType == "" {
				continue
			}
			// Agent subagent_type is surfaced under ConfigSkill so that it can
			// be correlated with Skill manifest entries if desired, but we use
			// a dedicated constant when one exists.  For now map to ConfigSkill
			// as agents are a kind of skill invocation.
			events = append(events, types.UsageEvent{
				ConfigType: types.ConfigSkill,
				Name:       inp.SubagentType,
				SessionID:  sessionID,
				Timestamp:  ts,
				Cwd:        raw.Cwd,
			})
		}
	}

	return events, blocks
}

// extractTouchedDirs examines tool_use content blocks for file-oriented tools
// (Read, Edit, Write, Glob, Grep) and returns deduplicated directory paths
// extracted from their input fields. Only absolute paths are included.
func extractTouchedDirs(blocks []rawContentBlock) []string {
	seen := make(map[string]struct{})
	for _, block := range blocks {
		if block.Type != "tool_use" || len(block.Input) == 0 {
			continue
		}

		var dir string
		switch block.Name {
		case "Read", "Edit", "Write":
			var inp rawFileInput
			if err := json.Unmarshal(block.Input, &inp); err != nil || inp.FilePath == "" {
				continue
			}
			dir = filepath.Dir(inp.FilePath)
		case "Glob", "Grep":
			var inp rawFileInput
			if err := json.Unmarshal(block.Input, &inp); err != nil || inp.Path == "" {
				continue
			}
			dir = inp.Path
		default:
			continue
		}

		if !filepath.IsAbs(dir) {
			continue
		}
		dir = filepath.Clean(dir)
		seen[dir] = struct{}{}
	}

	out := make([]string, 0, len(seen))
	for d := range seen {
		out = append(out, d)
	}
	return out
}

func parseUserLine(raw rawLine, sessionID string, ts time.Time) []types.UsageEvent {
	if len(raw.Message) == 0 {
		return nil
	}

	var msg rawMessage
	if err := json.Unmarshal(raw.Message, &msg); err != nil {
		return nil
	}

	// Collect all text from the content field regardless of whether it is a
	// plain string or an array of text blocks.
	text := extractText(msg.Content)
	if text == "" {
		return nil
	}

	matches := commandNameRe.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return nil
	}

	var events []types.UsageEvent
	for _, m := range matches {
		if len(m) < 2 || m[1] == "" {
			continue
		}
		name := m[1]
		events = append(events, types.UsageEvent{
			ConfigType: types.ConfigCommand,
			Name:       name,
			SessionID:  sessionID,
			Timestamp:  ts,
			Cwd:        raw.Cwd,
		})
		// Also emit as ConfigSkill with the slash stripped, because
		// skills invoked via slash command are logged with <command-name>
		// tags but registered in the manifest as ConfigSkill.
		skillName := strings.TrimPrefix(name, "/")
		if skillName != name && skillName != "" {
			events = append(events, types.UsageEvent{
				ConfigType: types.ConfigSkill,
				Name:       skillName,
				SessionID:  sessionID,
				Timestamp:  ts,
				Cwd:        raw.Cwd,
			})
		}
	}
	return events
}

func parseProgressLine(raw rawLine, sessionID string, ts time.Time) (types.UsageEvent, bool) {
	if len(raw.Data) == 0 {
		return types.UsageEvent{}, false
	}

	var d rawProgressData
	if err := json.Unmarshal(raw.Data, &d); err != nil {
		return types.UsageEvent{}, false
	}

	if d.Type != "hook_progress" || d.HookName == "" {
		return types.UsageEvent{}, false
	}

	return types.UsageEvent{
		ConfigType: types.ConfigHook,
		Name:       d.HookName,
		SessionID:  sessionID,
		Timestamp:  ts,
		Cwd:        raw.Cwd,
	}, true
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// decodeContentBlocks returns the slice of content blocks regardless of
// whether the raw JSON encodes content as a string or an array.  When content
// is a plain string the function returns nil (no tool_use blocks possible).
func decodeContentBlocks(raw json.RawMessage) []rawContentBlock {
	if len(raw) == 0 {
		return nil
	}

	// A JSON array starts with '['.
	if raw[0] != '[' {
		return nil
	}

	var blocks []rawContentBlock
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return nil
	}
	return blocks
}

// extractText returns all text content from a content field that may be
// either a plain JSON string or an array of content blocks.
func extractText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	// Plain string.
	if raw[0] == '"' {
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return ""
		}
		return s
	}

	// Array of blocks – concatenate all text fields.
	if raw[0] == '[' {
		var blocks []rawContentBlock
		if err := json.Unmarshal(raw, &blocks); err != nil {
			return ""
		}
		var sb strings.Builder
		for _, b := range blocks {
			if b.Text != "" {
				sb.WriteString(b.Text)
			}
		}
		return sb.String()
	}

	return ""
}

// mcpServerName extracts the server component from a tool name of the form
// mcp__<server>__<operation>.
//
// Example: mcp__supabase__query → "supabase"
func mcpServerName(toolName string) string {
	parts := strings.Split(toolName, "__")
	// parts[0] == "mcp", parts[1] == server name
	if len(parts) < 2 {
		return ""
	}
	return parts[1]
}

// parseTimestamp attempts to parse an ISO 8601 timestamp string.  It returns
// the zero time when parsing fails rather than propagating an error.
func parseTimestamp(s string) time.Time {
	if s == "" {
		return time.Time{}
	}

	// RFC3339Nano covers the most common ISO 8601 variant used by Claude.
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		// Try without fractional seconds.
		t, err = time.Parse(time.RFC3339, s)
		if err != nil {
			return time.Time{}
		}
	}
	return t
}

// cwdsFromSet converts the deduplication set into a stable string slice.
func cwdsFromSet(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for cwd := range set {
		out = append(out, cwd)
	}
	return out
}
