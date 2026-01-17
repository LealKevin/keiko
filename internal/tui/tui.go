package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/LealKevin/keiko/internal/config"
	"github.com/LealKevin/keiko/internal/tui/pages/settings"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type focus int

const (
	focusTabs focus = iota
	focusContainer
)

type model struct {
	config *config.Config

	settings *settings.Model

	Tabs       []string
	TabContent []string

	activeTab int
	focus     focus

	width  int
	height int
}

func New(config *config.Config, openDeckSelector bool) *model {
	settingsModel := settings.New(config, openDeckSelector)

	m := &model{
		Tabs:       []string{"Home", "Settings"},
		TabContent: []string{"HomeTab", "SettingsTab"},
		focus:      focusTabs,

		settings: settingsModel,

		config: config,
	}

	if openDeckSelector {
		m.activeTab = 1
		m.focus = focusContainer
	}

	return m
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case settings.BlurSettingsMsg:
		m.focus = focusTabs
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if m.focus == focusContainer {
			if m.activeTab == 1 { // settings tab
				_, cmd := m.settings.Update(msg)
				return m, cmd
			}
		}

		switch keypress := msg.String(); keypress {
		case "ctrl+c", "q", "esc":
			if m.focus == focusContainer {
				m.activeTab = 0
				m.focus = focusTabs
				return m, tea.Quit
			}
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
	containerMargin = 10
)

func (m model) View() string {
	var content string
	switch m.activeTab {
	case 0:
		content = m.TabContent[0]
	case 1:
		content = m.settings.View(m.focus == focusContainer)
	}

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

	line := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render(strings.Repeat("─", m.width))
	doc.WriteString("\n" + line + "\n")

	if m.focus == focusTabs {
		doc.WriteString(content)
	} else {
		doc.WriteString(content)
	}

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Left,
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
