package service

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

// RateCard defines the pricing and emission factors.
type RateCard struct {
	ComputeCADPerMs      float64 `yaml:"compute_cad_per_ms"`
	EnergyCADPerJoule    float64 `yaml:"energy_cad_per_joule"`
	CarbonGPerJoule      float64 `yaml:"carbon_g_per_joule"`
	TargetHitRate        float64 `yaml:"target_hit_rate"`
	DecayStep            int     `yaml:"decay_step"`
	RefineIntervalEvents int     `yaml:"refine_interval_events"`
}

// RateService manages the financial and environmental configuration.
type RateService struct {
	Card RateCard
}

// NewRateService loads rates from a YAML file.
func NewRateService(configPath string) (*RateService, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read rate card: %w", err)
	}

	var card RateCard
	if err := yaml.Unmarshal(data, &card); err != nil {
		return nil, fmt.Errorf("failed to parse rate card: %w", err)
	}

	return &RateService{Card: card}, nil
}

// CalculateEconomicImpact returns the cost and carbon for a given event.
func (rs *RateService) CalculateEconomicImpact(latencyMs int64, joules float64) (float64, float64) {
	cost := (float64(latencyMs) * rs.Card.ComputeCADPerMs) + (joules * rs.Card.EnergyCADPerJoule)
	carbon := joules * rs.Card.CarbonGPerJoule
	return cost, carbon
}
