package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/benoit/rss2kindle/internal/store"
)

func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	secret := make([]byte, 32)
	s, err := store.OpenMemory(secret)
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestFeedsAndArticles(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	// Catégories
	cat, err := s.CreateCategory(ctx, "Tech")
	if err != nil {
		t.Fatalf("CreateCategory: %v", err)
	}

	// Flux
	f := &store.Feed{
		CategoryID:      &cat.ID,
		URL:             "https://example.com/rss",
		Title:           "Example Tech",
		Enabled:         true,
		IncludeInDigest: true,
		MaxArticles:     10,
	}
	if err := s.CreateFeed(ctx, f); err != nil {
		t.Fatalf("CreateFeed: %v", err)
	}

	// Articles
	now := time.Now().UTC()
	articles := []store.NewArticle{
		{
			GUIDHash:    "hash1",
			URL:         "https://example.com/art1",
			Title:       "Introduction à Go et Kindle",
			Summary:     "Résumé court de l'article 1",
			ContentHTML: "<p>Texte complet de l'article 1</p>",
			PlainText:   "Introduction à Go et Kindle Texte complet de l'article 1",
			PublishedAt: now.Add(-2 * time.Hour),
		},
		{
			GUIDHash:    "hash2",
			URL:         "https://example.com/art2",
			Title:       "Deuxième article intéressant",
			Summary:     "Résumé court de l'article 2",
			ContentHTML: "<p>Texte complet de l'article 2</p>",
			PlainText:   "Deuxième article intéressant Texte complet de l'article 2",
			PublishedAt: now.Add(-1 * time.Hour),
		},
	}

	n, err := s.InsertArticles(ctx, f.ID, articles)
	if err != nil {
		t.Fatalf("InsertArticles: %v", err)
	}
	if n != 2 {
		t.Fatalf("expected 2 inserted, got %d", n)
	}

	// Dédoublonnage
	n2, err := s.InsertArticles(ctx, f.ID, articles)
	if err != nil {
		t.Fatalf("InsertArticles dedup: %v", err)
	}
	if n2 != 0 {
		t.Fatalf("expected 0 inserted on duplicate, got %d", n2)
	}

	// Recherche FTS5
	page, err := s.ListArticles(ctx, store.ArticleFilter{Query: "Kindle"})
	if err != nil {
		t.Fatalf("ListArticles FTS: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].Title != "Introduction à Go et Kindle" {
		t.Fatalf("expected 1 result matching Kindle, got %d", len(page.Items))
	}

	// Candidats pour digest
	candidates, err := s.DigestCandidates(ctx, now.Add(-24*time.Hour), false, 100)
	if err != nil {
		t.Fatalf("DigestCandidates: %v", err)
	}
	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(candidates))
	}

	// Marquer comme envoyé
	if err := s.MarkSent(ctx, []int64{candidates[0]}, time.Now()); err != nil {
		t.Fatalf("MarkSent: %v", err)
	}

	candidatesAfter, err := s.DigestCandidates(ctx, now.Add(-24*time.Hour), false, 100)
	if err != nil {
		t.Fatalf("DigestCandidates after: %v", err)
	}
	if len(candidatesAfter) != 1 {
		t.Fatalf("expected 1 remaining candidate, got %d", len(candidatesAfter))
	}
}

func TestCollections(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	f := &store.Feed{URL: "https://example.com/rss", Title: "Feed"}
	_ = s.CreateFeed(ctx, f)
	_, _ = s.InsertArticles(ctx, f.ID, []store.NewArticle{
		{GUIDHash: "h1", Title: "A1", PublishedAt: time.Now()},
		{GUIDHash: "h2", Title: "A2", PublishedAt: time.Now()},
	})

	page, _ := s.ListArticles(ctx, store.ArticleFilter{})
	a1 := page.Items[0].ID
	a2 := page.Items[1].ID

	col, err := s.CreateCollection(ctx, "À lire ce soir")
	if err != nil {
		t.Fatalf("CreateCollection: %v", err)
	}

	if err := s.AddToCollection(ctx, col.ID, []int64{a1, a2}); err != nil {
		t.Fatalf("AddToCollection: %v", err)
	}

	ids, err := s.CollectionArticleIDs(ctx, col.ID)
	if err != nil {
		t.Fatalf("CollectionArticleIDs: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("expected 2 items, got %d", len(ids))
	}

	// Reorder
	if err := s.ReorderCollection(ctx, col.ID, []int64{a2, a1}); err != nil {
		t.Fatalf("ReorderCollection: %v", err)
	}
	idsAfter, _ := s.CollectionArticleIDs(ctx, col.ID)
	if idsAfter[0] != a2 || idsAfter[1] != a1 {
		t.Fatalf("expected [a2, a1], got %v", idsAfter)
	}
}

func TestSettingsCrypto(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	st := store.DefaultSettings()
	st.KindleEmail = "mydevice@kindle.com"
	st.SenderEmail = "digest@mymail.com"
	st.SMTPHost = "smtp.mailgun.org"
	st.SMTPUser = "user"
	st.SMTPPassword = "super-secret-password-1234"

	saved, err := s.SaveSettings(ctx, st, false)
	if err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}

	loaded, err := s.GetSettings(ctx)
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}

	if loaded.KindleEmail != "mydevice@kindle.com" {
		t.Errorf("KindleEmail mismatch")
	}
	if loaded.SMTPPassword != "super-secret-password-1234" {
		t.Errorf("expected decrypted password %q, got %q", "super-secret-password-1234", loaded.SMTPPassword)
	}
	if !saved.MailConfigured() {
		t.Errorf("MailConfigured should be true")
	}
}
