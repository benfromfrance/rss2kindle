package api

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/benoit/rss2kindle/internal/config"
	"github.com/benoit/rss2kindle/internal/delivery"
	"github.com/benoit/rss2kindle/internal/feed"
	"github.com/benoit/rss2kindle/internal/mailer"
	"github.com/benoit/rss2kindle/internal/refresher"
	"github.com/benoit/rss2kindle/internal/scheduler"
	"github.com/benoit/rss2kindle/internal/store"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Server struct {
	cfg       *config.Config
	store     *store.Store
	fetcher   *feed.Fetcher
	refresher *refresher.Refresher
	pipeline  *delivery.Pipeline
	scheduler *scheduler.Scheduler
	mailer    *mailer.Mailer
	webFS     fs.FS

	router chi.Router
}

func NewServer(
	cfg *config.Config,
	s *store.Store,
	f *feed.Fetcher,
	r *refresher.Refresher,
	p *delivery.Pipeline,
	sch *scheduler.Scheduler,
	m *mailer.Mailer,
	webFS fs.FS,
) *Server {
	srv := &Server{
		cfg:       cfg,
		store:     s,
		fetcher:   f,
		refresher: r,
		pipeline:  p,
		scheduler: sch,
		mailer:    m,
		webFS:     webFS,
	}
	srv.routes()
	return srv
}

func (s *Server) Handler() http.Handler {
	return s.router
}

func (s *Server) routes() {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Dev CORS si configuré
	if s.cfg.DevCORSOrigin != "" {
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.Header().Set("Access-Control-Allow-Origin", s.cfg.DevCORSOrigin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token")
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				if req.Method == http.MethodOptions {
					w.WriteHeader(http.StatusOK)
					return
				}
				next.ServeHTTP(w, req)
			})
		})
	}

	// Routes publiques
	r.Post("/api/auth/login", s.handleLogin)
	r.Post("/api/auth/logout", s.handleLogout)
	r.Get("/api/auth/me", s.handleAuthMe)

	// Routes protégées
	r.Group(func(protected chi.Router) {
		protected.Use(s.requireAuth)

		// Catégories
		protected.Get("/api/categories", s.handleListCategories)
		protected.Post("/api/categories", s.handleCreateCategory)
		protected.Put("/api/categories/{id}", s.handleUpdateCategory)
		protected.Delete("/api/categories/{id}", s.handleDeleteCategory)

		// Flux
		protected.Get("/api/feeds", s.handleListFeeds)
		protected.Post("/api/feeds", s.handleCreateFeed)
		protected.Get("/api/feeds/{id}", s.handleGetFeed)
		protected.Put("/api/feeds/{id}", s.handleUpdateFeed)
		protected.Delete("/api/feeds/{id}", s.handleDeleteFeed)
		protected.Post("/api/feeds/validate", s.handleValidateFeed)
		protected.Post("/api/feeds/{id}/refresh", s.handleRefreshFeed)
		protected.Post("/api/feeds/refresh-all", s.handleRefreshAllFeeds)

		// OPML
		protected.Get("/api/opml/export", s.handleExportOPML)
		protected.Post("/api/opml/import", s.handleImportOPML)

		// Articles
		protected.Get("/api/articles", s.handleListArticles)
		protected.Get("/api/articles/{id}", s.handleGetArticle)
		protected.Patch("/api/articles/{id}", s.handlePatchArticle)
		protected.Post("/api/articles/bulk", s.handleBulkArticles)
		protected.Post("/api/articles/{id}/fulltext", s.handleExtractFullText)
		protected.Post("/api/articles/mark-all-read", s.handleMarkAllRead)

		// Compteurs
		protected.Get("/api/counts", s.handleGetCounts)

		// Collections
		protected.Get("/api/collections", s.handleListCollections)
		protected.Post("/api/collections", s.handleCreateCollection)
		protected.Get("/api/collections/{id}", s.handleGetCollection)
		protected.Put("/api/collections/{id}", s.handleUpdateCollection)
		protected.Delete("/api/collections/{id}", s.handleDeleteCollection)
		protected.Post("/api/collections/{id}/items", s.handleCollectionItems)

		// Livraisons
		protected.Get("/api/deliveries", s.handleListDeliveries)
		protected.Post("/api/deliveries", s.handleCreateDelivery)
		protected.Get("/api/deliveries/{id}", s.handleGetDelivery)
		protected.Post("/api/deliveries/{id}/resend", s.handleResendDelivery)
		protected.Get("/api/deliveries/{id}/epub", s.handleDownloadEpub)
		protected.Delete("/api/deliveries/{id}", s.handleDeleteDelivery)

		// Paramètres
		protected.Get("/api/settings", s.handleGetSettings)
		protected.Put("/api/settings", s.handleUpdateSettings)
		protected.Post("/api/settings/test-email", s.handleTestEmail)

		// Statut général & stats
		protected.Get("/api/status", s.handleStatus)
	})

	// Service du frontend SPA React (si webFS fourni)
	if s.webFS != nil {
		fileServer := http.FileServer(http.FS(s.webFS))
		indexHTML, _ := fs.ReadFile(s.webFS, "index.html")

		r.Get("/*", func(w http.ResponseWriter, req *http.Request) {
			path := strings.TrimPrefix(req.URL.Path, "/")
			if path != "" && path != "index.html" {
				if f, err := s.webFS.Open(path); err == nil {
					_ = f.Close()
					fileServer.ServeHTTP(w, req)
					return
				}
			}
			// Servir directement index.html pour la racine et toutes les routes SPA
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(indexHTML)
		})
	}

	s.router = r
}
