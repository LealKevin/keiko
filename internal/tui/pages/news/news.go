package news

import (
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

	listWidth := width / 4
	contentWidth := width - listWidth - 3

	m.list.SetSize(listWidth, height-2)
	m.article.SetSize(contentWidth, height-6)
	m.translation.SetWidth(contentWidth)
}

func (m *Model) View() string {
	listWidth := m.width / 4
	contentWidth := m.width - listWidth - 3

	listStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderRight(true).
		Width(listWidth).
		Height(m.height - 2)

	leftPane := listStyle.Render(m.list.View())

	var rightPane string
	if m.offline {
		rightPane = "Cannot connect to news server\nPress 'r' to retry"
	} else if m.loading {
		rightPane = "Loading..."
	} else {
		articleView := m.article.View()
		translationView := m.translation.View(m.article.SelectedToken())
		rightPane = lipgloss.JoinVertical(lipgloss.Left, articleView, translationView)
	}

	rightStyle := lipgloss.NewStyle().
		Width(contentWidth).
		Height(m.height - 2)

	rightPane = rightStyle.Render(rightPane)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
}

func (m *Model) Mode() Mode {
	return m.mode
}
