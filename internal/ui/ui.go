package ui

import (
	"fmt"
	"log"
	"os/exec"

	"github.com/LealKevin/keiko/internal/config"
	"github.com/LealKevin/keiko/internal/data"
	"github.com/LealKevin/keiko/internal/service"
)

type statusBar struct {
	svc         service.VocabService
	cfg         *config.Config
	currentWord *data.Word
}

type StatusBarUI interface {
	Init()
	Update(content string)
	Refresh(levels []int) error
	Close()
}

func NewStatusBar(svc service.VocabService, cfg *config.Config) *statusBar {
	return &statusBar{
		svc: svc,
		cfg: cfg,
	}
}

func (s *statusBar) Init() {
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

func (s *statusBar) Redraw() error {
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

func (s *statusBar) Refresh() error {
	levels := s.cfg.UserConfig.JLPTLevel
	word, err := s.svc.GetNextWord(levels)
	if err != nil {
		return err
	}
	s.currentWord = &word

	return s.Redraw()
}

func (s *statusBar) Update(content string) {
	if err := exec.Command("tmux", "set", "-g", "status-format[1]", content).Run(); err != nil {
		log.Printf("tmux update failed: %v", err)
	}
}

func (s *statusBar) Close() {
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
