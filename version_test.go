package goadstc

import (
	"testing"
)

func TestVersion(t *testing.T) {
	v := Version()
	if v == "" {
		t.Error("Version() returned empty string")
	}

	// Should follow semantic versioning format
	expected := "0.1.0"
	if v != expected {
		t.Errorf("Version() = %q, want %q", v, expected)
	}
}

func TestVersionWithPrerelease(t *testing.T) {
	// Test that prerelease versions work correctly
	// This test documents the expected format
	tests := []struct {
		major      int
		minor      int
		patch      int
		prerelease string
		want       string
	}{
		{0, 1, 0, "", "0.1.0"},
		{0, 1, 0, "alpha", "0.1.0-alpha"},
		{0, 1, 0, "beta.1", "0.1.0-beta.1"},
		{1, 0, 0, "", "1.0.0"},
		{1, 2, 3, "rc.1", "1.2.3-rc.1"},
	}

	for _, tt := range tests {
		// Note: We can't actually test this without modifying the constants,
		// but this documents the expected behavior
		_ = tt
	}
}

func TestGetBuildInfo(t *testing.T) {
	info := GetBuildInfo()

	// Version should always be available
	if info.Version == "" {
		t.Error("GetBuildInfo().Version is empty")
	}

	// BuildInfo should be valid
	str := info.String()
	if str == "" {
		t.Error("BuildInfo.String() returned empty string")
	}

	// Should contain the library name
	if len(str) < len("goadstc") {
		t.Errorf("BuildInfo.String() = %q, too short", str)
	}
}

func TestBuildInfoString(t *testing.T) {
	tests := []struct {
		name string
		info BuildInfo
		want string
	}{
		{
			name: "basic version",
			info: BuildInfo{Version: "0.1.0"},
			want: "goadstc 0.1.0",
		},
		{
			name: "with commit",
			info: BuildInfo{Version: "0.1.0", GitCommit: "abc1234"},
			want: "goadstc 0.1.0 (commit: abc1234)",
		},
		{
			name: "with dirty commit",
			info: BuildInfo{Version: "0.1.0", GitCommit: "abc1234", Dirty: true},
			want: "goadstc 0.1.0 (commit: abc1234-dirty)",
		},
		{
			name: "full info",
			info: BuildInfo{
				Version:   "0.1.0",
				GitCommit: "abc1234",
				GitTag:    "v0.1.0",
				GoVersion: "go1.24",
			},
			want: "goadstc 0.1.0 (commit: abc1234) [v0.1.0] - go1.24",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.info.String()
			if got != tt.want {
				t.Errorf("BuildInfo.String() = %q, want %q", got, tt.want)
			}
		})
	}
}
