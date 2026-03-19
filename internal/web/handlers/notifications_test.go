package handlers

import "testing"

func TestNotificationConfigMap_Valid(t *testing.T) {
	m := notificationConfigMap(`{"token":"abc","chat_id":"123"}`)
	if m["token"] != "abc" {
		t.Errorf("token = %q, want %q", m["token"], "abc")
	}
	if m["chat_id"] != "123" {
		t.Errorf("chat_id = %q, want %q", m["chat_id"], "123")
	}
}

func TestNotificationConfigMap_InvalidJSON(t *testing.T) {
	// must not panic; returns empty map
	m := notificationConfigMap("not-json")
	if len(m) != 0 {
		t.Errorf("expected empty map for invalid JSON, got %v", m)
	}
}

func TestNotificationConfigMap_Empty(t *testing.T) {
	m := notificationConfigMap("")
	if len(m) != 0 {
		t.Errorf("expected empty map for empty string, got %v", m)
	}
}

func TestNotificationConfigMap_EmptyObject(t *testing.T) {
	m := notificationConfigMap("{}")
	if len(m) != 0 {
		t.Errorf("expected empty map for '{}', got %v", m)
	}
}
