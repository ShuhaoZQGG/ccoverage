package usage

import "testing"

func TestEncodeRepoPath(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"/Users/foo/project", "-Users-foo-project"},
		{"/a/b/c", "-a-b-c"},
		{"relative/path", "relative-path"},
	}
	for _, tt := range tests {
		got := encodeRepoPath(tt.input)
		if got != tt.want {
			t.Errorf("encodeRepoPath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
