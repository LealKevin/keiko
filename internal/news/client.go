package news

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type NewsListItem struct {
	ID          int       `json:"id"`
	NhkID       string    `json:"nhk_id"`
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	PublishedAt time.Time `json:"published_at"`
}

type Token struct {
	Kana        string `json:"kana"`
	Furigana    string `json:"furigana"`
	BaseForm    string `json:"base_form"`
	Translation string `json:"translation"`
}

type Paragraph struct {
	ID       int     `json:"id"`
	Position int     `json:"position"`
	RawText  string  `json:"raw_text"`
	Tokens   []Token `json:"tokens"`
}

type NewsDetail struct {
	ID          int         `json:"id"`
	NhkID       string      `json:"nhk_id"`
	Title       string      `json:"title"`
	URL         string      `json:"url"`
	PublishedAt time.Time   `json:"published_at"`
	Paragraphs  []Paragraph `json:"paragraphs"`
}

func (c *Client) GetNewsList(limit, offset int) ([]NewsListItem, error) {
	url := fmt.Sprintf("%s/api/v1/news?limit=%d&offset=%d", c.baseURL, limit, offset)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch news list: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var items []NewsListItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return items, nil
}

func (c *Client) GetNewsDetail(id int) (*NewsDetail, error) {
	url := fmt.Sprintf("%s/api/v1/news/%d", c.baseURL, id)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch news detail: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var detail NewsDetail
	if err := json.NewDecoder(resp.Body).Decode(&detail); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &detail, nil
}

func (c *Client) IsAvailable() bool {
	url := fmt.Sprintf("%s/health", c.baseURL)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
