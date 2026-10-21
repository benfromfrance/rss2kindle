package store

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"time"
)

// Settings are user-editable application settings, persisted as JSON.
type Settings struct {
	// Kindle / e-mail
	KindleEmail  string `json:"kindle_email"`
	SenderEmail  string `json:"sender_email"`
	SenderName   string `json:"sender_name"`
	SMTPHost     string `json:"smtp_host"`
	SMTPPort     int    `json:"smtp_port"`
	SMTPUser     string `json:"smtp_user"`
	SMTPPassword string `json:"smtp_password,omitempty"` // plaintext only in memory / API input
	SMTPSecurity string `json:"smtp_security"`           // starttls | tls | none
	SMTPPassEnc  string `json:"smtp_password_enc,omitempty"`

	// Daily digest
	DigestEnabled     bool   `json:"digest_enabled"`
	DigestTime        string `json:"digest_time"` // HH:MM
	DigestDays        []int  `json:"digest_days"` // 0=Sunday … 6=Saturday
	Timezone          string `json:"timezone"`
	DigestWindowHours int    `json:"digest_window_hours"`
	DigestExcludeRead bool   `json:"digest_exclude_read"`
	DigestSkipEmpty   bool   `json:"digest_skip_empty"`
	DigestTitle       string `json:"digest_title"`
	DigestMaxArticles int    `json:"digest_max_articles"`

	// Fetching
	RefreshMinutes int `json:"refresh_minutes"`

	// Content
	ImagesEnabled   bool   `json:"images_enabled"`
	ImagesGrayscale bool   `json:"images_grayscale"`
	ImageMaxWidth   int    `json:"image_max_width"`
	Language        string `json:"language"`

	// Retention
	RetentionDays int `json:"retention_days"`
}

// DefaultSettings returns sensible defaults.
func DefaultSettings() Settings {
	return Settings{
		SMTPPort:          587,
		SMTPSecurity:      "starttls",
		SenderName:        "RSS to Kindle",
		DigestEnabled:     true,
		DigestTime:        "01:00",
		DigestDays:        []int{0, 1, 2, 3, 4, 5, 6},
		Timezone:          "Europe/Paris",
		DigestWindowHours: 24,
		DigestSkipEmpty:   true,
		DigestTitle:       "RSS Digest",
		DigestMaxArticles: 150,
		RefreshMinutes:    30,
		ImagesEnabled:     true,
		ImagesGrayscale:   true,
		ImageMaxWidth:     600,
		Language:          "fr",
		RetentionDays:     30,
	}
}

var timeRe = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)

// Validate normalises and validates settings.
func (st *Settings) Validate() error {
	st.KindleEmail = strings.TrimSpace(st.KindleEmail)
	st.SenderEmail = strings.TrimSpace(st.SenderEmail)
	if st.KindleEmail != "" {
		if _, err := mail.ParseAddress(st.KindleEmail); err != nil {
			return errors.New("adresse Kindle invalide")
		}
	}
	if st.SenderEmail != "" {
		if _, err := mail.ParseAddress(st.SenderEmail); err != nil {
			return errors.New("adresse d'expéditeur invalide")
		}
	}
	if !timeRe.MatchString(st.DigestTime) {
		return errors.New("heure du digest invalide (HH:MM)")
	}
	if _, err := time.LoadLocation(st.Timezone); err != nil {
		return fmt.Errorf("fuseau horaire invalide: %s", st.Timezone)
	}
	switch st.SMTPSecurity {
	case "starttls", "tls", "none":
	default:
		st.SMTPSecurity = "starttls"
	}
	for _, d := range st.DigestDays {
		if d < 0 || d > 6 {
			return errors.New("jour invalide")
		}
	}
	clamp := func(v *int, lo, hi, def int) {
		if *v == 0 {
			*v = def
		}
		if *v < lo {
			*v = lo
		}
		if *v > hi {
			*v = hi
		}
	}
	clamp(&st.SMTPPort, 1, 65535, 587)
	clamp(&st.DigestWindowHours, 1, 24*30, 24)
	clamp(&st.DigestMaxArticles, 1, 1000, 150)
	clamp(&st.RefreshMinutes, 5, 24*60, 30)
	clamp(&st.ImageMaxWidth, 200, 2000, 600)
	clamp(&st.RetentionDays, 1, 3650, 30)
	if strings.TrimSpace(st.DigestTitle) == "" {
		st.DigestTitle = "RSS Digest"
	}
	if st.Language == "" {
		st.Language = "fr"
	}
	return nil
}

// Location returns the configured timezone.
func (st Settings) Location() *time.Location {
	loc, err := time.LoadLocation(st.Timezone)
	if err != nil {
		return time.UTC
	}
	return loc
}

// MailConfigured reports whether enough settings exist to send e-mail.
func (st Settings) MailConfigured() bool {
	return st.KindleEmail != "" && st.SenderEmail != "" && st.SMTPHost != ""
}

// GetSettings loads settings (with decrypted SMTP password).
func (s *Store) GetSettings(ctx context.Context) (Settings, error) {
	st := DefaultSettings()
	var raw string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = 'app'`).Scan(&raw)
	if err == sql.ErrNoRows {
		return st, nil
	}
	if err != nil {
		return st, err
	}
	if err := json.Unmarshal([]byte(raw), &st); err != nil {
		return st, err
	}
	if st.SMTPPassEnc != "" {
		pw, err := s.decrypt(st.SMTPPassEnc)
		if err == nil {
			st.SMTPPassword = pw
		}
	}
	st.SMTPPassEnc = ""
	return st, nil
}

// SaveSettings persists settings. An empty SMTPPassword keeps the existing one unless clearPassword is set.
func (s *Store) SaveSettings(ctx context.Context, st Settings, clearPassword bool) (Settings, error) {
	if err := st.Validate(); err != nil {
		return st, err
	}
	if st.SMTPPassword == "" && !clearPassword {
		cur, err := s.GetSettings(ctx)
		if err != nil {
			return st, err
		}
		st.SMTPPassword = cur.SMTPPassword
	}
	toStore := st
	toStore.SMTPPassEnc = ""
	if st.SMTPPassword != "" {
		enc, err := s.encrypt(st.SMTPPassword)
		if err != nil {
			return st, err
		}
		toStore.SMTPPassEnc = enc
	}
	toStore.SMTPPassword = ""
	raw, err := json.Marshal(toStore)
	if err != nil {
		return st, err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO settings(key, value) VALUES ('app', ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`, string(raw))
	return st, err
}

func (s *Store) encrypt(plain string) (string, error) {
	block, err := aes.NewCipher(s.secret)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(gcm.Seal(nonce, nonce, []byte(plain), nil)), nil
}

func (s *Store) decrypt(enc string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(enc)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(s.secret)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", errors.New("ciphertext too short")
	}
	out, err := gcm.Open(nil, raw[:gcm.NonceSize()], raw[gcm.NonceSize():], nil)
	return string(out), err
}
