package scraper

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseJapaneseDate(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  *time.Time
	}{
		{
			name:  "full date with time",
			input: "2025年1月15日 12時30分",
			want:  timePtr(time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)),
		},
		{
			name:  "date only",
			input: "2025年1月15日",
			want:  timePtr(time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)),
		},
		{
			name:  "single digit month and day",
			input: "2025年1月5日",
			want:  timePtr(time.Date(2025, 1, 5, 0, 0, 0, 0, time.UTC)),
		},
		{
			name:  "double digit month",
			input: "2024年12月25日 15時00分",
			want:  timePtr(time.Date(2024, 12, 25, 0, 0, 0, 0, time.UTC)),
		},
		{
			name:  "invalid format",
			input: "January 15, 2025",
			want:  nil,
		},
		{
			name:  "empty string",
			input: "",
			want:  nil,
		},
		{
			name:  "partial date",
			input: "2025年1月",
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseJapaneseDate(tt.input)
			if tt.want == nil {
				assert.Nil(t, got)
			} else {
				require.NotNil(t, got)
				assert.Equal(t, *tt.want, *got)
			}
		})
	}
}

func TestIsRateLimitError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "429 error",
			err:  errors.New("HTTP 429: Too Many Requests"),
			want: true,
		},
		{
			name: "RESOURCE_EXHAUSTED error",
			err:  errors.New("RESOURCE_EXHAUSTED: quota exceeded"),
			want: true,
		},
		{
			name: "contains 429",
			err:  errors.New("request failed with status 429"),
			want: true,
		},
		{
			name: "regular error",
			err:  errors.New("connection timeout"),
			want: false,
		},
		{
			name: "500 error",
			err:  errors.New("HTTP 500: Internal Server Error"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isRateLimitError(tt.err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func timePtr(t time.Time) *time.Time {
	return &t
}
