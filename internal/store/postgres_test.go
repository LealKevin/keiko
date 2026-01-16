package store

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewsExists(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	s := NewWithDB(db)
	ctx := context.Background()

	tests := []struct {
		name      string
		nhkID     string
		setupMock func()
		want      bool
		wantErr   bool
	}{
		{
			name:  "news exists",
			nhkID: "ne123",
			setupMock: func() {
				mock.ExpectQuery("SELECT EXISTS").
					WithArgs("ne123").
					WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
			},
			want:    true,
			wantErr: false,
		},
		{
			name:  "news does not exist",
			nhkID: "ne999",
			setupMock: func() {
				mock.ExpectQuery("SELECT EXISTS").
					WithArgs("ne999").
					WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
			},
			want:    false,
			wantErr: false,
		},
		{
			name:  "database error",
			nhkID: "ne123",
			setupMock: func() {
				mock.ExpectQuery("SELECT EXISTS").
					WithArgs("ne123").
					WillReturnError(sql.ErrConnDone)
			},
			want:    false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			got, err := s.NewsExists(ctx, tt.nhkID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestInsertNews(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	s := NewWithDB(db)
	ctx := context.Background()

	publishedAt := time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)
	news := &News{
		NHKID:       "ne123",
		Title:       "Test News",
		URL:         "http://example.com",
		PublishedAt: &publishedAt,
	}
	paragraphs := []Paragraph{
		{
			Position: 0,
			RawText:  "First paragraph",
			Tokens:   []Token{{Kana: "テスト", Translation: "test"}},
		},
	}

	t.Run("successful insert", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectQuery("INSERT INTO news").
			WithArgs(news.NHKID, news.Title, news.URL, news.PublishedAt).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
		mock.ExpectExec("INSERT INTO paragraphs").
			WithArgs(1, 0, "First paragraph", sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := s.InsertNews(ctx, news, paragraphs)

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("rollback on error", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectQuery("INSERT INTO news").
			WithArgs(news.NHKID, news.Title, news.URL, news.PublishedAt).
			WillReturnError(sql.ErrConnDone)
		mock.ExpectRollback()

		err := s.InsertNews(ctx, news, paragraphs)

		assert.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGetNewsList(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	s := NewWithDB(db)
	ctx := context.Background()

	publishedAt := time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)

	t.Run("returns news list", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "nhk_id", "title", "url", "published_at"}).
			AddRow(1, "ne123", "Test News 1", "http://example.com/1", publishedAt).
			AddRow(2, "ne456", "Test News 2", "http://example.com/2", publishedAt)

		mock.ExpectQuery("SELECT id, nhk_id, title, url, published_at FROM news").
			WithArgs(10, 0).
			WillReturnRows(rows)

		news, err := s.GetNewsList(ctx, 10, 0)

		assert.NoError(t, err)
		assert.Len(t, news, 2)
		assert.Equal(t, "ne123", news[0].NHKID)
		assert.Equal(t, "ne456", news[1].NHKID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns empty list", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "nhk_id", "title", "url", "published_at"})

		mock.ExpectQuery("SELECT id, nhk_id, title, url, published_at FROM news").
			WithArgs(10, 0).
			WillReturnRows(rows)

		news, err := s.GetNewsList(ctx, 10, 0)

		assert.NoError(t, err)
		assert.Empty(t, news)
		assert.NotNil(t, news)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGetNewsByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	s := NewWithDB(db)
	ctx := context.Background()

	now := time.Now()
	publishedAt := time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)

	t.Run("returns news with paragraphs", func(t *testing.T) {
		newsRow := sqlmock.NewRows([]string{"id", "nhk_id", "title", "url", "published_at", "fetched_at", "created_at"}).
			AddRow(1, "ne123", "Test News", "http://example.com", publishedAt, now, now)

		paragraphRows := sqlmock.NewRows([]string{"id", "news_id", "position", "raw_text", "tokens", "created_at"}).
			AddRow(1, 1, 0, "First paragraph", `[{"kana":"テスト","translation":"test"}]`, now).
			AddRow(2, 1, 1, "Second paragraph", `[{"kana":"二番目","translation":"second"}]`, now)

		mock.ExpectQuery("SELECT id, nhk_id, title, url, published_at, fetched_at, created_at FROM news").
			WithArgs(1).
			WillReturnRows(newsRow)

		mock.ExpectQuery("SELECT id, news_id, position, raw_text, tokens, created_at FROM paragraphs").
			WithArgs(1).
			WillReturnRows(paragraphRows)

		news, err := s.GetNewsByID(ctx, 1)

		assert.NoError(t, err)
		assert.Equal(t, "ne123", news.NHKID)
		assert.Len(t, news.Paragraphs, 2)
		assert.Equal(t, "First paragraph", news.Paragraphs[0].RawText)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error when not found", func(t *testing.T) {
		mock.ExpectQuery("SELECT id, nhk_id, title, url, published_at, fetched_at, created_at FROM news").
			WithArgs(999).
			WillReturnError(sql.ErrNoRows)

		news, err := s.GetNewsByID(ctx, 999)

		assert.Error(t, err)
		assert.Nil(t, news)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGetLastRun(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	s := NewWithDB(db)
	ctx := context.Background()

	t.Run("returns last run time", func(t *testing.T) {
		expected := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

		mock.ExpectQuery("SELECT value FROM scheduler_state").
			WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(expected))

		lastRun, err := s.GetLastRun(ctx)

		assert.NoError(t, err)
		assert.Equal(t, expected, lastRun)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error when no record", func(t *testing.T) {
		mock.ExpectQuery("SELECT value FROM scheduler_state").
			WillReturnError(sql.ErrNoRows)

		_, err := s.GetLastRun(ctx)

		assert.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestSetLastRun(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	s := NewWithDB(db)
	ctx := context.Background()

	t.Run("sets last run time", func(t *testing.T) {
		runTime := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

		mock.ExpectExec("INSERT INTO scheduler_state").
			WithArgs(runTime).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := s.SetLastRun(ctx, runTime)

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
