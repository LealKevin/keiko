package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRateLimiterAllow(t *testing.T) {
	rl := &RateLimiter{
		requests:    make(map[string][]time.Time),
		maxRequests: 3,
		windowSecs:  60,
	}

	ip := "192.168.1.1"

	assert.True(t, rl.allow(ip), "first request should be allowed")
	assert.True(t, rl.allow(ip), "second request should be allowed")
	assert.True(t, rl.allow(ip), "third request should be allowed")
	assert.False(t, rl.allow(ip), "fourth request should be blocked")

	otherIP := "192.168.1.2"
	assert.True(t, rl.allow(otherIP), "different IP should be allowed")
}

func TestRateLimiterMiddleware(t *testing.T) {
	rl := &RateLimiter{
		requests:    make(map[string][]time.Time),
		maxRequests: 2,
		windowSecs:  60,
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := rl.Middleware(handler)

	makeRequest := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()
		middleware.ServeHTTP(w, req)
		return w
	}

	w := makeRequest()
	assert.Equal(t, http.StatusOK, w.Code)

	w = makeRequest()
	assert.Equal(t, http.StatusOK, w.Code)

	w = makeRequest()
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.Equal(t, "60", w.Header().Get("Retry-After"))
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		want       string
	}{
		{
			name:       "X-Forwarded-For single IP",
			headers:    map[string]string{"X-Forwarded-For": "203.0.113.195"},
			remoteAddr: "192.168.1.1:12345",
			want:       "203.0.113.195",
		},
		{
			name:       "X-Forwarded-For multiple IPs",
			headers:    map[string]string{"X-Forwarded-For": "203.0.113.195, 70.41.3.18, 150.172.238.178"},
			remoteAddr: "192.168.1.1:12345",
			want:       "203.0.113.195",
		},
		{
			name:       "X-Real-IP",
			headers:    map[string]string{"X-Real-IP": "203.0.113.195"},
			remoteAddr: "192.168.1.1:12345",
			want:       "203.0.113.195",
		},
		{
			name:       "fallback to RemoteAddr",
			headers:    map[string]string{},
			remoteAddr: "192.168.1.1:12345",
			want:       "192.168.1.1:12345",
		},
		{
			name:       "X-Forwarded-For takes precedence over X-Real-IP",
			headers:    map[string]string{"X-Forwarded-For": "10.0.0.1", "X-Real-IP": "10.0.0.2"},
			remoteAddr: "192.168.1.1:12345",
			want:       "10.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteAddr
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			got := getClientIP(req)
			assert.Equal(t, tt.want, got)
		})
	}
}
