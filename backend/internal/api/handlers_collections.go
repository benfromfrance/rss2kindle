package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (s *Server) handleListCollections(w http.ResponseWriter, r *http.Request) {
	cols, err := s.store.ListCollections(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, cols)
}

func (s *Server) handleGetCollection(w http.ResponseWriter, r *http.Request) {
	id, err := parseInt64(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "identifiant invalide")
		return
	}
	col, err := s.store.GetCollection(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "collection introuvable")
		return
	}
	writeJSON(w, http.StatusOK, col)
}

type collectionReq struct {
	Name string `json:"name"`
}

func (s *Server) handleCreateCollection(w http.ResponseWriter, r *http.Request) {
	var req collectionReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "données invalides")
		return
	}
	col, err := s.store.CreateCollection(r.Context(), req.Name)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, col)
}

func (s *Server) handleUpdateCollection(w http.ResponseWriter, r *http.Request) {
	id, err := parseInt64(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "identifiant invalide")
		return
	}
	var req collectionReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "données invalides")
		return
	}
	if err := s.store.RenameCollection(r.Context(), id, req.Name); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleDeleteCollection(w http.ResponseWriter, r *http.Request) {
	id, err := parseInt64(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "identifiant invalide")
		return
	}
	if err := s.store.DeleteCollection(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

type collectionItemsReq struct {
	Action     string  `json:"action"` // add | remove | reorder
	ArticleIDs []int64 `json:"article_ids"`
}

func (s *Server) handleCollectionItems(w http.ResponseWriter, r *http.Request) {
	id, err := parseInt64(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "identifiant invalide")
		return
	}

	var req collectionItemsReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "données invalides")
		return
	}

	ctx := r.Context()
	switch req.Action {
	case "add":
		err = s.store.AddToCollection(ctx, id, req.ArticleIDs)
	case "remove":
		err = s.store.RemoveFromCollection(ctx, id, req.ArticleIDs)
	case "reorder":
		err = s.store.ReorderCollection(ctx, id, req.ArticleIDs)
	default:
		writeError(w, http.StatusBadRequest, "action inconnue")
		return
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
