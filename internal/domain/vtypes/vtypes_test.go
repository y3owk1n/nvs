package vtypes_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/y3owk1n/nvs/internal/domain/vtypes"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name        string
		versionName string
		versionType vtypes.Type
		identifier  string
		commitHash  string
		wantName    string
		wantType    vtypes.Type
		wantID      string
		wantCommit  string
	}{
		{
			name:        "stable version",
			versionName: "stable",
			versionType: vtypes.TypeStable,
			identifier:  "v0.10.0",
			commitHash:  "",
			wantName:    "stable",
			wantType:    vtypes.TypeStable,
			wantID:      "v0.10.0",
			wantCommit:  "",
		},
		{
			name:        "nightly version",
			versionName: "nightly",
			versionType: vtypes.TypeNightly,
			identifier:  "nightly-2024-12-04",
			commitHash:  "abc123def456",
			wantName:    "nightly",
			wantType:    vtypes.TypeNightly,
			wantID:      "nightly-2024-12-04",
			wantCommit:  "abc123def456",
		},
		{
			name:        "commit hash",
			versionName: "1a2b3c4",
			versionType: vtypes.TypeCommit,
			identifier:  "1a2b3c4",
			commitHash:  "1a2b3c4d5e6f7890abcdef1234567890abcdef12",
			wantName:    "1a2b3c4",
			wantType:    vtypes.TypeCommit,
			wantID:      "1a2b3c4",
			wantCommit:  "1a2b3c4d5e6f7890abcdef1234567890abcdef12",
		},
		{
			name:        "specific tag",
			versionName: "v0.9.5",
			versionType: vtypes.TypeTag,
			identifier:  "v0.9.5",
			commitHash:  "",
			wantName:    "v0.9.5",
			wantType:    vtypes.TypeTag,
			wantID:      "v0.9.5",
			wantCommit:  "",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			ver := vtypes.New(
				testCase.versionName,
				testCase.versionType,
				testCase.identifier,
				testCase.commitHash,
			)

			if got := ver.Name(); got != testCase.wantName {
				t.Errorf("Name() = %v, want %v", got, testCase.wantName)
			}

			if got := ver.Type(); got != testCase.wantType {
				t.Errorf("Type() = %v, want %v", got, testCase.wantType)
			}

			if got := ver.Identifier(); got != testCase.wantID {
				t.Errorf("Identifier() = %v, want %v", got, testCase.wantID)
			}

			if got := ver.CommitHash(); got != testCase.wantCommit {
				t.Errorf("CommitHash() = %v, want %v", got, testCase.wantCommit)
			}
		})
	}
}

