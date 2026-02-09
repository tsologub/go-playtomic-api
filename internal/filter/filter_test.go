package filter

import (
	"testing"

	"github.com/rafa-garcia/go-playtomic-api/internal/config"
	"github.com/rafa-garcia/go-playtomic-api/models"
)

func TestApply_Blacklist(t *testing.T) {
	tournaments := []models.Tournament{
		{TournamentID: "1", Name: "Open Padel Mix", AvailablePlaces: 4, Status: "PENDING"},
		{TournamentID: "2", Name: "Ladies Only Cup", AvailablePlaces: 2, Status: "PENDING"},
		{TournamentID: "3", Name: "Torneo Femenino", AvailablePlaces: 3, Status: "PENDING"},
		{TournamentID: "4", Name: "Summer Championship", AvailablePlaces: 1, Status: "PENDING"},
	}

	f := config.TournamentFilter{
		TenantID:  "t1",
		Blacklist: []string{"ladies", "femenino"},
	}

	result := Apply(tournaments, f)

	if len(result) != 2 {
		t.Fatalf("expected 2 tournaments, got %d", len(result))
	}
	if result[0].TournamentID != "1" {
		t.Errorf("expected first result ID '1', got %q", result[0].TournamentID)
	}
	if result[1].TournamentID != "4" {
		t.Errorf("expected second result ID '4', got %q", result[1].TournamentID)
	}
}

func TestApply_BlacklistCaseInsensitive(t *testing.T) {
	tournaments := []models.Tournament{
		{TournamentID: "1", Name: "LADIES Tournament", AvailablePlaces: 2},
	}

	f := config.TournamentFilter{
		TenantID:  "t1",
		Blacklist: []string{"ladies"},
	}

	result := Apply(tournaments, f)

	if len(result) != 0 {
		t.Fatalf("expected 0 tournaments, got %d", len(result))
	}
}

func TestApply_MinAvailablePlaces(t *testing.T) {
	tournaments := []models.Tournament{
		{TournamentID: "1", Name: "Tournament A", AvailablePlaces: 0},
		{TournamentID: "2", Name: "Tournament B", AvailablePlaces: 1},
		{TournamentID: "3", Name: "Tournament C", AvailablePlaces: 5},
	}

	f := config.TournamentFilter{
		TenantID:           "t1",
		MinAvailablePlaces: 2,
	}

	result := Apply(tournaments, f)

	if len(result) != 1 {
		t.Fatalf("expected 1 tournament, got %d", len(result))
	}
	if result[0].TournamentID != "3" {
		t.Errorf("expected result ID '3', got %q", result[0].TournamentID)
	}
}

func TestApply_NoFilters(t *testing.T) {
	tournaments := []models.Tournament{
		{TournamentID: "1", Name: "Tournament A", AvailablePlaces: 2},
		{TournamentID: "2", Name: "Tournament B", AvailablePlaces: 0},
	}

	f := config.TournamentFilter{
		TenantID: "t1",
	}

	result := Apply(tournaments, f)

	if len(result) != 2 {
		t.Fatalf("expected 2 tournaments, got %d", len(result))
	}
}

func TestApply_EmptyInput(t *testing.T) {
	f := config.TournamentFilter{TenantID: "t1"}

	result := Apply(nil, f)

	if len(result) != 0 {
		t.Fatalf("expected 0 tournaments, got %d", len(result))
	}
}

func TestApply_CombinedFilters(t *testing.T) {
	tournaments := []models.Tournament{
		{TournamentID: "1", Name: "Open Padel", AvailablePlaces: 3},
		{TournamentID: "2", Name: "Ladies Night", AvailablePlaces: 5},
		{TournamentID: "3", Name: "Summer Open", AvailablePlaces: 0},
		{TournamentID: "4", Name: "Mixed Doubles", AvailablePlaces: 2},
	}

	f := config.TournamentFilter{
		TenantID:           "t1",
		MinAvailablePlaces: 1,
		Blacklist:          []string{"ladies"},
	}

	result := Apply(tournaments, f)

	if len(result) != 2 {
		t.Fatalf("expected 2 tournaments, got %d", len(result))
	}
	if result[0].TournamentID != "1" {
		t.Errorf("expected first result ID '1', got %q", result[0].TournamentID)
	}
	if result[1].TournamentID != "4" {
		t.Errorf("expected second result ID '4', got %q", result[1].TournamentID)
	}
}
