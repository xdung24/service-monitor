package config

import (
	"testing"
	"time"
)

func TestLoad_Defaults(t *testing.T) {
	// Ensure none of the env vars are set.
	for _, key := range []string{"LISTEN_ADDR", "DB_PATH", "DATA_DIR", "SECRET_KEY", "SECRET_KEY_SEED", "SECRET_KEY_SALT"} {
		t.Setenv(key, "")
	}
	dataDir := t.TempDir()
	t.Setenv("DATA_DIR", dataDir)

	cfg := Load()

	if cfg.ListenAddr != ":3001" {
		t.Errorf("ListenAddr: got %q, want %q", cfg.ListenAddr, ":3001")
	}
	if cfg.DBPath != "./data/conductor.db" {
		t.Errorf("DBPath: got %q, want %q", cfg.DBPath, "./data/conductor.db")
	}
	if cfg.DataDir != dataDir {
		t.Errorf("DataDir: got %q, want %q", cfg.DataDir, dataDir)
	}
	if len(cfg.SecretKey) != 32 {
		t.Errorf("SecretKey length: got %d, want 32", len(cfg.SecretKey))
	}
}

func TestLoad_EnvOverrides(t *testing.T) {
	t.Setenv("LISTEN_ADDR", ":8080")
	t.Setenv("DB_PATH", "/tmp/test.db")
	t.Setenv("DATA_DIR", "/tmp/data")
	t.Setenv("SECRET_KEY", "super-secret")

	cfg := Load()

	if cfg.ListenAddr != ":8080" {
		t.Errorf("ListenAddr: got %q, want %q", cfg.ListenAddr, ":8080")
	}
	if cfg.DBPath != "/tmp/test.db" {
		t.Errorf("DBPath: got %q, want %q", cfg.DBPath, "/tmp/test.db")
	}
	if cfg.DataDir != "/tmp/data" {
		t.Errorf("DataDir: got %q, want %q", cfg.DataDir, "/tmp/data")
	}
	if cfg.SecretKey != "super-secret" {
		t.Errorf("SecretKey: got %q, want %q", cfg.SecretKey, "super-secret")
	}
}

func TestLoad_AutoSecret_StableWithinDay(t *testing.T) {
	t.Setenv("SECRET_KEY", "")
	t.Setenv("SECRET_KEY_SEED", "my-very-strong-seed-value-1234567890")
	t.Setenv("SECRET_KEY_SALT", "test-salt")

	origNow := nowUTC
	nowUTC = func() time.Time { return time.Date(2026, 3, 27, 9, 0, 0, 0, time.UTC) }
	t.Cleanup(func() { nowUTC = origNow })

	cfg1 := Load()
	cfg2 := Load()

	if len(cfg1.SecretKey) != 32 {
		t.Fatalf("first SecretKey length: got %d, want 32", len(cfg1.SecretKey))
	}
	if cfg1.SecretKey != cfg2.SecretKey {
		t.Fatalf("expected same key for same day, got %q and %q", cfg1.SecretKey, cfg2.SecretKey)
	}
}

func TestLoad_AutoSecret_RotatesNextDay(t *testing.T) {
	t.Setenv("SECRET_KEY", "")
	t.Setenv("SECRET_KEY_SEED", "my-very-strong-seed-value-1234567890")
	t.Setenv("SECRET_KEY_SALT", "test-salt")

	origNow := nowUTC
	nowUTC = func() time.Time { return time.Date(2026, 3, 27, 9, 0, 0, 0, time.UTC) }
	first := Load().SecretKey
	nowUTC = func() time.Time { return time.Date(2026, 3, 28, 9, 0, 0, 0, time.UTC) }
	second := Load().SecretKey
	t.Cleanup(func() { nowUTC = origNow })

	if len(first) != 32 || len(second) != 32 {
		t.Fatalf("expected both keys to be 32 chars, got %d and %d", len(first), len(second))
	}
	if first == second {
		t.Fatalf("expected different key on next day, both were %q", first)
	}
}

func TestLoad_AutoSecret_SeedRequiredForDeterministicAcrossRestarts(t *testing.T) {
	t.Setenv("SECRET_KEY", "")
	t.Setenv("SECRET_KEY_SEED", "")
	t.Setenv("SECRET_KEY_SALT", "")

	first := Load().SecretKey
	second := Load().SecretKey

	if len(first) != 32 || len(second) != 32 {
		t.Fatalf("expected both keys to be 32 chars, got %d and %d", len(first), len(second))
	}
	if first == second {
		t.Fatalf("expected random fallback keys to differ when seed is missing")
	}
}

func TestLoad_AutoSecret_UsesDefaultSalt(t *testing.T) {
	t.Setenv("SECRET_KEY", "")
	t.Setenv("SECRET_KEY_SEED", "seed-abc")
	t.Setenv("SECRET_KEY_SALT", "")

	origNow := nowUTC
	nowUTC = func() time.Time { return time.Date(2026, 3, 27, 1, 2, 3, 0, time.UTC) }
	t.Cleanup(func() { nowUTC = origNow })

	got := Load().SecretKey
	want := deriveDailyKey("seed-abc", "conductor-v1", nowUTC())
	if got != want {
		t.Fatalf("expected key derived with default salt, got %q want %q", got, want)
	}
}
