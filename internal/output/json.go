package output

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/ShuhaoZQGG/ccoverage/internal/types"
)

// RenderManifestJSON serialises a manifest as indented JSON and writes it to w.
func RenderManifestJSON(manifest *types.Manifest, w io.Writer) error {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("output: marshal manifest to JSON: %w", err)
	}
	if _, err := fmt.Fprintf(w, "%s\n", data); err != nil {
		return fmt.Errorf("output: write JSON: %w", err)
	}
	return nil
}

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
