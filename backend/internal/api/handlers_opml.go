package api

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/benoit/rss2kindle/internal/store"
)

type opml struct {
	XMLName xml.Name `xml:"opml"`
	Version string   `xml:"version,attr"`
	Head    opmlHead `xml:"head"`
	Body    opmlBody `xml:"body"`
}

type opmlHead struct {
	Title string `xml:"title"`
}

type opmlBody struct {
	Outlines []opmlOutline `xml:"outline"`
}

type opmlOutline struct {
	Text     string        `xml:"text,attr"`
	Title    string        `xml:"title,attr,omitempty"`
	Type     string        `xml:"type,attr,omitempty"`
	XMLURL   string        `xml:"xmlUrl,attr,omitempty"`
	HTMLURL  string        `xml:"htmlUrl,attr,omitempty"`
	Outlines []opmlOutline `xml:"outline,omitempty"`
}

func (s *Server) handleExportOPML(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	feeds, err := s.store.ListFeeds(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	cats, err := s.store.ListCategories(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	catMap := make(map[int64]string)
	for _, c := range cats {
		catMap[c.ID] = c.Name
	}

	// Regroupement par catégorie
	byCat := make(map[string][]store.Feed)
	var uncategorized []store.Feed

	for _, f := range feeds {
		if f.CategoryID != nil && catMap[*f.CategoryID] != "" {
			name := catMap[*f.CategoryID]
			byCat[name] = append(byCat[name], f)
		} else {
			uncategorized = append(uncategorized, f)
		}
	}

	doc := opml{
		Version: "2.0",
		Head:    opmlHead{Title: "RSS to Kindle Subscriptions"},
	}

	for _, c := range cats {
		catFeeds := byCat[c.Name]
		if len(catFeeds) == 0 {
			continue
		}
		var children []opmlOutline
		for _, f := range catFeeds {
			children = append(children, opmlOutline{
				Text:    f.Title,
				Title:   f.Title,
				Type:    "rss",
				XMLURL:  f.URL,
				HTMLURL: f.SiteURL,
			})
		}
		doc.Body.Outlines = append(doc.Body.Outlines, opmlOutline{
			Text:     c.Name,
			Title:    c.Name,
			Outlines: children,
		})
	}

	for _, f := range uncategorized {
		doc.Body.Outlines = append(doc.Body.Outlines, opmlOutline{
			Text:    f.Title,
			Title:   f.Title,
			Type:    "rss",
			XMLURL:  f.URL,
			HTMLURL: f.SiteURL,
		})
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"rss2kindle_%s.opml\"", time.Now().Format("2006-01-02")))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(xml.Header))
	_ = xml.NewEncoder(w).Encode(doc)
}

func (s *Server) handleImportOPML(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "fichier OPML manquant ou trop volumineux")
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "champ 'file' requis")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusBadRequest, "lecture du fichier impossible")
		return
	}

	var doc opml
	if err := xml.Unmarshal(data, &doc); err != nil {
		writeError(w, http.StatusBadRequest, "format XML/OPML invalide: "+err.Error())
		return
	}

	ctx := r.Context()
	imported := 0

	var parseOutline func(o opmlOutline, catID *int64)
	parseOutline = func(o opmlOutline, catID *int64) {
		xmlURL := strings.TrimSpace(o.XMLURL)
		if xmlURL != "" {
			title := strings.TrimSpace(o.Title)
			if title == "" {
				title = strings.TrimSpace(o.Text)
			}
			f := &store.Feed{
				CategoryID:      catID,
				URL:             xmlURL,
				Title:           title,
				SiteURL:         strings.TrimSpace(o.HTMLURL),
				Enabled:         true,
				IncludeInDigest: true,
				MaxArticles:     20,
			}
			if err := s.store.CreateFeed(ctx, f); err == nil {
				imported++
			}
		}

		if len(o.Outlines) > 0 {
			// C'est un dossier / catégorie
			catName := strings.TrimSpace(o.Title)
			if catName == "" {
				catName = strings.TrimSpace(o.Text)
			}
			var nextCatID = catID
			if catName != "" && xmlURL == "" {
				if c, err := s.store.CreateCategory(ctx, catName); err == nil {
					nextCatID = &c.ID
				}
			}
			for _, child := range o.Outlines {
				parseOutline(child, nextCatID)
			}
		}
	}

	for _, o := range doc.Body.Outlines {
		parseOutline(o, nil)
	}

	// Lancement en arrière-plan d'un rafraîchissement
	go s.refresher.RefreshAll(context.Background())

	writeJSON(w, http.StatusOK, map[string]any{
		"imported": imported,
		"message":  fmt.Sprintf("%d flux importés avec succès", imported),
	})
}
