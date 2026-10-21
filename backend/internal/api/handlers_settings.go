package api

import (
	"encoding/json"
	"net/http"

	"github.com/benoit/rss2kindle/internal/store"
)

func (s *Server) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	st, err := s.store.GetSettings(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	// Masquer le mot de passe SMTP dans la réponse JSON
	hasPass := st.SMTPPassword != ""
	st.SMTPPassword = ""
	type settingsResp struct {
		store.Settings
		HasSMTPPassword bool `json:"has_smtp_password"`
	}
	writeJSON(w, http.StatusOK, settingsResp{
		Settings:        st,
		HasSMTPPassword: hasPass,
	})
}

type updateSettingsReq struct {
	store.Settings
	ClearSMTPPassword bool `json:"clear_smtp_password"`
}

func (s *Server) handleUpdateSettings(w http.ResponseWriter, r *http.Request) {
	var req updateSettingsReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "données invalides")
		return
	}

	saved, err := s.store.SaveSettings(r.Context(), req.Settings, req.ClearSMTPPassword)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Notifier le scheduler de recharger le cron
	_ = s.scheduler.ReloadSettings(r.Context())

	hasPass := saved.SMTPPassword != ""
	saved.SMTPPassword = ""
	writeJSON(w, http.StatusOK, map[string]any{
		"settings":          saved,
		"has_smtp_password": hasPass,
	})
}

type testEmailReq struct {
	Recipient    string `json:"recipient"`
	SMTPHost     string `json:"smtp_host"`
	SMTPPort     int    `json:"smtp_port"`
	SMTPUser     string `json:"smtp_user"`
	SMTPPassword string `json:"smtp_password"`
	SMTPSecurity string `json:"smtp_security"`
	SenderEmail  string `json:"sender_email"`
	SenderName   string `json:"sender_name"`
	KindleEmail  string `json:"kindle_email"`
}

func (s *Server) handleTestEmail(w http.ResponseWriter, r *http.Request) {
	var req testEmailReq
	_ = json.NewDecoder(r.Body).Decode(&req)

	st, err := s.store.GetSettings(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Surcharges depuis la requête directe si saisies dans l'UI
	if req.SMTPHost != "" {
		st.SMTPHost = req.SMTPHost
	}
	if req.SMTPPort > 0 {
		st.SMTPPort = req.SMTPPort
	}
	if req.SMTPUser != "" {
		st.SMTPUser = req.SMTPUser
	}
	if req.SMTPPassword != "" {
		st.SMTPPassword = req.SMTPPassword
	}
	if req.SMTPSecurity != "" {
		st.SMTPSecurity = req.SMTPSecurity
	}
	if req.SenderEmail != "" {
		st.SenderEmail = req.SenderEmail
	}
	if req.SenderName != "" {
		st.SenderName = req.SenderName
	}
	if req.KindleEmail != "" {
		st.KindleEmail = req.KindleEmail
	}

	// Détermination du destinataire de test
	dest := req.Recipient
	if dest == "" {
		dest = req.SenderEmail
	}
	if dest == "" {
		dest = req.KindleEmail
	}
	if dest == "" {
		dest = st.SenderEmail
	}
	if dest == "" {
		dest = st.KindleEmail
	}

	if dest == "" {
		writeError(w, http.StatusBadRequest, "veuillez renseigner au moins une adresse email (expéditeur ou Kindle) pour recevoir le test")
		return
	}

	if err := s.mailer.SendTest(r.Context(), st, dest); err != nil {
		writeError(w, http.StatusBadRequest, "échec envoi test SMTP: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "E-mail de test envoyé avec succès à " + dest,
	})
}
