package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/benoit/rss2kindle/internal/store"
	"github.com/go-chi/chi/v5"
)

// Catégories

func (s *Server) handleListCategories(w http.ResponseWriter, r *http.Request) {
	cats, err := s.store.ListCategories(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, cats)
}

type categoryReq struct {
	Name     string `json:"name"`
	Position int    `json:"position"`
}

func (s *Server) handleCreateCategory(w http.ResponseWriter, r *http.Request) {
	var req categoryReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "requête invalide")
		return
	}
	cat, err := s.store.CreateCategory(r.Context(), req.Name)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, cat)
}

func (s *Server) handleUpdateCategory(w http.ResponseWriter, r *http.Request) {
	id, err := parseInt64(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "identifiant invalide")
		return
	}
	var req categoryReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "requête invalide")
		return
	}
	err = s.store.UpdateCategory(r.Context(), store.Category{
		ID:       id,
		Name:     req.Name,
		Position: req.Position,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleDeleteCategory(w http.ResponseWriter, r *http.Request) {
	id, err := parseInt64(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "identifiant invalide")
		return
	}
	if err := s.store.DeleteCategory(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// Flux

func (s *Server) handleListFeeds(w http.ResponseWriter, r *http.Request) {
	feeds, err := s.store.ListFeeds(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, feeds)
}

func (s *Server) handleGetFeed(w http.ResponseWriter, r *http.Request) {
	id, err := parseInt64(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "identifiant invalide")
		return
	}
	feed, err := s.store.GetFeed(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "flux introuvable")
		return
	}
	writeJSON(w, http.StatusOK, feed)
}

type createFeedReq struct {
	URL             string `json:"url"`
	Title           string `json:"title"`
	CategoryID      *int64 `json:"category_id"`
	Enabled         bool   `json:"enabled"`
	FullText        bool   `json:"full_text"`
	IncludeInDigest bool   `json:"include_in_digest"`
	MaxArticles     int    `json:"max_articles"`
}

func (s *Server) handleCreateFeed(w http.ResponseWriter, r *http.Request) {
	var req createFeedReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "requête invalide")
		return
	}
	if req.URL == "" {
		writeError(w, http.StatusBadRequest, "URL requise")
		return
	}

	// Découverte du flux
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	disc, err := s.fetcher.Discover(ctx, req.URL)
	if err != nil {
		writeError(w, http.StatusBadRequest, "impossible de récupérer ce flux: "+err.Error())
		return
	}

	title := req.Title
	if title == "" {
		title = disc.Title
	}

	f := &store.Feed{
		CategoryID:      req.CategoryID,
		URL:             disc.FeedURL,
		Title:           title,
		SiteURL:         disc.SiteURL,
		Enabled:         true,
		FullText:        req.FullText,
		IncludeInDigest: true,
		MaxArticles:     req.MaxArticles,
	}
	if req.MaxArticles <= 0 {
		f.MaxArticles = 20
	}

	if err := s.store.CreateFeed(r.Context(), f); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Enregistrement des premiers articles
	if len(disc.Items) > 0 {
		_, _ = s.store.InsertArticles(r.Context(), f.ID, disc.Items)
		_ = s.store.UpdateFeedFetchState(r.Context(), f.ID, disc.ETag, disc.LastModified, "", disc.SiteURL, disc.Title)
	}

	writeJSON(w, http.StatusCreated, f)
}

func (s *Server) handleUpdateFeed(w http.ResponseWriter, r *http.Request) {
	id, err := parseInt64(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "identifiant invalide")
		return
	}
	var f store.Feed
	if err := json.NewDecoder(r.Body).Decode(&f); err != nil {
		writeError(w, http.StatusBadRequest, "requête invalide")
		return
	}
	f.ID = id

	if err := s.store.UpdateFeed(r.Context(), &f); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleDeleteFeed(w http.ResponseWriter, r *http.Request) {
	id, err := parseInt64(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "identifiant invalide")
		return
	}
	if err := s.store.DeleteFeed(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

type validateReq struct {
	URL string `json:"url"`
}

func (s *Server) handleValidateFeed(w http.ResponseWriter, r *http.Request) {
	var req validateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "requête invalide")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	res, err := s.fetcher.Discover(ctx, req.URL)
	if err != nil {
		writeError(w, http.StatusBadRequest, "flux introuvable: "+err.Error())
		return
	}

	type sampleItem struct {
		Title       string `json:"title"`
		URL         string `json:"url"`
		Author      string `json:"author"`
		PublishedAt string `json:"published_at"`
		Summary     string `json:"summary"`
	}
	var samples []sampleItem
	for i, it := range res.Items {
		if i >= 5 {
			break
		}
		samples = append(samples, sampleItem{
			Title:       it.Title,
			URL:         it.URL,
			Author:      it.Author,
			PublishedAt: it.PublishedAt.Format("02/01/2006 15:04"),
			Summary:     it.Summary,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"title":         res.Title,
		"feed_url":      res.FeedURL,
		"site_url":      res.SiteURL,
		"article_count": len(res.Items),
		"samples":       samples,
	})
}

func (s *Server) handleRefreshFeed(w http.ResponseWriter, r *http.Request) {
	id, err := parseInt64(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "identifiant invalide")
		return
	}
	feed, err := s.store.GetFeed(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "flux introuvable")
		return
	}

	newCount, err := s.refresher.RefreshFeed(r.Context(), feed)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"new_articles": newCount})
}

func (s *Server) handleRefreshAllFeeds(w http.ResponseWriter, r *http.Request) {
	go s.refresher.RefreshAll(context.Background())
	writeJSON(w, http.StatusOK, map[string]string{"message": "rafraîchissement démarré"})
}
