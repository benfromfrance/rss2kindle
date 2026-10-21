// Package extract cleans article HTML and extracts full text with Readability.
package extract

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/PuerkitoBio/goquery"
	readability "github.com/go-shiori/go-readability"
	"github.com/microcosm-cc/bluemonday"
)

// UserAgent is sent on every outgoing request.
const UserAgent = "Mozilla/5.0 (compatible; RSS2Kindle/1.0; +https://github.com/benoit/rss2kindle)"

var policy = func() *bluemonday.Policy {
	p := bluemonday.UGCPolicy()
	p.AllowAttrs("src", "alt", "title", "width", "height").OnElements("img")
	p.AllowElements("figure", "figcaption", "picture", "section", "article", "aside", "mark", "small", "sub", "sup")
	p.RequireNoFollowOnLinks(false)
	p.AddTargetBlankToFullyQualifiedLinks(false)
	return p
}()

// NewHTTPClient returns a client suited for fetching feeds and pages.
func NewHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}
}

// Sanitize removes unsafe markup and resolves relative URLs against base.
func Sanitize(html, base string) string {
	if strings.TrimSpace(html) == "" {
		return ""
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err == nil {
		baseURL, _ := url.Parse(base)
		doc.Find("script, style, iframe, object, embed, form, noscript, svg, video, audio, button, input").Remove()
		doc.Find("img").Each(func(_ int, s *goquery.Selection) {
			// Lazy-loaded images often hide the real URL.
			for _, attr := range []string{"data-src", "data-original", "data-lazy-src"} {
				if v, ok := s.Attr(attr); ok && v != "" {
					s.SetAttr("src", v)
					break
				}
			}
			if src, ok := s.Attr("src"); ok {
				s.SetAttr("src", resolve(baseURL, src))
			}
			s.RemoveAttr("srcset")
		})
		doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
			href, _ := s.Attr("href")
			s.SetAttr("href", resolve(baseURL, href))
		})
		// Remove tracking pixels.
		doc.Find(`img[width="1"], img[height="1"]`).Remove()
		if h, err := doc.Find("body").Html(); err == nil {
			html = h
		}
	}
	return strings.TrimSpace(policy.Sanitize(html))
}

func resolve(base *url.URL, ref string) string {
	ref = strings.TrimSpace(ref)
	if base == nil || ref == "" {
		return ref
	}
	u, err := url.Parse(ref)
	if err != nil {
		return ref
	}
	return base.ResolveReference(u).String()
}

var spaceRe = regexp.MustCompile(`\s+`)

// PlainText strips tags and collapses whitespace.
func PlainText(html string) string {
	if html == "" {
		return ""
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return ""
	}
	doc.Find("script, style").Remove()
	return strings.TrimSpace(spaceRe.ReplaceAllString(doc.Text(), " "))
}

// Truncate shortens s to at most n runes, adding an ellipsis.
func Truncate(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	r := []rune(s)
	cut := string(r[:n])
	if i := strings.LastIndex(cut, " "); i > n/2 {
		cut = cut[:i]
	}
	return strings.TrimSpace(cut) + "…"
}

// FirstImage returns the src of the first image in html.
func FirstImage(html string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return ""
	}
	src, _ := doc.Find("img[src]").First().Attr("src")
	return src
}

// FullText downloads pageURL and extracts the main content with Readability.
// Returns sanitized HTML.
func FullText(ctx context.Context, client *http.Client, pageURL string) (string, error) {
	u, err := url.Parse(pageURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return "", fmt.Errorf("URL invalide: %q", pageURL)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	body := io.LimitReader(resp.Body, 10<<20)
	art, err := readability.FromReader(body, resp.Request.URL)
	if err != nil {
		return "", fmt.Errorf("readability: %w", err)
	}
	clean := Sanitize(art.Content, resp.Request.URL.String())
	if PlainText(clean) == "" {
		return "", fmt.Errorf("aucun contenu extrait")
	}
	return clean, nil
}
