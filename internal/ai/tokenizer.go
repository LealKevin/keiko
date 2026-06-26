package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/LealKevin/keiko/internal/store"
	"google.golang.org/genai"
)

const apiTimeout = 2 * time.Minute

const systemPrompt = `Act as an expert Japanese Linguist. Your task is to perform morphological analysis on Japanese text for a language learning app.

### RULES:
1. TOKENIZATION: Split text into the smallest logical morphological units.
2. SEPARATION: Grammatical particles (助詞) and auxiliary verbs (助動詞) MUST be separate objects.
3. VERB STEMS: If a verb is conjugated (e.g., 飲んで), the "Kana" should be the conjugated form ("飲んで") but the "BaseForm" must be the dictionary form ("飲む").
4. PUNCTUATION: Include punctuation as separate tokens.
5. NO EXPLANATION: Output ONLY valid JSON. No prose.

### SCHEMA:
type Token struct {
    Kana        string
    Furigana    string
    BaseForm    string
    Translation string
}

### EXAMPLE:
Input: "食べています。"
Output: [
    {"Kana": "食べて", "Furigana": "たべて", "BaseForm": "食べる", "Translation": "to eat (te-form)"},
    {"Kana": "い", "Furigana": "い", "BaseForm": "いる", "Translation": "[auxiliary: progressive state]"},
    {"Kana": "ます", "Furigana": "ます", "BaseForm": "ます", "Translation": "[polite auxiliary]"},
    {"Kana": "。", "Furigana": "。", "BaseForm": "。", "Translation": "."}
]

### INPUT TEXT:
`

const batchSystemPrompt = `Act as an expert Japanese Linguist. Your task is to perform morphological analysis on multiple Japanese paragraphs for a language learning app.

### RULES:
1. TOKENIZATION: Split text into the smallest logical morphological units.
2. SEPARATION: Grammatical particles (助詞) and auxiliary verbs (助動詞) MUST be separate objects.
3. VERB STEMS: If a verb is conjugated (e.g., 飲んで), the "Kana" should be the conjugated form ("飲んで") but the "BaseForm" must be the dictionary form ("飲む").
4. PUNCTUATION: Include punctuation as separate tokens.
5. NO EXPLANATION: Output ONLY valid JSON. No prose.
6. PRESERVE ORDER: Return paragraphs in the same order as input.

### SCHEMA:
type Token struct {
    Kana        string
    Furigana    string
    BaseForm    string
    Translation string
}

type Paragraph struct {
    Index  int
    Tokens []Token
}

### EXAMPLE:
Input paragraphs:
[0]: "食べています。"
[1]: "飲みます。"

Output:
{
  "paragraphs": [
    {"index": 0, "tokens": [
      {"Kana": "食べて", "Furigana": "たべて", "BaseForm": "食べる", "Translation": "to eat (te-form)"},
      {"Kana": "い", "Furigana": "い", "BaseForm": "いる", "Translation": "[auxiliary: progressive state]"},
      {"Kana": "ます", "Furigana": "ます", "BaseForm": "ます", "Translation": "[polite auxiliary]"},
      {"Kana": "。", "Furigana": "。", "BaseForm": "。", "Translation": "."}
    ]},
    {"index": 1, "tokens": [
      {"Kana": "飲み", "Furigana": "のみ", "BaseForm": "飲む", "Translation": "to drink"},
      {"Kana": "ます", "Furigana": "ます", "BaseForm": "ます", "Translation": "[polite auxiliary]"},
      {"Kana": "。", "Furigana": "。", "BaseForm": "。", "Translation": "."}
    ]}
  ]
}

### INPUT PARAGRAPHS:
`

type Tokenizer struct {
	client *genai.Client
}

