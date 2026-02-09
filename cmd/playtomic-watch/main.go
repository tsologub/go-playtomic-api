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

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	var bot *telegram.Bot
	if *telegramToken != "" && *telegramChatID != "" {
		bot = telegram.NewBot(*telegramToken, *telegramChatID)
	}

	c := client.NewClient(
		client.WithTimeout(*timeout),
		client.WithBaseURL(client.DefaultBaseUrlV2),
	)

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	var matched []models.Tournament
	for _, tf := range cfg.Tournaments {
		tournaments, err := fetchTournaments(ctx, c, tf)
		if err != nil {
			log.Printf("Error fetching tournaments for tenant %s: %v", tf.TenantID, err)
			continue
		}

		matched = append(matched, filter.Apply(tournaments, tf)...)
	}

	if len(matched) == 0 {
		msg := "No matching tournaments found."
		fmt.Println(msg)
		notify(bot, msg)
		return
	}

	var sb strings.Builder
	for _, t := range matched {
		printTournament(t)
		formatTournament(&sb, t)
	}
	notify(bot, sb.String())
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
