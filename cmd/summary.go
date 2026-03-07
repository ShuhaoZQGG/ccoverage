package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/shuhaozhang/ccoverage/internal/coverage"
	"github.com/shuhaozhang/ccoverage/internal/output"
	"github.com/shuhaozhang/ccoverage/internal/scanner"
	"github.com/shuhaozhang/ccoverage/internal/types"
	"github.com/shuhaozhang/ccoverage/internal/usage"
	"github.com/spf13/cobra"
)

var summaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Print a one-line coverage summary (designed for hook use)",
	Long:  "Prints a one-line coverage summary to stderr and exits with code 2. Designed for a SessionEnd hook so the message is shown to the user when the session ends.",
	RunE:  runSummary,
}

func init() {
	rootCmd.AddCommand(summaryCmd)
}

func runSummary(cmd *cobra.Command, args []string) error {
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		log.Printf("ccoverage: resolve repo path: %v", err)
		return nil
	}

	// Read hook input from stdin (with timeout to avoid blocking on TTY).
	readHookInput()

	report, err := buildSummaryReport(absPath)
	if err != nil || report == nil {
		return nil
	}

	// Render one-liner into a buffer.
	var buf bytes.Buffer
	output.RenderOneLine(report, &buf)
	line := strings.TrimSpace(buf.String())
	if line == "" {
		return nil
	}

	// Write to stderr and exit 2 — SessionEnd hooks show stderr to the user.
	fmt.Fprintln(os.Stderr, line)
	os.Exit(2)
	return nil
}

type hookInputData struct {
	SessionID string `json:"session_id"`
}

// readHookInput reads and discards the hook JSON payload from stdin with a timeout.
func readHookInput() {
	ch := make(chan struct{}, 1)
	go func() {
		var input hookInputData
		_ = json.NewDecoder(os.Stdin).Decode(&input)
		ch <- struct{}{}
	}()
	select {
	case <-ch:
	case <-time.After(500 * time.Millisecond):
	}
}

func buildSummaryReport(absPath string) (*types.CoverageReport, error) {
	manifest, err := scanner.BuildManifest(absPath)
	if err != nil {
		log.Printf("ccoverage: scan: %v", err)
		return nil, nil
	}
	if len(manifest.Items) == 0 {
		return nil, nil
	}

	sessionFiles, err := usage.LocateSessionFiles(absPath, lookbackDays)
	if err != nil {
		log.Printf("ccoverage: locate sessions: %v", err)
		return nil, nil
	}

	report, err := coverage.Analyze(manifest, sessionFiles, lookbackDays, 2)
	if err != nil {
		log.Printf("ccoverage: analyze: %v", err)
		return nil, nil
	}
	return report, nil
}
