package news

import (
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/LealKevin/keiko/internal/db"
	"github.com/LealKevin/keiko/internal/news"
)

type Mode int

const (
	ModeList Mode = iota
	ModeReading
)

type Model struct {
	client      *news.Client
	db          *db.DB
	list        list.Model
	article     *ArticleView
	translation *TranslationPanel
	mode        Mode
	width       int
	height      int
	currentItem *NewsItem
	offline     bool
	loading     bool
}

type newsListMsg struct {
	items []news.NewsListItem
	err   error
}

type newsDetailMsg struct {
	detail *news.NewsDetail
	err    error
}

func New(client *news.Client, db *db.DB) *Model {
	delegate := NewItemDelegate()
	l := list.New([]list.Item{}, delegate, 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowFilter(false)
	l.SetShowHelp(false)

	return &Model{
		client:      client,
		db:          db,
		list:        l,
		article:     NewArticleView(),
		translation: NewTranslationPanel(),
		mode:        ModeList,
	}
}

func (m *Model) Init() tea.Cmd {
	return m.fetchNewsList()
}

func (m *Model) fetchNewsList() tea.Cmd {
	return func() tea.Msg {
		items, err := m.client.GetNewsList(20, 0)
		return newsListMsg{items: items, err: err}
	}
}

func (m *Model) fetchNewsDetail(id int) tea.Cmd {
	return func() tea.Msg {
		detail, err := m.client.GetNewsDetail(id)
		return newsDetailMsg{detail: detail, err: err}
	}
}

func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case newsListMsg:
		if msg.err != nil {
			m.offline = true
			return m, nil
		}
		m.offline = false

		readIDs, _ := m.db.GetReadNewsIDs()

		items := make([]list.Item, len(msg.items))
		for i, item := range msg.items {
			items[i] = NewsItem{
				ID:          item.ID,
				NhkID:       item.NhkID,
				Title:       item.Title,
				PublishedAt: item.PublishedAt,
				IsRead:      readIDs[item.NhkID],
			}
		}
		m.list.SetItems(items)
		return m, nil

	case newsDetailMsg:
		m.loading = false
		if msg.err != nil {
			return m, nil
		}
		m.article.SetArticle(msg.detail)
		m.mode = ModeReading
		return m, nil
	}

	return m.handleKeyMsg(msg)
}

func (m *Model) handleKeyMsg(msg tea.Msg) (*Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch m.mode {
	case ModeList:
		switch keyMsg.String() {
		case "enter":
			if item, ok := m.list.SelectedItem().(NewsItem); ok {
				m.currentItem = &item
				m.loading = true
				return m, m.fetchNewsDetail(item.ID)
			}
		case "r":
			return m, m.fetchNewsList()
		default:
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)
			return m, cmd
		}

	case ModeReading:
		switch keyMsg.String() {
		case "esc", "q":
			if m.currentItem != nil {
				m.db.MarkNewsAsRead(m.currentItem.NhkID)
				items := m.list.Items()
				for i, item := range items {
					if ni, ok := item.(NewsItem); ok && ni.NhkID == m.currentItem.NhkID {
						ni.IsRead = true
						items[i] = ni
						break
					}
				}
				m.list.SetItems(items)
			}
			m.mode = ModeList
			m.currentItem = nil
			return m, nil
		case "h":
			m.article.MoveLeft()
		case "l":
			m.article.MoveRight()
		case "j":
			m.article.MoveDown()
		case "k":
			m.article.MoveUp()
		}
	}

	return m, nil
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height

	listWidth := 30
	if width > 120 {
		listWidth = 35
	}
	contentWidth := width - listWidth - 1

	m.list.SetSize(listWidth, height-2)
	m.article.SetSize(contentWidth, height-6)
	m.translation.SetWidth(contentWidth)
}

func (m *Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	listWidth := 30
	if m.width > 120 {
		listWidth = 35
	}
	contentWidth := m.width - listWidth - 1

	borderColor := lipgloss.Color("240")
	borderStyle := lipgloss.NewStyle().Foreground(borderColor)

	listStyle := lipgloss.NewStyle().Width(listWidth)
	listView := m.list.View()
	listLines := strings.Split(listStyle.Render(listView), "\n")

	translationHeight := 2
	articleHeight := m.height - translationHeight - 1

	var rightContent string
	if m.offline {
		rightContent = "Cannot connect to news server\nPress 'r' to retry"
	} else if m.loading {
		rightContent = "Loading..."
	} else {
		rightContent = m.article.View()
	}

	articleStyle := lipgloss.NewStyle().Width(contentWidth).Height(articleHeight)
	articleRendered := articleStyle.Render(rightContent)
	articleLines := strings.Split(articleRendered, "\n")

	separator := borderStyle.Render(strings.Repeat("─", contentWidth))

	translationStyle := lipgloss.NewStyle().Width(contentWidth).Height(translationHeight)
	translationRendered := translationStyle.Render(m.translation.View(m.article.SelectedToken()))
	translationLines := strings.Split(translationRendered, "\n")

	var rightLines []string
	rightLines = append(rightLines, articleLines...)
	rightLines = append(rightLines, separator)
	rightLines = append(rightLines, translationLines...)

	var output strings.Builder
	for i := 0; i < m.height; i++ {
		left := ""
		if i < len(listLines) {
			left = listLines[i]
		}
		left = lipgloss.NewStyle().Width(listWidth).Render(left)

		right := ""
		if i < len(rightLines) {
			right = rightLines[i]
		}

		output.WriteString(left)
		output.WriteString(borderStyle.Render("│"))
		output.WriteString(right)
		if i < m.height-1 {
			output.WriteString("\n")
		}
	}

	return output.String()
}

func (m *Model) Mode() Mode {
	return m.mode
}
