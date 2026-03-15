package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// SlackProvider sends notifications to a Slack channel via Incoming Webhooks.
type SlackProvider struct{}

// Send posts a formatted message to the configured Slack Incoming Webhook URL.
// Required config fields: url
func (p *SlackProvider) Send(ctx context.Context, cfg map[string]string, e Event) error {
	webhookURL, err := RequiredField(cfg, "url")
	if err != nil {
		return err
	}

	color := "#2ecc71" // green = UP
	if e.Status == 0 {
		color = "#e74c3c" // red = DOWN
	}

	payload := map[string]interface{}{
		"attachments": []map[string]interface{}{
			{
				"color":    color,
				"fallback": fmt.Sprintf("%s is %s", e.MonitorName, e.StatusText()),
				"title":    fmt.Sprintf("%s is %s", e.MonitorName, e.StatusText()),
				"fields": []map[string]interface{}{
					{"title": "URL", "value": e.MonitorURL, "short": false},
					{"title": "Latency", "value": fmt.Sprintf("%d ms", e.LatencyMs), "short": true},
					{"title": "Message", "value": e.Message, "short": true},
				},
				"footer": "Service Monitor",
				"ts":     time.Now().Unix(),
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("slack: marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("slack: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "service-monitor/1.0")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("slack: send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("slack: server returned %d", resp.StatusCode)
	}
	return nil
}
