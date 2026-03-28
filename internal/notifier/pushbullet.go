package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

var pushbulletAPIURL = "https://api.pushbullet.com/v2/pushes"

// PushbulletProvider sends alerts to Pushbullet.
type PushbulletProvider struct{}

// Send delivers a note push via Pushbullet API.
// Required config fields: token
// Optional config fields: device
func (p *PushbulletProvider) Send(ctx context.Context, cfg map[string]string, e Event) error {
	token, err := RequiredField(cfg, "token")
	if err != nil {
		return err
	}

	title := fmt.Sprintf("%s is %s", e.MonitorName, e.StatusText())
	bodyText := fmt.Sprintf("URL: %s\nLatency: %d ms\n%s", e.MonitorURL, e.LatencyMs, e.Message)

	payload := map[string]interface{}{
		"type":  "note",
		"title": title,
		"body":  bodyText,
	}
	if device := cfg["device"]; device != "" {
		payload["device_iden"] = device
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("pushbullet: marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, pushbulletAPIURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("pushbullet: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Access-Token", token)
	req.Header.Set("User-Agent", "conductor/1.0")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("pushbullet: send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("pushbullet: server returned %d", resp.StatusCode)
	}
	return nil
}
