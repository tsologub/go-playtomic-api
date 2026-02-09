package models

import (
	"net/url"
	"strings"
)

type Tournament struct {
	TournamentID    string `json:"tournament_id"`
	Name            string `json:"name"`
	Visibility      string `json:"visibility"`
	AvailablePlaces int    `json:"available_places"`
	Status          string `json:"status"`
}

type SearchTournamentsParams struct {
	AvailablePlaces    bool
	RegistrationStatus string
	Status             string
	TenantID           string
	Visibility         string
}

func (p *SearchTournamentsParams) ToURLValues() url.Values {
	values := url.Values{}

	if p.AvailablePlaces {
		values.Set("available_places", "true")
	}

	if rs := strings.TrimSpace(p.RegistrationStatus); rs != "" {
		values.Set("registration_status", rs)
	}
	if s := strings.TrimSpace(p.Status); s != "" {
		values.Set("status", s)
	}
	if t := strings.TrimSpace(p.TenantID); t != "" {
		values.Set("tenant_id", t)
	}
	if v := strings.TrimSpace(p.Visibility); v != "" {
		values.Set("visibility", v)
	}

	return values
}
