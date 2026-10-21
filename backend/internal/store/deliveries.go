package store

import (
	"context"
	"database/sql"
	"time"
)

// Delivery kinds.
const (
	KindDigest     = "digest"
	KindSelection  = "selection"
	KindCollection = "collection"
	KindStarred    = "starred"
	KindResend     = "resend"
)

// Delivery statuses.
const (
	StatusQueued   = "queued"
	StatusBuilding = "building"
	StatusSending  = "sending"
	StatusSent     = "sent"
	StatusReady    = "ready" // built but not e-mailed (download only)
	StatusEmpty    = "empty"
	StatusFailed   = "failed"
)

// Delivery is a generated (and possibly e-mailed) EPUB.
type Delivery struct {
	ID           int64      `json:"id"`
	Kind         string     `json:"kind"`
	Title        string     `json:"title"`
	Status       string     `json:"status"`
	Send         bool       `json:"send"`
	ArticleCount int        `json:"article_count"`
	SizeBytes    int64      `json:"size_bytes"`
	EpubPath     string     `json:"-"`
	HasEpub      bool       `json:"has_epub"`
	Error        string     `json:"error"`
	Recipient    string     `json:"recipient"`
	ParentID     *int64     `json:"parent_id"`
	CreatedAt    time.Time  `json:"created_at"`
	FinishedAt   *time.Time `json:"finished_at"`
	ArticleIDs   []int64    `json:"article_ids,omitempty"`
}

const deliveryCols = `id, kind, title, status, send, article_count, size_bytes, epub_path, error, recipient, parent_id, created_at, finished_at`

func scanDelivery(sc interface{ Scan(...any) error }) (*Delivery, error) {
	var d Delivery
	var parent, finished sql.NullInt64
	var created int64
	if err := sc.Scan(&d.ID, &d.Kind, &d.Title, &d.Status, &d.Send, &d.ArticleCount, &d.SizeBytes, &d.EpubPath,
		&d.Error, &d.Recipient, &parent, &created, &finished); err != nil {
		return nil, err
	}
	if parent.Valid {
		d.ParentID = &parent.Int64
	}
	d.CreatedAt = fromUnix(created)
	d.FinishedAt = nullUnix(finished)
	d.HasEpub = d.EpubPath != ""
	return &d, nil
}

// CreateDelivery inserts a queued delivery with its ordered article list.
func (s *Store) CreateDelivery(ctx context.Context, d *Delivery) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	d.CreatedAt = time.Now().UTC()
	if d.Status == "" {
		d.Status = StatusQueued
	}
	d.ArticleCount = len(d.ArticleIDs)
	res, err := tx.ExecContext(ctx, `INSERT INTO deliveries(kind, title, status, send, article_count, parent_id, created_at, epub_path)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, d.Kind, d.Title, d.Status, b2i(d.Send), d.ArticleCount, d.ParentID, unix(d.CreatedAt), d.EpubPath)
	if err != nil {
		return err
	}
	d.ID, _ = res.LastInsertId()
	for i, aid := range d.ArticleIDs {
		if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO delivery_articles(delivery_id, article_id, position) VALUES (?, ?, ?)`, d.ID, aid, i); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// UpdateDeliveryStatus sets the status (and optionally error) of a delivery.
func (s *Store) UpdateDeliveryStatus(ctx context.Context, id int64, status, errMsg string) error {
	q := `UPDATE deliveries SET status = ?, error = ?`
	args := []any{status, errMsg}
	switch status {
	case StatusSent, StatusReady, StatusEmpty, StatusFailed:
		q += `, finished_at = ?`
		args = append(args, unix(time.Now()))
	}
	q += ` WHERE id = ?`
	args = append(args, id)
	_, err := s.db.ExecContext(ctx, q, args...)
	return err
}

// SetDeliveryResult records the built file.
func (s *Store) SetDeliveryResult(ctx context.Context, id int64, epubPath string, size int64, articleCount int, recipient string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE deliveries SET epub_path = ?, size_bytes = ?, article_count = ?, recipient = ? WHERE id = ?`,
		epubPath, size, articleCount, recipient, id)
	return err
}

// GetDelivery returns a delivery with its article ids.
func (s *Store) GetDelivery(ctx context.Context, id int64) (*Delivery, error) {
	d, err := scanDelivery(s.db.QueryRowContext(ctx, `SELECT `+deliveryCols+` FROM deliveries WHERE id = ?`, id))
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	d.ArticleIDs, err = s.queryIDs(ctx, `SELECT article_id FROM delivery_articles WHERE delivery_id = ? ORDER BY position`, id)
	return d, err
}

// ListDeliveries returns the most recent deliveries.
func (s *Store) ListDeliveries(ctx context.Context, limit int) ([]Delivery, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `SELECT `+deliveryCols+` FROM deliveries ORDER BY created_at DESC, id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Delivery{}
	for rows.Next() {
		d, err := scanDelivery(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *d)
	}
	return out, rows.Err()
}

// DeleteDelivery removes a delivery record.
func (s *Store) DeleteDelivery(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM deliveries WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return mustAffect(res)
}

// ExpiredEpubs returns deliveries whose EPUB files are older than the cutoff.
func (s *Store) ExpiredEpubs(ctx context.Context, before time.Time) ([]Delivery, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+deliveryCols+` FROM deliveries WHERE epub_path <> '' AND created_at < ?`, unix(before))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Delivery{}
	for rows.Next() {
		d, err := scanDelivery(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *d)
	}
	return out, rows.Err()
}

// ClearEpubPath forgets the file of a delivery.
func (s *Store) ClearEpubPath(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `UPDATE deliveries SET epub_path = '' WHERE id = ?`, id)
	return err
}

// FailStaleDeliveries marks deliveries left in-flight by a previous process as failed.
func (s *Store) FailStaleDeliveries(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `UPDATE deliveries SET status = ?, error = 'interrompu (redémarrage du serveur)', finished_at = ?
		WHERE status IN (?, ?, ?)`, StatusFailed, unix(time.Now()), StatusQueued, StatusBuilding, StatusSending)
	return err
}

// LastDelivery returns the most recent delivery of a kind, if any.
func (s *Store) LastDelivery(ctx context.Context, kind string) (*Delivery, error) {
	q := `SELECT ` + deliveryCols + ` FROM deliveries`
	var args []any
	if kind != "" {
		q += ` WHERE kind = ?`
		args = append(args, kind)
	}
	q += ` ORDER BY created_at DESC, id DESC LIMIT 1`
	d, err := scanDelivery(s.db.QueryRowContext(ctx, q, args...))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return d, err
}
