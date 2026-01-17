package settings

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/LealKevin/keiko/internal/anki"
	"github.com/LealKevin/keiko/internal/config"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type settingsView int

const (
	viewMain settingsView = iota
	viewDeckSelector
)

type field int

const (
	fieldLoopInterval field = iota
	fieldJLPTLevel
	fieldVisibility
	fieldAnkiDeck
	fieldCount
)

type Model struct {
	config *config.Config

	focus            field
	jlptCursor       int
	visibilityCursor int

	visibilityLabels []string

	loopIntervalInput textinput.Model

	currentView       settingsView
	deckCursor        int
	availableDecks    []anki.DeckInfo
	ankiConnected     bool
	ankiClient        *anki.Client
	quitOnDeckSelect  bool // true when opened via --deck-selector
}

func createInput(config *config.Config, field field) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = "10"
	ti.CharLimit = 4
	ti.Width = 10
	ti.SetValue(strconv.Itoa(config.UserConfig.LoopInterval))

	return ti
}

func New(config *config.Config, openDeckSelector bool) *Model {
	loopIntervalInput := createInput(config, fieldLoopInterval)
	visibilityLabels := []string{"Furigana", "Translation", "JLPT Level"}

	ankiClient := anki.NewClient()
	ankiConnected := ankiClient.IsConnected()

	var availableDecks []anki.DeckInfo
	if ankiConnected {
		decks, err := ankiClient.GetDecksWithStats()
		if err == nil {
			availableDecks = decks
		}
	}

	m := &Model{
		config: config,
		focus:  fieldLoopInterval,

		loopIntervalInput: loopIntervalInput,
		visibilityLabels:  visibilityLabels,

		ankiClient:       ankiClient,
		ankiConnected:    ankiConnected,
		availableDecks:   availableDecks,
		quitOnDeckSelect: openDeckSelector,
	}

	if openDeckSelector {
		m.focus = fieldAnkiDeck
		m.currentView = viewDeckSelector
	}

	return m
}

func (m *Model) Init() tea.Cmd {
	return nil
}

type BlurSettingsMsg struct{}

func (m *Model) blurSettings() tea.Cmd {
	return func() tea.Msg {
		return BlurSettingsMsg{}
	}
}

func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.currentView == viewDeckSelector {
			return m.updateDeckSelector(msg)
		}
		return m.updateMainView(msg)
	}
	var cmd tea.Cmd
	m.loopIntervalInput, cmd = m.loopIntervalInput.Update(msg)
	return m, cmd
}

func (m *Model) updateDeckSelector(msg tea.KeyMsg) (*Model, tea.Cmd) {
	switch keypress := msg.String(); keypress {
	case "ctrl+c", "q", "esc":
		if m.quitOnDeckSelect {
			return m, tea.Quit
		}
		m.currentView = viewMain
		return m, nil
	case "down", "j":
		if len(m.availableDecks) > 0 {
			m.deckCursor = min(m.deckCursor+1, len(m.availableDecks)-1)
		}
	case "up", "k":
		m.deckCursor = max(m.deckCursor-1, 0)
	case "enter":
		if len(m.availableDecks) > 0 && m.deckCursor < len(m.availableDecks) {
			m.config.UserConfig.AnkiDeck = m.availableDecks[m.deckCursor].Name
			m.config.UserConfig.AnkiModeEnabled = true
			m.config.Save()
			if m.quitOnDeckSelect {
				return m, tea.Quit
			}
			m.currentView = viewMain
		}
	}
	return m, nil
}

