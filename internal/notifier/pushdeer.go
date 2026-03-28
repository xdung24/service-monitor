package notifier

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// PushDeerProvider sends alerts via PushDeer.
type PushDeerProvider struct{}

// Send calls PushDeer push API.
// Required config fields: push_key
// Optional config fields: server_url (default https://api2.pushdeer.com)
func (p *PushDeerProvider) Send(ctx context.Context, cfg map[string]string, e Event) error {
	pushKey, err := RequiredField(cfg, "push_key")
	if err != nil {
		return err
	}

	serverURL := cfg["server_url"]
	if serverURL == "" {
		serverURL = "https://api2.pushdeer.com"
	}
	serverURL = strings.TrimRight(serverURL, "/")

	title := fmt.Sprintf("%s is %s", e.MonitorName, e.StatusText())
	bodyText := fmt.Sprintf("URL: %s\nLatency: %d ms\n%s", e.MonitorURL, e.LatencyMs, e.Message)

	v := url.Values{}
	v.Set("pushkey", pushKey)
	v.Set("text", title)
	v.Set("desp", bodyText)
	v.Set("type", "text")

	endpoint := serverURL + "/message/push"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+v.Encode(), nil)
	if err != nil {
		return fmt.Errorf("pushdeer: create request: %w", err)
	}
	req.Header.Set("User-Agent", "conductor/1.0")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("pushdeer: send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("pushdeer: server returned %d", resp.StatusCode)
	}
	return nil
}
