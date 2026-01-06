package tui

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/LealKevin/keiko/internal/config"
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

	Tabs       []string
	TabContent []string

	activeTab int
	focus     focus

	width  int
	height int
}

func New(config *config.Config) *model {
	return &model{
		Tabs:       []string{"Home", "Settings"},
		TabContent: []string{"HomeTab", "SettingsTab"},
		focus:      focusTabs,

		config: config,
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

		// testing purposes
		case "t", "T":
			m.config.UserConfig.LoopInterval++
			m.config.Save()
			return m, nil
		case "s", "S":
			m.config.UserConfig.LoopInterval--
			m.config.Save()
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

	containerInactive = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("240")).
				Margin(1, 1)

	containerActive = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("205")).
			Margin(1, 1)

	containerMargin = 10
)

func (m model) settingsView() string {
	var doc strings.Builder

	doc.WriteString(lipgloss.NewStyle().Bold(true).Render("Settings"))
	doc.WriteString("\n\n")

	label := lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("Loop Interval: ")
	value := lipgloss.NewStyle().Foreground(lipgloss.Color("81")).Render(strconv.Itoa(m.config.UserConfig.LoopInterval))

	doc.WriteString(label + value)
	doc.WriteString("\n\n")
	doc.WriteString(lipgloss.NewStyle().Italic(true).Render("(Press T to increase, S to decrease)"))

	return doc.String()
}

func (m model) View() string {
	var content string
	switch m.activeTab {
	case 0:
		content = m.TabContent[0]
	case 1:
		content = m.settingsView()
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

	doc.WriteString("\n")

	if m.focus == focusTabs {
		doc.WriteString(containerInactive.Width(m.width - containerMargin).Height(m.height - containerMargin).Render(content))
	} else {
		doc.WriteString(containerActive.Width(m.width - 10).Height(m.height - 10).Render(content))
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
