package usage

import (
	"path/filepath"
	"testing"

	"github.com/shuhaozhang/ccoverage/internal/types"
)

func TestParseSessionFile(t *testing.T) {
	path, err := filepath.Abs("../../testdata/sample_session.jsonl")
	if err != nil {
		t.Fatal(err)
	}

	events, cwds, touchedDirs, err := ParseSessionFile(path)
	if err != nil {
		t.Fatalf("ParseSessionFile: %v", err)
	}

	if len(cwds) == 0 {
		t.Error("expected at least one cwd")
	}

	// Check we got the expected event types.
	found := map[types.ConfigType][]string{}
	for _, e := range events {
		found[e.ConfigType] = append(found[e.ConfigType], e.Name)
	}

	if names, ok := found[types.ConfigSkill]; !ok || !contains(names, "db-migration") {
		t.Errorf("expected Skill:db-migration, got %v", found[types.ConfigSkill])
	}
	if names, ok := found[types.ConfigMCP]; !ok || !contains(names, "supabase") {
		t.Errorf("expected MCP:supabase, got %v", found[types.ConfigMCP])
	}
	if names, ok := found[types.ConfigCommand]; !ok || !contains(names, "/deploy") {
		t.Errorf("expected Command:/deploy, got %v", found[types.ConfigCommand])
	}
	if names, ok := found[types.ConfigHook]; !ok || !contains(names, "PreToolUse:Bash") {
		t.Errorf("expected Hook:PreToolUse:Bash, got %v", found[types.ConfigHook])
	}

	// Check touched dirs from file-oriented tool_use blocks.
	if !contains(touchedDirs, "/Users/test/myproject/src/api") {
		t.Errorf("expected touched dir /Users/test/myproject/src/api, got %v", touchedDirs)
	}
}

func TestExtractTouchedDirs(t *testing.T) {
	tests := []struct {
		name   string
		blocks []rawContentBlock
		want   []string
	}{
		{
			name: "Read tool",
			blocks: []rawContentBlock{
				{Type: "tool_use", Name: "Read", Input: []byte(`{"file_path":"/repo/src/main.go"}`)},
			},
			want: []string{"/repo/src"},
		},
		{
			name: "Edit tool",
			blocks: []rawContentBlock{
				{Type: "tool_use", Name: "Edit", Input: []byte(`{"file_path":"/repo/pkg/handler.go","old_string":"a","new_string":"b"}`)},
			},
			want: []string{"/repo/pkg"},
		},
		{
			name: "Write tool",
			blocks: []rawContentBlock{
				{Type: "tool_use", Name: "Write", Input: []byte(`{"file_path":"/repo/out/result.txt","content":"hello"}`)},
			},
			want: []string{"/repo/out"},
		},
		{
			name: "Glob tool with path",
			blocks: []rawContentBlock{
				{Type: "tool_use", Name: "Glob", Input: []byte(`{"pattern":"*.go","path":"/repo/internal"}`)},
			},
			want: []string{"/repo/internal"},
		},
		{
			name: "Grep tool with path",
			blocks: []rawContentBlock{
				{Type: "tool_use", Name: "Grep", Input: []byte(`{"pattern":"TODO","path":"/repo/cmd"}`)},
			},
			want: []string{"/repo/cmd"},
		},
		{
			name: "relative path rejected",
			blocks: []rawContentBlock{
				{Type: "tool_use", Name: "Read", Input: []byte(`{"file_path":"src/main.go"}`)},
			},
			want: nil,
		},
		{
			name: "non-file tool ignored",
			blocks: []rawContentBlock{
				{Type: "tool_use", Name: "Bash", Input: []byte(`{"command":"ls"}`)},
			},
			want: nil,
		},
		{
			name: "text block ignored",
			blocks: []rawContentBlock{
				{Type: "text", Text: "hello"},
			},
			want: nil,
		},
		{
			name: "deduplication",
			blocks: []rawContentBlock{
				{Type: "tool_use", Name: "Read", Input: []byte(`{"file_path":"/repo/src/a.go"}`)},
				{Type: "tool_use", Name: "Edit", Input: []byte(`{"file_path":"/repo/src/b.go","old_string":"x","new_string":"y"}`)},
			},
			want: []string{"/repo/src"},
		},
		{
			name: "Glob without path ignored",
			blocks: []rawContentBlock{
				{Type: "tool_use", Name: "Glob", Input: []byte(`{"pattern":"*.go"}`)},
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTouchedDirs(tt.blocks)
			if len(got) == 0 && len(tt.want) == 0 {
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			// For single-element results, direct comparison is fine.
			// For multi-element, check containment.
			for _, w := range tt.want {
				if !contains(got, w) {
					t.Errorf("missing %q in %v", w, got)
				}
			}
		})
	}
}

func TestMcpServerName(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"mcp__supabase__query", "supabase"},
		{"mcp__github__list_repos", "github"},
		{"mcp", ""},
		{"Skill", ""},
	}
	for _, tt := range tests {
		got := mcpServerName(tt.input)
		if got != tt.want {
			t.Errorf("mcpServerName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
