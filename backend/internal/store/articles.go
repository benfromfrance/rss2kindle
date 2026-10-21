package store

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Article is a stored feed entry.
type Article struct {
	ID              int64      `json:"id"`
	FeedID          int64      `json:"feed_id"`
	FeedTitle       string     `json:"feed_title"`
	GUIDHash        string     `json:"-"`
	URL             string     `json:"url"`
	Title           string     `json:"title"`
	Author          string     `json:"author"`
	Summary         string     `json:"summary"`
	ContentHTML     string     `json:"content_html,omitempty"`
	FullContentHTML string     `json:"full_content_html,omitempty"`
	ImageURL        string     `json:"image_url"`
	PublishedAt     time.Time  `json:"published_at"`
	FetchedAt       time.Time  `json:"fetched_at"`
	IsRead          bool       `json:"is_read"`
	IsStarred       bool       `json:"is_starred"`
	SentAt          *time.Time `json:"sent_at"`
	HasFullContent  bool       `json:"has_full_content"`
}

// NewArticle is the input for InsertArticles.
type NewArticle struct {
	GUIDHash    string
	URL         string
	Title       string
	Author      string
	Summary     string
	ContentHTML string
	PlainText   string // used for full-text search
	ImageURL    string
	PublishedAt time.Time
}

// InsertArticles stores new articles for a feed, ignoring duplicates. Returns the number inserted.
func (s *Store) InsertArticles(ctx context.Context, feedID int64, items []NewArticle) (int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	now := unix(time.Now())
	inserted := 0
	for _, it := range items {
		res, err := tx.ExecContext(ctx, `
			INSERT INTO articles(feed_id, guid_hash, url, title, author, summary, content_html, image_url, published_at, fetched_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(feed_id, guid_hash) DO NOTHING`,
			feedID, it.GUIDHash, it.URL, it.Title, it.Author, it.Summary, it.ContentHTML, it.ImageURL, unix(it.PublishedAt), now)
		if err != nil {
			return inserted, err
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			continue
		}
		id, _ := res.LastInsertId()
		if _, err := tx.ExecContext(ctx, `INSERT INTO articles_fts(rowid, title, body) VALUES (?, ?, ?)`, id, it.Title, it.PlainText); err != nil {
			return inserted, err
		}
		inserted++
	}
	return inserted, tx.Commit()
}

// ArticleFilter describes a list query.
type ArticleFilter struct {
	View         string // all | unread | today | starred | unsent
	FeedID       int64
	CategoryID   int64
	CollectionID int64
	Query        string
	Cursor       string
	Limit        int
	TZ           *time.Location
}

// ArticlePage is a page of results.
type ArticlePage struct {
	Items      []Article `json:"items"`
	NextCursor string    `json:"next_cursor"`
}

const articleListCols = `a.id, a.feed_id, f.title, a.url, a.title, a.author, a.summary, a.image_url,
	a.published_at, a.fetched_at, a.is_read, a.is_starred, a.sent_at, (a.full_content_html <> '')`

func scanArticle(sc interface{ Scan(...any) error }, extra ...any) (*Article, error) {
	var a Article
	var pub, fetched int64
	var sent sql.NullInt64
	dest := []any{&a.ID, &a.FeedID, &a.FeedTitle, &a.URL, &a.Title, &a.Author, &a.Summary, &a.ImageURL,
		&pub, &fetched, &a.IsRead, &a.IsStarred, &sent, &a.HasFullContent}
	dest = append(dest, extra...)
	if err := sc.Scan(dest...); err != nil {
		return nil, err
	}
	a.PublishedAt, a.FetchedAt, a.SentAt = fromUnix(pub), fromUnix(fetched), nullUnix(sent)
	return &a, nil
}

