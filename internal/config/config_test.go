package config

import (
	"os"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	// Ensure none of the env vars are set.
	for _, key := range []string{"LISTEN_ADDR", "DB_PATH", "DATA_DIR", "SECRET_KEY"} {
		t.Setenv(key, "")
	}

	cfg := Load()

	if cfg.ListenAddr != ":3001" {
		t.Errorf("ListenAddr: got %q, want %q", cfg.ListenAddr, ":3001")
	}
	if cfg.DBPath != "./data/conductor.db" {
		t.Errorf("DBPath: got %q, want %q", cfg.DBPath, "./data/conductor.db")
	}
	if cfg.DataDir != "./data" {
		t.Errorf("DataDir: got %q, want %q", cfg.DataDir, "./data")
	}
	if cfg.SecretKey != "change-me-in-production" {
		t.Errorf("SecretKey: got %q, want %q", cfg.SecretKey, "change-me-in-production")
	}
}

func TestLoad_EnvOverrides(t *testing.T) {
	os.Setenv("LISTEN_ADDR", ":8080")
	os.Setenv("DB_PATH", "/tmp/test.db")
	os.Setenv("DATA_DIR", "/tmp/data")
	os.Setenv("SECRET_KEY", "super-secret")
	t.Cleanup(func() {
		os.Unsetenv("LISTEN_ADDR")
		os.Unsetenv("DB_PATH")
		os.Unsetenv("DATA_DIR")
		os.Unsetenv("SECRET_KEY")
	})

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
