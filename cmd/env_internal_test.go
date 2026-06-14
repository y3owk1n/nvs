package cmd

import (
	"testing"
)

func TestPathListContains(t *testing.T) {
	t.Parallel()

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
			list: "/foo:/bar:/baz",
			item: "/foo",
			want: true,
		},
		{
			name: "middle of many",
			list: "/foo:/bar:/baz",
			item: "/bar",
			want: true,
		},
		{
			name: "last of many",
			list: "/foo:/bar:/baz",
			item: "/baz",
			want: true,
		},
		{
			name: "missing",
			list: "/foo:/bar:/baz",
			item: "/qux",
			want: false,
		},
		{
			// Substring-only match (the false positive that
			// strings.Contains on the raw PATH would have
			// reported as a hit).
			name: "substring does not match",
			list: "/foo-extra:/bar",
			item: "/foo",
			want: false,
		},
		{
			// Item is a prefix of an existing entry but not
			// equal to it.
			name: "prefix does not match",
			list: "/foobar:/baz",
			item: "/foo",
			want: false,
		},
		{
			// Windows-style separator. The test asserts that
			// we use the platform separator correctly; on
			// non-Windows this is treated as one big entry.
			name: "trailing empty entry",
			list: "/foo:/bar:",
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
