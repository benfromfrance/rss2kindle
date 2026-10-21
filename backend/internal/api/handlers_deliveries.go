package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/benoit/rss2kindle/internal/store"
	"github.com/go-chi/chi/v5"
)

func (s *Server) handleListDeliveries(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	deliveries, err := s.store.ListDeliveries(r.Context(), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, deliveries)
}

func (s *Server) handleGetDelivery(w http.ResponseWriter, r *http.Request) {
	id, err := parseInt64(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "identifiant invalide")
		return
	}
	deliv, err := s.store.GetDelivery(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "livraison introuvable")
		return
	}
	writeJSON(w, http.StatusOK, deliv)
}

type createDeliveryReq struct {
	Kind         string  `json:"kind"` // digest | selection | collection | starred
	Title        string  `json:"title"`
	ArticleIDs   []int64 `json:"article_ids"`
	CollectionID *int64  `json:"collection_id"`
	Send         *bool   `json:"send"` // default true
}

func (s *Server) handleCreateDelivery(w http.ResponseWriter, r *http.Request) {
	var req createDeliveryReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "données invalides")
		return
	}

	send := true
	if req.Send != nil {
		send = *req.Send
	}

	ctx := r.Context()
	var deliv *store.Delivery
	var err error

	switch req.Kind {
	case store.KindDigest, "":
		deliv, err = s.pipeline.QueueDigest(ctx)
		if deliv == nil && err == nil {
			writeJSON(w, http.StatusOK, map[string]string{"message": "aucun nouvel article pour le digest"})
			return
		}
	case store.KindSelection:
		deliv, err = s.pipeline.QueueSelection(ctx, req.Title, req.ArticleIDs, send)
	case store.KindCollection:
		if req.CollectionID == nil {
			writeError(w, http.StatusBadRequest, "collection_id requis")
			return
		}
		deliv, err = s.pipeline.QueueCollection(ctx, *req.CollectionID, send)
	case store.KindStarred:
		starredIDs, sErr := s.store.StarredArticleIDs(ctx, true)
		if sErr != nil {
			writeError(w, http.StatusInternalServerError, sErr.Error())
			return
		}
		if len(starredIDs) == 0 {
			writeError(w, http.StatusBadRequest, "aucun article favori non envoyé")
			return
		}
		title := req.Title
		if title == "" {
			title = "Articles Favoris"
		}
		deliv, err = s.pipeline.QueueSelection(ctx, title, starredIDs, send)
	default:
		writeError(w, http.StatusBadRequest, "type de livraison non pris en charge: "+req.Kind)
		return
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusAccepted, deliv)
}

type resendReq struct {
	Regenerate bool `json:"regenerate"`
}

func (s *Server) handleResendDelivery(w http.ResponseWriter, r *http.Request) {
	id, err := parseInt64(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "identifiant invalide")
		return
	}

	var req resendReq
	_ = json.NewDecoder(r.Body).Decode(&req)

	deliv, err := s.pipeline.QueueResend(r.Context(), id, req.Regenerate)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusAccepted, deliv)
}

func (s *Server) handleDownloadEpub(w http.ResponseWriter, r *http.Request) {
	id, err := parseInt64(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "identifiant invalide")
		return
	}

	deliv, err := s.store.GetDelivery(r.Context(), id)
	if err != nil || deliv.EpubPath == "" {
		writeError(w, http.StatusNotFound, "fichier EPUB introuvable")
		return
	}

	file, err := os.Open(deliv.EpubPath)
	if err != nil {
		writeError(w, http.StatusNotFound, "le fichier physique n'existe plus sur le serveur")
		return
	}
	defer file.Close()

	stat, _ := file.Stat()
	filename := filepath.Base(deliv.EpubPath)
	w.Header().Set("Content-Type", "application/epub+zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	if stat != nil {
		w.Header().Set("Content-Length", strconv.FormatInt(stat.Size(), 10))
	}

	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, file)
}

func (s *Server) handleDeleteDelivery(w http.ResponseWriter, r *http.Request) {
	id, err := parseInt64(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "identifiant invalide")
		return
	}

	deliv, _ := s.store.GetDelivery(r.Context(), id)
	if deliv != nil && deliv.EpubPath != "" {
		_ = os.Remove(deliv.EpubPath)
	}

	if err := s.store.DeleteDelivery(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
