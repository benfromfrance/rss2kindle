package ebook_test

import (
	"context"
	"testing"
	"time"

	"github.com/benoit/rss2kindle/internal/ebook"
)

func TestGenerateCover(t *testing.T) {
	sources := []string{"Hacker News", "Le Monde", "The Verge"}
	coverData, err := ebook.GenerateCover("RSS Digest", time.Now(), sources, 42)
	if err != nil {
		t.Fatalf("GenerateCover failed: %v", err)
	}
	if len(coverData) < 1000 {
		t.Fatalf("cover JPEG too small: %d bytes", len(coverData))
	}
}

func TestBuildEpub(t *testing.T) {
	ctx := context.Background()
	articles := []ebook.Article{
		{
			Title:       "Premier article de test",
			FeedTitle:   "Source 1",
			Author:      "Alice",
			URL:         "https://example.com/1",
			PublishedAt: time.Now(),
			HTML:        "<p>Bonjour le monde. Ceci est un test de <strong>génération EPUB</strong>.</p>",
		},
		{
			Title:       "Deuxième article avec liste",
			FeedTitle:   "Source 2",
			Author:      "Bob",
			URL:         "https://example.com/2",
			PublishedAt: time.Now(),
			HTML:        "<ul><li>Élément 1</li><li>Élément 2</li></ul>",
		},
	}

	opts := ebook.Options{
		Title:       "Test Book",
		Lang:        "fr",
		GroupByFeed: true,
		Images:      false,
		Location:    time.UTC,
	}

	data, stats, err := ebook.Build(ctx, articles, opts)
	if err != nil {
		t.Fatalf("Build EPUB failed: %v", err)
	}

	if stats.Articles != 2 {
		t.Fatalf("expected 2 articles, got %d", stats.Articles)
	}
	if len(data) < 2000 {
		t.Fatalf("EPUB binary too small: %d bytes", len(data))
	}

	// Signature zip standard "PK\x03\x04"
	if string(data[0:2]) != "PK" {
		t.Fatalf("expected PK zip magic header, got %q", string(data[0:2]))
	}
}
