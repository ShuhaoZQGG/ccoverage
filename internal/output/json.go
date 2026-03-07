package output

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/shuhaozhang/ccoverage/internal/types"
)

// RenderJSON serialises report as indented JSON and writes it to w, followed
// by a trailing newline. Any encoding or write error is returned to the caller.
func RenderJSON(report *types.CoverageReport, w io.Writer) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("output: marshal report to JSON: %w", err)
	}

	if _, err := fmt.Fprintf(w, "%s\n", data); err != nil {
		return fmt.Errorf("output: write JSON: %w", err)
	}

	return nil
}
