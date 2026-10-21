// Package ebook builds Kindle-compatible EPUB files from articles.
package ebook

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"html"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/benoit/rss2kindle/internal/imageproc"
	epub "github.com/go-shiori/go-epub"
	xhtml "golang.org/x/net/html"
)

// Article is the input for a book chapter.
type Article struct {
	Title       string
	FeedTitle   string
	Author      string
	URL         string
	PublishedAt time.Time
	HTML        string
}

// Options controls book generation.
type Options struct {
	Title          string
	Lang           string
	GroupByFeed    bool
	Images         bool
	Grayscale      bool
	ImageMaxWidth  int
	MaxImageBytes  int64 // total image budget
	Location       *time.Location
	HTTPClient     *http.Client
	ImageWorkers   int
}

// Stats describes the generated book.
type Stats struct {
	Articles int
	Images   int
	Size     int64
}

const css = `
body { font-family: serif; line-height: 1.45; margin: 0 2%; }
h1 { font-size: 1.5em; line-height: 1.2; margin: 0.4em 0 0.3em; page-break-before: always; }
h1.feed { font-size: 1.8em; text-align: center; margin-top: 30%; }
h2 { font-size: 1.25em; } h3 { font-size: 1.1em; }
p { margin: 0 0 0.7em; text-align: justify; }
.meta { font-size: 0.8em; color: #444; margin-bottom: 1.2em; font-family: sans-serif; text-align: left; }
.source { font-size: 0.8em; margin-top: 1.5em; border-top: 1px solid #999; padding-top: 0.5em; font-family: sans-serif; }
img { max-width: 100%; height: auto; display: block; margin: 0.6em auto; }
figure { margin: 0.8em 0; } figcaption { font-size: 0.8em; color: #555; text-align: center; }
blockquote { margin: 0.8em 1.2em; font-style: italic; }
pre, code { font-family: monospace; font-size: 0.85em; white-space: pre-wrap; }
table { border-collapse: collapse; } td, th { border: 1px solid #999; padding: 0.2em 0.4em; }
ul.toc { list-style: none; padding: 0; } ul.toc li { margin: 0.35em 0; }
ul.toc li.feed { font-weight: bold; margin-top: 0.9em; }
ul.toc li.art { margin-left: 1em; }
`

