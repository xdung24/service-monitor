package models

import (
	"context"
	"database/sql"
	"time"
)

// DowntimeEvent records a contiguous down period for a monitor.
type DowntimeEvent struct {
	ID        int64
	MonitorID int64
	StartedAt time.Time
	EndedAt   *time.Time // nil = incident still open
	DurationS *int       // seconds; nil until closed
}

// DowntimeEventStore provides access to the downtime_events table.
type DowntimeEventStore struct{ db *sql.DB }

// NewDowntimeEventStore creates a new DowntimeEventStore backed by db.
func NewDowntimeEventStore(db *sql.DB) *DowntimeEventStore {
	return &DowntimeEventStore{db: db}
}

// OpenIncident inserts a new open incident when a monitor transitions to DOWN.
// Idempotent: no-op if an open incident already exists for that monitor.
func (s *DowntimeEventStore) OpenIncident(monitorID int64, at time.Time) error {
	_, err := s.db.ExecContext(context.Background(), `
		INSERT INTO downtime_events (monitor_id, started_at)
		SELECT ?, ?
		WHERE NOT EXISTS (
			SELECT 1 FROM downtime_events
			WHERE monitor_id = ? AND ended_at IS NULL
		)
	`, monitorID, at.UTC(), monitorID)
	return err
}

// CloseIncident closes the open incident for a monitor when it recovers to UP.
// No-op if there is no open incident.
func (s *DowntimeEventStore) CloseIncident(monitorID int64, at time.Time) error {
	_, err := s.db.ExecContext(context.Background(), `
		UPDATE downtime_events
		SET ended_at   = ?,
		    duration_s = CAST((julianday(?) - julianday(started_at)) * 86400 AS INTEGER)
		WHERE monitor_id = ? AND ended_at IS NULL
	`, at.UTC(), at.UTC(), monitorID)
	return err
}

// ListSince returns all events that overlap the window [since, now], oldest-first.
// An open (still-ongoing) incident is always included.
func (s *DowntimeEventStore) ListSince(monitorID int64, since time.Time) ([]*DowntimeEvent, error) {
	rows, err := s.db.QueryContext(context.Background(), `
		SELECT id, monitor_id, started_at, ended_at, duration_s
		FROM downtime_events
		WHERE monitor_id = ?
		  AND (ended_at IS NULL OR ended_at >= ?)
		ORDER BY started_at ASC
	`, monitorID, since.UTC())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*DowntimeEvent
	for rows.Next() {
		e := &DowntimeEvent{}
		var endedAt sql.NullTime
		var durS sql.NullInt64
		if err := rows.Scan(&e.ID, &e.MonitorID, &e.StartedAt, &endedAt, &durS); err != nil {
			return nil, err
		}
		if endedAt.Valid {
			e.EndedAt = &endedAt.Time
		}
		if durS.Valid {
			v := int(durS.Int64)
			e.DurationS = &v
		}
		events = append(events, e)
	}
	return events, rows.Err()
}
