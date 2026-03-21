package models

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// openTestDB opens an in-memory SQLite database for testing.
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:?_foreign_keys=ON")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	db.SetMaxOpenConns(1)
	return db
}

func createHeartbeatTable(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.ExecContext(context.Background(), `
		CREATE TABLE heartbeats (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			monitor_id INTEGER  NOT NULL,
			status     INTEGER  NOT NULL,
			latency_ms INTEGER  NOT NULL DEFAULT 0,
			message    TEXT     NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("create heartbeats table: %v", err)
	}
}

func createMaintenanceTables(t *testing.T, db *sql.DB) {
	t.Helper()
	stmts := []string{
		`CREATE TABLE maintenance_windows (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			name       TEXT     NOT NULL DEFAULT '',
			start_time DATETIME NOT NULL,
			end_time   DATETIME NOT NULL,
			active     INTEGER  NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		)`,
		`CREATE TABLE monitor_maintenance (
			window_id  INTEGER NOT NULL,
			monitor_id INTEGER NOT NULL
		)`,
	}
	for _, s := range stmts {
		if _, err := db.ExecContext(context.Background(), s); err != nil {
			t.Fatalf("create maintenance tables: %v", err)
		}
	}
}

// TestHeartbeatLatestSinceUTC verifies that LatestSince correctly filters
// heartbeats using UTC timestamp comparison.  This guards against the bug
// where passing a non-UTC time.Time to SQLite breaks the lexicographic text
// comparison against UTC-stored values (e.g. "10:00Z" vs "17:00+07:00").
func TestHeartbeatLatestSinceUTC(t *testing.T) {
	db := openTestDB(t)
	createHeartbeatTable(t, db)
	store := NewHeartbeatStore(db)

	// Base reference point: 10:00 UTC
	base := time.Date(2026, 3, 21, 10, 0, 0, 0, time.UTC)

	// Insert three heartbeats at base, base+1h, base+2h (all UTC, as the
	// scheduler always creates heartbeats with time.Now().UTC()).
	for i := 0; i < 3; i++ {
		if err := store.Insert(&Heartbeat{
			MonitorID: 1,
			Status:    1,
			LatencyMs: 10,
			CreatedAt: base.Add(time.Duration(i) * time.Hour),
		}); err != nil {
			t.Fatalf("insert heartbeat %d: %v", i, err)
		}
	}

	tests := []struct {
		name  string
		since time.Time
		want  int // expected number of results
	}{
		{"since 1h before base", base.Add(-1 * time.Hour), 3},
		{"since exactly at base", base, 3},
		{"since 30m after base", base.Add(30 * time.Minute), 2}, // base+1h and base+2h
		{"since 90m after base", base.Add(90 * time.Minute), 1}, // only base+2h
		{"since after all records", base.Add(3 * time.Hour), 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			beats, err := store.LatestSince(1, tc.since, 100)
			if err != nil {
				t.Fatalf("LatestSince: %v", err)
			}
			if len(beats) != tc.want {
				t.Errorf("LatestSince(since=%v) = %d beats, want %d", tc.since, len(beats), tc.want)
			}
		})
	}
}

// TestHeartbeatUptimePercentUTC verifies that UptimePercent correctly counts
// UP heartbeats within a UTC time window and excludes records outside it.
func TestHeartbeatUptimePercentUTC(t *testing.T) {
	db := openTestDB(t)
	createHeartbeatTable(t, db)
	store := NewHeartbeatStore(db)

	base := time.Date(2026, 3, 21, 10, 0, 0, 0, time.UTC)

	// Within range: 2 UP (base, base+30m) + 1 DOWN (base+60m)
	// Outside range: 1 UP (base-2h) — must be excluded from the calculation.
	for _, hb := range []struct {
		status int
		offset time.Duration
	}{
		{1, 0},
		{1, 30 * time.Minute},
		{0, 60 * time.Minute},
		{1, -2 * time.Hour}, // before the since boundary
	} {
		if err := store.Insert(&Heartbeat{
			MonitorID: 1,
			Status:    hb.status,
			LatencyMs: 10,
			CreatedAt: base.Add(hb.offset),
		}); err != nil {
			t.Fatalf("insert heartbeat: %v", err)
		}
	}

	// since = base-1h captures the 3 in-range records, not the one at base-2h.
	pct, err := store.UptimePercent(1, base.Add(-1*time.Hour))
	if err != nil {
		t.Fatalf("UptimePercent: %v", err)
	}

	const want = 2.0 / 3.0 * 100 // 2 UP out of 3 in-range records
	if pct < want-0.01 || pct > want+0.01 {
		t.Errorf("UptimePercent = %.4f, want %.4f (2/3 × 100)", pct, want)
	}
}

// TestIsInMaintenanceUTC verifies that IsInMaintenance correctly evaluates
// UTC-stored window boundaries when queried with UTC times.  Maintenance
// windows are stored with UTC start_time/end_time (the handler applies
// .UTC() before calling Create) so the SQL text comparison must also use UTC.
func TestIsInMaintenanceUTC(t *testing.T) {
	db := openTestDB(t)
	createMaintenanceTables(t, db)
	store := NewMaintenanceStore(db)

	// Window: 10:00–12:00 UTC.  StartTime/EndTime are set to UTC to match
	// production behaviour (maintenance.go applies .UTC() before Create).
	windowStart := time.Date(2026, 3, 21, 10, 0, 0, 0, time.UTC)
	windowEnd := windowStart.Add(2 * time.Hour)

	wID, err := store.Create(&MaintenanceWindow{
		Name:      "test window",
		StartTime: windowStart,
		EndTime:   windowEnd,
		Active:    true,
	})
	if err != nil {
		t.Fatalf("create window: %v", err)
	}
	if err := store.SetMonitors(wID, []int64{1}); err != nil {
		t.Fatalf("set monitors: %v", err)
	}

	tests := []struct {
		name    string
		t       time.Time
		inMaint bool
	}{
		{"1h before start", windowStart.Add(-1 * time.Hour), false},
		{"at start boundary", windowStart, true},
		{"midpoint of window", windowStart.Add(1 * time.Hour), true},
		{"at end boundary", windowEnd, true},
		{"1h after end", windowEnd.Add(1 * time.Hour), false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Pass tc.t.UTC() as production code does (scheduler.go).
			got, err := store.IsInMaintenance(1, tc.t.UTC())
			if err != nil {
				t.Fatalf("IsInMaintenance: %v", err)
			}
			if got != tc.inMaint {
				t.Errorf("IsInMaintenance(%v) = %v, want %v", tc.t, got, tc.inMaint)
			}
		})
	}
}
