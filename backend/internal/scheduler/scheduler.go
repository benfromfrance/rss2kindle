package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/benoit/rss2kindle/internal/delivery"
	"github.com/benoit/rss2kindle/internal/store"
	"github.com/robfig/cron/v3"
)

// Scheduler orchestre les tâches périodiques : digest quotidien et nettoyage.
type Scheduler struct {
	store    *store.Store
	pipeline *delivery.Pipeline

	cron    *cron.Cron
	entryID cron.EntryID
	mu      sync.Mutex
}

func New(s *store.Store, p *delivery.Pipeline) *Scheduler {
	return &Scheduler{
		store:    s,
		pipeline: p,
	}
}

// Start initialise le planificateur de tâches.
func (s *Scheduler) Start(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.setupCron(ctx)

	// Lancement d'une goroutine de nettoyage quotidien
	go func() {
		ticker := time.NewTicker(6 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.cleanup(ctx)
			}
		}
	}()
}

// ReloadSettings reprogramme le cron avec les paramètres actualisés.
func (s *Scheduler) ReloadSettings(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cron != nil {
		s.cron.Stop()
	}

	s.setupCron(ctx)
	return nil
}

func (s *Scheduler) setupCron(ctx context.Context) {
	st, err := s.store.GetSettings(ctx)
	if err != nil {
		slog.Error("scheduler: chargement settings", "err", err)
		return
	}

	loc := st.Location()
	s.cron = cron.New(cron.WithLocation(loc))

	if st.DigestEnabled && st.DigestTime != "" {
		parts := strings.Split(st.DigestTime, ":")
		if len(parts) == 2 {
			hour, _ := strconv.Atoi(parts[0])
			min, _ := strconv.Atoi(parts[1])

			// Construction du cron spec (ex: "0 1 * * 0,1,2,3,4,5,6")
			daysStr := "*"
			if len(st.DigestDays) > 0 && len(st.DigestDays) < 7 {
				var ds []string
				for _, d := range st.DigestDays {
					ds = append(ds, strconv.Itoa(d))
				}
				daysStr = strings.Join(ds, ",")
			}

			cronSpec := fmt.Sprintf("%d %d * * %s", min, hour, daysStr)

			id, err := s.cron.AddFunc(cronSpec, func() {
				slog.Info("déclenchement planifié du digest Kindle", "time", st.DigestTime, "tz", st.Timezone)
				if _, err := s.pipeline.QueueDigest(context.Background()); err != nil {
					slog.Error("échec déclenchement digest planifié", "err", err)
				}
			})
			if err != nil {
				slog.Error("scheduler: cron spec invalide", "spec", cronSpec, "err", err)
			} else {
				s.entryID = id
				slog.Info("digest programmé", "spec", cronSpec, "tz", st.Timezone)
			}
		}
	}

	s.cron.Start()
}

// NextRun renvoie l'heure de la prochaine exécution planifiée du digest.
func (s *Scheduler) NextRun() *time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cron == nil {
		return nil
	}
	entry := s.cron.Entry(s.entryID)
	if entry.Next.IsZero() {
		return nil
	}
	t := entry.Next
	return &t
}

func (s *Scheduler) cleanup(ctx context.Context) {
	st, err := s.store.GetSettings(ctx)
	if err != nil {
		return
	}

	retention := time.Duration(st.RetentionDays) * 24 * time.Hour
	cutoff := time.Now().Add(-retention)

	// Purge articles
	if n, err := s.store.PurgeOldArticles(ctx, cutoff); err == nil && n > 0 {
		slog.Info("nettoyage: articles purgés", "count", n)
	}

	// Purge fichiers EPUB expirés
	expired, err := s.store.ExpiredEpubs(ctx, cutoff)
	if err == nil {
		for _, d := range expired {
			if d.EpubPath != "" {
				_ = os.Remove(d.EpubPath)
				_ = s.store.ClearEpubPath(ctx, d.ID)
			}
		}
	}
}
