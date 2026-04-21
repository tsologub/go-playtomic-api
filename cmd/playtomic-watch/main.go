package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/rafa-garcia/go-playtomic-api/client"
	"github.com/rafa-garcia/go-playtomic-api/internal/config"
	"github.com/rafa-garcia/go-playtomic-api/internal/filter"
	"github.com/rafa-garcia/go-playtomic-api/internal/state"
	"github.com/rafa-garcia/go-playtomic-api/internal/telegram"
	"github.com/rafa-garcia/go-playtomic-api/models"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to configuration file")
	timeout := flag.Duration("timeout", 30*time.Second, "HTTP request timeout")
	telegramToken := flag.String("telegram-token", "", "Telegram bot token")
	telegramChatID := flag.String("telegram-chat-id", "", "Telegram chat ID")
	tournamentStatePath := flag.String("tournament-state", "tournament-state.json", "path to tournament state file")
	classStatePath := flag.String("class-state", "class-state.json", "path to class state file")
	courtStatePath := flag.String("court-state", "court-state.json", "path to court state file")
	flag.Parse()

	// Check for subcommand
	args := flag.Args()
	if len(args) == 0 {
		printUsage()
		log.Fatalf("Error: subcommand required")
	}

	subcommand := args[0]
	if subcommand != "tournaments" && subcommand != "classes" && subcommand != "courts" {
		printUsage()
		log.Fatalf("Error: invalid subcommand '%s'", subcommand)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	var bot *telegram.Bot
	if *telegramToken != "" && *telegramChatID != "" {
		bot = telegram.NewBot(*telegramToken, *telegramChatID)
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	var sb strings.Builder
	var notificationState *state.State

	switch subcommand {
	case "tournaments":
		if len(cfg.Tournaments) == 0 {
			log.Fatalf("No tournament filters configured in %s", *configPath)
		}

		// Initialize and load state
		notificationState = state.New(*tournamentStatePath)
		if err := notificationState.Load(); err != nil {
			log.Fatalf("Failed to load tournament state: %v", err)
		}
		defer func() {
			if err := notificationState.Save(); err != nil {
				log.Printf("Failed to save tournament state: %v", err)
			}
		}()

		// Create client for v2 API (tournaments)
		v2Client := client.NewClient(
			client.WithTimeout(*timeout),
			client.WithBaseURL(client.DefaultBaseUrlV2),
		)

		var matchedTournaments []models.Tournament
		for _, tf := range cfg.Tournaments {
			tournaments, err := fetchTournaments(ctx, v2Client, tf)
			if err != nil {
				log.Printf("Error fetching tournaments for tenant %s: %v", tf.TenantID, err)
				continue
			}

			matchedTournaments = append(matchedTournaments, filter.Apply(tournaments, tf)...)
		}

		if len(matchedTournaments) == 0 {
			fmt.Println("No matching tournaments found.")
			return
		}

		for _, t := range matchedTournaments {
			printTournament(t)

			// Check if we should notify about this tournament
			if notificationState.ShouldNotify(t.TournamentID, t.AvailablePlaces) {
				log.Printf("📢 Found new tournament '%s', sending notification", t.Name)
				formatTournament(&sb, t)
			} else {
				log.Printf("✓ Tournament '%s' already in state, skipping notification", t.Name)
			}

			// Update state with current information
			notificationState.Update(t.TournamentID, t.AvailablePlaces)
		}

	case "classes":
		if len(cfg.Classes) == 0 {
			log.Fatalf("No class filters configured in %s", *configPath)
		}

		// Initialize and load state
		notificationState = state.New(*classStatePath)
		if err := notificationState.Load(); err != nil {
			log.Fatalf("Failed to load class state: %v", err)
		}
		defer func() {
			if err := notificationState.Save(); err != nil {
				log.Printf("Failed to save class state: %v", err)
			}
		}()

		// Create client for v1 API (classes)
		v1Client := client.NewClient(
			client.WithTimeout(*timeout),
			client.WithBaseURL(client.DefaultBaseUrlV1),
		)

		var matchedClasses []models.Class
		for _, cf := range cfg.Classes {
			classes, err := fetchClasses(ctx, v1Client, cf)
			if err != nil {
				log.Printf("Error fetching classes for tenant %s: %v", cf.TenantID, err)
				continue
			}

			matchedClasses = append(matchedClasses, filter.ApplyClasses(classes, cf)...)
		}

		if len(matchedClasses) == 0 {
			fmt.Println("No matching classes found.")
			return
		}

		for _, c := range matchedClasses {
			printClass(c)

			// Calculate available places for classes
			availablePlaces := 0
			if c.CourseSummary != nil {
				availablePlaces = c.CourseSummary.MaxPlayers - len(c.RegistrationInfo.Registrations)
			}

			// Get class name for logging
			className := "Unknown"
			if c.CourseSummary != nil {
				className = c.CourseSummary.Name
			}

			// Check if we should notify about this class
			if notificationState.ShouldNotify(c.AcademyClassID, availablePlaces) {
				log.Printf("📢 Found new class '%s', sending notification", className)
				formatClass(&sb, c)
			} else {
				log.Printf("✓ Class '%s' already in state, skipping notification", className)
			}

			// Update state with current information
			notificationState.Update(c.AcademyClassID, availablePlaces)
		}

	case "courts":
		if len(cfg.Courts) == 0 {
			log.Fatalf("No court filters configured in %s", *configPath)
		}

		courtState := state.New(*courtStatePath)
		if err := courtState.Load(); err != nil {
			log.Fatalf("Failed to load court state: %v", err)
		}
		defer func() {
			if err := courtState.Save(); err != nil {
				log.Printf("Failed to save court state: %v", err)
			}
		}()

		v1Client := client.NewClient(
			client.WithTimeout(*timeout),
			client.WithBaseURL(client.DefaultBaseUrlV1),
		)

		berlinLoc, err := time.LoadLocation("Europe/Berlin")
		if err != nil {
			log.Fatalf("Failed to load Europe/Berlin timezone: %v", err)
		}

		var totalMatched int
		now := time.Now().UTC()

		for _, cf := range cfg.Courts {
			clubName := tenantName(cf.TenantID)
			var clubMatches int

			for day := 0; day <= 14; day++ {
				date := now.AddDate(0, 0, day)
				availability, err := fetchCourtAvailability(ctx, v1Client, cf, date)
				if err != nil {
					log.Printf("Error fetching courts for tenant %s on %s: %v",
						cf.TenantID, date.Format("2006-01-02"), err)
					continue
				}

				matched := filter.ApplyCourts(availability, cf)
				for _, court := range matched {
					for _, slot := range court.Slots {
						printCourtSlot(clubName, court, slot, berlinLoc)

						slotKey := court.ResourceID + "|" + court.StartDate + "|" + slot.StartTime
						if courtState.ShouldNotify(slotKey, 1) {
							log.Printf("📢 New court slot %s at %s, sending notification", court.ResourceID, slot.StartTime)
							formatCourtSlot(&sb, clubName, court, slot, berlinLoc)
						} else {
							log.Printf("✓ Court slot %s at %s already in state, skipping notification", court.ResourceID, slot.StartTime)
						}
						courtState.Update(slotKey, 1)
						clubMatches++
						totalMatched++
					}
				}
			}

			if clubMatches == 0 {
				fmt.Printf("No available courts found for %s.\n", clubName)
			}
		}

		if totalMatched == 0 {
			fmt.Println("No available courts found.")
			return
		}
	}

	if sb.Len() > 0 {
		log.Println("📧 Sending notification with new items")
		notify(bot, sb.String())
	} else {
		log.Println("ℹ️  No new items to notify about")
	}
}

// tenantNames maps known tenant IDs to human-readable club names.
var tenantNames = map[string]string{
	"8b818dae-aacb-4ea3-aa7b-0e77b1149c85": "Charlotte-Mitte",
	"9fea856e-7d1a-4cae-9831-79015318967b": "PBC",
	"4a3497a5-f9bd-43eb-9aaa-a972a856b3d2": "PadelBros",
}

func tenantName(id string) string {
	if name, ok := tenantNames[id]; ok {
		return name
	}
	return id
}

func printUsage() {
	fmt.Println("Usage: playtomic-watch [OPTIONS] <tournaments|classes|courts>")
	fmt.Println("\nSubcommands:")
	fmt.Println("  tournaments    Search for tournaments")
	fmt.Println("  classes        Search for classes")
	fmt.Println("  courts         Search for available courts")
	fmt.Println("\nOptions:")
	flag.PrintDefaults()
}

func fetchTournaments(ctx context.Context, c *client.Client, tf config.TournamentFilter) ([]models.Tournament, error) {
	params := &models.SearchTournamentsParams{
		TenantID: tf.TenantID,
	}

	if tf.Visibility != "" {
		params.Visibility = tf.Visibility
	}
	if tf.RegistrationStatus != "" {
		params.RegistrationStatus = tf.RegistrationStatus
	}
	if tf.Status != "" {
		params.Status = tf.Status
	}
	if tf.MinAvailablePlaces > 0 {
		params.AvailablePlaces = true
	}

	return c.GetTournaments(ctx, params)
}

func fetchClasses(ctx context.Context, c *client.Client, cf config.ClassFilter) ([]models.Class, error) {
	params := &models.SearchClassesParams{
		TenantIDs: []string{cf.TenantID},
	}

	if cf.CourseVisibility != "" {
		params.CourseVisibility = cf.CourseVisibility
	}
	if cf.ShowOnlyAvailable {
		params.ShowOnlyAvailable = true
	}
	if cf.Status != "" {
		params.Status = cf.Status
	}
	if cf.Type != "" {
		params.Type = cf.Type
	}

	return c.GetClasses(ctx, params)
}

func notify(bot *telegram.Bot, msg string) {
	if bot == nil {
		return
	}
	if err := bot.Send(msg); err != nil {
		log.Printf("Failed to send Telegram message: %v", err)
	}
}
func printTournament(t models.Tournament) {
	fmt.Printf("--- Tournament ---\n")
	fmt.Printf("  ID:               %s\n", t.TournamentID)
	fmt.Printf("  Name:             %s\n", t.Name)
	fmt.Printf("  Status:           %s\n", t.Status)
	fmt.Printf("  Visibility:       %s\n", t.Visibility)
	fmt.Printf("  Available Places: %d\n", t.AvailablePlaces)
	fmt.Println()
}

func formatTournament(sb *strings.Builder, t models.Tournament) {
	fmt.Fprintf(sb, "🏆 %s\n", t.Name)
	fmt.Fprintf(sb, "  Status: %s\n", t.Status)
	fmt.Fprintf(sb, "  Places: %d\n", t.AvailablePlaces)
	sb.WriteString("\n")
}

func printClass(c models.Class) {
	fmt.Printf("--- Class ---\n")
	fmt.Printf("  ID:          %s\n", c.AcademyClassID)
	if c.CourseSummary != nil {
		fmt.Printf("  Course:      %s\n", c.CourseSummary.Name)
	}
	fmt.Printf("  Type:        %s\n", c.Type)
	fmt.Printf("  Status:      %s\n", c.Status)
	fmt.Printf("  Start:       %s\n", c.StartDate)
	fmt.Printf("  End:         %s\n", c.EndDate)
	if len(c.Coaches) > 0 {
		fmt.Printf("  Coaches:     ")
		for i, coach := range c.Coaches {
			if i > 0 {
				fmt.Printf(", ")
			}
			fmt.Printf("%s", coach.Name)
		}
		fmt.Println()
	}
	fmt.Printf("  Registrations: %d\n", len(c.RegistrationInfo.Registrations))
	fmt.Println()
}

// fetchCourtAvailability queries the availability endpoint for a single day.
// The API requires a window ≤25h, so we use prev-day 22:00 UTC to curr-day 21:59:59 UTC,
// which maps to exactly one midnight-to-midnight window in Europe/Berlin (CET/CEST).
func fetchCourtAvailability(ctx context.Context, c *client.Client, cf config.CourtFilter, day time.Time) ([]models.CourtAvailability, error) {
	startMin := time.Date(day.Year(), day.Month(), day.Day()-1, 22, 0, 0, 0, time.UTC)
	startMax := time.Date(day.Year(), day.Month(), day.Day(), 21, 59, 59, 0, time.UTC)

	params := &models.SearchAvailabilityParams{
		TenantID: cf.TenantID,
		SportID:  cf.SportID,
		StartMin: startMin.Format("2006-01-02T15:04:05"),
		StartMax: startMax.Format("2006-01-02T15:04:05"),
	}
	return c.GetAvailability(ctx, params)
}

func printCourtSlot(clubName string, court models.CourtAvailability, slot models.Slot, loc *time.Location) {
	t := parseSlotTime(court.StartDate, slot.StartTime, loc)
	fmt.Printf("--- Court Available ---\n")
	fmt.Printf("  Club:     %s\n", clubName)
	fmt.Printf("  Court:    %s\n", court.ResourceID)
	fmt.Printf("  Time:     %s\n", t.Format("Mon 02 Jan 2006 15:04 MST"))
	fmt.Printf("  Duration: %d min\n", slot.Duration)
	fmt.Printf("  Price:    %s\n", slot.Price)
	fmt.Println()
}

func formatCourtSlot(sb *strings.Builder, clubName string, court models.CourtAvailability, slot models.Slot, loc *time.Location) {
	t := parseSlotTime(court.StartDate, slot.StartTime, loc)
	fmt.Fprintf(sb, "🎾 %s\n", clubName)
	fmt.Fprintf(sb, "  Court: %s\n", court.ResourceID)
	fmt.Fprintf(sb, "  Time: %s\n", t.Format("Mon 02 Jan 2006 15:04 MST"))
	fmt.Fprintf(sb, "  Duration: %d min | Price: %s\n", slot.Duration, slot.Price)
	sb.WriteString("\n")
}

// parseSlotTime combines the API date ("2006-01-02") and time ("15:04:05") strings
// (both UTC) into a time.Time converted to the given location.
func parseSlotTime(date, slotTime string, loc *time.Location) time.Time {
	raw := date + "T" + slotTime + "Z"
	t, err := time.Parse("2006-01-02T15:04:05Z", raw)
	if err != nil {
		return time.Time{}
	}
	return t.In(loc)
}

func formatClass(sb *strings.Builder, c models.Class) {
	if c.CourseSummary != nil {
		fmt.Fprintf(sb, "🎓 %s\n", c.CourseSummary.Name)
	} else {
		fmt.Fprintf(sb, "🎓 Class\n")
	}
	fmt.Fprintf(sb, "  Type: %s\n", c.Type)
	fmt.Fprintf(sb, "  Status: %s\n", c.Status)
	fmt.Fprintf(sb, "  Start: %s\n", formatBerlinTime(c.StartDate))
	if len(c.Coaches) > 0 {
		fmt.Fprintf(sb, "  Coach: %s\n", c.Coaches[0].Name)
	}
	fmt.Fprintf(sb, "  Registrations: %d\n", len(c.RegistrationInfo.Registrations))
	sb.WriteString("\n")
}

// formatBerlinTime converts an ISO date string to Berlin timezone format
func formatBerlinTime(dateStr string) string {
	// Parse the ISO 8601 date string
	t, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		// If parsing fails, try without timezone info
		t, err = time.Parse("2006-01-02T15:04:05", dateStr)
		if err != nil {
			// If still fails, return original string
			return dateStr
		}
	}

	// Load Berlin timezone
	berlin, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		// Fallback to original if timezone loading fails
		return dateStr
	}

	// Convert to Berlin time
	berlinTime := t.In(berlin)

	// Format as: Mon 16 Feb, 09:00 CET
	// The timezone name will automatically be CET or CEST depending on DST
	return berlinTime.Format("Mon 02 Jan, 15:04 MST")
}
