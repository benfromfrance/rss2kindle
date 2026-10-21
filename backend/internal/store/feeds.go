package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

// ErrNotFound is returned when a row does not exist.
var ErrNotFound = errors.New("not found")

// Category groups feeds.
type Category struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Position int    `json:"position"`
}

// Feed is an RSS/Atom subscription.
type Feed struct {
	ID              int64      `json:"id"`
	CategoryID      *int64     `json:"category_id"`
	URL             string     `json:"url"`
	Title           string     `json:"title"`
	SiteURL         string     `json:"site_url"`
	Enabled         bool       `json:"enabled"`
	FullText        bool       `json:"full_text"`
	IncludeInDigest bool       `json:"include_in_digest"`
	MaxArticles     int        `json:"max_articles"`
	ETag            string     `json:"-"`
	LastModified    string     `json:"-"`
	LastFetchedAt   *time.Time `json:"last_fetched_at"`
	LastError       string     `json:"last_error"`
	CreatedAt       time.Time  `json:"created_at"`
	UnreadCount     int        `json:"unread_count"`
	ArticleCount    int        `json:"article_count"`
}

// ---------- categories ----------

// ListCategories returns all categories ordered by position then name.
func (s *Store) ListCategories(ctx context.Context) ([]Category, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, position FROM categories ORDER BY position, name COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Category{}
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Name, &c.Position); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// CreateCategory inserts a category, or returns the existing one with the same name.
func (s *Store) CreateCategory(ctx context.Context, name string) (*Category, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("category name is required")
	}
	var c Category
	err := s.db.QueryRowContext(ctx, `SELECT id, name, position FROM categories WHERE name = ? COLLATE NOCASE`, name).Scan(&c.ID, &c.Name, &c.Position)
	if err == nil {
		return &c, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}
	var pos int
	_ = s.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(position), -1) + 1 FROM categories`).Scan(&pos)
	res, err := s.db.ExecContext(ctx, `INSERT INTO categories(name, position) VALUES (?, ?)`, name, pos)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &Category{ID: id, Name: name, Position: pos}, nil
}

// UpdateCategory renames / repositions a category.
func (s *Store) UpdateCategory(ctx context.Context, c Category) error {
	res, err := s.db.ExecContext(ctx, `UPDATE categories SET name = ?, position = ? WHERE id = ?`, strings.TrimSpace(c.Name), c.Position, c.ID)
	if err != nil {
		return err
	}
	return mustAffect(res)
}

// DeleteCategory removes a category; its feeds become uncategorized.
func (s *Store) DeleteCategory(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM categories WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return mustAffect(res)
}

// ---------- feeds ----------

const feedCols = `f.id, f.category_id, f.url, f.title, f.site_url, f.enabled, f.full_text, f.include_in_digest,
	f.max_articles, f.etag, f.last_modified, f.last_fetched_at, f.last_error, f.created_at`

func scanFeed(sc interface{ Scan(...any) error }, extra ...any) (*Feed, error) {
	var f Feed
	var cat sql.NullInt64
	var last sql.NullInt64
	var created int64
	dest := []any{&f.ID, &cat, &f.URL, &f.Title, &f.SiteURL, &f.Enabled, &f.FullText, &f.IncludeInDigest,
		&f.MaxArticles, &f.ETag, &f.LastModified, &last, &f.LastError, &created}
	dest = append(dest, extra...)
	if err := sc.Scan(dest...); err != nil {
		return nil, err
	}
	if cat.Valid {
		f.CategoryID = &cat.Int64
	}
	f.LastFetchedAt = nullUnix(last)
	f.CreatedAt = fromUnix(created)
	return &f, nil
}

// ListFeeds returns all feeds with unread/article counts.
func (s *Store) ListFeeds(ctx context.Context) ([]Feed, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+feedCols+`,
			(SELECT COUNT(*) FROM articles a WHERE a.feed_id = f.id AND a.is_read = 0),
			(SELECT COUNT(*) FROM articles a WHERE a.feed_id = f.id)
		FROM feeds f ORDER BY f.title COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Feed{}
	for rows.Next() {
		var unread, total int
		f, err := scanFeed(rows, &unread, &total)
		if err != nil {
			return nil, err
		}
		f.UnreadCount, f.ArticleCount = unread, total
		out = append(out, *f)
	}
	return out, rows.Err()
}

// ListEnabledFeeds returns feeds that should be refreshed.
func (s *Store) ListEnabledFeeds(ctx context.Context) ([]Feed, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+feedCols+` FROM feeds f WHERE f.enabled = 1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Feed{}
	for rows.Next() {
		f, err := scanFeed(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *f)
	}
	return out, rows.Err()
}

// GetFeed returns a single feed.
func (s *Store) GetFeed(ctx context.Context, id int64) (*Feed, error) {
	f, err := scanFeed(s.db.QueryRowContext(ctx, `SELECT `+feedCols+` FROM feeds f WHERE f.id = ?`, id))
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return f, err
}

// CreateFeed inserts a new feed.
func (s *Store) CreateFeed(ctx context.Context, f *Feed) error {
	if f.MaxArticles <= 0 {
		f.MaxArticles = 20
	}
	f.CreatedAt = time.Now().UTC()
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO feeds(category_id, url, title, site_url, enabled, full_text, include_in_digest, max_articles, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		f.CategoryID, f.URL, f.Title, f.SiteURL, b2i(f.Enabled), b2i(f.FullText), b2i(f.IncludeInDigest), f.MaxArticles, unix(f.CreatedAt))
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return errors.New("ce flux existe déjà")
		}
		return err
	}
	f.ID, _ = res.LastInsertId()
	return nil
}

// UpdateFeed updates user-editable fields.
func (s *Store) UpdateFeed(ctx context.Context, f *Feed) error {
	res, err := s.db.ExecContext(ctx, `
		UPDATE feeds SET category_id = ?, url = ?, title = ?, enabled = ?, full_text = ?, include_in_digest = ?, max_articles = ?
		WHERE id = ?`,
		f.CategoryID, f.URL, f.Title, b2i(f.Enabled), b2i(f.FullText), b2i(f.IncludeInDigest), f.MaxArticles, f.ID)
	if err != nil {
		return err
	}
	return mustAffect(res)
}

// UpdateFeedFetchState records the result of a fetch.
func (s *Store) UpdateFeedFetchState(ctx context.Context, id int64, etag, lastModified, fetchErr string, siteURL, title string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE feeds SET etag = ?, last_modified = ?, last_error = ?, last_fetched_at = ?,
			site_url = CASE WHEN ? <> '' THEN ? ELSE site_url END,
			title = CASE WHEN title = '' AND ? <> '' THEN ? ELSE title END
		WHERE id = ?`,
		etag, lastModified, fetchErr, unix(time.Now()), siteURL, siteURL, title, title, id)
	return err
}

// DeleteFeed removes a feed and its articles.
func (s *Store) DeleteFeed(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM feeds WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return mustAffect(res)
}

func mustAffect(res sql.Result) error {
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
