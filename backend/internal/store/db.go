// Package store implements persistence on top of SQLite.
package store

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite" // pure-Go SQLite driver (with FTS5)
)

// Store wraps the database handle.
type Store struct {
	db     *sql.DB
	secret []byte
}

// Open opens (and migrates) the SQLite database located in dataDir.
func Open(dataDir string, secret []byte) (*Store, error) {
	path := filepath.Join(dataDir, "rss2kindle.db")
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=synchronous(NORMAL)", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	// SQLite allows a single writer; a single connection keeps things simple and safe.
	db.SetMaxOpenConns(1)
	s := &Store{db: db, secret: secret}
	if err := s.migrate(context.Background()); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

// OpenMemory opens an in-memory database (used by tests).
func OpenMemory(secret []byte) (*Store, error) {
	db, err := sql.Open("sqlite", "file::memory:?_pragma=foreign_keys(1)")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	s := &Store{db: db, secret: secret}
	if err := s.migrate(context.Background()); err != nil {
		return nil, err
	}
	return s, nil
}

// Close closes the database.
func (s *Store) Close() error { return s.db.Close() }

var migrations = []string{
	// v1: initial schema
	`
CREATE TABLE categories (
	id       INTEGER PRIMARY KEY,
	name     TEXT NOT NULL UNIQUE,
	position INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE feeds (
	id                INTEGER PRIMARY KEY,
	category_id       INTEGER REFERENCES categories(id) ON DELETE SET NULL,
	url               TEXT NOT NULL UNIQUE,
	title             TEXT NOT NULL DEFAULT '',
	site_url          TEXT NOT NULL DEFAULT '',
	enabled           INTEGER NOT NULL DEFAULT 1,
	full_text         INTEGER NOT NULL DEFAULT 0,
	include_in_digest INTEGER NOT NULL DEFAULT 1,
	max_articles      INTEGER NOT NULL DEFAULT 20,
	etag              TEXT NOT NULL DEFAULT '',
	last_modified     TEXT NOT NULL DEFAULT '',
	last_fetched_at   INTEGER,
	last_error        TEXT NOT NULL DEFAULT '',
	created_at        INTEGER NOT NULL
);

CREATE TABLE articles (
	id                INTEGER PRIMARY KEY,
	feed_id           INTEGER NOT NULL REFERENCES feeds(id) ON DELETE CASCADE,
	guid_hash         TEXT NOT NULL,
	url               TEXT NOT NULL DEFAULT '',
	title             TEXT NOT NULL DEFAULT '',
	author            TEXT NOT NULL DEFAULT '',
	summary           TEXT NOT NULL DEFAULT '',
	content_html      TEXT NOT NULL DEFAULT '',
	full_content_html TEXT NOT NULL DEFAULT '',
	image_url         TEXT NOT NULL DEFAULT '',
	published_at      INTEGER NOT NULL,
	fetched_at        INTEGER NOT NULL,
	is_read           INTEGER NOT NULL DEFAULT 0,
	is_starred        INTEGER NOT NULL DEFAULT 0,
	starred_at        INTEGER,
	sent_at           INTEGER,
	UNIQUE(feed_id, guid_hash)
);
CREATE INDEX idx_articles_published ON articles(published_at DESC, id DESC);
CREATE INDEX idx_articles_feed ON articles(feed_id, published_at DESC);
CREATE INDEX idx_articles_starred ON articles(is_starred) WHERE is_starred = 1;
CREATE INDEX idx_articles_unread ON articles(is_read) WHERE is_read = 0;
CREATE INDEX idx_articles_unsent ON articles(sent_at) WHERE sent_at IS NULL;

CREATE VIRTUAL TABLE articles_fts USING fts5(title, body, tokenize='unicode61 remove_diacritics 2');
CREATE TRIGGER articles_fts_delete AFTER DELETE ON articles BEGIN
	DELETE FROM articles_fts WHERE rowid = old.id;
END;

CREATE TABLE collections (
	id         INTEGER PRIMARY KEY,
	name       TEXT NOT NULL,
	created_at INTEGER NOT NULL
);

CREATE TABLE collection_items (
	collection_id INTEGER NOT NULL REFERENCES collections(id) ON DELETE CASCADE,
	article_id    INTEGER NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
	position      INTEGER NOT NULL DEFAULT 0,
	added_at      INTEGER NOT NULL,
	PRIMARY KEY (collection_id, article_id)
);
CREATE INDEX idx_collection_items_article ON collection_items(article_id);

CREATE TABLE deliveries (
	id            INTEGER PRIMARY KEY,
	kind          TEXT NOT NULL,
	title         TEXT NOT NULL,
	status        TEXT NOT NULL,
	send          INTEGER NOT NULL DEFAULT 1,
	article_count INTEGER NOT NULL DEFAULT 0,
	size_bytes    INTEGER NOT NULL DEFAULT 0,
	epub_path     TEXT NOT NULL DEFAULT '',
	error         TEXT NOT NULL DEFAULT '',
	recipient     TEXT NOT NULL DEFAULT '',
	parent_id     INTEGER REFERENCES deliveries(id) ON DELETE SET NULL,
	created_at    INTEGER NOT NULL,
	finished_at   INTEGER
);
CREATE INDEX idx_deliveries_created ON deliveries(created_at DESC);

CREATE TABLE delivery_articles (
	delivery_id INTEGER NOT NULL REFERENCES deliveries(id) ON DELETE CASCADE,
	article_id  INTEGER NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
	position    INTEGER NOT NULL DEFAULT 0,
	PRIMARY KEY (delivery_id, article_id)
);

CREATE TABLE settings (
	key   TEXT PRIMARY KEY,
	value TEXT NOT NULL
);
`,
}

func (s *Store) migrate(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_version (version INTEGER NOT NULL)`); err != nil {
		return err
	}
	var version int
	err := s.db.QueryRowContext(ctx, `SELECT version FROM schema_version LIMIT 1`).Scan(&version)
	if err == sql.ErrNoRows {
		if _, err := s.db.ExecContext(ctx, `INSERT INTO schema_version(version) VALUES (0)`); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	for i := version; i < len(migrations); i++ {
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, migrations[i]); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %d: %w", i+1, err)
		}
		if _, err := tx.ExecContext(ctx, `UPDATE schema_version SET version = ?`, i+1); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

// --- time helpers: timestamps are stored as unix seconds ---

func unix(t time.Time) int64 { return t.UTC().Unix() }

func fromUnix(v int64) time.Time { return time.Unix(v, 0).UTC() }

func nullUnix(n sql.NullInt64) *time.Time {
	if !n.Valid {
		return nil
	}
	t := fromUnix(n.Int64)
	return &t
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}
