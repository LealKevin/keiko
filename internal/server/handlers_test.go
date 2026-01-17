package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/LealKevin/keiko/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseIntParam(t *testing.T) {
	tests := []struct {
		name       string
		queryParam string
		defaultVal int
		want       int
	}{
		{"valid integer", "10", 5, 10},
		{"empty string returns default", "", 5, 5},
		{"invalid string returns default", "abc", 5, 5},
		{"negative integer", "-5", 10, -5},
		{"zero", "0", 10, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/?key="+tt.queryParam, nil)
			got := parseIntParam(req, "key", tt.defaultVal)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]string{"status": "ok"}

	writeJSON(w, http.StatusOK, data)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&response))
	assert.Equal(t, "ok", response["status"])
}

func TestWriteError(t *testing.T) {
	w := httptest.NewRecorder()

	writeError(w, http.StatusBadRequest, "invalid request")

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&response))
	assert.Equal(t, "invalid request", response["error"])
}

func TestHandleHealth(t *testing.T) {
	tests := []struct {
		name          string
		setupMock     func(mock sqlmock.Sqlmock)
		wantStatus    int
		wantDBStatus  string
		wantAppStatus string
	}{
		{
			name: "healthy - db connected",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectPing()
			},
			wantStatus:    http.StatusOK,
			wantDBStatus:  "connected",
			wantAppStatus: "ok",
		},
		{
			name: "degraded - db disconnected",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectPing().WillReturnError(context.DeadlineExceeded)
			},
			wantStatus:    http.StatusOK,
			wantDBStatus:  "disconnected",
			wantAppStatus: "degraded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
			require.NoError(t, err)
			defer db.Close()

			tt.setupMock(mock)

			s := store.NewWithDB(db)
			server := New(s)

			req := httptest.NewRequest("GET", "/health", nil)
			w := httptest.NewRecorder()

			server.handleHealth(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)

			var response map[string]string
			require.NoError(t, json.NewDecoder(w.Body).Decode(&response))
			assert.Equal(t, tt.wantAppStatus, response["status"])
			assert.Equal(t, tt.wantDBStatus, response["db"])
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestHandleGetNews(t *testing.T) {
	publishedAt := time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		query      string
		setupMock  func(mock sqlmock.Sqlmock)
		wantStatus int
		wantLen    int
	}{
		{
			name:  "default pagination",
			query: "",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "nhk_id", "title", "url", "published_at"}).
					AddRow(1, "ne123", "Test News", "http://example.com", publishedAt).
					AddRow(2, "ne456", "Test News 2", "http://example.com/2", publishedAt)
				mock.ExpectQuery("SELECT id, nhk_id, title, url, published_at FROM news").
					WithArgs(10, 0).
					WillReturnRows(rows)
			},
			wantStatus: http.StatusOK,
			wantLen:    2,
		},
		{
			name:  "custom pagination",
			query: "?limit=5&offset=10",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "nhk_id", "title", "url", "published_at"})
				mock.ExpectQuery("SELECT id, nhk_id, title, url, published_at FROM news").
					WithArgs(5, 10).
					WillReturnRows(rows)
			},
			wantStatus: http.StatusOK,
			wantLen:    0,
		},
		{
			name:  "limit clamped to 100",
			query: "?limit=200",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "nhk_id", "title", "url", "published_at"})
				mock.ExpectQuery("SELECT id, nhk_id, title, url, published_at FROM news").
					WithArgs(100, 0).
					WillReturnRows(rows)
			},
			wantStatus: http.StatusOK,
			wantLen:    0,
		},
		{
			name:  "negative limit clamped to 1",
			query: "?limit=-5",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "nhk_id", "title", "url", "published_at"})
				mock.ExpectQuery("SELECT id, nhk_id, title, url, published_at FROM news").
					WithArgs(1, 0).
					WillReturnRows(rows)
			},
			wantStatus: http.StatusOK,
			wantLen:    0,
		},
		{
			name:  "negative offset clamped to 0",
			query: "?offset=-10",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "nhk_id", "title", "url", "published_at"})
				mock.ExpectQuery("SELECT id, nhk_id, title, url, published_at FROM news").
					WithArgs(10, 0).
					WillReturnRows(rows)
			},
			wantStatus: http.StatusOK,
			wantLen:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			tt.setupMock(mock)

			s := store.NewWithDB(db)
			server := New(s)

			req := httptest.NewRequest("GET", "/api/v1/news"+tt.query, nil)
			w := httptest.NewRecorder()

			server.handleGetNews(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)

			var response []store.NewsList
			require.NoError(t, json.NewDecoder(w.Body).Decode(&response))
			assert.Len(t, response, tt.wantLen)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestHandleGetNewsById(t *testing.T) {
	now := time.Now()
	publishedAt := time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		id         string
		setupMock  func(mock sqlmock.Sqlmock)
		wantStatus int
	}{
		{
			name: "valid ID returns news",
			id:   "1",
			setupMock: func(mock sqlmock.Sqlmock) {
				newsRow := sqlmock.NewRows([]string{"id", "nhk_id", "title", "url", "published_at", "fetched_at", "created_at"}).
					AddRow(1, "ne123", "Test News", "http://example.com", publishedAt, now, now)
				mock.ExpectQuery("SELECT id, nhk_id, title, url, published_at, fetched_at, created_at FROM news").
					WithArgs(1).
					WillReturnRows(newsRow)

				paragraphRows := sqlmock.NewRows([]string{"id", "news_id", "position", "raw_text", "tokens", "created_at"}).
					AddRow(1, 1, 0, "Test paragraph", `[{"kana":"テスト","furigana":"","base_form":"テスト","translation":"test"}]`, now)
				mock.ExpectQuery("SELECT id, news_id, position, raw_text, tokens, created_at FROM paragraphs").
					WithArgs(1).
					WillReturnRows(paragraphRows)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid ID returns bad request",
			id:         "abc",
			setupMock:  func(mock sqlmock.Sqlmock) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "not found ID returns 404",
			id:   "999",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT id, nhk_id, title, url, published_at, fetched_at, created_at FROM news").
					WithArgs(999).
					WillReturnError(sql.ErrNoRows)
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "database error returns 500",
			id:   "999",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT id, nhk_id, title, url, published_at, fetched_at, created_at FROM news").
					WithArgs(999).
					WillReturnError(context.DeadlineExceeded)
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			tt.setupMock(mock)

			s := store.NewWithDB(db)
			server := New(s)

			req := httptest.NewRequest("GET", "/api/v1/news/"+tt.id, nil)
			req.SetPathValue("id", tt.id)
			w := httptest.NewRecorder()

			server.handleGetNewsById(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
