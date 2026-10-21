package delivery

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/benoit/rss2kindle/internal/ebook"
	"github.com/benoit/rss2kindle/internal/extract"
	"github.com/benoit/rss2kindle/internal/mailer"
	"github.com/benoit/rss2kindle/internal/store"
)

// Pipeline gère la file d'attente et l'exécution asynchrone des livraisons.
type Pipeline struct {
	store      *store.Store
	mailer     *mailer.Mailer
	dataDir    string
	httpClient *http.Client

	queue   chan int64
	mu      sync.Mutex
	running bool
}

func NewPipeline(s *store.Store, m *mailer.Mailer, dataDir string) *Pipeline {
	return &Pipeline{
		store:      s,
		mailer:     m,
		dataDir:    dataDir,
		httpClient: extract.NewHTTPClient(30 * time.Second),
		queue:      make(chan int64, 100),
	}
}

// Start lance le worker en arrière-plan qui traite les livraisons séquentielles.
func (p *Pipeline) Start(ctx context.Context) {
	// Nettoyage au démarrage : passer en failed toute livraison bloquée
	_ = p.store.FailStaleDeliveries(ctx)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case deliveryID := <-p.queue:
				p.process(ctx, deliveryID)
			}
		}
	}()
}

// Enqueue place une livraison en file d'attente de traitement.
func (p *Pipeline) Enqueue(deliveryID int64) {
	select {
	case p.queue <- deliveryID:
	default:
		slog.Error("file d'attente de livraison saturée", "delivery_id", deliveryID)
	}
}

// QueueDigest génère et enfile la livraison quotidienne de digest.
func (p *Pipeline) QueueDigest(ctx context.Context) (*store.Delivery, error) {
	st, err := p.store.GetSettings(ctx)
	if err != nil {
		return nil, err
	}

	since := time.Now().Add(-time.Duration(st.DigestWindowHours) * time.Hour)
	candidates, err := p.store.DigestCandidates(ctx, since, st.DigestExcludeRead, st.DigestMaxArticles)
	if err != nil {
		return nil, fmt.Errorf("recherche articles digest: %w", err)
	}

	if len(candidates) == 0 && st.DigestSkipEmpty {
		slog.Info("aucun nouvel article pour le digest, livraison ignorée")
		return nil, nil
	}

	title := fmt.Sprintf("%s — %s", st.DigestTitle, ebook.FormatDateFR(time.Now().In(st.Location())))
	deliv := &store.Delivery{
		Kind:       store.KindDigest,
		Title:      title,
		Status:     store.StatusQueued,
		Send:       true,
		ArticleIDs: candidates,
	}

	if err := p.store.CreateDelivery(ctx, deliv); err != nil {
		return nil, err
	}

	p.Enqueue(deliv.ID)
	return deliv, nil
}

// QueueSelection enfile une sélection d'articles choisis manuellement par l'utilisateur.
func (p *Pipeline) QueueSelection(ctx context.Context, title string, articleIDs []int64, send bool) (*store.Delivery, error) {
	if len(articleIDs) == 0 {
		return nil, fmt.Errorf("aucun article sélectionné")
	}
	if title == "" {
		st, _ := p.store.GetSettings(ctx)
		title = fmt.Sprintf("Sélection Kindle — %s", time.Now().In(st.Location()).Format("02/01/2006 15:04"))
	}

	deliv := &store.Delivery{
		Kind:       store.KindSelection,
		Title:      title,
		Status:     store.StatusQueued,
		Send:       send,
		ArticleIDs: articleIDs,
	}

	if err := p.store.CreateDelivery(ctx, deliv); err != nil {
		return nil, err
	}

	p.Enqueue(deliv.ID)
	return deliv, nil
}

// QueueCollection enfile les articles d'une collection.
func (p *Pipeline) QueueCollection(ctx context.Context, collectionID int64, send bool) (*store.Delivery, error) {
	col, err := p.store.GetCollection(ctx, collectionID)
	if err != nil {
		return nil, err
	}

	ids, err := p.store.CollectionArticleIDs(ctx, collectionID)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("la collection %q est vide", col.Name)
	}

	deliv := &store.Delivery{
		Kind:       store.KindCollection,
		Title:      col.Name,
		Status:     store.StatusQueued,
		Send:       send,
		ArticleIDs: ids,
	}

	if err := p.store.CreateDelivery(ctx, deliv); err != nil {
		return nil, err
	}

	p.Enqueue(deliv.ID)
	return deliv, nil
}

// QueueResend réenvoie une livraison existante, avec option de regénération.
func (p *Pipeline) QueueResend(ctx context.Context, originalID int64, regenerate bool) (*store.Delivery, error) {
	orig, err := p.store.GetDelivery(ctx, originalID)
	if err != nil {
		return nil, err
	}

	if !regenerate && orig.EpubPath != "" {
		if _, err := os.Stat(orig.EpubPath); err == nil {
			// Le fichier existe, on crée une copie livraison immédiate
			deliv := &store.Delivery{
				Kind:       store.KindResend,
				Title:      orig.Title + " (Renvoi)",
				Status:     store.StatusQueued,
				Send:       true,
				ParentID:   &orig.ID,
				EpubPath:   orig.EpubPath,
				ArticleIDs: orig.ArticleIDs,
			}
			if err := p.store.CreateDelivery(ctx, deliv); err != nil {
				return nil, err
			}
			p.Enqueue(deliv.ID)
			return deliv, nil
		}
	}

	// Regénération complète nécessaire
	deliv := &store.Delivery{
		Kind:       store.KindResend,
		Title:      orig.Title + " (Regénéré)",
		Status:     store.StatusQueued,
		Send:       true,
		ParentID:   &orig.ID,
		ArticleIDs: orig.ArticleIDs,
	}
	if err := p.store.CreateDelivery(ctx, deliv); err != nil {
		return nil, err
	}

	p.Enqueue(deliv.ID)
	return deliv, nil
}

