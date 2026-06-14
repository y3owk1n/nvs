package cmd

import "testing"

func TestLineHasPathComponent(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		line   string
		target string
		want   bool
	}{
		{
			name:   "exact match at line start",
			line:   "/home/u/.local/bin",
			target: "/home/u/.local/bin",
			want:   true,
		},
		{
			name:   "exact match in export statement (bash)",
			line:   `export PATH="$PATH:/home/u/.local/bin"`,
			target: "/home/u/.local/bin",
			want:   true,
		},
		{
			name:   "exact match in fish set",
			line:   `set -gx PATH $PATH /home/u/.local/bin`,
			target: "/home/u/.local/bin",
			want:   true,
		},
		{
			name:   "delimited by colon on both sides (PATH entry)",
			line:   `export PATH=/foo:/home/u/.local/bin:/bar`,
			target: "/home/u/.local/bin",
			want:   true,
		},
		{
			name:   "subpath of longer directory should NOT match (the bug)",
			line:   `export PATH="$PATH:/home/u/.local/bin-extra"`,
			target: "/home/u/.local/bin",
			want:   false,
		},
		{
			// '/' is a path-component boundary, so /home/u/.local/bin is
			// a distinct prefix component of /home/u/.local/bin/nvim. The
			// wrapper rcFileContainsPathComponent relies on this to
			// recognize that a parent directory of a path already in PATH
			// implies the user does not need a fresh export.
			name:   "subpath with extra component delimited by '/' matches as prefix",
			line:   `export PATH="$PATH:/home/u/.local/bin/nvim"`,
			target: "/home/u/.local/bin",
			want:   true,
		},
		{
			// lineHasPathComponent does not filter by PATH content; that
			// is rcFileContainsPathComponent's job. A non-PATH line that
			// happens to contain the target still matches.
			name:   "appears in a non-PATH comment (still matches as path component)",
			line:   "# I removed /home/u/.local/bin from PATH",
			target: "/home/u/.local/bin",
			want:   true,
		},
		{
			name:   "appears as longer token in PATH line should NOT match",
			line:   `export PATH="$PATH:/home/u/.local/binaries"`,
			target: "/home/u/.local/bin",
			want:   false,
		},
		{
			name:   "preceded by alpha char should NOT match",
			line:   `export PATH="$PATH:prefix/home/u/.local/bin"`,
			target: "/home/u/.local/bin",
			want:   false,
		},
		{
			name:   "delimited by quotes is fine",
			line:   `export PATH="/home/u/.local/bin"`,
			target: "/home/u/.local/bin",
			want:   true,
		},
		{
			name:   "underscore is treated as path-component char (no match)",
			line:   `export PATH="$PATH:/home/u/.local/bin_old"`,
			target: "/home/u/.local/bin",
			want:   false,
		},
		{
			name:   "empty line",
			line:   "",
			target: "/home/u/.local/bin",
			want:   false,
		},
	}

	for idx := range cases {
		test := cases[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := lineHasPathComponent(test.line, test.target)
			if got != test.want {
				t.Errorf(
					"lineHasPathComponent(%q, %q) = %v, want %v",
					test.line,
					test.target,
					got,
					test.want,
				)
			}
		})
	}
}

func TestRcFileContainsPathComponent(t *testing.T) {
	t.Parallel()

	const (
		binDir = "/home/u/.local/bin"
		subDir = "/home/u/.local/bin-extra"
	)

	cases := []struct {
		name    string
		content string
		target  string
		want    bool
	}{
		{
			name: "bash export present",
			content: `# my rc
export PATH="$PATH:` + binDir + `"
`,
			target: binDir,
			want:   true,
		},
		{
			name: "fish set present",
			content: `set -gx PATH $PATH ` + binDir + `
`,
			target: binDir,
			want:   true,
		},
		{
			name: "subpath-only entry would have been substring-matched by bug",
			content: `# my rc
export PATH="$PATH:` + subDir + `"
`,
			target: binDir,
			want:   false,
		},
		{
			name: "comment-only should not count",
			content: `# removed ` + binDir + ` from PATH
export PATH="$PATH:/usr/local/bin"
`,
			target: binDir,
			want:   false,
		},
		{
			name:    "empty content",
			content: "",
			target:  binDir,
			want:    false,
		},
		{
			name:    "empty target",
			content: "export PATH=\"$PATH:/foo\"",
			target:  "",
			want:    false,
		},
		{
			name: "multiple lines, real entry in middle",
			content: `# header
# more comments
export PATH="$PATH:/opt/bin:` + binDir + `:/usr/bin"
`,
			target: binDir,
			want:   true,
		},
	}

	for idx := range cases {
		test := cases[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := rcFileContainsPathComponent(test.content, test.target)
			if got != test.want {
				t.Errorf(
					"rcFileContainsPathComponent() = %v, want %v",
					got,
					test.want,
				)
			}
		})
	}
}

func TestIsPathComponentByte(t *testing.T) {
	t.Parallel()

	cases := []struct {
		chr  byte
		want bool
	}{
		{'a', true},
		{'z', true},
		{'Z', true},
		{'0', true},
		{'9', true},
		{'_', true},
		{'-', true},
		{'.', true},
		{'/', false},
		{':', false},
		{' ', false},
		{'"', false},
		{'\'', false},
		{'$', false},
		{'=', false},
		{'\n', false},
		{0, false},
	}

	for idx := range cases {
		test := cases[idx]
		t.Run(string(test.chr), func(t *testing.T) {
			t.Parallel()

			got := isPathComponentByte(test.chr)
			if got != test.want {
				t.Errorf("isPathComponentByte(%q) = %v, want %v", test.chr, got, test.want)
			}
		})
	}
}
