package api

import (
	"net/http"
	"time"

	"github.com/benoit/rss2kindle/internal/refresher"
	"github.com/benoit/rss2kindle/internal/store"
)

type StatusResponse struct {
	NextDigestRun *time.Time       `json:"next_digest_run"`
	LastDigest    *store.Delivery  `json:"last_digest"`
	Refresher     refresher.Status `json:"refresher"`
	Counts        *store.Counts    `json:"counts"`
	Stats         []store.DailyStat `json:"stats"`
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	st, _ := s.store.GetSettings(ctx)
	loc := st.Location()

	nextRun := s.scheduler.NextRun()
	refStatus := s.refresher.Status(ctx)
	lastDigest, _ := s.store.LastDelivery(ctx, store.KindDigest)
	counts, _ := s.store.GetCounts(ctx, loc)
	stats, _ := s.store.Stats(ctx, 14, loc)

	writeJSON(w, http.StatusOK, StatusResponse{
		NextDigestRun: nextRun,
		LastDigest:    lastDigest,
		Refresher:     refStatus,
		Counts:        counts,
		Stats:         stats,
	})
}
