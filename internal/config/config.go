package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Tournaments []TournamentFilter `yaml:"tournaments"`
	Classes     []ClassFilter      `yaml:"classes"`
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

type ClassFilter struct {
	TenantID          string   `yaml:"tenant_id"`
	CourseVisibility  string   `yaml:"course_visibility"`
	ShowOnlyAvailable bool     `yaml:"show_only_available"`
	Status            string   `yaml:"status"`
	Type              string   `yaml:"type"`
	CoachNames        []string `yaml:"coach_names"`
	PlayerName        string   `yaml:"player_name"`
	CourseNames       []string `yaml:"course_names"`
	Blacklist         []string `yaml:"blacklist"`
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
	if len(c.Tournaments) == 0 && len(c.Classes) == 0 {
		return fmt.Errorf("at least one tournament or class filter is required")
	}

	for i, t := range c.Tournaments {
		if t.TenantID == "" {
			return fmt.Errorf("tournaments[%d]: tenant_id is required", i)
		}
	}

	for i, cl := range c.Classes {
		if cl.TenantID == "" {
			return fmt.Errorf("classes[%d]: tenant_id is required", i)
		}
	}

	return nil
}
