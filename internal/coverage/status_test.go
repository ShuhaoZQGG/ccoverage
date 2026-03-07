package coverage

import (
	"testing"

	"github.com/shuhaozhang/ccoverage/internal/types"
)

func TestClassify(t *testing.T) {
	tests := []struct {
		name      string
		exists    bool
		activations int
		threshold int
		want      types.Status
	}{
		{"orphaned", false, 5, 2, types.StatusOrphaned},
		{"dormant", true, 0, 2, types.StatusDormant},
		{"underused", true, 1, 2, types.StatusUnderused},
		{"underused_at_threshold", true, 2, 2, types.StatusUnderused},
		{"active", true, 3, 2, types.StatusActive},
		{"active_high", true, 100, 2, types.StatusActive},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := types.ManifestItem{Exists: tt.exists}
			usage := types.UsageSummary{TotalActivations: tt.activations}
			got := Classify(item, usage, tt.threshold)
			if got != tt.want {
				t.Errorf("Classify() = %q, want %q", got, tt.want)
			}
		})
	}
}
