package anki

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const (
	BaseURL         = "http://localhost:8765"
	RefreshInterval = 30 * time.Second
)

type Client struct {
	http *http.Client
}

type DeckInfo struct {
	Name     string
	DueCount int
}

type CardInfo struct {
	CardID   int64
	DeckName string
	Question string
	Answer   string
}

func NewClient() *Client {
	return &Client{
		http: &http.Client{Timeout: 5 * time.Second},
	}
}

type ankiRequest struct {
	Action  string      `json:"action"`
	Version int         `json:"version"`
	Params  interface{} `json:"params,omitempty"`
}

type ankiResponse struct {
	Result json.RawMessage `json:"result"`
	Error  *string         `json:"error"`
}

func (c *Client) call(action string, params interface{}) (json.RawMessage, error) {
	req := ankiRequest{
		Action:  action,
		Version: 6,
		Params:  params,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Post(BaseURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var ar ankiResponse
	if err := json.NewDecoder(resp.Body).Decode(&ar); err != nil {
		return nil, err
	}

	if ar.Error != nil {
		return nil, fmt.Errorf("anki error: %s", *ar.Error)
	}

	return ar.Result, nil
}

func (c *Client) IsConnected() bool {
	_, err := c.call("version", nil)
	return err == nil
}

func (c *Client) GetDecksWithStats() ([]DeckInfo, error) {
	result, err := c.call("deckNamesAndIds", nil)
	if err != nil {
		return nil, err
	}

	var deckMap map[string]int64
	if err := json.Unmarshal(result, &deckMap); err != nil {
		return nil, err
	}

	var decks []DeckInfo
	for name := range deckMap {
		dueCount, err := c.getDueCount(name)
		if err != nil {
			dueCount = 0
		}
		decks = append(decks, DeckInfo{Name: name, DueCount: dueCount})
	}

	return decks, nil
}

func (c *Client) getDueCount(deck string) (int, error) {
	result, err := c.call("getDeckStats", map[string]interface{}{
		"decks": []string{deck},
	})
	if err != nil {
		return 0, err
	}

	var stats map[string]struct {
		NewCount    int `json:"new_count"`
		LearnCount  int `json:"learn_count"`
		ReviewCount int `json:"review_count"`
	}
	if err := json.Unmarshal(result, &stats); err != nil {
		return 0, err
	}

	for _, s := range stats {
		return s.NewCount + s.LearnCount + s.ReviewCount, nil
	}
	return 0, nil
}

func (c *Client) GetDueCards(deck string) ([]int64, error) {
	result, err := c.call("findCards", map[string]interface{}{
		"query": fmt.Sprintf("deck:\"%s\" is:due", deck),
	})
	if err != nil {
		return nil, err
	}

	var cardIDs []int64
	if err := json.Unmarshal(result, &cardIDs); err != nil {
		return nil, err
	}

	return cardIDs, nil
}

func (c *Client) GetCardInfo(cardID int64) (*CardInfo, error) {
	result, err := c.call("cardsInfo", map[string]interface{}{
		"cards": []int64{cardID},
	})
	if err != nil {
		return nil, err
	}

	var cards []struct {
		CardID   int64  `json:"cardId"`
		DeckName string `json:"deckName"`
		Question string `json:"question"`
		Answer   string `json:"answer"`
	}
	if err := json.Unmarshal(result, &cards); err != nil {
		return nil, err
	}

	if len(cards) == 0 {
		return nil, fmt.Errorf("card not found: %d", cardID)
	}

	card := cards[0]
	return &CardInfo{
		CardID:   card.CardID,
		DeckName: card.DeckName,
		Question: StripHTML(card.Question),
		Answer:   StripHTML(card.Answer),
	}, nil
}

func (c *Client) AnswerCard(cardID int64, ease int) error {
	_, err := c.call("answerCards", map[string]interface{}{
		"answers": []map[string]interface{}{
			{"cardId": cardID, "ease": ease},
		},
	})
	return err
}

var (
	soundRegex = regexp.MustCompile(`\[sound:[^\]]+\]`)
	brRegex    = regexp.MustCompile(`<br\s*/?>`)
	tagRegex   = regexp.MustCompile(`<[^>]+>`)
	spaceRegex = regexp.MustCompile(`\s+`)
)

func StripHTML(htmlStr string) string {
	text := soundRegex.ReplaceAllString(htmlStr, "[audio]")
	text = brRegex.ReplaceAllString(text, " ")
	text = tagRegex.ReplaceAllString(text, "")
	text = html.UnescapeString(text)
	text = spaceRegex.ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)

	if text == "" {
		return "[media card]"
	}
	return text
}
