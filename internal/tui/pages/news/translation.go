package news

import (
	"fmt"

	"github.com/LealKevin/keiko/internal/news"
)

type TranslationPanel struct {
	width int
}

func NewTranslationPanel() *TranslationPanel {
	return &TranslationPanel{}
}

func (t *TranslationPanel) SetWidth(width int) {
	t.width = width
}

func (t *TranslationPanel) View(token *news.Token) string {
	if token == nil {
		return "Navigate with h/l/j/k to explore tokens"
	}

	line1 := fmt.Sprintf("%s【%s】%s", token.BaseForm, token.Furigana, token.Translation)

	line2 := ""
	if token.Kana != token.BaseForm {
		line2 = fmt.Sprintf("Form: %s", token.Kana)
	}

	content := line1
	if line2 != "" {
		content += "\n" + line2
	}

	return content
}
