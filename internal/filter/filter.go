package filter

import (
	"strings"

	"github.com/rafa-garcia/go-playtomic-api/internal/config"
	"github.com/rafa-garcia/go-playtomic-api/models"
)

// ApplyClasses returns the subset of classes that match the given filter criteria.
func ApplyClasses(classes []models.Class, f config.ClassFilter) []models.Class {
	var result []models.Class
	for _, c := range classes {
		if matchClass(c, f) {
			result = append(result, c)
		}
	}
	return result
}

func matchClass(c models.Class, f config.ClassFilter) bool {
	// Filter by coach name
	if len(f.CoachNames) > 0 && !hasAnyCoach(c, f.CoachNames) {
		return false
	}

	// Filter out if player is already registered
	if f.PlayerName != "" && isRegistered(c, f.PlayerName) {
		return false
	}

	// Filter by course name (whitelist)
	if len(f.CourseNames) > 0 && c.CourseSummary != nil {
		if !isInCourseNames(c.CourseSummary.Name, f.CourseNames) {
			return false
		}
	}

	// Filter by blacklist
	if c.CourseSummary != nil && isBlacklisted(c.CourseSummary.Name, f.Blacklist) {
		return false
	}

	return true
}

func hasAnyCoach(c models.Class, names []string) bool {
	for _, name := range names {
		lower := strings.ToLower(name)

		for _, coach := range c.Coaches {
			if strings.Contains(strings.ToLower(coach.Name), lower) {
				return true
			}
		}
	}
	return false
}

func isRegistered(c models.Class, name string) bool {
	lower := strings.ToLower(name)
	for _, reg := range c.RegistrationInfo.Registrations {
		if strings.Contains(strings.ToLower(reg.Player.Name), lower) {
			return true
		}
	}
	return false
}

func isInCourseNames(courseName string, courseNames []string) bool {
	lower := strings.ToLower(courseName)
	for _, cn := range courseNames {
		if strings.Contains(lower, strings.ToLower(cn)) {
			return true
		}
	}
	return false
}

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
