package version_test

import (
	"testing"

	"github.com/y3owk1n/nvs/internal/domain/version"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name        string
		versionName string
		versionType version.Type
		identifier  string
		commitHash  string
		wantName    string
		wantType    version.Type
		wantID      string
		wantCommit  string
	}{
		{
			name:        "stable version",
			versionName: "stable",
			versionType: version.TypeStable,
			identifier:  "v0.10.0",
			commitHash:  "",
			wantName:    "stable",
			wantType:    version.TypeStable,
			wantID:      "v0.10.0",
			wantCommit:  "",
		},
		{
			name:        "nightly version",
			versionName: "nightly",
			versionType: version.TypeNightly,
			identifier:  "nightly-2024-12-04",
			commitHash:  "abc123def456",
			wantName:    "nightly",
			wantType:    version.TypeNightly,
			wantID:      "nightly-2024-12-04",
			wantCommit:  "abc123def456",
		},
		{
			name:        "commit hash",
			versionName: "1a2b3c4",
			versionType: version.TypeCommit,
			identifier:  "1a2b3c4",
			commitHash:  "1a2b3c4d5e6f7890abcdef1234567890abcdef12",
			wantName:    "1a2b3c4",
			wantType:    version.TypeCommit,
			wantID:      "1a2b3c4",
			wantCommit:  "1a2b3c4d5e6f7890abcdef1234567890abcdef12",
		},
		{
			name:        "specific tag",
			versionName: "v0.9.5",
			versionType: version.TypeTag,
			identifier:  "v0.9.5",
			commitHash:  "",
			wantName:    "v0.9.5",
			wantType:    version.TypeTag,
			wantID:      "v0.9.5",
			wantCommit:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := version.New(tt.versionName, tt.versionType, tt.identifier, tt.commitHash)

			if got := v.Name(); got != tt.wantName {
				t.Errorf("Name() = %v, want %v", got, tt.wantName)
			}

			if got := v.Type(); got != tt.wantType {
				t.Errorf("Type() = %v, want %v", got, tt.wantType)
			}

			if got := v.Identifier(); got != tt.wantID {
				t.Errorf("Identifier() = %v, want %v", got, tt.wantID)
			}

			if got := v.CommitHash(); got != tt.wantCommit {
				t.Errorf("CommitHash() = %v, want %v", got, tt.wantCommit)
			}
		})
	}
}

func TestTypeString(t *testing.T) {
	tests := []struct {
		name string
		t    version.Type
		want string
	}{
		{"stable", version.TypeStable, "stable"},
		{"nightly", version.TypeNightly, "nightly"},
		{"commit", version.TypeCommit, "commit"},
		{"tag", version.TypeTag, "tag"},
		{"unknown", version.Type(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.t.String(); got != tt.want {
				t.Errorf("Type.String() = %v, want %v", got, tt.want)
			}
		})
	}
}
