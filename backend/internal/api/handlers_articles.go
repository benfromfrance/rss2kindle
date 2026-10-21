package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/benoit/rss2kindle/internal/extract"
	"github.com/benoit/rss2kindle/internal/store"
	"github.com/go-chi/chi/v5"
)

func (s *Server) handleListArticles(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	feedID, _ := strconv.ParseInt(q.Get("feed_id"), 10, 64)
	catID, _ := strconv.ParseInt(q.Get("category_id"), 10, 64)
	colID, _ := strconv.ParseInt(q.Get("collection_id"), 10, 64)

	st, _ := s.store.GetSettings(r.Context())

	fl := store.ArticleFilter{
		View:         q.Get("view"),
		FeedID:       feedID,
		CategoryID:   catID,
		CollectionID: colID,
		Query:        q.Get("q"),
		Cursor:       q.Get("cursor"),
		Limit:        limit,
		TZ:           st.Location(),
	}

	page, err := s.store.ListArticles(r.Context(), fl)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func (s *Server) handleGetArticle(w http.ResponseWriter, r *http.Request) {
	id, err := parseInt64(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "identifiant invalide")
		return
	}

	art, err := s.store.GetArticle(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "article introuvable")
		return
	}
	writeJSON(w, http.StatusOK, art)
}

func (s *Server) handlePatchArticle(w http.ResponseWriter, r *http.Request) {
	id, err := parseInt64(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "identifiant invalide")
		return
	}

	var patch store.ArticlePatch
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		writeError(w, http.StatusBadRequest, "données invalides")
		return
	}

	if err := s.store.PatchArticles(r.Context(), []int64{id}, patch); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

type bulkReq struct {
	Action       string  `json:"action"` // mark_read | mark_unread | star | unstar | add_collection | remove_collection
	ArticleIDs   []int64 `json:"article_ids"`
	CollectionID *int64  `json:"collection_id,omitempty"`
}

func (s *Server) handleBulkArticles(w http.ResponseWriter, r *http.Request) {
	var req bulkReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "données invalides")
		return
	}
	if len(req.ArticleIDs) == 0 {
		writeJSON(w, http.StatusOK, map[string]int{"affected": 0})
		return
	}

	ctx := r.Context()
	switch req.Action {
	case "mark_read":
		t := true
		_ = s.store.PatchArticles(ctx, req.ArticleIDs, store.ArticlePatch{IsRead: &t})
	case "mark_unread":
		f := false
		_ = s.store.PatchArticles(ctx, req.ArticleIDs, store.ArticlePatch{IsRead: &f})
	case "star":
		t := true
		_ = s.store.PatchArticles(ctx, req.ArticleIDs, store.ArticlePatch{IsStarred: &t})
	case "unstar":
		f := false
		_ = s.store.PatchArticles(ctx, req.ArticleIDs, store.ArticlePatch{IsStarred: &f})
	case "add_collection":
		if req.CollectionID != nil {
			_ = s.store.AddToCollection(ctx, *req.CollectionID, req.ArticleIDs)
		}
	case "remove_collection":
		if req.CollectionID != nil {
			_ = s.store.RemoveFromCollection(ctx, *req.CollectionID, req.ArticleIDs)
		}
	default:
		writeError(w, http.StatusBadRequest, "action inconnue")
		return
	}

	writeJSON(w, http.StatusOK, map[string]int{"affected": len(req.ArticleIDs)})
}

func (s *Server) handleExtractFullText(w http.ResponseWriter, r *http.Request) {
	id, err := parseInt64(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "identifiant invalide")
		return
	}

	art, err := s.store.GetArticle(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "article introuvable")
		return
	}

	if art.URL == "" {
		writeError(w, http.StatusBadRequest, "l'article ne possède pas d'URL source")
		return
	}

	full, err := extract.FullText(r.Context(), extract.NewHTTPClient(25*time.Second), art.URL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "échec d'extraction du texte intégral: "+err.Error())
		return
	}

	plain := extract.PlainText(full)
	if err := s.store.SetFullContent(r.Context(), id, full, plain); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"full_content_html": full,
	})
}

type markAllReadReq struct {
	FeedID     int64 `json:"feed_id"`
	CategoryID int64 `json:"category_id"`
}

func (s *Server) handleMarkAllRead(w http.ResponseWriter, r *http.Request) {
	var req markAllReadReq
	_ = json.NewDecoder(r.Body).Decode(&req)

	affected, err := s.store.MarkAllRead(r.Context(), req.FeedID, req.CategoryID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"affected": affected})
}

func (s *Server) handleGetCounts(w http.ResponseWriter, r *http.Request) {
	st, _ := s.store.GetSettings(r.Context())
	counts, err := s.store.GetCounts(r.Context(), st.Location())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, counts)
}
