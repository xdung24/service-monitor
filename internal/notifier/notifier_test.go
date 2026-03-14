package notifier

import "testing"

func TestRequiredField_Present(t *testing.T) {
	cfg := map[string]string{"url": "http://example.com"}
	v, err := RequiredField(cfg, "url")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != "http://example.com" {
		t.Fatalf("expected %q, got %q", "http://example.com", v)
	}
}

func TestRequiredField_Missing(t *testing.T) {
	cfg := map[string]string{}
	_, err := RequiredField(cfg, "token")
	if err == nil {
		t.Fatal("expected error for missing key")
	}
}

func TestRequiredField_EmptyValue(t *testing.T) {
	cfg := map[string]string{"token": ""}
	_, err := RequiredField(cfg, "token")
	if err == nil {
		t.Fatal("expected error for empty value")
	}
}

func TestEventStatusText(t *testing.T) {
	tests := []struct {
		status int
		want   string
	}{
		{1, "UP"},
		{0, "DOWN"},
		{-1, "DOWN"},
		{2, "DOWN"},
	}
	for _, tt := range tests {
		e := Event{Status: tt.status}
		got := e.StatusText()
		if got != tt.want {
			t.Errorf("Event{Status:%d}.StatusText() = %q, want %q", tt.status, got, tt.want)
		}
	}
}
