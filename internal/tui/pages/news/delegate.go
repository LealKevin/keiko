package news

import (
	"fmt"
	"io"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type NewsItem struct {
	ID          int
	NhkID       string
	Title       string
	PublishedAt time.Time
	IsRead      bool
}

func (n NewsItem) FilterValue() string {
	return n.Title
}

type itemDelegate struct{}

func NewItemDelegate() list.ItemDelegate {
	return itemDelegate{}
}

func (d itemDelegate) Height() int {
	return 2
}

func (d itemDelegate) Spacing() int {
	return 0
}

func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}

func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(NewsItem)
	if !ok {
		return
	}

	readIndicator := "● "
	if item.IsRead {
		readIndicator = "  "
	}

	dateStr := item.PublishedAt.Format("01/02")

	title := item.Title
	maxTitleWidth := 24
	runes := []rune(title)
	for i := range runes {
		if lipgloss.Width(string(runes[:i+1])) > maxTitleWidth {
			title = string(runes[:i]) + "…"
			break
		}
	}

	line1 := fmt.Sprintf("%s%s", readIndicator, dateStr)
	line2 := fmt.Sprintf("   %s", title)

	selected := index == m.Index()

	style := lipgloss.NewStyle()
	if selected {
		style = style.Foreground(lipgloss.Color("205")).Bold(true)
	} else {
		style = style.Foreground(lipgloss.Color("240"))
	}

	fmt.Fprint(w, style.Render(line1)+"\n"+style.Render(line2))
}
