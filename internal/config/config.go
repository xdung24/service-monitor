package config

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"time"
)

var nowUTC = func() time.Time {
	return time.Now().UTC()
}

// Config holds all application configuration sourced from environment variables.
// Runtime settings (e.g. registration_enabled) are stored in the database and
// editable by the admin via the web UI.
type Config struct {
	ListenAddr    string
	DBPath        string
	DataDir       string
	SecretKey     string
	SecureCookies bool
	SessionMaxAge time.Duration

	// System SMTP — used for transactional emails (invite, password reset, etc.).
	// All fields are empty by default; setting SystemSMTPHost enables sending.
	SystemSMTPHost     string
	SystemSMTPPort     string
	SystemSMTPUsername string
	SystemSMTPPassword string
	SystemSMTPFrom     string
	SystemSMTPTLS      string // "true" (STARTTLS, default) or "false"
	SystemSMTPBCC      string // optional BCC added to every outgoing message
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	dataDir := getEnv("DATA_DIR", "./data")
	secretKey := resolveSecretKey()

	return &Config{
		ListenAddr:    getEnv("LISTEN_ADDR", ":3001"),
		DBPath:        getEnv("DB_PATH", "./data/conductor.db"),
		DataDir:       dataDir,
		SecretKey:     secretKey,
		SecureCookies: getEnvBool("SECURE_COOKIES", false),
		SessionMaxAge: getEnvDuration("SESSION_MAX_AGE", 24*time.Hour),

		SystemSMTPHost:     os.Getenv("SYSTEM_SMTP_HOST"),
		SystemSMTPPort:     getEnv("SYSTEM_SMTP_PORT", "587"),
		SystemSMTPUsername: os.Getenv("SYSTEM_SMTP_USERNAME"),
		SystemSMTPPassword: os.Getenv("SYSTEM_SMTP_PASSWORD"),
		SystemSMTPFrom:     os.Getenv("SYSTEM_SMTP_FROM"),
		SystemSMTPTLS:      getEnv("SYSTEM_SMTP_TLS", "true"),
		SystemSMTPBCC:      os.Getenv("SYSTEM_SMTP_BCC"),
	}
}

func resolveSecretKey() string {
	if v := os.Getenv("SECRET_KEY"); v != "" {
		return v
	}

	seed := os.Getenv("SECRET_KEY_SEED")
	if seed != "" {
		salt := getEnv("SECRET_KEY_SALT", "conductor-v1")
		return deriveDailyKey(seed, salt, nowUTC())
	}

	return randomHex32()
}

func randomHex32() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}

func deriveDailyKey(seed, salt string, now time.Time) string {
	day := now.UTC().Format("2006-01-02")
	msg := fmt.Sprintf("%s:%s", salt, day)

	mac := hmac.New(sha256.New, []byte(seed))
	_, _ = mac.Write([]byte(msg))
	sum := hex.EncodeToString(mac.Sum(nil))

	// 32 hex chars == 128 bits, matches SECRET_KEY length policy.
	return sum[:32]
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}
