package data

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

type Word struct {
	ID       int    `json:"id"`
	Word     string `json:"word"`
	Meaning  string `json:"meaning"`
	Furigana string `json:"furigana"`
	Romaji   string `json:"romaji"`
	Level    int    `json:"level"`
}

var (
	ErrorFetchingWords = errors.New("error fetching words")
	ErrorParsingWords  = errors.New("error parsing words")
)

func FetchWords() ([]Word, error) {
	response, err := http.Get("https://jlpt-vocab-api.vercel.app/api/words/all")
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrorFetchingWords, err)
	}
	defer response.Body.Close()

	words := []Word{}
	err = json.NewDecoder(response.Body).Decode(&words)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrorParsingWords, err)
	}

	return words, nil
}
