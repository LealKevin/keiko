package news

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/LealKevin/keiko/internal/news"
)

type ArticleView struct {
	detail *news.NewsDetail
	tokens []news.Token
	lines  []tokenLine
	cursor int
	width  int
	height int
}

type tokenLine struct {
	startIdx int
	endIdx   int
}

func NewArticleView() *ArticleView {
	return &ArticleView{}
}

func (a *ArticleView) SetArticle(detail *news.NewsDetail) {
	a.detail = detail
	a.cursor = 0
	a.tokens = nil
	a.lines = nil

	if detail == nil {
		return
	}

	for _, para := range detail.Paragraphs {
		a.tokens = append(a.tokens, para.Tokens...)
	}

	a.computeLines()
}

func (a *ArticleView) SetSize(width, height int) {
	a.width = width
	a.height = height
	a.computeLines()
}

func (a *ArticleView) computeLines() {
	if len(a.tokens) == 0 || a.width == 0 {
		return
	}

	a.lines = nil
	lineStart := 0
	lineWidth := 0

	for i, token := range a.tokens {
		tokenWidth := len([]rune(token.Kana)) + 1

		if lineWidth+tokenWidth > a.width && lineStart < i {
			a.lines = append(a.lines, tokenLine{startIdx: lineStart, endIdx: i - 1})
			lineStart = i
			lineWidth = tokenWidth
		} else {
			lineWidth += tokenWidth
		}
	}

	if lineStart < len(a.tokens) {
		a.lines = append(a.lines, tokenLine{startIdx: lineStart, endIdx: len(a.tokens) - 1})
	}
}

func (a *ArticleView) MoveLeft() {
	if a.cursor > 0 {
		a.cursor--
	}
}

func (a *ArticleView) MoveRight() {
	if a.cursor < len(a.tokens)-1 {
		a.cursor++
	}
}

func (a *ArticleView) MoveDown() {
	currentLine := a.currentLineIndex()
	if currentLine < len(a.lines)-1 {
		a.cursor = a.lines[currentLine+1].startIdx
	}
}

func (a *ArticleView) MoveUp() {
	currentLine := a.currentLineIndex()
	if currentLine > 0 {
		a.cursor = a.lines[currentLine-1].startIdx
	}
}

func (a *ArticleView) currentLineIndex() int {
	for i, line := range a.lines {
		if a.cursor >= line.startIdx && a.cursor <= line.endIdx {
			return i
		}
	}
	return 0
}

func (a *ArticleView) SelectedToken() *news.Token {
	if a.cursor < 0 || a.cursor >= len(a.tokens) {
		return nil
	}
	return &a.tokens[a.cursor]
}

func (a *ArticleView) View() string {
	if a.detail == nil {
		return "No article selected"
	}

	if len(a.tokens) == 0 {
		return "No content available"
	}

	titleStyle := lipgloss.NewStyle().Bold(true).MarginBottom(1)
	normalStyle := lipgloss.NewStyle()
	selectedStyle := lipgloss.NewStyle().Background(lipgloss.Color("205")).Foreground(lipgloss.Color("0"))

	var sb strings.Builder
	sb.WriteString(titleStyle.Render(a.detail.Title))
	sb.WriteString("\n\n")

	for i, token := range a.tokens {
		style := normalStyle
		if i == a.cursor {
			style = selectedStyle
		}
		sb.WriteString(style.Render(token.Kana))
		sb.WriteString(" ")
	}

	return sb.String()
}
