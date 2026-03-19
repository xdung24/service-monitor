package notifier

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNtfyProvider_Send(t *testing.T) {
	var (
		gotTitle    string
		gotPriority string
		gotBody     string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTitle = r.Header.Get("Title")
		gotPriority = r.Header.Get("Priority")
		raw, _ := io.ReadAll(r.Body)
		gotBody = string(raw)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	p := &NtfyProvider{}
	err := p.Send(context.Background(), map[string]string{
		"topic":  "alerts",
		"server": srv.URL,
	}, Event{
		MonitorName: "My DB",
		MonitorURL:  "postgres://db:5432",
		Status:      0,
		LatencyMs:   0,
		Message:     "connection refused",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(gotTitle, "My DB") {
		t.Errorf("title should contain monitor name, got %q", gotTitle)
	}
	if !strings.Contains(gotTitle, "DOWN") {
		t.Errorf("title should mention DOWN, got %q", gotTitle)
	}
	if gotPriority != "urgent" {
		t.Errorf("DOWN priority: want urgent, got %q", gotPriority)
	}
	if !strings.Contains(gotBody, "connection refused") {
		t.Errorf("body should contain message, got %q", gotBody)
	}
}

func TestNtfyProvider_UPPriority(t *testing.T) {
	var gotPriority string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPriority = r.Header.Get("Priority")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	p := &NtfyProvider{}
	_ = p.Send(context.Background(), map[string]string{"topic": "alerts", "server": srv.URL},
		Event{Status: 1, MonitorName: "API"})
	if gotPriority != "default" {
		t.Errorf("UP priority: want default, got %q", gotPriority)
	}
}

func TestNtfyProvider_DefaultServer(t *testing.T) {
	// Just verify that missing server doesn't panic / error on missing topic only
	p := &NtfyProvider{}
	err := p.Send(context.Background(), map[string]string{}, Event{})
	if err == nil {
		t.Fatal("expected error for missing topic")
	}
}

func TestNtfyProvider_BearerToken(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	p := &NtfyProvider{}
	_ = p.Send(context.Background(), map[string]string{
		"topic":  "secure",
		"server": srv.URL,
		"token":  "mytoken123",
	}, Event{Status: 1, MonitorName: "X"})

	if gotAuth != "Bearer mytoken123" {
		t.Errorf("expected Bearer token, got %q", gotAuth)
	}
}

func TestNtfyProvider_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	p := &NtfyProvider{}
	err := p.Send(context.Background(), map[string]string{"topic": "t", "server": srv.URL}, Event{})
	if err == nil {
		t.Fatal("expected error for 403")
	}
}
