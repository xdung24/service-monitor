package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// DiscordProvider sends notifications to a Discord channel via Webhooks.
type DiscordProvider struct{}

// Send posts an embed message to the configured Discord Webhook URL.
// Required config fields: url
func (p *DiscordProvider) Send(ctx context.Context, cfg map[string]string, e Event) error {
	webhookURL, err := RequiredField(cfg, "url")
	if err != nil {
		return err
	}

	color := 0x2ecc71 // green = UP
	if e.Status == 0 {
		color = 0xe74c3c // red = DOWN
	}

	payload := map[string]interface{}{
		"username": "Service Monitor",
		"embeds": []map[string]interface{}{
			{
				"title":       fmt.Sprintf("%s is %s", e.MonitorName, e.StatusText()),
				"description": e.Message,
				"color":       color,
				"fields": []map[string]interface{}{
					{"name": "URL", "value": e.MonitorURL, "inline": false},
					{"name": "Latency", "value": fmt.Sprintf("%d ms", e.LatencyMs), "inline": true},
				},
				"footer":    map[string]string{"text": "Service Monitor"},
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("discord: marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("discord: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "conductor/1.0")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("discord: send: %w", err)
	}
	defer resp.Body.Close()

	// Discord returns 204 No Content on success
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("discord: server returned %d", resp.StatusCode)
	}
	return nil
}