func NewTokenizer(apiKey string) (*Tokenizer, error) {
	ctx := context.Background()
	cfg := &genai.ClientConfig{APIKey: apiKey}

	client, err := genai.NewClient(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	return &Tokenizer{client: client}, nil
}

type TokenizeResponse struct {
	Tokens []struct {
		Kana        string `json:"Kana"`
		Furigana    string `json:"Furigana"`
		BaseForm    string `json:"BaseForm"`
		Translation string `json:"Translation"`
	} `json:"tokens"`
}

func (t *Tokenizer) Tokenize(ctx context.Context, text string) ([]store.Token, error) {
	schema := &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"tokens": {
				Type: genai.TypeArray,
				Items: &genai.Schema{
					Type:     genai.TypeObject,
					Required: []string{"Kana", "Furigana", "BaseForm", "Translation"},
					Properties: map[string]*genai.Schema{
						"Kana":        {Type: genai.TypeString},
						"Furigana":    {Type: genai.TypeString},
						"BaseForm":    {Type: genai.TypeString},
						"Translation": {Type: genai.TypeString},
					},
				},
			},
		},
		Required: []string{"tokens"},
	}

	config := &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
		ResponseSchema:   schema,
	}

	prompt := systemPrompt + text

	result, err := t.client.Models.GenerateContent(
		ctx,
		"gemini-3-flash-preview",
		genai.Text(prompt),
		config,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %w", err)
	}

	if result == nil || len(result.Candidates) == 0 || result.Candidates[0].Content == nil {
		return nil, fmt.Errorf("empty response from Gemini API")
	}

	var response TokenizeResponse
	if err := json.Unmarshal([]byte(result.Text()), &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	tokens := make([]store.Token, len(response.Tokens))
	for i, t := range response.Tokens {
		tokens[i] = store.Token{
			Kana:        t.Kana,
			Furigana:    t.Furigana,
			BaseForm:    t.BaseForm,
			Translation: t.Translation,
		}
	}

	return tokens, nil
}

type BatchTokenizeResponse struct {
	Paragraphs []struct {
		Index  int `json:"index"`
		Tokens []struct {
			Kana        string `json:"Kana"`
			Furigana    string `json:"Furigana"`
			BaseForm    string `json:"BaseForm"`
			Translation string `json:"Translation"`
		} `json:"tokens"`
	} `json:"paragraphs"`
}

func (t *Tokenizer) TokenizeBatch(ctx context.Context, texts []string) ([][]store.Token, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(ctx, apiTimeout)
	defer cancel()

	var promptBuilder strings.Builder
	promptBuilder.WriteString(batchSystemPrompt)
	for i, text := range texts {
		promptBuilder.WriteString(fmt.Sprintf("[%d]: %q\n", i, text))
	}

	schema := &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"paragraphs": {
				Type: genai.TypeArray,
				Items: &genai.Schema{
					Type:     genai.TypeObject,
					Required: []string{"index", "tokens"},
					Properties: map[string]*genai.Schema{
						"index": {Type: genai.TypeInteger},
						"tokens": {
							Type: genai.TypeArray,
							Items: &genai.Schema{
								Type:     genai.TypeObject,
								Required: []string{"Kana", "Furigana", "BaseForm", "Translation"},
								Properties: map[string]*genai.Schema{
									"Kana":        {Type: genai.TypeString},
									"Furigana":    {Type: genai.TypeString},
									"BaseForm":    {Type: genai.TypeString},
									"Translation": {Type: genai.TypeString},
								},
							},
						},
					},
				},
			},
		},
		Required: []string{"paragraphs"},
	}

	config := &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
		ResponseSchema:   schema,
	}

	result, err := t.client.Models.GenerateContent(
		ctx,
		"gemini-3-flash-preview",
		genai.Text(promptBuilder.String()),
		config,
	)
	if err != nil {
		return nil, fmt.Errorf("API error: %w", err)
	}

	if result == nil || len(result.Candidates) == 0 || result.Candidates[0].Content == nil {
		return nil, fmt.Errorf("empty response from API")
	}

	responseText := result.Text()
	if responseText == "" {
		return nil, fmt.Errorf("empty response text from API")
	}

	var response BatchTokenizeResponse
	if err := json.Unmarshal([]byte(responseText), &response); err != nil {
		return nil, fmt.Errorf("JSON parse error: %w (response: %.100s...)", err, responseText)
	}

	if len(response.Paragraphs) != len(texts) {
		return nil, fmt.Errorf("paragraph count mismatch: expected %d, got %d", len(texts), len(response.Paragraphs))
	}

	allTokens := make([][]store.Token, len(texts))
	for _, p := range response.Paragraphs {
		if p.Index < 0 || p.Index >= len(texts) {
			return nil, fmt.Errorf("invalid paragraph index %d (expected 0-%d)", p.Index, len(texts)-1)
		}
		tokens := make([]store.Token, len(p.Tokens))
		for i, t := range p.Tokens {
			tokens[i] = store.Token{
				Kana:        t.Kana,
				Furigana:    t.Furigana,
				BaseForm:    t.BaseForm,
				Translation: t.Translation,
			}
		}
		allTokens[p.Index] = tokens
	}

	for i, tokens := range allTokens {
		if tokens == nil {
			return nil, fmt.Errorf("missing tokens for paragraph %d", i)
		}
	}

	return allTokens, nil
}
