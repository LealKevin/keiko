package settings

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/LealKevin/keiko/internal/config"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type field int

const (
	fieldLoopInterval field = iota
	fieldJLPTLevel
	fieldCount
)

type Model struct {
	config *config.Config

	focus      field
	jlptCursor int

	loopIntervalInput textinput.Model
}

func createInput(config *config.Config, field field) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = "10"
	ti.CharLimit = 4
	ti.Width = 10
	ti.SetValue(strconv.Itoa(config.UserConfig.LoopInterval))

	return ti
}

func New(config *config.Config) *Model {
	loopIntervalInput := createInput(config, fieldLoopInterval)

	return &Model{
		config: config,
		focus:  fieldLoopInterval,

		loopIntervalInput: loopIntervalInput,
	}
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
			if m.focus == fieldJLPTLevel {
				if m.jlptCursor == len(JLPTLEVELS)-1 {
					return m, nil
				}
				m.jlptCursor = max(m.jlptCursor+1, 0)
			}

			if m.focus == fieldLoopInterval {
				m.config.IncreaseInterval()
				m.config.Save()
			}
			return m, nil
		case "left", "h":
			if m.focus == fieldJLPTLevel {
				if m.jlptCursor == 0 {
					return m, nil
				}
				m.jlptCursor = min(m.jlptCursor-1, len(JLPTLEVELS)-1)
			}

			if m.focus == fieldLoopInterval {
				m.config.DecreaseInterval()
				m.config.Save()
			}
			return m, nil
		case " ", "enter":
			if m.focus == fieldJLPTLevel {
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
			}
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.loopIntervalInput, cmd = m.loopIntervalInput.Update(msg)
	return m, cmd
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
	var doc strings.Builder

	interval := lipgloss.JoinHorizontal(lipgloss.Center, []string{
		m.renderField("Loop Interval: ", focused && m.focus == fieldLoopInterval),
		m.renderField(m.displayIntervalFormat(m.config.UserConfig.LoopInterval), focused && m.focus == fieldLoopInterval),
	}...)

	jlpt := lipgloss.JoinHorizontal(lipgloss.Center, []string{
		m.renderField("JLPT Level: ", focused && m.focus == fieldJLPTLevel),
		m.renderJLPTField(focused),
	}...)

	doc.WriteString(interval)
	doc.WriteString("\n")
	doc.WriteString(jlpt)

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

func (m *Model) displayIntervalFormat(seconds int) string {
	var doc strings.Builder
	doc.WriteString(fmt.Sprintf("%02d:%02d", seconds/60, seconds%60))
	return doc.String()
}
