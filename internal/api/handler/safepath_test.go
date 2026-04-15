package handler

import "testing"

// TestIsSafeRelPath pins the edge cases that must be rejected. Kept
// separate from files_test.go so the hardened-path rules are easy to
// locate and extend.
func TestIsSafeRelPath(t *testing.T) {
	cases := []struct {
		in   string
		safe bool
	}{
		// Happy paths.
		{"foo/bar.png", true},
		{"nested/dir/file.mp3", true},
		{"manifest.json", true},

		// Obvious unsafe forms.
		{"", false},
		{"../etc/passwd", false},
		{"foo/../bar", false},
		{"/absolute/unix", false},
		{`\absolute\win`, false},

		// Windows drive letter masquerading as relative.
		{"C:foo.txt", false},
		{"d:bar", false},

		// NUL-byte truncation attack.
		{"safe.txt\x00../../etc/passwd", false},

		// Control-character injection (CR/LF).
		{"ok\rline.txt", false},
		{"newline\n.txt", false},

		// Just dots.
		{"..", false},
		{"...", true}, // not a traversal segment, just a filename
	}

	for _, tc := range cases {
		got := isSafeRelPath(tc.in)
		if got != tc.safe {
			t.Errorf("isSafeRelPath(%q) = %v, want %v", tc.in, got, tc.safe)
		}
	}
}
