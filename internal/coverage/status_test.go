package coverage

import (
	"testing"

	"github.com/ShuhaoZQGG/ccoverage/internal/types"
)

func TestClassify(t *testing.T) {
	tests := []struct {
		name        string
		activations int
		threshold   int
		want        types.Status
	}{
		{"dormant", 0, 2, types.StatusDormant},
		{"underused", 1, 2, types.StatusUnderused},
		{"underused_at_threshold", 2, 2, types.StatusUnderused},
		{"active", 3, 2, types.StatusActive},
		{"active_high", 100, 2, types.StatusActive},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := types.ManifestItem{}
			usage := types.UsageSummary{TotalActivations: tt.activations}
			got := Classify(item, usage, tt.threshold)
			if got != tt.want {
				t.Errorf("Classify() = %q, want %q", got, tt.want)
			}
		})
	}
}
