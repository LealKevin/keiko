package ui

import (
	"fmt"
	"os/exec"

	"github.com/LealKevin/keiko/internal/service"
)

type statusBar struct {
	svc service.VocabService
}

type StatusBarUI interface {
	Init()
	Update(content string)
	Refresh(levels []int) error
	Close()
}

func NewStatusBar(svc service.VocabService) StatusBarUI {
	return &statusBar{
		svc: svc,
	}
}

func (s *statusBar) Init() {
	exec.Command("tmux", "set", "-g", "status", "2").Run()
	exec.Command("tmux", "set", "-g", "status-format[1]", "").Run()
}

const (
	fillColor = "#1a1a2e"
	bgColor   = "#1a1a2e"
	fgColor   = "#e94560"
	align     = "centre"
)

func (s *statusBar) Refresh(levels []int) error {
	word, err := s.svc.GetNextWord(levels)
	if err != nil {
		return err
	}

	content := fmt.Sprintf("#[fill=%s bg=%s,fg=%s,align=%s] %s (%s) — %s — JLPT N%d", fillColor, bgColor, fgColor, align, word.Word, word.Furigana, word.Meaning, word.Level)
	s.Update(content)

	return nil
}

func (s *statusBar) Update(content string) {
	exec.Command("tmux", "set", "-g", "status-format[1]", content).Run()
}

func (s *statusBar) Close() {
	exec.Command("tmux", "set", "-g", "status", "1").Run()
	exec.Command("tmux", "set", "-g", "status-format[1]", "").Run()
	exec.Command("tmux", "set", "-g", "status", "on").Run()
	fmt.Println("UI closed")
}
