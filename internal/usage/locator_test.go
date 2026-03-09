package usage

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLatestSessionFile(t *testing.T) {
	t.Run("no files returns empty string", func(t *testing.T) {
		// Use a temp dir as the fake home; the project dir won't exist.
		home := t.TempDir()
		t.Setenv("HOME", home)

		got, err := LatestSessionFile("/nonexistent/repo")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "" {
			t.Errorf("expected empty string, got %q", got)
		}
	})

	t.Run("returns newest file", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)

		encoded := EncodeRepoPath("/my/repo")
		projectDir := filepath.Join(home, ".claude", "projects", encoded)
		if err := os.MkdirAll(projectDir, 0o755); err != nil {
			t.Fatal(err)
		}

		// Create two JSONL files with different modification times.
		older := filepath.Join(projectDir, "session-old.jsonl")
		newer := filepath.Join(projectDir, "session-new.jsonl")

		for _, f := range []string{older, newer} {
			if err := os.WriteFile(f, []byte(`{"type":"user","cwd":"/my/repo"}`+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}

		// Make the "older" file clearly older.
		oldTime := time.Now().Add(-1 * time.Hour)
		if err := os.Chtimes(older, oldTime, oldTime); err != nil {
			t.Fatal(err)
		}

		got, err := LatestSessionFile("/my/repo")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != newer {
			t.Errorf("got %q, want %q", got, newer)
		}
	})

	t.Run("picks subagent file when newer", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)

		encoded := EncodeRepoPath("/my/repo2")
		projectDir := filepath.Join(home, ".claude", "projects", encoded)
		subDir := filepath.Join(projectDir, "sub")
		for _, d := range []string{projectDir, subDir} {
			if err := os.MkdirAll(d, 0o755); err != nil {
				t.Fatal(err)
			}
		}

		topLevel := filepath.Join(projectDir, "session-top.jsonl")
		subAgent := filepath.Join(subDir, "session-sub.jsonl")

		for _, f := range []string{topLevel, subAgent} {
			if err := os.WriteFile(f, []byte(`{"type":"user","cwd":"/my/repo2"}`+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}

		// Make top-level file older.
		oldTime := time.Now().Add(-2 * time.Hour)
		if err := os.Chtimes(topLevel, oldTime, oldTime); err != nil {
			t.Fatal(err)
		}

		got, err := LatestSessionFile("/my/repo2")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != subAgent {
			t.Errorf("got %q, want %q", got, subAgent)
		}
	})
}

func TestEncodeRepoPath(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"/Users/foo/project", "-Users-foo-project"},
		{"/a/b/c", "-a-b-c"},
		{"relative/path", "relative-path"},
	}
	for _, tt := range tests {
		got := EncodeRepoPath(tt.input)
		if got != tt.want {
			t.Errorf("EncodeRepoPath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
