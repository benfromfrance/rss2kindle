package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const cookieName = "r2k_session"

type Session struct {
	Subject   string `json:"sub"`
	ExpiresAt int64  `json:"exp"`
}

// signSession produit un cookie signé HMAC-SHA256.
func signSession(secret []byte, duration time.Duration) string {
	sess := Session{
		Subject:   "admin",
		ExpiresAt: time.Now().Add(duration).Unix(),
	}
	payload, _ := json.Marshal(sess)
	b64Payload := base64.RawURLEncoding.EncodeToString(payload)

	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(b64Payload))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return fmt.Sprintf("%s.%s", b64Payload, sig)
}

// verifySession valide le cookie signé.
func verifySession(token string, secret []byte) bool {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return false
	}
	b64Payload, sig := parts[0], parts[1]

	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(b64Payload))
	expectedSig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(sig), []byte(expectedSig)) {
		return false
	}

	raw, err := base64.RawURLEncoding.DecodeString(b64Payload)
	if err != nil {
		return false
	}

	var sess Session
	if err := json.Unmarshal(raw, &sess); err != nil {
		return false
	}

	return time.Now().Unix() < sess.ExpiresAt
}

func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(cookieName)
		if err != nil || !verifySession(cookie.Value, s.cfg.Secret) {
			writeError(w, http.StatusUnauthorized, "authentification requise")
			return
		}
		next.ServeHTTP(w, r)
	})
}

type loginRequest struct {
	Password string `json:"password"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "données invalides")
		return
	}

	if req.Password != s.cfg.AdminPassword {
		writeError(w, http.StatusUnauthorized, "mot de passe incorrect")
		return
	}

	token := signSession(s.cfg.Secret, 30*24*time.Hour)
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(30 * 24 * time.Hour / time.Second),
	})

	writeJSON(w, http.StatusOK, map[string]any{"authenticated": true})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
	writeJSON(w, http.StatusOK, map[string]any{"authenticated": false})
}

func (s *Server) handleAuthMe(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(cookieName)
	authed := err == nil && verifySession(cookie.Value, s.cfg.Secret)
	writeJSON(w, http.StatusOK, map[string]any{"authenticated": authed})
}

// Helpers json response
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func parseInt64(val string) (int64, error) {
	return strconv.ParseInt(val, 10, 64)
}
