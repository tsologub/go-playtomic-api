package filter

import (
	"strings"

	"github.com/rafa-garcia/go-playtomic-api/internal/config"
	"github.com/rafa-garcia/go-playtomic-api/models"
)

// Apply returns the subset of tournaments that match the given filter criteria.
func Apply(tournaments []models.Tournament, f config.TournamentFilter) []models.Tournament {
	var result []models.Tournament
	for _, t := range tournaments {
		if match(t, f) {
			result = append(result, t)
		}
	}
	return result
}

func match(t models.Tournament, f config.TournamentFilter) bool {
	if f.MinAvailablePlaces > 0 && t.AvailablePlaces < f.MinAvailablePlaces {
		return false
	}

	if isBlacklisted(t.Name, f.Blacklist) {
		return false
	}

	if f.PlayerName != "" && hasPlayer(t, f.PlayerName) {
		return false
	}

	return true
}

func hasPlayer(t models.Tournament, name string) bool {
	lower := strings.ToLower(name)
	for _, team := range t.Teams {
		for _, p := range team.Players {
			if strings.ToLower(p.Name) == lower {
				return true
			}
		}
	}
	return false
}

func isBlacklisted(name string, blacklist []string) bool {
	lower := strings.ToLower(name)
	for _, b := range blacklist {
		if strings.Contains(lower, strings.ToLower(b)) {
			return true
		}
	}
	return false
}
