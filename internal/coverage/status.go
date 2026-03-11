// Package coverage classifies manifest items and builds coverage reports by
// joining manifest data with session usage summaries.
package coverage

import "github.com/ShuhaoZQGG/ccoverage/internal/types"

// Classify returns the Status for a single manifest item given its observed
// usage and the configured underuse threshold.
//
// Classification rules (evaluated in order):
//  1. If the item was never activated → StatusDormant
//  2. If total activations are at or below threshold → StatusUnderused
//  3. Otherwise → StatusActive
func Classify(item types.ManifestItem, usage types.UsageSummary, threshold int) types.Status {
	if usage.TotalActivations == 0 {
		return types.StatusDormant
	}
	if usage.TotalActivations <= threshold {
		return types.StatusUnderused
	}
	return types.StatusActive
}
