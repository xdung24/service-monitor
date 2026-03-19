package notifier

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDiscordProvider_SendUP(t *testing.T) {
	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	p := &DiscordProvider{}
	err := p.Send(context.Background(), map[string]string{"url": srv.URL}, Event{
		MonitorName: "API",
		MonitorURL:  "https://api.example.com",
		Status:      1,
		LatencyMs:   55,
		Message:     "200 OK",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	embeds, ok := gotBody["embeds"].([]interface{})
	if !ok || len(embeds) == 0 {
		t.Fatal("expected embeds in payload")
	}
	embed := embeds[0].(map[string]interface{})
	// 0x2ecc71 = 3066993 in float64 (JSON number)
	if embed["color"] != float64(0x2ecc71) {
		t.Errorf("UP color: want %d, got %v", 0x2ecc71, embed["color"])
	}
}

func TestDiscordProvider_SendDOWN(t *testing.T) {
	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	p := &DiscordProvider{}
	_ = p.Send(context.Background(), map[string]string{"url": srv.URL}, Event{Status: 0, MonitorName: "API"})

	embeds := gotBody["embeds"].([]interface{})
	embed := embeds[0].(map[string]interface{})
	if embed["color"] != float64(0xe74c3c) {
		t.Errorf("DOWN color: want %d, got %v", 0xe74c3c, embed["color"])
	}
}

func TestDiscordProvider_MissingURL(t *testing.T) {
	p := &DiscordProvider{}
	err := p.Send(context.Background(), map[string]string{}, Event{})
	if err == nil {
		t.Fatal("expected error for missing url")
	}
}
