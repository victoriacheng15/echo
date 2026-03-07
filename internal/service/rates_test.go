package service

import (
	"os"
	"testing"
)

func TestRateService(t *testing.T) {
	configContent := `
compute_usd_per_ms: 0.1
energy_usd_per_joule: 0.01
carbon_g_per_joule: 0.5
`
	tmpFile := "test_rates.yml"
	err := os.WriteFile(tmpFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}
	defer os.Remove(tmpFile)

	rs, err := NewRateService(tmpFile)
	if err != nil {
		t.Fatalf("Failed to create RateService: %v", err)
	}

	t.Run("CalculateImpact", func(t *testing.T) {
		latency := int64(10)
		joules := 2.0
		
		// cost = (10 * 0.1) + (2.0 * 0.01) = 1.0 + 0.02 = 1.02
		// carbon = 2.0 * 0.5 = 1.0
		cost, carbon := rs.CalculateEconomicImpact(latency, joules)
		
		if cost != 1.02 {
			t.Errorf("Expected cost 1.02, got %f", cost)
		}
		if carbon != 1.0 {
			t.Errorf("Expected carbon 1.0, got %f", carbon)
		}
	})
}
