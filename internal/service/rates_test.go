package service

import (
	"os"
	"testing"
)

func TestRateService(t *testing.T) {
	tests := []struct {
		name          string
		configContent string
		latency       int64
		joules        float64
		wantCost      float64
		wantCarbon    float64
	}{
		{
			name: "StandardCalculation",
			configContent: `
compute_cad_per_ms: 0.1
energy_cad_per_joule: 0.01
carbon_g_per_joule: 0.5
`,
			latency:    10,
			joules:     2.0,
			wantCost:   1.02, // (10 * 0.1) + (2.0 * 0.01)
			wantCarbon: 1.0,  // 2.0 * 0.5
		},
		{
			name: "ZeroLatency",
			configContent: `
compute_cad_per_ms: 0.1
energy_cad_per_joule: 0.01
carbon_g_per_joule: 0.5
`,
			latency:    0,
			joules:     2.0,
			wantCost:   0.02,
			wantCarbon: 1.0,
		},
		{
			name: "ZeroEnergy",
			configContent: `
compute_cad_per_ms: 0.1
energy_cad_per_joule: 0.01
carbon_g_per_joule: 0.5
`,
			latency:    10,
			joules:     0,
			wantCost:   1.0,
			wantCarbon: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile := "test_rates_" + tt.name + ".yml"
			err := os.WriteFile(tmpFile, []byte(tt.configContent), 0644)
			if err != nil {
				t.Fatalf("Failed to write test config: %v", err)
			}
			defer os.Remove(tmpFile)

			rs, err := NewRateService(tmpFile)
			if err != nil {
				t.Fatalf("Failed to create RateService: %v", err)
			}

			cost, carbon := rs.CalculateEconomicImpact(tt.latency, tt.joules)

			if cost != tt.wantCost {
				t.Errorf("got cost %f, want %f", cost, tt.wantCost)
			}
			if carbon != tt.wantCarbon {
				t.Errorf("got carbon %f, want %f", carbon, tt.wantCarbon)
			}
		})
	}
}