// ListArticles returns a page of articles matching the filter (newest first, cursor-paginated).
func (s *Store) ListArticles(ctx context.Context, fl ArticleFilter) (*ArticlePage, error) {
	if fl.Limit <= 0 || fl.Limit > 200 {
		fl.Limit = 50
	}
	var where []string
	var args []any
	join := ""
	order := "a.published_at DESC, a.id DESC"

	switch fl.View {
	case "unread":
		where = append(where, "a.is_read = 0")
	case "starred":
		where = append(where, "a.is_starred = 1")
	case "unsent":
		where = append(where, "a.sent_at IS NULL")
	case "today":
		loc := fl.TZ
		if loc == nil {
			loc = time.UTC
		}
		now := time.Now().In(loc)
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
		where = append(where, "a.published_at >= ?")
		args = append(args, unix(start))
	}
	if fl.FeedID > 0 {
		where = append(where, "a.feed_id = ?")
		args = append(args, fl.FeedID)
	}
	if fl.CategoryID > 0 {
		where = append(where, "f.category_id = ?")
		args = append(args, fl.CategoryID)
	}
	if fl.CollectionID > 0 {
		join += " JOIN collection_items ci ON ci.article_id = a.id AND ci.collection_id = ?"
		args = append([]any{fl.CollectionID}, args...)
		order = "ci.position, ci.added_at"
	}
	if q := strings.TrimSpace(fl.Query); q != "" {
		where = append(where, "a.id IN (SELECT rowid FROM articles_fts WHERE articles_fts MATCH ?)")
		args = append(args, ftsQuery(q))
	}
	usesCursor := fl.CollectionID == 0
	if usesCursor && fl.Cursor != "" {
		ts, id, err := decodeCursor(fl.Cursor)
		if err == nil {
			where = append(where, "(a.published_at < ? OR (a.published_at = ? AND a.id < ?))")
			args = append(args, ts, ts, id)
		}
	}
	q := `SELECT ` + articleListCols + ` FROM articles a JOIN feeds f ON f.id = a.feed_id` + join
	if len(where) > 0 {
		q += " WHERE " + strings.Join(where, " AND ")
	}
	limit := fl.Limit
	if !usesCursor {
		limit = 1000
	}
	q += fmt.Sprintf(" ORDER BY %s LIMIT %d", order, limit+1)

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	page := &ArticlePage{Items: []Article{}}
	for rows.Next() {
		a, err := scanArticle(rows)
		if err != nil {
			return nil, err
		}
		page.Items = append(page.Items, *a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(page.Items) > limit {
		page.Items = page.Items[:limit]
		last := page.Items[len(page.Items)-1]
		page.NextCursor = encodeCursor(unix(last.PublishedAt), last.ID)
	}
	return page, nil
}

// ftsQuery turns free text into a safe FTS5 prefix query.
func ftsQuery(q string) string {
	var parts []string
	for _, w := range strings.Fields(q) {
		w = strings.Map(func(r rune) rune {
			if r == '"' {
				return -1
			}
			return r
		}, w)
		if w != "" {
			parts = append(parts, `"`+w+`"*`)
		}
	}
	return strings.Join(parts, " ")
}

func encodeCursor(ts, id int64) string {
	return base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf("%d:%d", ts, id)))
}

func decodeCursor(c string) (int64, int64, error) {
	raw, err := base64.RawURLEncoding.DecodeString(c)
	if err != nil {
		return 0, 0, err
	}
	parts := strings.SplitN(string(raw), ":", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("bad cursor")
	}
	ts, err1 := strconv.ParseInt(parts[0], 10, 64)
	id, err2 := strconv.ParseInt(parts[1], 10, 64)
	if err1 != nil || err2 != nil {
		return 0, 0, fmt.Errorf("bad cursor")
	}
	return ts, id, nil
}

// GetArticle returns a full article including content.
func (s *Store) GetArticle(ctx context.Context, id int64) (*Article, error) {
	var content, full string
	a, err := scanArticle(s.db.QueryRowContext(ctx, `SELECT `+articleListCols+`, a.content_html, a.full_content_html
		FROM articles a JOIN feeds f ON f.id = a.feed_id WHERE a.id = ?`, id), &content, &full)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	a.ContentHTML, a.FullContentHTML = content, full
	return a, nil
}

