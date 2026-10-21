// Package refresher periodically fetches all enabled feeds and stores new articles.
package refresher

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/benoit/rss2kindle/internal/feed"
	"github.com/benoit/rss2kindle/internal/store"
)

// Refresher keeps the article database up to date.
type Refresher struct {
	store   *store.Store
	fetcher *feed.Fetcher
	workers int

	running atomic.Bool
	mu      sync.Mutex
	lastRun time.Time
	trigger chan struct{}
}

// New creates a Refresher.
func New(s *store.Store, f *feed.Fetcher) *Refresher {
	return &Refresher{store: s, fetcher: f, workers: 8, trigger: make(chan struct{}, 1)}
}

// Status describes the refresher state.
type Status struct {
	Running bool       `json:"running"`
	LastRun *time.Time `json:"last_run"`
	NextRun *time.Time `json:"next_run"`
}

// Status returns the current state.
func (r *Refresher) Status(ctx context.Context) Status {
	r.mu.Lock()
	defer r.mu.Unlock()
	st := Status{Running: r.running.Load()}
	if !r.lastRun.IsZero() {
		lr := r.lastRun
		st.LastRun = &lr
		if set, err := r.store.GetSettings(ctx); err == nil {
			nr := lr.Add(time.Duration(set.RefreshMinutes) * time.Minute)
			st.NextRun = &nr
		}
	}
	return st
}

// Run loops until ctx is cancelled, refreshing at the configured interval.
func (r *Refresher) Run(ctx context.Context) {
	// First refresh shortly after startup.
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		case <-r.trigger:
		}
		r.RefreshAll(ctx)
		interval := 30 * time.Minute
		if set, err := r.store.GetSettings(ctx); err == nil {
			interval = time.Duration(set.RefreshMinutes) * time.Minute
		}
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		timer.Reset(interval)
	}
}

// Trigger asks the background loop to refresh now (non-blocking).
func (r *Refresher) Trigger() {
	select {
	case r.trigger <- struct{}{}:
	default:
	}
}

// RefreshAll fetches every enabled feed. Concurrent calls are coalesced.
func (r *Refresher) RefreshAll(ctx context.Context) {
	if !r.running.CompareAndSwap(false, true) {
		return
	}
	defer r.running.Store(false)
	feeds, err := r.store.ListEnabledFeeds(ctx)
	if err != nil {
		slog.Error("refresh: list feeds", "err", err)
		return
	}
	start := time.Now()
	jobs := make(chan store.Feed)
	var wg sync.WaitGroup
	var total atomic.Int64
	for i := 0; i < r.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for f := range jobs {
				n, _ := r.RefreshFeed(ctx, &f)
				total.Add(int64(n))
			}
		}()
	}
	for _, f := range feeds {
		jobs <- f
	}
	close(jobs)
	wg.Wait()
	r.mu.Lock()
	r.lastRun = time.Now()
	r.mu.Unlock()
	slog.Info("refresh done", "feeds", len(feeds), "new_articles", total.Load(), "took", time.Since(start).Round(time.Millisecond))
}

// RefreshFeed fetches a single feed and stores its new articles.
func (r *Refresher) RefreshFeed(ctx context.Context, f *store.Feed) (int, error) {
	fctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	res, err := r.fetcher.Fetch(fctx, f.URL, f.ETag, f.LastModified)
	if errors.Is(err, feed.ErrNotModified) {
		_ = r.store.UpdateFeedFetchState(ctx, f.ID, f.ETag, f.LastModified, "", "", "")
		return 0, nil
	}
	if err != nil {
		slog.Warn("refresh feed failed", "feed", f.URL, "err", err)
		_ = r.store.UpdateFeedFetchState(ctx, f.ID, f.ETag, f.LastModified, err.Error(), "", "")
		return 0, err
	}
	n, err := r.store.InsertArticles(ctx, f.ID, res.Items)
	if err != nil {
		_ = r.store.UpdateFeedFetchState(ctx, f.ID, res.ETag, res.LastModified, err.Error(), res.SiteURL, res.Title)
		return n, err
	}
	_ = r.store.UpdateFeedFetchState(ctx, f.ID, res.ETag, res.LastModified, "", res.SiteURL, res.Title)
	return n, nil
}
