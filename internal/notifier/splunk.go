package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// SplunkProvider sends events to Splunk HTTP Event Collector.
type SplunkProvider struct{}

// Send posts an event to Splunk HEC.
// Required config fields: url, token
func (p *SplunkProvider) Send(ctx context.Context, cfg map[string]string, e Event) error {
	endpoint, err := RequiredField(cfg, "url")
	if err != nil {
		return err
	}
	token, err := RequiredField(cfg, "token")
	if err != nil {
		return err
	}

	payload := map[string]interface{}{
		"time": float64(time.Now().UTC().Unix()),
		"event": map[string]interface{}{
			"monitor_id":   e.MonitorID,
			"monitor_name": e.MonitorName,
			"monitor_url":  e.MonitorURL,
			"status":       e.StatusText(),
			"latency_ms":   e.LatencyMs,
			"message":      e.Message,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("splunk: marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("splunk: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Splunk "+token)
	req.Header.Set("User-Agent", "conductor/1.0")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("splunk: send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("splunk: server returned %d", resp.StatusCode)
	}
	return nil
}
