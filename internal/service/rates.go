package service

// RateCard defines the pricing and emission factors.
type RateCard struct {
	ComputeCADPerMs      float64
	EnergyCADPerJoule    float64
	CarbonGPerJoule      float64
	TargetHitRate        float64
	DecayStep            int
	RefineIntervalEvents int
}

// DefaultRateCard provides Canada-based FinOps and GreenOps benchmarks.
// These are embedded directly in the binary to ensure the analytics engine
// is "Zero-Config" and always functional regardless of working directory.
var DefaultRateCard = RateCard{
	ComputeCADPerMs:      0.0001,      // $0.10 CAD per second
	EnergyCADPerJoule:    0.000000042, // $0.15 CAD per kWh
	CarbonGPerJoule:      0.000042,    // 150g CO2e per kWh
	TargetHitRate:        0.15,
	DecayStep:            1,
	RefineIntervalEvents: 50,
}

// RateService manages the financial and environmental configuration.
type RateService struct {
	Card RateCard
}

// NewRateService initializes a RateService with the default card.
func NewRateService() *RateService {
	return &RateService{Card: DefaultRateCard}
}

// CalculateEconomicImpact returns the cost and carbon for a given event.
func (rs *RateService) CalculateEconomicImpact(latencyMs int64, joules float64) (float64, float64) {
	cost := (float64(latencyMs) * rs.Card.ComputeCADPerMs) + (joules * rs.Card.EnergyCADPerJoule)
	carbon := joules * rs.Card.CarbonGPerJoule
	return cost, carbon
}