func (p *Pipeline) process(ctx context.Context, id int64) {
	deliv, err := p.store.GetDelivery(ctx, id)
	if err != nil {
		slog.Error("livraison introuvable pour traitement", "id", id, "err", err)
		return
	}

	st, err := p.store.GetSettings(ctx)
	if err != nil {
		_ = p.store.UpdateDeliveryStatus(ctx, id, store.StatusFailed, fmt.Sprintf("lecture réglages: %v", err))
		return
	}

	// Si aucun article
	if len(deliv.ArticleIDs) == 0 {
		_ = p.store.UpdateDeliveryStatus(ctx, id, store.StatusEmpty, "aucun article à inclure")
		return
	}

	epubPath := deliv.EpubPath
	var epubSize int64

	// Si pas encore d'EPUB ou regénération demandée
	if epubPath == "" || !fileExists(epubPath) {
		_ = p.store.UpdateDeliveryStatus(ctx, id, store.StatusBuilding, "")

		// Récupération des articles complets
		rawArticles, err := p.store.GetArticles(ctx, deliv.ArticleIDs)
		if err != nil {
			_ = p.store.UpdateDeliveryStatus(ctx, id, store.StatusFailed, fmt.Sprintf("chargement articles: %v", err))
			return
		}
		if len(rawArticles) == 0 {
			_ = p.store.UpdateDeliveryStatus(ctx, id, store.StatusEmpty, "articles supprimés ou introuvables")
			return
		}

		// Préparation des articles pour le livre
		var bookArticles []ebook.Article
		for _, a := range rawArticles {
			contentHTML := a.ContentHTML
			if a.FullContentHTML != "" {
				contentHTML = a.FullContentHTML
			}

			// Si le flux demande le plein texte par défaut et que nous ne l'avons pas encore
			feed, fErr := p.store.GetFeed(ctx, a.FeedID)
			if fErr == nil && feed.FullText && a.FullContentHTML == "" && a.URL != "" {
				if full, err := extract.FullText(ctx, p.httpClient, a.URL); err == nil && full != "" {
					contentHTML = full
					_ = p.store.SetFullContent(ctx, a.ID, full, extract.PlainText(full))
				}
			}

			bookArticles = append(bookArticles, ebook.Article{
				Title:       a.Title,
				FeedTitle:   a.FeedTitle,
				Author:      a.Author,
				URL:         a.URL,
				PublishedAt: a.PublishedAt,
				HTML:        contentHTML,
			})
		}

		// Options de génération
		opts := ebook.Options{
			Title:         deliv.Title,
			Lang:          st.Language,
			GroupByFeed:   deliv.Kind == store.KindDigest,
			Images:        st.ImagesEnabled,
			Grayscale:     st.ImagesGrayscale,
			ImageMaxWidth: st.ImageMaxWidth,
			Location:      st.Location(),
			HTTPClient:    p.httpClient,
		}

		epubBytes, stats, err := ebook.Build(ctx, bookArticles, opts)
		if err != nil {
			_ = p.store.UpdateDeliveryStatus(ctx, id, store.StatusFailed, fmt.Sprintf("création epub: %v", err))
			return
		}

		// Sauvegarde du fichier EPUB
		epubFilename := fmt.Sprintf("delivery_%d_%d.epub", id, time.Now().Unix())
		targetPath := filepath.Join(p.dataDir, "epub", epubFilename)
		if err := os.WriteFile(targetPath, epubBytes, 0o644); err != nil {
			_ = p.store.UpdateDeliveryStatus(ctx, id, store.StatusFailed, fmt.Sprintf("enregistrement fichier: %v", err))
			return
		}

		epubPath = targetPath
		epubSize = stats.Size
		_ = p.store.SetDeliveryResult(ctx, id, epubPath, epubSize, stats.Articles, st.KindleEmail)
	} else {
		// Le fichier existait déjà
		if fi, err := os.Stat(epubPath); err == nil {
			epubSize = fi.Size()
		}
	}

	// Envoi par email si demandé
	if deliv.Send {
		_ = p.store.UpdateDeliveryStatus(ctx, id, store.StatusSending, "")

		if err := p.mailer.SendEpub(ctx, st, deliv.Title, epubPath); err != nil {
			slog.Error("échec envoi mail livraison", "id", id, "err", err)
			_ = p.store.UpdateDeliveryStatus(ctx, id, store.StatusFailed, fmt.Sprintf("envoi e-mail: %v", err))
			return
		}

		// Succès de l'envoi : on estampille les articles comme envoyés
		_ = p.store.MarkSent(ctx, deliv.ArticleIDs, time.Now())
		_ = p.store.UpdateDeliveryStatus(ctx, id, store.StatusSent, "")
		slog.Info("livraison envoyée avec succès au Kindle", "id", id, "dest", st.KindleEmail, "size", epubSize)
	} else {
		// Pas d'envoi demandé, prêt pour téléchargement direct
		_ = p.store.UpdateDeliveryStatus(ctx, id, store.StatusReady, "")
		slog.Info("livraison générée (prête au téléchargement)", "id", id, "size", epubSize)
	}
}

func fileExists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}