// Build generates the EPUB and returns its bytes.
func Build(ctx context.Context, arts []Article, o Options) ([]byte, Stats, error) {
	var st Stats
	if len(arts) == 0 {
		return nil, st, fmt.Errorf("aucun article")
	}
	if o.Location == nil {
		o.Location = time.UTC
	}
	if o.Lang == "" {
		o.Lang = "fr"
	}
	if o.HTTPClient == nil {
		o.HTTPClient = &http.Client{Timeout: 20 * time.Second}
	}
	if o.MaxImageBytes <= 0 {
		o.MaxImageBytes = 35 << 20
	}
	if o.ImageWorkers <= 0 {
		o.ImageWorkers = 6
	}
	now := time.Now().In(o.Location)

	book, err := epub.NewEpub(o.Title)
	if err != nil {
		return nil, st, err
	}
	book.SetLang(o.Lang)
	book.SetAuthor("RSS to Kindle")
	book.SetDescription(fmt.Sprintf("%d articles — %s", len(arts), FormatDateFR(now)))

	cssPath, err := book.AddCSS(dataURL("text/css", []byte(css)), "style.css")
	if err != nil {
		return nil, st, fmt.Errorf("css: %w", err)
	}

	// Cover
	sources := uniqueFeeds(arts)
	coverJPG, err := GenerateCover(o.Title, now, sources, len(arts))
	if err != nil {
		return nil, st, fmt.Errorf("cover: %w", err)
	}
	coverPath, err := book.AddImage(dataURL("image/jpeg", coverJPG), "cover.jpg")
	if err != nil {
		return nil, st, err
	}
	if err := book.SetCover(coverPath, ""); err != nil {
		return nil, st, err
	}

	// Prepare article bodies (images processed concurrently).
	bodies := make([]string, len(arts))
	imgs := newImageBudget(o.MaxImageBytes)
	var mu sync.Mutex // guards book.AddImage
	imgCounter := 0
	sem := make(chan struct{}, o.ImageWorkers)
	var wg sync.WaitGroup
	for i := range arts {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			bodies[i] = prepareContent(ctx, arts[i], o, sem, func(data []byte) (string, error) {
				if !imgs.take(int64(len(data))) {
					return "", fmt.Errorf("budget images dépassé")
				}
				mu.Lock()
				defer mu.Unlock()
				imgCounter++
				return book.AddImage(dataURL("image/jpeg", data), fmt.Sprintf("img%04d.jpg", imgCounter))
			})
		}(i)
	}
	wg.Wait()
	st.Images = imgCounter

	// Table of contents page.
	type chapter struct {
		file  string
		art   int
		group string
	}
	chapters := make([]chapter, len(arts))
	for i := range arts {
		chapters[i] = chapter{file: fmt.Sprintf("article%04d.xhtml", i+1), art: i, group: arts[i].FeedTitle}
	}
	var toc strings.Builder
	toc.WriteString(`<h1>Sommaire</h1><ul class="toc">`)
	lastGroup := "\x00"
	for _, c := range chapters {
		if o.GroupByFeed && c.group != lastGroup {
			fmt.Fprintf(&toc, `<li class="feed">%s</li>`, html.EscapeString(c.group))
			lastGroup = c.group
		}
		cls := ""
		if o.GroupByFeed {
			cls = ` class="art"`
		}
		fmt.Fprintf(&toc, `<li%s><a href="%s">%s</a></li>`, cls, c.file, html.EscapeString(arts[c.art].Title))
	}
	toc.WriteString(`</ul>`)
	if _, err := book.AddSection(toc.String(), "Sommaire", "toc.xhtml", cssPath); err != nil {
		return nil, st, err
	}

	// Chapters
	parent := ""
	lastGroup = "\x00"
	feedN := 0
	for _, c := range chapters {
		a := arts[c.art]
		if o.GroupByFeed && c.group != lastGroup {
			feedN++
			parent = fmt.Sprintf("feed%03d.xhtml", feedN)
			body := fmt.Sprintf(`<h1 class="feed">%s</h1>`, html.EscapeString(c.group))
			if _, err := book.AddSection(body, c.group, parent, cssPath); err != nil {
				return nil, st, err
			}
			lastGroup = c.group
		}
		chapterBody := renderArticle(a, bodies[c.art], o.Location)
		if o.GroupByFeed {
			_, err = book.AddSubSection(parent, chapterBody, a.Title, c.file, cssPath)
		} else {
			_, err = book.AddSection(chapterBody, a.Title, c.file, cssPath)
		}
		if err != nil {
			return nil, st, fmt.Errorf("chapitre %q: %w", a.Title, err)
		}
	}

	var buf bytes.Buffer
	if _, err := book.WriteTo(&buf); err != nil {
		return nil, st, err
	}
	st.Articles = len(arts)
	st.Size = int64(buf.Len())
	return buf.Bytes(), st, nil
}

func renderArticle(a Article, body string, loc *time.Location) string {
	var meta []string
	if a.FeedTitle != "" {
		meta = append(meta, html.EscapeString(a.FeedTitle))
	}
	if a.Author != "" {
		meta = append(meta, html.EscapeString(a.Author))
	}
	if !a.PublishedAt.IsZero() {
		meta = append(meta, a.PublishedAt.In(loc).Format("02/01/2006 15:04"))
	}
	var b strings.Builder
	fmt.Fprintf(&b, `<h1>%s</h1>`, html.EscapeString(a.Title))
	if len(meta) > 0 {
		fmt.Fprintf(&b, `<p class="meta">%s</p>`, strings.Join(meta, " · "))
	}
	b.WriteString(body)
	if a.URL != "" {
		fmt.Fprintf(&b, `<p class="source">Source : <a href="%s">%s</a></p>`, html.EscapeString(a.URL), html.EscapeString(shortURL(a.URL)))
	}
	return b.String()
}

