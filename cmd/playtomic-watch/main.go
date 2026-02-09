package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/rafa-garcia/go-playtomic-api/client"
	"github.com/rafa-garcia/go-playtomic-api/internal/config"
	"github.com/rafa-garcia/go-playtomic-api/internal/filter"
	"github.com/rafa-garcia/go-playtomic-api/models"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to configuration file")
	timeout := flag.Duration("timeout", 30*time.Second, "HTTP request timeout")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	c := client.NewClient(
		client.WithTimeout(*timeout),
		client.WithBaseURL(client.DefaultBaseUrlV2),
	)

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	found := 0
	for _, tf := range cfg.Tournaments {
		tournaments, err := fetchTournaments(ctx, c, tf)
		if err != nil {
			log.Printf("Error fetching tournaments for tenant %s: %v", tf.TenantID, err)
			continue
		}

		matched := filter.Apply(tournaments, tf)
		for _, t := range matched {
			printTournament(t, tf.TenantID)
			found++
		}
	}

	if found == 0 {
		fmt.Println("No matching tournaments found.")
	}
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

func printTournament(t models.Tournament, tenantID string) {
	fmt.Printf("--- Tournament ---\n")
	fmt.Printf("  ID:               %s\n", t.TournamentID)
	fmt.Printf("  Name:             %s\n", t.Name)
	fmt.Printf("  Status:           %s\n", t.Status)
	fmt.Printf("  Visibility:       %s\n", t.Visibility)
	fmt.Printf("  Available Places: %d\n", t.AvailablePlaces)
	fmt.Printf("  Tenant:           %s\n", tenantID)
	fmt.Println()
}
