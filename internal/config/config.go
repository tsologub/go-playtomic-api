package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Tournaments []TournamentFilter `yaml:"tournaments"`
}

type TournamentFilter struct {
	TenantID           string   `yaml:"tenant_id"`
	Visibility         string   `yaml:"visibility"`
	RegistrationStatus string   `yaml:"registration_status"`
	Status             string   `yaml:"status"`
	MinAvailablePlaces int      `yaml:"min_available_places"`
	Blacklist          []string `yaml:"blacklist"`
	PlayerName         string   `yaml:"player_name"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

func (c *Config) validate() error {
	if len(c.Tournaments) == 0 {
		return fmt.Errorf("at least one tournament filter is required")
	}

	for i, t := range c.Tournaments {
		if t.TenantID == "" {
			return fmt.Errorf("tournaments[%d]: tenant_id is required", i)
		}
	}

	return nil
}
