package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

// Collection is a named, ordered list of articles.
type Collection struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	CreatedAt    time.Time `json:"created_at"`
	ArticleCount int       `json:"article_count"`
}

// ListCollections returns collections with their item counts.
func (s *Store) ListCollections(ctx context.Context) ([]Collection, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT c.id, c.name, c.created_at,
		(SELECT COUNT(*) FROM collection_items ci WHERE ci.collection_id = c.id)
		FROM collections c ORDER BY c.name COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Collection{}
	for rows.Next() {
		var c Collection
		var created int64
		if err := rows.Scan(&c.ID, &c.Name, &created, &c.ArticleCount); err != nil {
			return nil, err
		}
		c.CreatedAt = fromUnix(created)
		out = append(out, c)
	}
	return out, rows.Err()
}

// GetCollection returns a single collection.
func (s *Store) GetCollection(ctx context.Context, id int64) (*Collection, error) {
	var c Collection
	var created int64
	err := s.db.QueryRowContext(ctx, `SELECT c.id, c.name, c.created_at,
		(SELECT COUNT(*) FROM collection_items ci WHERE ci.collection_id = c.id)
		FROM collections c WHERE c.id = ?`, id).Scan(&c.ID, &c.Name, &created, &c.ArticleCount)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	c.CreatedAt = fromUnix(created)
	return &c, nil
}

// CreateCollection creates a collection.
func (s *Store) CreateCollection(ctx context.Context, name string) (*Collection, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("collection name is required")
	}
	now := time.Now().UTC()
	res, err := s.db.ExecContext(ctx, `INSERT INTO collections(name, created_at) VALUES (?, ?)`, name, unix(now))
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &Collection{ID: id, Name: name, CreatedAt: now}, nil
}

// RenameCollection renames a collection.
func (s *Store) RenameCollection(ctx context.Context, id int64, name string) error {
	res, err := s.db.ExecContext(ctx, `UPDATE collections SET name = ? WHERE id = ?`, strings.TrimSpace(name), id)
	if err != nil {
		return err
	}
	return mustAffect(res)
}

// DeleteCollection removes a collection (articles are kept).
func (s *Store) DeleteCollection(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM collections WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return mustAffect(res)
}

// AddToCollection appends articles to a collection (existing ones are kept in place).
func (s *Store) AddToCollection(ctx context.Context, id int64, articleIDs []int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var pos int
	_ = tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(position), -1) + 1 FROM collection_items WHERE collection_id = ?`, id).Scan(&pos)
	now := unix(time.Now())
	for _, aid := range articleIDs {
		res, err := tx.ExecContext(ctx, `INSERT INTO collection_items(collection_id, article_id, position, added_at) VALUES (?, ?, ?, ?)
			ON CONFLICT DO NOTHING`, id, aid, pos, now)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n > 0 {
			pos++
		}
	}
	return tx.Commit()
}

// RemoveFromCollection removes articles from a collection.
func (s *Store) RemoveFromCollection(ctx context.Context, id int64, articleIDs []int64) error {
	for _, aid := range articleIDs {
		if _, err := s.db.ExecContext(ctx, `DELETE FROM collection_items WHERE collection_id = ? AND article_id = ?`, id, aid); err != nil {
			return err
		}
	}
	return nil
}

// ReorderCollection sets the item order to match articleIDs.
func (s *Store) ReorderCollection(ctx context.Context, id int64, articleIDs []int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for i, aid := range articleIDs {
		if _, err := tx.ExecContext(ctx, `UPDATE collection_items SET position = ? WHERE collection_id = ? AND article_id = ?`, i, id, aid); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// CollectionArticleIDs returns ordered article ids of a collection.
func (s *Store) CollectionArticleIDs(ctx context.Context, id int64) ([]int64, error) {
	return s.queryIDs(ctx, `SELECT article_id FROM collection_items WHERE collection_id = ? ORDER BY position, added_at`, id)
}

// StarredArticleIDs returns starred article ids (optionally only unsent), oldest star first.
func (s *Store) StarredArticleIDs(ctx context.Context, onlyUnsent bool) ([]int64, error) {
	q := `SELECT id FROM articles WHERE is_starred = 1`
	if onlyUnsent {
		q += ` AND sent_at IS NULL`
	}
	q += ` ORDER BY starred_at, id`
	return s.queryIDs(ctx, q)
}

// ArticleCollections returns ids of collections containing the article.
func (s *Store) ArticleCollections(ctx context.Context, articleID int64) ([]int64, error) {
	return s.queryIDs(ctx, `SELECT collection_id FROM collection_items WHERE article_id = ?`, articleID)
}

func (s *Store) queryIDs(ctx context.Context, q string, args ...any) ([]int64, error) {
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