func (m *Model) updateMainView(msg tea.KeyMsg) (*Model, tea.Cmd) {
	switch keypress := msg.String(); keypress {
	case "ctrl+c", "q", "esc":
		return m, m.blurSettings()
	case "down", "j", "m":
		m.focus = min(m.focus+1, fieldCount-1)
	case "up", "k", ",":
		if m.focus == 0 {
			return m, m.blurSettings()
		}
		m.focus = max(m.focus-1, 0)
		return m, nil
	case "right", "l":
		switch m.focus {
		case fieldJLPTLevel:
			if m.jlptCursor == len(JLPTLEVELS)-1 {
				return m, nil
			}
			m.jlptCursor = max(m.jlptCursor+1, 0)
		case fieldVisibility:
			if m.visibilityCursor == len(m.visibilityLabels)-1 {
				return m, nil
			}
			m.visibilityCursor = max(m.visibilityCursor+1, 0)
		}
		return m, nil
	case "left", "h":
		switch m.focus {
		case fieldJLPTLevel:
			if m.jlptCursor == 0 {
				return m, nil
			}
			m.jlptCursor = min(m.jlptCursor-1, len(JLPTLEVELS)-1)
		case fieldVisibility:
			if m.visibilityCursor == 0 {
				return m, nil
			}
			m.visibilityCursor = min(m.visibilityCursor-1, len(m.visibilityLabels)-1)
		}
		return m, nil
	case " ", "enter":
		switch m.focus {
		case fieldJLPTLevel:
			if slices.Contains(m.config.UserConfig.JLPTLevel, JLPTLEVELS[m.jlptCursor]) {
				m.config.UserConfig.JLPTLevel = slices.DeleteFunc(m.config.UserConfig.JLPTLevel, func(i int) bool {
					return i == JLPTLEVELS[m.jlptCursor]
				})
				m.config.Save()
				return m, nil
			} else {
				m.config.UserConfig.JLPTLevel = append(m.config.UserConfig.JLPTLevel, JLPTLEVELS[m.jlptCursor])
				m.config.Save()
				return m, nil
			}
		case fieldVisibility:
			if m.visibilityCursor == 0 {
				m.config.ToggleFurigana()
			} else if m.visibilityCursor == 1 {
				m.config.ToggleTranslation()
			} else if m.visibilityCursor == 2 {
				m.config.ToggleJLPTLevel()
			}
		case fieldAnkiDeck:
			m.currentView = viewDeckSelector
			m.deckCursor = 0
			for i, d := range m.availableDecks {
				if d.Name == m.config.UserConfig.AnkiDeck {
					m.deckCursor = i
					break
				}
			}
		}
		return m, nil
	}
	return m, nil
}

var (
	activeField = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Padding(0, 1)
	inactiveField = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Padding(0, 1)

	JLPTactiveField = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Padding(0, 1)
	JLPTinactiveField = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240")).
				Padding(0, 1)

	JLPTLEVELS = []int{5, 4, 3, 2, 1}
)

func (m *Model) View(focused bool) string {
	if m.currentView == viewDeckSelector {
		return m.renderDeckSelector()
	}
	return m.renderMainView(focused)
}

func (m *Model) renderMainView(focused bool) string {
	var doc strings.Builder

	interval := lipgloss.JoinHorizontal(lipgloss.Center, []string{
		m.renderField("Loop Interval: ", focused && m.focus == fieldLoopInterval),
		m.renderField(m.displayIntervalFormat(m.config.UserConfig.LoopInterval), focused && m.focus == fieldLoopInterval),
	}...)

	jlpt := lipgloss.JoinHorizontal(lipgloss.Center, []string{
		m.renderField("JLPT Level: ", focused && m.focus == fieldJLPTLevel),
		m.renderJLPTField(focused),
	}...)

	visibility := lipgloss.JoinHorizontal(lipgloss.Center, []string{
		m.renderField("Visibility: ", focused && m.focus == fieldVisibility),
		m.renderVisibilityField(focused),
	}...)

	ankiDeck := lipgloss.JoinHorizontal(lipgloss.Center, []string{
		m.renderField("Anki Deck: ", focused && m.focus == fieldAnkiDeck),
		m.renderAnkiDeckField(focused),
	}...)

	doc.WriteString(interval)
	doc.WriteString("\n")
	doc.WriteString(jlpt)
	doc.WriteString("\n")
	doc.WriteString(visibility)
	doc.WriteString("\n")
	doc.WriteString(ankiDeck)

	return doc.String()
}

func (m *Model) renderAnkiDeckField(focused bool) string {
	var display string
	if !m.ankiConnected {
		display = "(Anki not connected)"
	} else if m.config.UserConfig.AnkiDeck == "" {
		display = "(none selected) Press Enter to choose"
	} else {
		dueCount := 0
		for _, d := range m.availableDecks {
			if d.Name == m.config.UserConfig.AnkiDeck {
				dueCount = d.DueCount
				break
			}
		}
		display = fmt.Sprintf("%s (%d due) Press Enter to change", m.config.UserConfig.AnkiDeck, dueCount)
	}

	if focused && m.focus == fieldAnkiDeck {
		return activeField.Render(display)
	}
	return inactiveField.Render(display)
}

