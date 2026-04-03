package service

import (
	"math"
	"testing"
)

func TestRateService(t *testing.T) {
	const epsilon = 1e-10

	tests := []struct {
		name       string
		card       RateCard
		latency    int64
		joules     float64
		wantCost   float64
		wantCarbon float64
	}{
		{
			name: "StandardCalculation",
			card: RateCard{
				ComputeCADPerMs:   0.1,
				EnergyCADPerJoule: 0.01,
				CarbonGPerJoule:   0.5,
			},
			latency:    10,
			joules:     2.0,
			wantCost:   1.02, // (10 * 0.1) + (2.0 * 0.01)
			wantCarbon: 1.0,  // 2.0 * 0.5
		},
		{
			name: "ZeroLatency",
			card: RateCard{
				ComputeCADPerMs:   0.1,
				EnergyCADPerJoule: 0.01,
				CarbonGPerJoule:   0.5,
			},
			latency:    0,
			joules:     2.0,
			wantCost:   0.02,
			wantCarbon: 1.0,
		},
		{
			name: "ZeroEnergy",
			card: RateCard{
				ComputeCADPerMs:   0.1,
				EnergyCADPerJoule: 0.01,
				CarbonGPerJoule:   0.5,
			},
			latency:    10,
			joules:     0,
			wantCost:   1.0,
			wantCarbon: 0,
		},
		{
			name:       "DefaultRates",
			card:       DefaultRateCard,
			latency:    1000, // 1 second
			joules:     10,
			wantCost:   (1000 * 0.0001) + (10 * 0.000000042),
			wantCarbon: 10 * 0.000042,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rs := &RateService{Card: tt.card}
			cost, carbon := rs.CalculateEconomicImpact(tt.latency, tt.joules)

			if math.Abs(cost-tt.wantCost) > epsilon {
				t.Errorf("got cost %f, want %f", cost, tt.wantCost)
			}
			if math.Abs(carbon-tt.wantCarbon) > epsilon {
				t.Errorf("got carbon %f, want %f", carbon, tt.wantCarbon)
			}
		})
	}
}
