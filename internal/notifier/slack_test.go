package notifier

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSlackProvider_SendUP(t *testing.T) {
	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	p := &SlackProvider{}
	err := p.Send(context.Background(), map[string]string{"url": srv.URL}, Event{
		MonitorName: "My Service",
		MonitorURL:  "https://example.com",
		Status:      1,
		LatencyMs:   42,
		Message:     "200 OK",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	attachments, ok := gotBody["attachments"].([]interface{})
	if !ok || len(attachments) == 0 {
		t.Fatal("expected attachments in payload")
	}
	att := attachments[0].(map[string]interface{})
	if att["color"] != "#2ecc71" {
		t.Errorf("UP color: want #2ecc71, got %v", att["color"])
	}
}

func TestSlackProvider_SendDOWN(t *testing.T) {
	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	p := &SlackProvider{}
	_ = p.Send(context.Background(), map[string]string{"url": srv.URL}, Event{
		MonitorName: "My Service",
		Status:      0,
	})

	attachments := gotBody["attachments"].([]interface{})
	att := attachments[0].(map[string]interface{})
	if att["color"] != "#e74c3c" {
		t.Errorf("DOWN color: want #e74c3c, got %v", att["color"])
	}
}

func TestSlackProvider_MissingURL(t *testing.T) {
	p := &SlackProvider{}
	err := p.Send(context.Background(), map[string]string{}, Event{})
	if err == nil {
		t.Fatal("expected error for missing url")
	}
}

func TestSlackProvider_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	p := &SlackProvider{}
	err := p.Send(context.Background(), map[string]string{"url": srv.URL}, Event{Status: 1})
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}
