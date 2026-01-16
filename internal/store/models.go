package store

import (
	"encoding/json"
	"time"
)

type News struct {
	ID          int        `json:"id"`
	NHKID       string     `json:"nhk_id"`
	Title       string     `json:"title"`
	URL         string     `json:"url"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	FetchedAt   time.Time  `json:"fetched_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

type NewsWithParagraphs struct {
	News
	Paragraphs []Paragraph `json:"paragraphs"`
}

type Paragraph struct {
	ID        int       `json:"id"`
	NewsID    int       `json:"news_id"`
	Position  int       `json:"position"`
	RawText   string    `json:"raw_text"`
	Tokens    []Token   `json:"tokens"`
	CreatedAt time.Time `json:"created_at"`
}

type Token struct {
	Kana        string `json:"kana"`
	Furigana    string `json:"furigana"`
	BaseForm    string `json:"base_form"`
	Translation string `json:"translation"`
}

type TokensJSON []Token

func (t TokensJSON) Value() ([]byte, error) {
	return json.Marshal(t)
}

func (t *TokensJSON) Scan(value interface{}) error {
	if value == nil {
		*t = []Token{}
		return nil
	}
	return json.Unmarshal(value.([]byte), t)
}

type NewsList struct {
	ID          int        `json:"id"`
	NHKID       string     `json:"nhk_id"`
	Title       string     `json:"title"`
	URL         string     `json:"url"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
}
