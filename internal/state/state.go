package state

import (
	"encoding/json"
	"os"
	"time"
)

// Entry represents a tracked tournament or class
type Entry struct {
	LastSeen        time.Time `json:"last_seen"`
	AvailablePlaces int       `json:"available_places"`
}

// State manages notification state for tournaments or classes
type State struct {
	entries  map[string]Entry
	filePath string
}

// New creates a new State instance
func New(filePath string) *State {
	return &State{
		entries:  make(map[string]Entry),
		filePath: filePath,
	}
}

// Load reads the state from the JSON file
func (s *State) Load() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist yet, start with empty state
			return nil
		}
		return err
	}

	if err := json.Unmarshal(data, &s.entries); err != nil {
		return err
	}

	return nil
}

// Save writes the state to the JSON file
func (s *State) Save() error {
	// Clean up old entries (older than 3 months)
	s.cleanup()

	data, err := json.MarshalIndent(s.entries, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.filePath, data, 0644)
}

// cleanup removes entries older than 3 months
func (s *State) cleanup() {
	threeMonthsAgo := time.Now().AddDate(0, -3, 0)
	for id, entry := range s.entries {
		if entry.LastSeen.Before(threeMonthsAgo) {
			delete(s.entries, id)
		}
	}
}

// ShouldNotify determines if a notification should be sent
// Returns true if:
// - ID is new (not in state)
// - Available places increased (someone dropped out)
func (s *State) ShouldNotify(id string, availablePlaces int) bool {
	entry, exists := s.entries[id]
	if !exists {
		// New tournament/class, should notify
		return true
	}

	// Notify if places increased (someone dropped out)
	return availablePlaces > entry.AvailablePlaces
}

// Update records the current state for an ID
func (s *State) Update(id string, availablePlaces int) {
	s.entries[id] = Entry{
		LastSeen:        time.Now(),
		AvailablePlaces: availablePlaces,
	}
}