func TestTypeString(t *testing.T) {
	tests := []struct {
		name string
		t    vtypes.Type
		want string
	}{
		{"stable", vtypes.TypeStable, "stable"},
		{"nightly", vtypes.TypeNightly, "nightly"},
		{"commit", vtypes.TypeCommit, "commit"},
		{"tag", vtypes.TypeTag, "tag"},
		{"unknown", vtypes.Type(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.t.String(); got != tt.want {
				t.Errorf("Type.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsCommitReference(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		// Valid branch names
		{"master branch", "master", true},
		{"main branch", "main", true},

		// Valid commit hashes (different lengths)
		{"7 char hex", "abc1234", true},
		{"8 char hex", "abc12345", true},
		{"40 char hex (full SHA)", "1234567890abcdef1234567890abcdef12345678", true},
		{"mixed case hex", "AbC1234", true},
		{"all digits", "1234567", true},
		{"all hex letters", "abcdefab", true},

		// Invalid inputs
		{"too short (6 chars)", "abc123", false},
		{"too long (41 chars)", "abc1234567890abcdef1234567890abcdef1234567", false},
		{"empty string", "", false},
		{"contains invalid char g", "abc123g", false},
		{"contains space", "abc 123", false},
		{"stable keyword", "stable", false},
		{"nightly keyword", "nightly", false},
		{"version tag", "v0.10.0", false},
		{"contains dash", "abc-123", false},
		{"contains underscore", "abc_123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := vtypes.IsCommitReference(tt.input); got != tt.want {
				t.Errorf("IsCommitReference(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsValidVersionName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		// Valid inputs we expect in the wild
		{"empty string", "", false},
		{"single dot", ".", false},
		{"double dot", "..", false},
		{"literal stable", "stable", true},
		{"literal nightly", "nightly", true},
		{"release tag vX.Y.Z", "v0.10.0", true},
		{"release tag with pre-release", "v0.10.0-beta1", true},
		{"master branch", "master", true},
		{"main branch", "main", true},
		{"7-char hex", "abc1234", true},
		{"40-char hex", "1234567890abcdef1234567890abcdef12345678", true},

		// Path-traversal payloads
		{"parent dir escape", "../etc/passwd", false},
		{"nested parent escape", "stable/../../etc/passwd", false},
		{"absolute path posix", "/etc/passwd", false},
		{"absolute path on Windows", "\\windows\\system32", false},
		{"backslash only", "..\\foo", false},
		{"trailing slash", "stable/", false},
		{"leading slash", "/stable", false},
		{"embedded slash", "stab/le", false},
		{"embedded backslash", `stab\le`, false},

		// Suspicious characters
		{"NUL byte", "v0.10.0\x00", false},
		{"newline", "v0.10.0\n", false},
		{"tab", "v0.10.0\t", false},
		{"space", "v 0.10.0", false},
		{"semicolon (shell metachar)", "stable;rm -rf /", false},
		{"ampersand", "stable&&evil", false},
		{"pipe", "stable|evil", false},
		{"dollar sign", "$HOME", false},
		{"backtick", "`evil`", false},
		{"colon", "C:\\foo", false},
		{"asterisk (glob)", "stable*", false},
		{"question mark (glob)", "stable?", false},
		{"bracket", "v0.10.[0]", false},
		{"non-ASCII (unicode)", "v0.10.0β", false},
		{"emoji", "v0.10.0🎉", false},
		{"DEL byte", "v0.10.0\x7f", false},

		// Edge cases
		{"just a dot and letter", ".v0.10.0", true},
		{"underscore", "v0_10_0", true},
		{"plus sign", "v0.10.0+build1", false}, // not in our allow-list
		{"tilde (home dir)", "~/stable", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := vtypes.IsValidVersionName(tt.input); got != tt.want {
				t.Errorf("IsValidVersionName(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidateVersionName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid stable", "stable", false},
		{"valid v0.10.0", "v0.10.0", false},
		{"valid commit hash", "abc1234", false},
		{"invalid: empty", "", true},
		{"invalid: parent escape", "../etc/passwd", true},
		{"invalid: absolute", "/etc/passwd", true},
		{"invalid: null byte", "v0.10.0\x00", true},
		{"invalid: space", "v 0.10.0", true},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			err := vtypes.ValidateVersionName(testCase.input)

			gotErr := err != nil
			if gotErr != testCase.wantErr {
				t.Errorf(
					"ValidateVersionName(%q) error = %v, wantErr %v",
					testCase.input,
					err,
					testCase.wantErr,
				)
			}
		})
	}
}

func TestValidateVersionName_InvalidIncludesInput(t *testing.T) {
	t.Parallel()

	err := vtypes.ValidateVersionName("../etc/passwd")
	if err == nil {
		t.Fatal("expected error for path-traversal input")
	}

	if !errors.Is(err, vtypes.ErrInvalidVersionName) {
		t.Errorf("expected error to wrap ErrInvalidVersionName, got %v", err)
	}

	// The error message should mention the rejected input so
	// the user knows exactly which character or segment
	// triggered the rejection.
	if !strings.Contains(err.Error(), "../etc/passwd") {
		t.Errorf("expected error message to include rejected input, got %q", err.Error())
	}
}

func TestNormalizeVersionForPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"stable alias", "stable", "stable"},
		{"nightly alias", "nightly", "nightly"},
		{"commit hash preserved", "abc1234", "abc1234"},
		{"master branch preserved", "master", "master"},
		{"main branch preserved", "main", "main"},
		{"bare version gets v prefix", "0.10.0", "v0.10.0"},
		{"already-prefixed version preserved", "v0.10.0", "v0.10.0"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := vtypes.NormalizeVersionForPath(test.input); got != test.want {
				t.Errorf("NormalizeVersionForPath(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}
