package cmd

import (
	"fmt"
	"runtime/debug"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func init() {
	if version == "dev" {
		if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "(devel)" {
			version = info.Main.Version
		}
	}
	rootCmd.Version = fmt.Sprintf("%s (commit %s, built %s)", version, commit, date)
}