func shortURL(u string) string {
	u = strings.TrimPrefix(strings.TrimPrefix(u, "https://"), "http://")
	if len(u) > 60 {
		u = u[:57] + "…"
	}
	return u
}

// prepareContent embeds images (or strips them) and returns XHTML-safe markup.
func prepareContent(ctx context.Context, a Article, o Options, sem chan struct{}, add func([]byte) (string, error)) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader("<html><body>" + a.HTML + "</body></html>"))
	if err != nil {
		return "<p>" + html.EscapeString(a.Title) + "</p>"
	}
	// Headings inside content shouldn't compete with the chapter title.
	doc.Find("h1").Each(func(_ int, s *goquery.Selection) { s.Get(0).Data = "h2" })
	// Drop empty links wrapping images? Keep but neutralise picture/source.
	doc.Find("source").Remove()

	if !o.Images {
		doc.Find("img").Remove()
	} else {
		nodes := doc.Find("img")
		results := make([]string, nodes.Length())
		var wg sync.WaitGroup
		nodes.Each(func(i int, s *goquery.Selection) {
			src, _ := s.Attr("src")
			if src == "" || strings.HasPrefix(src, "data:") {
				return
			}
			wg.Add(1)
			go func(i int, src string) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				ictx, cancel := context.WithTimeout(ctx, 20*time.Second)
				defer cancel()
				raw, err := imageproc.Download(ictx, o.HTTPClient, src, a.URL)
				if err != nil {
					slog.Debug("image download failed", "src", src, "err", err)
					return
				}
				jpg, _, _, err := imageproc.Convert(raw, imageproc.Options{MaxWidth: o.ImageMaxWidth, Grayscale: o.Grayscale})
				if err != nil {
					return
				}
				if path, err := add(jpg); err == nil {
					results[i] = path
				}
			}(i, src)
		})
		wg.Wait()
		// Apply DOM changes sequentially.
		nodes.Each(func(i int, s *goquery.Selection) {
			if results[i] == "" {
				s.Remove()
				return
			}
			s.SetAttr("src", results[i])
			s.RemoveAttr("width")
			s.RemoveAttr("height")
			if _, ok := s.Attr("alt"); !ok {
				s.SetAttr("alt", "")
			}
		})
	}
	return toXHTML(doc.Find("body"))
}

// toXHTML renders nodes using x/net/html, which produces well-formed void tags (<br/>),
// and replaces named entities with characters, as required by EPUB's XHTML.
func toXHTML(sel *goquery.Selection) string {
	var buf bytes.Buffer
	for _, n := range sel.Nodes {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			stripBadAttrs(c)
			_ = xhtml.Render(&buf, c)
		}
	}
	return buf.String()
}

// stripBadAttrs removes attributes whose names are not valid XML names.
func stripBadAttrs(n *xhtml.Node) {
	if n.Type == xhtml.ElementNode {
		attrs := n.Attr[:0]
		for _, a := range n.Attr {
			if validXMLName(a.Key) && a.Namespace == "" {
				attrs = append(attrs, a)
			}
		}
		n.Attr = attrs
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		stripBadAttrs(c)
	}
}

func validXMLName(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		ok := r == '_' || r == ':' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
		if i > 0 {
			ok = ok || r == '-' || r == '.' || (r >= '0' && r <= '9')
		}
		if !ok {
			return false
		}
	}
	return true
}

func uniqueFeeds(arts []Article) []string {
	seen := map[string]bool{}
	var out []string
	for _, a := range arts {
		if a.FeedTitle != "" && !seen[a.FeedTitle] {
			seen[a.FeedTitle] = true
			out = append(out, a.FeedTitle)
		}
	}
	return out
}

func dataURL(mime string, data []byte) string {
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data)
}

type imageBudget struct {
	mu   sync.Mutex
	left int64
}

func newImageBudget(n int64) *imageBudget { return &imageBudget{left: n} }

func (b *imageBudget) take(n int64) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if n > b.left {
		return false
	}
	b.left -= n
	return true
}
