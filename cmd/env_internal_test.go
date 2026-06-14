package cmd

import (
	"os"
	"testing"
)

// pathListSep is the platform's PATH separator (':' on Unix,
// ';' on Windows). Using it here instead of a hard-coded
// literal makes the same test data meaningful on every OS —
// without this, the tests pass on macOS/Linux but fail on
// Windows because strings.Split("/foo:/bar:/baz", ";")
// returns a single-element slice.
const pathListSep = string(os.PathListSeparator)

func TestPathListContains(t *testing.T) {
	t.Parallel()

	// Build a few well-known multi-entry PATH-style lists
	// using the platform separator so the same test data
	// works on Unix ('/foo:/bar:/baz') and Windows
	// ('C:/foo;C:/bar;C:/baz').
	manyItems := "/foo" + pathListSep + "/bar" + pathListSep + "/baz"
	manyItemsWithSubstring := "/foo-extra" + pathListSep + "/bar"
	manyItemsWithPrefix := "/foobar" + pathListSep + "/baz"
	trailingEmpty := "/foo" + pathListSep + "/bar" + pathListSep

	tests := []struct {
		name string
		list string
		item string
		want bool
	}{
		{
			name: "empty list",
			list: "",
			item: "/foo",
			want: false,
		},
		{
			name: "single match",
			list: "/foo",
			item: "/foo",
			want: true,
		},
		{
			name: "single no match",
			list: "/bar",
			item: "/foo",
			want: false,
		},
		{
			name: "first of many",
			list: manyItems,
			item: "/foo",
			want: true,
		},
		{
			name: "middle of many",
			list: manyItems,
			item: "/bar",
			want: true,
		},
		{
			name: "last of many",
			list: manyItems,
			item: "/baz",
			want: true,
		},
		{
			name: "missing",
			list: manyItems,
			item: "/qux",
			want: false,
		},
		{
			// Substring-only match (the false positive that
			// strings.Contains on the raw PATH would have
			// reported as a hit).
			name: "substring does not match",
			list: manyItemsWithSubstring,
			item: "/foo",
			want: false,
		},
		{
			// Item is a prefix of an existing entry but not
			// equal to it.
			name: "prefix does not match",
			list: manyItemsWithPrefix,
			item: "/foo",
			want: false,
		},
		{
			// Trailing separator produces a trailing empty
			// entry. The split should still find the real
			// entries.
			name: "trailing empty entry",
			list: trailingEmpty,
			item: "/bar",
			want: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := pathListContains(test.list, test.item)
			if got != test.want {
				t.Errorf(
					"pathListContains(%q, %q) = %v, want %v",
					test.list,
					test.item,
					got,
					test.want,
				)
			}
		})
	}
}

func TestShellQuote(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty",
			input: "",
			want:  "''",
		},
		{
			name:  "plain path",
			input: "/usr/local/bin",
			want:  "'/usr/local/bin'",
		},
		{
			name:  "path with dollar sign",
			input: "/path/with/$dollar",
			want:  "'/path/with/$dollar'",
		},
		{
			name:  "path with backtick",
			input: "/path/with/`backtick`",
			want:  "'/path/with/`backtick`'",
		},
		{
			name:  "path with backslash",
			input: `/path\with\backslash`,
			want:  `'/path\with\backslash'`,
		},
		{
			name:  "path with single quote",
			input: "/path/with/'quote",
			want:  `'/path/with/'\''quote'`,
		},
		{
			name:  "path with multiple single quotes",
			input: "a'b'c",
			want:  `'a'\''b'\''c'`,
		},
		{
			name:  "path with double quote",
			input: `/path/with/"double`,
			want:  `'/path/with/"double'`,
		},
		{
			name:  "path with newline",
			input: "/path/with/\nnewline",
			want:  "'/path/with/\nnewline'",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := shellQuote(test.input)
			if got != test.want {
				t.Errorf(
					"shellQuote(%q) = %q, want %q",
					test.input,
					got,
					test.want,
				)
			}
		})
	}
}
