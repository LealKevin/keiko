package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/LealKevin/keiko/internal/anki"
	"github.com/LealKevin/keiko/internal/config"
	"github.com/LealKevin/keiko/internal/data"
	"github.com/LealKevin/keiko/internal/db"
	"github.com/LealKevin/keiko/internal/news"
	"github.com/LealKevin/keiko/internal/service"
	"github.com/LealKevin/keiko/internal/tui"
	"github.com/LealKevin/keiko/internal/ui"
	tea "github.com/charmbracelet/bubbletea"

	hook "github.com/robotn/gohook"
)

const (
	KeyF2 = 0x003C // Settings
	KeyF3 = 0x003D // Toggle mode
	KeyF4 = 0x003E // Reveal answer
	KeyF5 = 0x003F // Again (ease 1)
	KeyF6 = 0x0040 // Good (ease 3)
)

var (
	tuiMode          = flag.Bool("tui", false, "Run in TUI mode")
	deckSelectorFlag = flag.Bool("deck-selector", false, "Open directly to deck selector")
)

func main() {
	configDir, _ := os.UserConfigDir()
	appDir := filepath.Join(configDir, "keiko")

	err := os.MkdirAll(appDir, os.ModePerm)
	if err != nil {
		fmt.Println(err)
		panic(err)
	}

	configFilePath := filepath.Join(appDir, "config.yaml")
	dbFilePath := filepath.Join(appDir, "keiko.db")

	c, err := config.New(configFilePath)
	if err != nil {
		fmt.Println(err)
		panic(err)
	}

	err = c.Init()
	if err != nil {
		fmt.Println(err)
		panic(err)
	}

	database, err := db.Open(dbFilePath)
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	defer database.Close()

	err = database.Migrate()
	if err != nil {
		fmt.Println(err)
		panic(err)
	}

	flag.Parse()
	if *tuiMode {
		runTui(c, database)
		return
	}

	allJLPTLevels := []int{1, 2, 3, 4, 5}

	count, err := database.GetWordsCount(allJLPTLevels)
	if err != nil {
		panic(err)
	}

	if count == 0 {
		words, err := data.FetchWords()
		if err != nil {
			panic(err)
		}

		err = database.SeedVocab(words)
		if err != nil {
			fmt.Println(err)
			panic(err)
		}
	}

	service := service.New(database)
	statusBar := ui.NewStatusBar(service, c)
	statusBar.Init()
	statusBar.Refresh()

	fmt.Println("Setup complete!")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	c.Watch()

	go keyboardListener(statusBar, c)

	// Background polling for Anki due count refresh
	go func() {
		for range time.Tick(anki.RefreshInterval) {
			statusBar.RefreshAnkiDueCount()
		}
	}()

	ticker := time.NewTicker(time.Second * time.Duration(c.UserConfig.LoopInterval))
	for {
		select {
		case <-ticker.C:
			if statusBar.Mode() == ui.VocabMode {
				statusBar.Refresh()
			}
			ticker.Reset(time.Second * time.Duration(c.UserConfig.LoopInterval))
		case <-c.Updated:
			ticker.Reset(time.Second * time.Duration(c.UserConfig.LoopInterval))
			statusBar.OnConfigChange()
		case <-sigChan:
			fmt.Println("Exiting...")
			statusBar.Close()
			os.Exit(0)
		}
	}
}

func runTui(cfg *config.Config, database *db.DB) {
	newsClient := news.NewClient(cfg.UserConfig.NewsServerURL)
	tuiModel := tui.New(cfg, database, newsClient, *deckSelectorFlag)
	if _, err := tea.NewProgram(tuiModel, tea.WithAltScreen()).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}

func openTui() {
	path, err := os.Executable()
	if err != nil {
		fmt.Println("Error getting executable path:", err)
		return
	}
	if err := exec.Command("tmux", "display-popup", "-w", "80%", "-h", "80%", "-E", path, "--tui").Run(); err != nil {
		fmt.Printf("tmux popup failed: %v\n", err)
	}
}

func openDeckSelector() {
	path, err := os.Executable()
	if err != nil {
		fmt.Println("Error getting executable path:", err)
		return
	}
	if err := exec.Command("tmux", "display-popup", "-w", "80%", "-h", "80%", "-E", path, "--tui", "--deck-selector").Run(); err != nil {
		fmt.Printf("tmux popup failed: %v\n", err)
	}
}

func keyboardListener(statusBar *ui.StatusBar, cfg *config.Config) {
	evChan := hook.Start()
	defer hook.End()

	for ev := range evChan {
		if ev.Kind != hook.KeyDown {
			continue
		}

		switch ev.Keycode {
		case KeyF2:
			openTui()
		case KeyF3:
			if statusBar.NeedsDeckSelector() {
				openDeckSelector()
			} else {
				statusBar.ToggleMode()
			}
		case KeyF4:
			statusBar.RevealAnswer()
		case KeyF5:
			statusBar.AnswerCard(1) // Again
		case KeyF6:
			statusBar.AnswerCard(3) // Good
		}
	}
}
