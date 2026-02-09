// Example showing basic usage of the Playtomic API client
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/rafa-garcia/go-playtomic-api/client"
	"github.com/rafa-garcia/go-playtomic-api/models"
)

func main() {
	// Create a client with options
	c := client.NewClient(
		client.WithTimeout(15*time.Second),
		client.WithRetries(2),
		client.WithBaseURL(client.DefaultBaseUrlV2),
	)

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Search for Tournaments
	fmt.Println("Searching for tournaments...")
	tournaments, err := searchTournaments(ctx, c)

	if len(tournaments) > 0 {
		fmt.Println("Found tournaments")
	}
	if err != nil {
		log.Fatalf("Error searching tournaments: %v", err)
	}
}

// searchClasses demonstrates searching for classes
func searchClasses(ctx context.Context, c *client.Client) ([]models.Class, error) {
	// Build search parameters
	classParams := &models.SearchClassesParams{
		Sort:             "start_date,ASC",
		Status:           "PENDING,IN_PROGRESS",
		Type:             "COURSE,PUBLIC",
		IncludeSummary:   true,
		Size:             100,
		Page:             0,
		CourseVisibility: "PUBLIC",
		FromStartDate:    time.Now().Format("2006-01-02") + "T00:00:00",
	}

	// Add tenant IDs if provided
	tenantID := os.Getenv("PLAYTOMIC_TENANT_ID")
	if tenantID != "" {
		classParams.TenantIDs = []string{tenantID}
	}

	return c.GetClasses(ctx, classParams)
}

func searchTournaments(ctx context.Context, c *client.Client) ([]models.Tournament, error) {
	tournamentParams := &models.SearchTournamentsParams{
		AvailablePlaces:    true,
		RegistrationStatus: "OPEN",
		Status:             "PENDING",
		TenantID:           "8b818dae-aacb-4ea3-aa7b-0e77b1149c85",
		Visibility:         "PUBLIC",
	}

	return c.GetTournaments(ctx, tournamentParams)
}

// searchMatches demonstrates searching for matches
func searchMatches(ctx context.Context, c *client.Client) ([]models.Match, error) {
	// Build search parameters
	matchParams := &models.SearchMatchesParams{
		Sort:          "start_date,DESC",
		HasPlayers:    true,
		SportID:       "PADEL",
		Visibility:    "VISIBLE",
		FromStartDate: time.Now().Format("2006-01-02") + "T00:00:00",
		Size:          100,
		Page:          0,
	}

	// Add tenant IDs if provided
	tenantID := os.Getenv("PLAYTOMIC_TENANT_ID")
	if tenantID != "" {
		matchParams.TenantIDs = []string{tenantID}
	}

	return c.GetMatches(ctx, matchParams)
}

// searchLessons demonstrates searching for lessons
func searchLessons(ctx context.Context, c *client.Client) ([]models.Lesson, error) {
	// Build search parameters
	lessonParams := &models.SearchLessonsParams{
		Sort:                 "start_date,ASC",
		Status:               "REGISTRATION_OPEN,REGISTRATION_CLOSED,IN_PROGRESS",
		TournamentVisibility: "PUBLIC",
		Size:                 100,
		Page:                 0,
		FromStartDate:        time.Now().Format("2006-01-02") + "T00:00:00",
	}

	// Add tenant ID if provided
	// Note: Lessons API only accepts a single tenant_id, not a list
	tenantID := os.Getenv("PLAYTOMIC_TENANT_ID")
	if tenantID != "" {
		lessonParams.TenantID = tenantID
	}

	return c.GetLessons(ctx, lessonParams)
}

// Helper function to get class title
func getClassTitle(class models.Class) string {
	if class.CourseSummary != nil && class.CourseSummary.Name != "" {
		return class.CourseSummary.Name
	}
	return class.Resource.Name
}

// Helper function to count registered players in a match
func countPlayers(match models.Match) int {
	count := 0
	for _, team := range match.Teams {
		count += len(team.Players)
	}
	return count
}

// Helper function to calculate total player slots in a match
func totalPlayerSlots(match models.Match) int {
	return match.MinPlayersPerTeam * len(match.Teams)
}
