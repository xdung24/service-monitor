package models

import "testing"

func TestNotificationLogStatusText(t *testing.T) {
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
		l := &NotificationLog{EventStatus: tt.status}
		got := l.StatusText()
		if got != tt.want {
			t.Errorf("NotificationLog{EventStatus:%d}.StatusText() = %q, want %q", tt.status, got, tt.want)
		}
	}
}
