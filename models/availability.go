package models

import "net/url"

// Slot represents a single available time slot for a court.
type Slot struct {
	StartTime string `json:"start_time"` // "21:00:00" UTC
	Duration  int    `json:"duration"`   // minutes
	Price     string `json:"price"`      // "36 EUR"
}

// CourtAvailability represents availability for a single court (resource) on a given date.
type CourtAvailability struct {
	ResourceID string `json:"resource_id"`
	StartDate  string `json:"start_date"` // "2026-04-10"
	Slots      []Slot `json:"slots"`
}

// SearchAvailabilityParams holds parameters for the /v1/availability endpoint.
type SearchAvailabilityParams struct {
	TenantID string
	SportID  string
	StartMin string // UTC datetime without timezone, e.g. "2026-04-09T22:00:00"
	StartMax string // UTC datetime without timezone, e.g. "2026-04-10T21:59:59"
}

func (p *SearchAvailabilityParams) ToURLValues() url.Values {
	values := url.Values{}
	values.Set("tenant_id", p.TenantID)
	values.Set("sport_id", p.SportID)
	values.Set("start_min", p.StartMin)
	values.Set("start_max", p.StartMax)
	return values
}
