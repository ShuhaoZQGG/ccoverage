package cmd

import "fmt"

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func init() {
	rootCmd.Version = fmt.Sprintf("%s (commit %s, built %s)", version, commit, date)
}
