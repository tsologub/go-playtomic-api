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
	"github.com/rafa-garcia/go-playtomic-api/internal/telegram"
	"github.com/rafa-garcia/go-playtomic-api/models"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to configuration file")
	timeout := flag.Duration("timeout", 30*time.Second, "HTTP request timeout")
	telegramToken := flag.String("telegram-token", "", "Telegram bot token")
	telegramChatID := flag.String("telegram-chat-id", "", "Telegram chat ID")
	flag.Parse()

	// Check for subcommand
	args := flag.Args()
	if len(args) == 0 {
		printUsage()
		log.Fatalf("Error: subcommand required")
	}

	subcommand := args[0]
	if subcommand != "tournaments" && subcommand != "classes" {
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

	switch subcommand {
	case "tournaments":
		if len(cfg.Tournaments) == 0 {
			log.Fatalf("No tournament filters configured in %s", *configPath)
		}

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
			formatTournament(&sb, t)
		}

	case "classes":
		if len(cfg.Classes) == 0 {
			log.Fatalf("No class filters configured in %s", *configPath)
		}

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
			formatClass(&sb, c)
		}
	}

	if sb.Len() > 0 {
		notify(bot, sb.String())
	}
}

func printUsage() {
	fmt.Println("Usage: playtomic-watch [OPTIONS] <tournaments|classes>")
	fmt.Println("\nSubcommands:")
	fmt.Println("  tournaments    Search for tournaments")
	fmt.Println("  classes        Search for classes")
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

func formatClass(sb *strings.Builder, c models.Class) {
	if c.CourseSummary != nil {
		fmt.Fprintf(sb, "🎓 %s\n", c.CourseSummary.Name)
	} else {
		fmt.Fprintf(sb, "🎓 Class\n")
	}
	fmt.Fprintf(sb, "  Type: %s\n", c.Type)
	fmt.Fprintf(sb, "  Status: %s\n", c.Status)
	fmt.Fprintf(sb, "  Start: %s\n", c.StartDate)
	if len(c.Coaches) > 0 {
		fmt.Fprintf(sb, "  Coach: %s\n", c.Coaches[0].Name)
	}
	fmt.Fprintf(sb, "  Registrations: %d\n", len(c.RegistrationInfo.Registrations))
	sb.WriteString("\n")
}
