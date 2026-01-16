package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	_ "github.com/lib/pq"
)

const migrationSQL = `
CREATE TABLE IF NOT EXISTS news (
    id            SERIAL PRIMARY KEY,
    nhk_id        VARCHAR(20) UNIQUE NOT NULL,
    title         TEXT NOT NULL,
    url           TEXT NOT NULL,
    published_at  TIMESTAMP,
    fetched_at    TIMESTAMP DEFAULT NOW(),
    created_at    TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS paragraphs (
    id         SERIAL PRIMARY KEY,
    news_id    INTEGER REFERENCES news(id) ON DELETE CASCADE,
    position   INTEGER NOT NULL,
    raw_text   TEXT NOT NULL,
    tokens     JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_news_nhk_id ON news(nhk_id);
CREATE INDEX IF NOT EXISTS idx_news_created_at ON news(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_paragraphs_news_id ON paragraphs(news_id);

CREATE TABLE IF NOT EXISTS scheduler_state (
    key        VARCHAR(50) PRIMARY KEY,
    value      TIMESTAMP NOT NULL,
    updated_at TIMESTAMP DEFAULT NOW()
);
`

type Store struct {
	db *sql.DB
}

func NewWithDB(db *sql.DB) *Store {
	return &Store{db: db}
}

func New(ctx context.Context, databaseURL string) (*Store, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}

	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Migrate(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, migrationSQL)
	return err
}

func (s *Store) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s *Store) NewsExists(ctx context.Context, nhkID string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx,
		"SELECT EXISTS(SELECT 1 FROM news WHERE nhk_id = $1)",
		nhkID,
	).Scan(&exists)
	return exists, err
}

func (s *Store) InsertNews(ctx context.Context, news *News, paragraphs []Paragraph) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var newsID int
	err = tx.QueryRowContext(ctx,
		`INSERT INTO news (nhk_id, title, url, published_at, fetched_at)
		 VALUES ($1, $2, $3, $4, NOW())
		 RETURNING id`,
		news.NHKID, news.Title, news.URL, news.PublishedAt,
	).Scan(&newsID)
	if err != nil {
		return err
	}

	for _, p := range paragraphs {
		tokensJSON, err := json.Marshal(p.Tokens)
		if err != nil {
			return err
		}

		_, err = tx.ExecContext(ctx,
			`INSERT INTO paragraphs (news_id, position, raw_text, tokens)
			 VALUES ($1, $2, $3, $4)`,
			newsID, p.Position, p.RawText, tokensJSON,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *Store) GetNewsList(ctx context.Context, limit, offset int) ([]NewsList, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, nhk_id, title, url, published_at
		 FROM news
		 ORDER BY created_at DESC
		 LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var news []NewsList
	for rows.Next() {
		var n NewsList
		if err := rows.Scan(&n.ID, &n.NHKID, &n.Title, &n.URL, &n.PublishedAt); err != nil {
			return nil, err
		}
		news = append(news, n)
	}

	if news == nil {
		news = []NewsList{}
	}

	return news, rows.Err()
}

func (s *Store) GetNewsByID(ctx context.Context, id int) (*NewsWithParagraphs, error) {
	news := &NewsWithParagraphs{}

	err := s.db.QueryRowContext(ctx,
		`SELECT id, nhk_id, title, url, published_at, fetched_at, created_at
		 FROM news
		 WHERE id = $1`,
		id,
	).Scan(&news.ID, &news.NHKID, &news.Title, &news.URL,
		&news.PublishedAt, &news.FetchedAt, &news.CreatedAt)
	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT id, news_id, position, raw_text, tokens, created_at
		 FROM paragraphs
		 WHERE news_id = $1
		 ORDER BY position`,
		id,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var p Paragraph
		var tokensJSON []byte
		if err := rows.Scan(&p.ID, &p.NewsID, &p.Position, &p.RawText, &tokensJSON, &p.CreatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(tokensJSON, &p.Tokens); err != nil {
			return nil, err
		}
		news.Paragraphs = append(news.Paragraphs, p)
	}

	if news.Paragraphs == nil {
		news.Paragraphs = []Paragraph{}
	}

	return news, rows.Err()
}

func (s *Store) GetLastRun(ctx context.Context) (time.Time, error) {
	var lastRun time.Time
	err := s.db.QueryRowContext(ctx,
		"SELECT value FROM scheduler_state WHERE key = 'last_run'",
	).Scan(&lastRun)
	return lastRun, err
}

func (s *Store) SetLastRun(ctx context.Context, t time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO scheduler_state (key, value, updated_at)
		 VALUES ('last_run', $1, NOW())
		 ON CONFLICT (key) DO UPDATE SET value = $1, updated_at = NOW()`,
		t,
	)
	return err
}
