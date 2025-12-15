package goadstc

import (
	"fmt"
	"runtime/debug"
)

const (
	// VersionMajor is the major version number.
	VersionMajor = 0

	// VersionMinor is the minor version number.
	VersionMinor = 1

	// VersionPatch is the patch version number.
	VersionPatch = 0

	// VersionPrerelease is the pre-release version string (e.g., "alpha", "beta", "rc.1").
	// Empty string for stable releases.
	VersionPrerelease = ""
)

// Version returns the semantic version string of the library.
func Version() string {
	v := fmt.Sprintf("%d.%d.%d", VersionMajor, VersionMinor, VersionPatch)
	if VersionPrerelease != "" {
		v += "-" + VersionPrerelease
	}
	return v
}

// BuildInfo contains version and build information.
type BuildInfo struct {
	Version   string
	GitCommit string
	GitTag    string
	BuildTime string
	GoVersion string
	Dirty     bool
}

// GetBuildInfo returns detailed build information including version and VCS details.
// When built with build info, it includes git commit hash and other metadata.
func GetBuildInfo() BuildInfo {
	info := BuildInfo{
		Version:   Version(),
		GoVersion: "",
	}

	// Try to get build info from runtime/debug
	if buildInfo, ok := debug.ReadBuildInfo(); ok {
		info.GoVersion = buildInfo.GoVersion

		// Extract VCS information from build settings
		for _, setting := range buildInfo.Settings {
			switch setting.Key {
			case "vcs.revision":
				info.GitCommit = setting.Value
				if len(setting.Value) > 7 {
					// Show short commit hash
					info.GitCommit = setting.Value[:7]
				}
			case "vcs.time":
				info.BuildTime = setting.Value
			case "vcs.modified":
				info.Dirty = setting.Value == "true"
			}
		}

		// Try to find git tag from module version
		if buildInfo.Main.Version != "" && buildInfo.Main.Version != "(devel)" {
			info.GitTag = buildInfo.Main.Version
		}
	}

	return info
}

// String returns a human-readable string representation of BuildInfo.
func (b BuildInfo) String() string {
	s := fmt.Sprintf("goadstc %s", b.Version)

	if b.GitCommit != "" {
		s += fmt.Sprintf(" (commit: %s", b.GitCommit)
		if b.Dirty {
			s += "-dirty"
		}
		s += ")"
	}

	if b.GitTag != "" {
		s += fmt.Sprintf(" [%s]", b.GitTag)
	}

	if b.GoVersion != "" {
		s += fmt.Sprintf(" - %s", b.GoVersion)
	}

	return s
}
