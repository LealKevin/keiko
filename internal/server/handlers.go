package server

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
)

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	status := "ok"
	dbStatus := "connected"

	if err := s.store.Ping(ctx); err != nil {
		dbStatus = "disconnected"
		status = "degraded"
	}

	response := map[string]string{
		"status": status,
		"db":     dbStatus,
	}

	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleGetNews(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	limit := parseIntParam(r, "limit", 10)
	offset := parseIntParam(r, "offset", 0)

	if limit < 1 {
		limit = 1
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	news, err := s.store.GetNewsList(ctx, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch news")
		return
	}

	writeJSON(w, http.StatusOK, news)
}

func (s *Server) handleGetNewsById(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid news ID")
		return
	}

	news, err := s.store.GetNewsByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "News not found")
		} else {
			writeError(w, http.StatusInternalServerError, "Failed to fetch news")
		}
		return
	}

	writeJSON(w, http.StatusOK, news)
}

func parseIntParam(r *http.Request, key string, defaultVal int) int {
	str := r.URL.Query().Get(key)
	if str == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(str)
	if err != nil {
		return defaultVal
	}
	return val
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
