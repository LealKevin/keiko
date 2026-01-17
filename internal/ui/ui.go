package ui

import (
	"fmt"
	"log"
	"os/exec"

	"github.com/LealKevin/keiko/internal/anki"
	"github.com/LealKevin/keiko/internal/config"
	"github.com/LealKevin/keiko/internal/data"
	"github.com/LealKevin/keiko/internal/service"
)

type Mode int

const (
	VocabMode Mode = iota
	AnkiMode
)

type AnkiState int

const (
	StateQuestion AnkiState = iota
	StateAnswer
	StateDisconnected
	StateDone
	StateNoDeck
)

type StatusBar struct {
	svc         service.VocabService
	cfg         *config.Config
	currentWord *data.Word

	mode        Mode
	ankiClient  *anki.Client
	ankiState   AnkiState
	currentCard *anki.CardInfo
	dueCards    []int64
	dueCount    int
}

type StatusBarUI interface {
	Init()
	Update(content string)
	Refresh(levels []int) error
	Close()
}

func NewStatusBar(svc service.VocabService, cfg *config.Config) *StatusBar {
	sb := &StatusBar{
		svc:        svc,
		cfg:        cfg,
		ankiClient: anki.NewClient(),
	}

	if cfg.UserConfig.AnkiModeEnabled {
		sb.mode = AnkiMode
		if cfg.UserConfig.AnkiDeck == "" {
			sb.ankiState = StateNoDeck
		} else if sb.ankiClient.IsConnected() {
			sb.fetchAnkiCards()
		} else {
			sb.ankiState = StateDisconnected
		}
	}

	return sb
}

func (s *StatusBar) Init() {
	if err := exec.Command("tmux", "set", "-g", "status", "2").Run(); err != nil {
		log.Printf("tmux status set failed: %v", err)
	}
	if err := exec.Command("tmux", "set", "-g", "status-format[1]", "").Run(); err != nil {
		log.Printf("tmux status-format set failed: %v", err)
	}
}

const (
	fillColor = "#1a1a2e"
	bgColor   = "#1a1a2e"
	fgColor   = "#e94560"
	align     = "centre"
)

func (s *StatusBar) Redraw() error {
	if s.mode == AnkiMode {
		return s.redrawAnki()
	}
	return s.redrawVocab()
}

func (s *StatusBar) redrawVocab() error {
	if s.currentWord == nil {
		return nil
	}
	word := s.currentWord

	translation := ""
	if s.cfg.UserConfig.IsTranslationVisible {
		translation = word.Meaning
	}

	furigana := ""
	if s.cfg.UserConfig.IsFuriganaVisible {
		furigana = fmt.Sprintf("【%s】", word.Furigana)
	}

	jlptLevel := ""
	if s.cfg.UserConfig.IsJLPTLevelVisible {
		jlptLevel = fmt.Sprintf("JLPT N%d", word.Level)
	}

	content := fmt.Sprintf("#[fill=%s bg=%s,fg=%s,align=%s] %s %s  %s %s", fillColor, bgColor, fgColor, align, word.Word, furigana, translation, jlptLevel)
	s.Update(content)

	return nil
}

func (s *StatusBar) redrawAnki() error {
	var left, center, right string

	switch s.ankiState {
	case StateNoDeck:
		left = "[Anki: no deck]"
		center = "Select deck in settings (F2)"
	case StateDisconnected:
		left = "[Anki: disconnected]"
		center = "Open Anki Desktop"
	case StateDone:
		left = s.formatPrefix()
		center = "All caught up!"
	case StateQuestion:
		if s.currentCard == nil {
			center = "[Anki: loading...]"
		} else {
			left = s.formatPrefix()
			center = truncateRunes(s.currentCard.Question, 40)
			right = "[F4]"
		}
	case StateAnswer:
		if s.currentCard == nil {
			center = "[Anki: loading...]"
		} else {
			left = s.formatPrefix()
			// Show word with reading and meaning
			wordWithReading := s.currentCard.Question
			if s.currentCard.Reading != "" {
				wordWithReading = fmt.Sprintf("%s【%s】", s.currentCard.Question, s.currentCard.Reading)
			}
			center = fmt.Sprintf("%s - %s", truncateRunes(wordWithReading, 30), truncateRunes(s.currentCard.Answer, 25))
			right = "[F5 ✗ | F6 ✓]"
		}
	}

	// Build tmux format with left/center/right alignment
	content := fmt.Sprintf("#[fill=%s,bg=%s,fg=%s]#[align=left] %s #[align=centre]%s#[align=right]%s ",
		fillColor, bgColor, fgColor, left, center, right)
	s.Update(content)
	return nil
}

func (s *StatusBar) Refresh() error {
	levels := s.cfg.UserConfig.JLPTLevel
	word, err := s.svc.GetNextWord(levels)
	if err != nil {
		return err
	}
	s.currentWord = &word

	return s.Redraw()
}

