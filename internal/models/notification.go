package models

import "time"

// Notification holds notification provider configuration.
type Notification struct {
	ID        int64     `db:"id"`
	Name      string    `db:"name"`
	Type      string    `db:"type"`
	Config    string    `db:"config"` // JSON
	Active    bool      `db:"active"`
	CreatedAt time.Time `db:"created_at"`
}

// NotificationLog records a single notification delivery attempt.
type NotificationLog struct {
	ID               int64     `db:"id"`
	MonitorID        *int64    `db:"monitor_id"`      // nullable (monitor may be deleted)
	NotificationID   *int64    `db:"notification_id"` // nullable
	MonitorName      string    `db:"monitor_name"`
	NotificationName string    `db:"notification_name"`
	EventStatus      int       `db:"event_status"` // 0=down, 1=up
	Success          bool      `db:"success"`
	Error            string    `db:"error"`
	CreatedAt        time.Time `db:"created_at"`
}

// StatusText returns "UP" or "DOWN" for the logged event.
func (l *NotificationLog) StatusText() string {
	if l.EventStatus == 1 {
		return "UP"
	}
	return "DOWN"
}
