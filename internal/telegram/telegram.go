package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const defaultAPIURL = "https://api.telegram.org"

type Bot struct {
	token  string
	chatID string
	client *http.Client
	apiURL string
}

func NewBot(token, chatID string) *Bot {
	return &Bot{
		token:  token,
		chatID: chatID,
		client: &http.Client{Timeout: 10 * time.Second},
		apiURL: defaultAPIURL,
	}
}

func (b *Bot) Send(text string) error {
	url := fmt.Sprintf("%s/bot%s/sendMessage", b.apiURL, b.token)

	payload := map[string]string{
		"chat_id": b.chatID,
		"text":    text,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling payload: %w", err)
	}

	resp, err := b.client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("sending telegram message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API returned status %d", resp.StatusCode)
	}

	return nil
}