func (s *StatusBar) Update(content string) {
	if err := exec.Command("tmux", "set", "-g", "status-format[1]", content).Run(); err != nil {
		log.Printf("tmux update failed: %v", err)
	}
}

func (s *StatusBar) Close() {
	if err := exec.Command("tmux", "set", "-g", "status", "1").Run(); err != nil {
		log.Printf("tmux close status set failed: %v", err)
	}
	if err := exec.Command("tmux", "set", "-g", "status-format[1]", "").Run(); err != nil {
		log.Printf("tmux close format set failed: %v", err)
	}
	if err := exec.Command("tmux", "set", "-g", "status", "on").Run(); err != nil {
		log.Printf("tmux close status on failed: %v", err)
	}
	fmt.Println("UI closed")
}

func truncateRunes(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) > maxRunes {
		return string(runes[:maxRunes-3]) + "..."
	}
	return s
}

func (s *StatusBar) formatPrefix() string {
	deckName := s.cfg.UserConfig.AnkiDeck
	maxDeckLen := 15
	runes := []rune(deckName)
	if len(runes) > maxDeckLen {
		deckName = string(runes[:maxDeckLen-2]) + ".."
	}
	return fmt.Sprintf("[%s: %d due]", deckName, s.dueCount)
}

func (s *StatusBar) fetchAnkiCards() {
	cards, err := s.ankiClient.GetDueCards(s.cfg.UserConfig.AnkiDeck)
	if err != nil {
		s.ankiState = StateDisconnected
		return
	}

	s.dueCards = cards

	dueCount, err := s.ankiClient.GetDueCount(s.cfg.UserConfig.AnkiDeck)
	if err != nil {
		s.dueCount = len(cards)
	} else {
		s.dueCount = dueCount
	}

	if len(cards) == 0 {
		s.ankiState = StateDone
		s.currentCard = nil
		return
	}

	s.fetchNextCard()
}

func (s *StatusBar) fetchNextCard() {
	if len(s.dueCards) == 0 {
		if s.dueCount > 0 {
			s.fetchAnkiCards()
			return
		}
		s.ankiState = StateDone
		s.currentCard = nil
		return
	}

	cardID := s.dueCards[0]
	s.dueCards = s.dueCards[1:]

	card, err := s.ankiClient.GetCardInfo(cardID)
	if err != nil {
		s.ankiState = StateDisconnected
		return
	}

	s.currentCard = card
	s.ankiState = StateQuestion
}

func (s *StatusBar) Mode() Mode {
	return s.mode
}

func (s *StatusBar) AnkiState() AnkiState {
	return s.ankiState
}

func (s *StatusBar) ToggleMode() {
	if s.cfg.UserConfig.AnkiDeck == "" {
		return
	}

	if s.mode == VocabMode {
		s.mode = AnkiMode
		if s.ankiClient.IsConnected() {
			s.fetchAnkiCards()
		} else {
			s.ankiState = StateDisconnected
		}
	} else {
		s.mode = VocabMode
	}

	s.cfg.UserConfig.AnkiModeEnabled = (s.mode == AnkiMode)
	s.cfg.Save()
	s.Redraw()
}

func (s *StatusBar) NeedsDeckSelector() bool {
	return s.cfg.UserConfig.AnkiDeck == ""
}

func (s *StatusBar) RevealAnswer() {
	if s.mode != AnkiMode || s.ankiState != StateQuestion {
		return
	}
	s.ankiState = StateAnswer
	s.Redraw()
}

func (s *StatusBar) AnswerCard(ease int) {
	if s.mode != AnkiMode || s.ankiState != StateAnswer || s.currentCard == nil {
		return
	}

	err := s.ankiClient.AnswerCard(s.currentCard.CardID, ease)
	if err != nil {
		s.ankiState = StateDisconnected
		s.Redraw()
		return
	}

	if count, err := s.ankiClient.GetDueCount(s.cfg.UserConfig.AnkiDeck); err == nil {
		s.dueCount = count
	}
	s.fetchNextCard()
	s.Redraw()
}

func (s *StatusBar) RefreshAnkiDueCount() {
	if s.mode != AnkiMode || s.ankiState == StateDisconnected {
		return
	}

	count, err := s.ankiClient.GetDueCount(s.cfg.UserConfig.AnkiDeck)
	if err != nil {
		s.ankiState = StateDisconnected
		s.Redraw()
		return
	}
	s.dueCount = count
}

func (s *StatusBar) OnConfigChange() {
	if s.mode == AnkiMode && s.cfg.UserConfig.AnkiDeck != "" {
		s.fetchAnkiCards()
	}
	s.Redraw()
}

func (s *StatusBar) AnkiClient() *anki.Client {
	return s.ankiClient
}