// GetArticles returns full articles in the order of ids (missing ids are skipped).
func (s *Store) GetArticles(ctx context.Context, ids []int64) ([]Article, error) {
	out := make([]Article, 0, len(ids))
	for _, id := range ids {
		a, err := s.GetArticle(ctx, id)
		if err == ErrNotFound {
			continue
		}
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, nil
}

// ArticlePatch holds optional flag updates.
type ArticlePatch struct {
	IsRead    *bool `json:"is_read"`
	IsStarred *bool `json:"is_starred"`
}

// PatchArticles applies flag updates to the given articles.
func (s *Store) PatchArticles(ctx context.Context, ids []int64, p ArticlePatch) error {
	if len(ids) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, id := range ids {
		if p.IsRead != nil {
			if _, err := tx.ExecContext(ctx, `UPDATE articles SET is_read = ? WHERE id = ?`, b2i(*p.IsRead), id); err != nil {
				return err
			}
		}
		if p.IsStarred != nil {
			if _, err := tx.ExecContext(ctx, `UPDATE articles SET is_starred = ?, starred_at = CASE WHEN ? = 1 THEN ? ELSE NULL END WHERE id = ?`,
				b2i(*p.IsStarred), b2i(*p.IsStarred), unix(time.Now()), id); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

// MarkAllRead marks every article matching the scope as read.
func (s *Store) MarkAllRead(ctx context.Context, feedID, categoryID int64) (int64, error) {
	q := `UPDATE articles SET is_read = 1 WHERE is_read = 0`
	var args []any
	if feedID > 0 {
		q += ` AND feed_id = ?`
		args = append(args, feedID)
	} else if categoryID > 0 {
		q += ` AND feed_id IN (SELECT id FROM feeds WHERE category_id = ?)`
		args = append(args, categoryID)
	}
	res, err := s.db.ExecContext(ctx, q, args...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// SetFullContent caches extracted full content for an article.
func (s *Store) SetFullContent(ctx context.Context, id int64, html, plain string) error {
	if _, err := s.db.ExecContext(ctx, `UPDATE articles SET full_content_html = ? WHERE id = ?`, html, id); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `UPDATE articles_fts SET body = ? WHERE rowid = ?`, plain, id)
	return err
}

// MarkSent stamps sent_at for the given articles.
func (s *Store) MarkSent(ctx context.Context, ids []int64, at time.Time) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, id := range ids {
		if _, err := tx.ExecContext(ctx, `UPDATE articles SET sent_at = ? WHERE id = ?`, unix(at), id); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// DigestCandidates returns ids of unsent articles for the daily digest.
func (s *Store) DigestCandidates(ctx context.Context, since time.Time, excludeRead bool, max int) ([]int64, error) {
	readFilter := ""
	if excludeRead {
		readFilter = ` AND a.is_read = 0`
	}
	// Group by category / feed so the book reads nicely, respecting each feed's max_articles.
	q := `SELECT id FROM (
		SELECT a.id, c.position AS cpos, f.title AS ftitle, a.published_at, f.max_articles,
			ROW_NUMBER() OVER (PARTITION BY a.feed_id ORDER BY a.published_at DESC) AS rn
		FROM articles a JOIN feeds f ON f.id = a.feed_id
		LEFT JOIN categories c ON c.id = f.category_id
		WHERE a.sent_at IS NULL AND f.include_in_digest = 1 AND f.enabled = 1 AND a.published_at >= ?` + readFilter + `
	) WHERE rn <= max_articles
	ORDER BY COALESCE(cpos, 1000000), ftitle COLLATE NOCASE, published_at DESC
	LIMIT ?`
	rows, err := s.db.QueryContext(ctx, q, unix(since), max)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// Counts summarises unread counts for the sidebar.
type Counts struct {
	Unread     int           `json:"unread"`
	Starred    int           `json:"starred"`
	Unsent     int           `json:"unsent"`
	Today      int           `json:"today"`
	Total      int           `json:"total"`
	ByFeed     map[int64]int `json:"by_feed"`
	ByCategory map[int64]int `json:"by_category"`
}

// GetCounts returns sidebar counters.
func (s *Store) GetCounts(ctx context.Context, loc *time.Location) (*Counts, error) {
	if loc == nil {
		loc = time.UTC
	}
	now := time.Now().In(loc)
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	c := &Counts{ByFeed: map[int64]int{}, ByCategory: map[int64]int{}}
	err := s.db.QueryRowContext(ctx, `SELECT
		COALESCE(SUM(is_read = 0), 0), COALESCE(SUM(is_starred = 1), 0), COALESCE(SUM(sent_at IS NULL), 0),
		COALESCE(SUM(published_at >= ?), 0), COUNT(*) FROM articles`, unix(start)).
		Scan(&c.Unread, &c.Starred, &c.Unsent, &c.Today, &c.Total)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT a.feed_id, f.category_id, COUNT(*) FROM articles a JOIN feeds f ON f.id = a.feed_id
		WHERE a.is_read = 0 GROUP BY a.feed_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var fid int64
		var cid sql.NullInt64
		var n int
		if err := rows.Scan(&fid, &cid, &n); err != nil {
			return nil, err
		}
		c.ByFeed[fid] = n
		if cid.Valid {
			c.ByCategory[cid.Int64] += n
		}
	}
	return c, rows.Err()
}

// PurgeOldArticles deletes read, unstarred, uncollected articles older than the cutoff.
func (s *Store) PurgeOldArticles(ctx context.Context, before time.Time) (int64, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM articles
		WHERE is_starred = 0 AND fetched_at < ? AND (is_read = 1 OR sent_at IS NOT NULL)
		AND id NOT IN (SELECT article_id FROM collection_items)`, unix(before))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// DailyStat is a per-day aggregate used by the dashboard.
type DailyStat struct {
	Day     string `json:"day"`
	Fetched int    `json:"fetched"`
	Sent    int    `json:"sent"`
}

// Stats returns article fetch/sent counts per day for the last n days.
func (s *Store) Stats(ctx context.Context, days int, loc *time.Location) ([]DailyStat, error) {
	if loc == nil {
		loc = time.UTC
	}
	now := time.Now().In(loc)
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, -(days - 1))
	out := make([]DailyStat, days)
	idx := map[string]int{}
	for i := 0; i < days; i++ {
		d := start.AddDate(0, 0, i).Format("2006-01-02")
		out[i] = DailyStat{Day: d}
		idx[d] = i
	}
	rows, err := s.db.QueryContext(ctx, `SELECT fetched_at, sent_at FROM articles WHERE fetched_at >= ? OR sent_at >= ?`, unix(start), unix(start))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var f int64
		var snt sql.NullInt64
		if err := rows.Scan(&f, &snt); err != nil {
			return nil, err
		}
		if i, ok := idx[fromUnix(f).In(loc).Format("2006-01-02")]; ok {
			out[i].Fetched++
		}
		if snt.Valid {
			if i, ok := idx[fromUnix(snt.Int64).In(loc).Format("2006-01-02")]; ok {
				out[i].Sent++
			}
		}
	}
	return out, rows.Err()
}
