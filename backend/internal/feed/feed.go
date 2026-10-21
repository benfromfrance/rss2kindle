// Package feed fetches and parses RSS / Atom / JSON feeds.
package feed

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/benoit/rss2kindle/internal/extract"
	"github.com/benoit/rss2kindle/internal/store"
	"github.com/mmcdole/gofeed"
)

// ErrNotModified is returned when the server answers 304.
var ErrNotModified = errors.New("not modified")

// Result is a parsed feed.
type Result struct {
	Title        string
	SiteURL      string
	FeedURL      string
	ETag         string
	LastModified string
	Items        []store.NewArticle
}

// Fetcher downloads feeds.
type Fetcher struct {
	Client *http.Client
}

// NewFetcher returns a Fetcher with a sensible timeout.
func NewFetcher() *Fetcher {
	return &Fetcher{Client: extract.NewHTTPClient(25 * time.Second)}
}

// Fetch downloads and parses a feed, using conditional headers when provided.
func (f *Fetcher) Fetch(ctx context.Context, feedURL, etag, lastModified string) (*Result, error) {
	body, resp, err := f.get(ctx, feedURL, etag, lastModified)
	if err != nil {
		return nil, err
	}
	parsed, err := gofeed.NewParser().Parse(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("flux illisible: %w", err)
	}
	res := &Result{
		Title:        strings.TrimSpace(parsed.Title),
		SiteURL:      parsed.Link,
		FeedURL:      resp.Request.URL.String(),
		ETag:         resp.Header.Get("ETag"),
		LastModified: resp.Header.Get("Last-Modified"),
	}
	base := parsed.Link
	if base == "" {
		base = res.FeedURL
	}
	for _, it := range parsed.Items {
		res.Items = append(res.Items, toArticle(it, base))
	}
	return res, nil
}

// Discover resolves a user-supplied URL to a feed URL. If the URL is an HTML
// page, it looks for <link rel="alternate"> feed declarations.
func (f *Fetcher) Discover(ctx context.Context, rawURL string) (*Result, error) {
	rawURL = strings.TrimSpace(rawURL)
	if !strings.Contains(rawURL, "://") {
		rawURL = "https://" + rawURL
	}
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return nil, fmt.Errorf("URL invalide")
	}
	body, resp, err := f.get(ctx, rawURL, "", "")
	if err != nil {
		return nil, err
	}
	if parsed, perr := gofeed.NewParser().Parse(bytes.NewReader(body)); perr == nil {
		res := &Result{Title: strings.TrimSpace(parsed.Title), SiteURL: parsed.Link, FeedURL: resp.Request.URL.String()}
		base := parsed.Link
		if base == "" {
			base = res.FeedURL
		}
		for _, it := range parsed.Items {
			res.Items = append(res.Items, toArticle(it, base))
		}
		return res, nil
	}
	// Not a feed: try HTML autodiscovery.
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("aucun flux trouvé à cette adresse")
	}
	var candidate string
	doc.Find(`link[rel="alternate"]`).EachWithBreak(func(_ int, s *goquery.Selection) bool {
		t, _ := s.Attr("type")
		href, _ := s.Attr("href")
		if href != "" && (strings.Contains(t, "rss") || strings.Contains(t, "atom") || strings.Contains(t, "feed+json")) {
			if ref, err := url.Parse(href); err == nil {
				candidate = resp.Request.URL.ResolveReference(ref).String()
				return false
			}
		}
		return true
	})
	if candidate == "" {
		return nil, fmt.Errorf("aucun flux RSS/Atom trouvé sur cette page")
	}
	return f.Fetch(ctx, candidate, "", "")
}

func (f *Fetcher) get(ctx context.Context, u, etag, lastModified string) ([]byte, *http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("User-Agent", extract.UserAgent)
	req.Header.Set("Accept", "application/rss+xml, application/atom+xml, application/feed+json, application/xml;q=0.9, text/html;q=0.8, */*;q=0.5")
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}
	if lastModified != "" {
		req.Header.Set("If-Modified-Since", lastModified)
	}
	resp, err := f.Client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotModified {
		return nil, resp, ErrNotModified
	}
	if resp.StatusCode >= 400 {
		return nil, resp, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 20<<20))
	return body, resp, err
}

func toArticle(it *gofeed.Item, base string) store.NewArticle {
	link := strings.TrimSpace(it.Link)
	if link != "" {
		if b, err := url.Parse(base); err == nil {
			if l, err := url.Parse(link); err == nil {
				link = b.ResolveReference(l).String()
			}
		}
	}
	raw := it.Content
	if strings.TrimSpace(raw) == "" {
		raw = it.Description
	}
	pageBase := link
	if pageBase == "" {
		pageBase = base
	}
	content := extract.Sanitize(raw, pageBase)
	plain := extract.PlainText(content)

	img := ""
	if it.Image != nil {
		img = it.Image.URL
	}
	if img == "" {
		for _, enc := range it.Enclosures {
			if strings.HasPrefix(enc.Type, "image/") {
				img = enc.URL
				break
			}
		}
	}
	if img == "" {
		img = extract.FirstImage(content)
	}

	published := time.Now().UTC()
	if it.PublishedParsed != nil {
		published = it.PublishedParsed.UTC()
	} else if it.UpdatedParsed != nil {
		published = it.UpdatedParsed.UTC()
	}
	// Guard against feeds with future dates.
	if published.After(time.Now().Add(24 * time.Hour)) {
		published = time.Now().UTC()
	}

	author := ""
	if it.Author != nil {
		author = it.Author.Name
	} else if len(it.Authors) > 0 && it.Authors[0] != nil {
		author = it.Authors[0].Name
	}

	title := strings.TrimSpace(extract.PlainText(it.Title))
	if title == "" {
		title = extract.Truncate(plain, 80)
	}
	if title == "" {
		title = "(sans titre)"
	}

	key := it.GUID
	if key == "" {
		key = link
	}
	if key == "" {
		key = title + published.String()
	}
	sum := sha1.Sum([]byte(key))

	return store.NewArticle{
		GUIDHash:    hex.EncodeToString(sum[:]),
		URL:         link,
		Title:       title,
		Author:      strings.TrimSpace(author),
		Summary:     extract.Truncate(plain, 280),
		ContentHTML: content,
		PlainText:   plain,
		ImageURL:    img,
		PublishedAt: published,
	}
}
