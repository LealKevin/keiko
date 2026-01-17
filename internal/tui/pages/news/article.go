package news

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/LealKevin/keiko/internal/news"
)

type ArticleView struct {
	detail     *news.NewsDetail
	tokens     []news.Token
	paraBreaks []int
	lines      []tokenLine
	cursor     int
	width      int
	height     int
}

type tokenLine struct {
	startIdx    int
	endIdx      int
	isParaBreak bool
}

func NewArticleView() *ArticleView {
	return &ArticleView{}
}

func (a *ArticleView) SetArticle(detail *news.NewsDetail) {
	a.detail = detail
	a.cursor = 0
	a.tokens = nil
	a.paraBreaks = nil
	a.lines = nil

	if detail == nil {
		return
	}

	for _, para := range detail.Paragraphs {
		if len(a.tokens) > 0 {
			a.paraBreaks = append(a.paraBreaks, len(a.tokens))
		}
		a.tokens = append(a.tokens, para.Tokens...)
	}

	a.computeLines()
}

func (a *ArticleView) SetSize(width, height int) {
	a.width = width
	a.height = height
	a.computeLines()
}

func (a *ArticleView) isParaBreak(idx int) bool {
	for _, b := range a.paraBreaks {
		if b == idx {
			return true
		}
	}
	return false
}

func (a *ArticleView) computeLines() {
	if len(a.tokens) == 0 || a.width == 0 {
		return
	}

	a.lines = nil
	lineStart := 0
	lineWidth := 0

	for i, token := range a.tokens {
		if a.isParaBreak(i) && lineStart < i {
			a.lines = append(a.lines, tokenLine{startIdx: lineStart, endIdx: i - 1})
			a.lines = append(a.lines, tokenLine{isParaBreak: true})
			lineStart = i
			lineWidth = 0
		}

		tokenWidth := lipgloss.Width(token.Kana) + 1

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
	for i := currentLine + 1; i < len(a.lines); i++ {
		if !a.lines[i].isParaBreak {
			a.cursor = a.lines[i].startIdx
			return
		}
	}
}

func (a *ArticleView) MoveUp() {
	currentLine := a.currentLineIndex()
	for i := currentLine - 1; i >= 0; i-- {
		if !a.lines[i].isParaBreak {
			a.cursor = a.lines[i].startIdx
			return
		}
	}
}

func (a *ArticleView) currentLineIndex() int {
	for i, line := range a.lines {
		if !line.isParaBreak && a.cursor >= line.startIdx && a.cursor <= line.endIdx {
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

	titleStyle := lipgloss.NewStyle().Bold(true)
	normalStyle := lipgloss.NewStyle()
	selectedStyle := lipgloss.NewStyle().Background(lipgloss.Color("205")).Foreground(lipgloss.Color("0"))

	var sb strings.Builder
	sb.WriteString(titleStyle.Render(a.detail.Title))
	sb.WriteString("\n\n")

	for _, line := range a.lines {
		if line.isParaBreak {
			sb.WriteString("\n")
			continue
		}
		for i := line.startIdx; i <= line.endIdx; i++ {
			style := normalStyle
			if i == a.cursor {
				style = selectedStyle
			}
			sb.WriteString(style.Render(a.tokens[i].Kana))
			if i < line.endIdx {
				sb.WriteString(" ")
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
