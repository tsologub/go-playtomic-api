package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Tournaments []TournamentFilter `yaml:"tournaments"`
	Classes     []ClassFilter      `yaml:"classes"`
	Courts      []CourtFilter      `yaml:"courts"`
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

// TimeWindow defines a time range of interest using HH:MM strings in UTC.
type TimeWindow struct {
	Start string `yaml:"start"` // e.g. "17:00"
	End   string `yaml:"end"`   // e.g. "20:00"
}

// CourtFilter holds configuration for querying court availability.
type CourtFilter struct {
	TenantID        string       `yaml:"tenant_id"`
	SportID         string       `yaml:"sport_id"`
	TimeWindows     []TimeWindow `yaml:"time_windows"`
	IgnoredCourtIDs []string     `yaml:"ignored_court_ids"`
	IgnoredDays     []string     `yaml:"ignored_days"`
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
	if len(c.Tournaments) == 0 && len(c.Classes) == 0 && len(c.Courts) == 0 {
		return fmt.Errorf("at least one tournament, class, or court filter is required")
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

	for i, ct := range c.Courts {
		if ct.TenantID == "" {
			return fmt.Errorf("courts[%d]: tenant_id is required", i)
		}
		if ct.SportID == "" {
			return fmt.Errorf("courts[%d]: sport_id is required", i)
		}
		if len(ct.TimeWindows) == 0 {
			return fmt.Errorf("courts[%d]: at least one time_window is required", i)
		}
	}

	return nil
}
