package notifier

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// NtfyProvider sends push notifications via ntfy.sh or a self-hosted ntfy server.
type NtfyProvider struct{}

// Send publishes a notification to the configured ntfy topic.
// Required config fields: topic
// Optional config fields: server (default https://ntfy.sh), token (Bearer access token)
func (p *NtfyProvider) Send(ctx context.Context, cfg map[string]string, e Event) error {
	topic, err := RequiredField(cfg, "topic")
	if err != nil {
		return err
	}

	server := cfg["server"]
	if server == "" {
		server = "https://ntfy.sh"
	}
	server = strings.TrimRight(server, "/")

	url := server + "/" + topic

	title := fmt.Sprintf("%s is %s", e.MonitorName, e.StatusText())
	body := fmt.Sprintf("URL: %s\nLatency: %d ms\n%s", e.MonitorURL, e.LatencyMs, e.Message)

	priority := "default"
	tags := "white_check_mark"
	if e.Status == 0 {
		priority = "urgent"
		tags = "rotating_light"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("ntfy: create request: %w", err)
	}
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Title", title)
	req.Header.Set("Priority", priority)
	req.Header.Set("Tags", tags)
	req.Header.Set("User-Agent", "service-monitor/1.0")

	if token := cfg["token"]; token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("ntfy: send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("ntfy: server returned %d", resp.StatusCode)
	}
	return nil
}
