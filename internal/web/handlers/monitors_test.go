package handlers

import (
	"encoding/json"
	"testing"

	"github.com/xdung24/service-monitor/internal/models"
)

func TestNotifSummaryMap_Webhook(t *testing.T) {
	cfg, _ := json.Marshal(map[string]string{"url": "https://hooks.example.com/abc"})
	notifs := []*models.Notification{
		{ID: 1, Type: "webhook", Config: string(cfg)},
	}
	m := notifSummaryMap(notifs)
	if m[1] != "hooks.example.com" {
		t.Errorf("webhook summary = %q, want %q", m[1], "hooks.example.com")
	}
}

func TestNotifSummaryMap_WebhookInvalidURL(t *testing.T) {
	cfg, _ := json.Marshal(map[string]string{"url": "not-a-url"})
	notifs := []*models.Notification{
		{ID: 2, Type: "webhook", Config: string(cfg)},
	}
	m := notifSummaryMap(notifs)
	if m[2] != "not-a-url" {
		t.Errorf("webhook invalid URL fallback = %q, want %q", m[2], "not-a-url")
	}
}

func TestNotifSummaryMap_Telegram(t *testing.T) {
	cfg, _ := json.Marshal(map[string]string{"chat_id": "123456789"})
	notifs := []*models.Notification{
		{ID: 3, Type: "telegram", Config: string(cfg)},
	}
	m := notifSummaryMap(notifs)
	if m[3] != "Chat: 123456789" {
		t.Errorf("telegram summary = %q, want %q", m[3], "Chat: 123456789")
	}
}

func TestNotifSummaryMap_Email(t *testing.T) {
	cfg, _ := json.Marshal(map[string]string{"to": "user@example.com"})
	notifs := []*models.Notification{
		{ID: 4, Type: "email", Config: string(cfg)},
	}
	m := notifSummaryMap(notifs)
	if m[4] != "→ user@example.com" {
		t.Errorf("email summary = %q, want %q", m[4], "→ user@example.com")
	}
}

func TestNotifSummaryMap_InvalidJSON(t *testing.T) {
	notifs := []*models.Notification{
		{ID: 5, Type: "webhook", Config: "invalid-json"},
	}
	// should not panic; entry simply won't be populated
	m := notifSummaryMap(notifs)
	if _, exists := m[5]; exists {
		t.Error("expected no entry for notification with invalid JSON config")
	}
}

func TestNotifSummaryMap_Empty(t *testing.T) {
	m := notifSummaryMap(nil)
	if len(m) != 0 {
		t.Errorf("expected empty map for nil input, got %v", m)
	}
}
