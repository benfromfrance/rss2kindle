// Package config loads runtime configuration from environment variables.
package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// Config holds process-level configuration. User-editable settings live in the database.
type Config struct {
	Addr          string
	DataDir       string
	AdminPassword string
	Secret        []byte // 32 bytes, used for AES-GCM and session signing
	DevCORSOrigin string
}

// Load reads configuration from the environment, applying sensible defaults.
func Load() (*Config, error) {
	c := &Config{
		Addr:          env("R2K_ADDR", ":8080"),
		DataDir:       env("R2K_DATA_DIR", "./data"),
		AdminPassword: os.Getenv("R2K_ADMIN_PASSWORD"),
		DevCORSOrigin: os.Getenv("R2K_DEV_CORS_ORIGIN"),
	}
	if err := os.MkdirAll(filepath.Join(c.DataDir, "epub"), 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	if c.AdminPassword == "" {
		c.AdminPassword = "admin"
		slog.Warn("R2K_ADMIN_PASSWORD not set, using default password 'admin' — change it!")
	}
	secret, err := loadSecret(c.DataDir)
	if err != nil {
		return nil, err
	}
	c.Secret = secret
	return c, nil
}

func env(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

// loadSecret uses R2K_SECRET (hex, 64 chars) or a persisted random secret in the data dir.
func loadSecret(dataDir string) ([]byte, error) {
	if v := os.Getenv("R2K_SECRET"); v != "" {
		b, err := hex.DecodeString(v)
		if err != nil || len(b) != 32 {
			return nil, fmt.Errorf("R2K_SECRET must be 64 hex characters (32 bytes)")
		}
		return b, nil
	}
	path := filepath.Join(dataDir, ".secret")
	if raw, err := os.ReadFile(path); err == nil {
		b, err := hex.DecodeString(strings.TrimSpace(string(raw)))
		if err == nil && len(b) == 32 {
			return b, nil
		}
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, []byte(hex.EncodeToString(b)), 0o600); err != nil {
		return nil, fmt.Errorf("persist secret: %w", err)
	}
	return b, nil
}