func (m *Model) renderDeckSelector() string {
	var doc strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	doc.WriteString(titleStyle.Render("Select Anki Deck"))
	doc.WriteString("\n\n")

	if !m.ankiConnected {
		doc.WriteString(inactiveField.Render("Anki not connected. Please open Anki Desktop."))
		doc.WriteString("\n\n")
		doc.WriteString(inactiveField.Render("Press Esc to go back"))
		return doc.String()
	}

	if len(m.availableDecks) == 0 {
		doc.WriteString(inactiveField.Render("No decks found in Anki."))
		doc.WriteString("\n\n")
		doc.WriteString(inactiveField.Render("Press Esc to go back"))
		return doc.String()
	}

	cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	unselectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	const maxDeckNameLen = 35

	for i, deck := range m.availableDecks {
		isCurrentDeck := deck.Name == m.config.UserConfig.AnkiDeck
		isCursor := i == m.deckCursor

		var prefix string
		if isCursor {
			prefix = "> "
		} else {
			prefix = "  "
		}

		var radio string
		if isCurrentDeck {
			radio = "(●) "
		} else {
			radio = "( ) "
		}

		deckName := truncateAndPad(deck.Name, maxDeckNameLen)
		line := fmt.Sprintf("%s%s%s %3d due", prefix, radio, deckName, deck.DueCount)

		if isCursor {
			doc.WriteString(cursorStyle.Render(line))
		} else if isCurrentDeck {
			doc.WriteString(selectedStyle.Render(line))
		} else {
			doc.WriteString(unselectedStyle.Render(line))
		}
		doc.WriteString("\n")
	}

	doc.WriteString("\n")
	doc.WriteString(inactiveField.Render("↑↓ navigate  Enter select  Esc cancel"))

	return doc.String()
}

func (m *Model) renderField(content string, focused bool) string {
	if focused {
		return activeField.Render(content)
	}
	return inactiveField.Render(content)
}

func (m *Model) renderJLPTField(focused bool) string {
	cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true).Underline(true)
	var levels []string
	for i, level := range JLPTLEVELS {
		isSelected := false
		for _, ul := range m.config.UserConfig.JLPTLevel {
			if level == ul {
				isSelected = true
				break
			}
		}

		str := fmt.Sprintf("N%d", level)

		if focused && m.focus == fieldJLPTLevel && i == m.jlptCursor {
			str = cursorStyle.Render(str)
		}

		if isSelected {
			levels = append(levels, JLPTactiveField.Render(str))
		} else {
			levels = append(levels, JLPTinactiveField.Render(str))
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, levels...)
}

func (m *Model) renderVisibilityField(focused bool) string {
	cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true).Underline(true)
	var visibility []string
	for i, label := range m.visibilityLabels {
		isSelected := false
		if m.config.UserConfig.IsFuriganaVisible && i == 0 {
			isSelected = true
		}
		if m.config.UserConfig.IsTranslationVisible && i == 1 {
			isSelected = true
		}
		if m.config.UserConfig.IsJLPTLevelVisible && i == 2 {
			isSelected = true
		}
		str := fmt.Sprintf("%s", label)

		if focused && m.focus == fieldVisibility && i == m.visibilityCursor {
			str = cursorStyle.Render(str)
		}

		if isSelected {
			visibility = append(visibility, JLPTactiveField.Render(str))
		} else {
			visibility = append(visibility, JLPTinactiveField.Render(str))
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, visibility...)
}

func (m *Model) displayIntervalFormat(seconds int) string {
	var doc strings.Builder
	doc.WriteString(fmt.Sprintf("%02d:%02d", seconds/60, seconds%60))
	return doc.String()
}

func truncateAndPad(s string, width int) string {
	var result []rune
	currentWidth := 0

	for _, r := range s {
		w := runeWidth(r)
		if currentWidth+w > width-2 {
			result = append(result, '.', '.')
			currentWidth += 2
			break
		}
		result = append(result, r)
		currentWidth += w
	}

	for currentWidth < width {
		result = append(result, ' ')
		currentWidth++
	}

	return string(result)
}

func runeWidth(r rune) int {
	if r >= 0x1100 &&
		(r <= 0x115F || r == 0x2329 || r == 0x232A ||
			(r >= 0x2E80 && r <= 0xA4CF && r != 0x303F) ||
			(r >= 0xAC00 && r <= 0xD7A3) ||
			(r >= 0xF900 && r <= 0xFAFF) ||
			(r >= 0xFE10 && r <= 0xFE19) ||
			(r >= 0xFE30 && r <= 0xFE6F) ||
			(r >= 0xFF00 && r <= 0xFF60) ||
			(r >= 0xFFE0 && r <= 0xFFE6) ||
			(r >= 0x20000 && r <= 0x2FFFD) ||
			(r >= 0x30000 && r <= 0x3FFFD)) {
		return 2
	}
	return 1
}
