package version

import (
	"fmt"
	"runtime"
)

// Build metadata injected by goreleaser or makefile
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// GetVersion returns a formatted version string
func GetVersion() string {
	if Version == "dev" {
		return "dev"
	}
	return Version
}

// GetVersionInfo returns detailed version information
func GetVersionInfo() string {
	if Version == "dev" {
		return fmt.Sprintf("Pulse dev (%s, %s)", runtime.GOOS, runtime.GOARCH)
	}
	return fmt.Sprintf("Pulse %s (commit: %s, built: %s, %s/%s)",
		Version, Commit, Date, runtime.GOOS, runtime.GOARCH)
}

// GetShortVersion returns a short version string for display
func GetShortVersion() string {
	if Version == "dev" {
		return "Pulse dev"
	}
	return fmt.Sprintf("Pulse %s", Version)
}