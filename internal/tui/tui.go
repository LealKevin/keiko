package tui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type focus int

const (
	focusTabs focus = iota
	focusContainer
)

type model struct {
	Tabs       []string
	TabContent []string
	activeTab  int
	focus      focus

	width  int
	height int
}

func New() *model {
	return &model{
		Tabs:       []string{"Home", "Settings"},
		TabContent: []string{"HomeTab", "SettingsTab"},
		focus:      focusTabs,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit
		case "right", "l", "n", "tab":
			if m.focus == focusTabs {
				m.activeTab = min(m.activeTab+1, len(m.Tabs)-1)
			}
			return m, nil
		case "left", "h", "p", "shift+tab":
			if m.focus == focusTabs {
				m.activeTab = max(m.activeTab-1, 0)
			}
			return m, nil
		case "down", "j", "m":
			m.focus = focusContainer

			return m, nil
		case "up", "k", ",":
			m.focus = focusTabs
			return m, nil
		}
	}

	return m, nil
}

var (
	activeColor = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Padding(0, 1)
	inactiveColor = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Padding(0, 1)

	container = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Margin(1, 1)

	containerActive = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("205")).
			Margin(1, 1)

	containerMargin = 10
)

func (m model) View() string {
	var doc strings.Builder

	if m.focus == focusContainer {
		for _, tab := range m.Tabs {
			doc.WriteString(inactiveColor.Render(tab))
		}
	} else {
		for i, tab := range m.Tabs {
			if i == m.activeTab {
				doc.WriteString(activeColor.Render(tab))
				continue
			}
			doc.WriteString(inactiveColor.Render(tab))

		}
	}

	doc.WriteString("\n")

	if m.focus == focusTabs {
		doc.WriteString(container.Width(m.width - containerMargin).Height(m.height - containerMargin).Render(m.TabContent[m.activeTab]))
	} else {
		doc.WriteString(containerActive.Width(m.width - 10).Height(m.height - 10).Render(m.TabContent[m.activeTab]))
	}

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		0,
		doc.String(),
	)
}

func (m model) Run() {
	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
