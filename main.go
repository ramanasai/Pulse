package main

import (
	"github.com/ramanasai/pulse/internal/ui"
	"github.com/ramanasai/pulse/internal/version"
)

// Build metadata injected by goreleaser or makefile
var (
	buildVersion = "dev"
	buildCommit  = "none"
	buildDate    = "unknown"
)

func init() {
	// Set version info
	version.Version = buildVersion
	version.Commit = buildCommit
	version.Date = buildDate
}

func main() { _ = ui.Run() }
