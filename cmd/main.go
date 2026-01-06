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

	"github.com/LealKevin/keiko/internal/config"
	"github.com/LealKevin/keiko/internal/data"
	"github.com/LealKevin/keiko/internal/db"
	"github.com/LealKevin/keiko/internal/service"
	"github.com/LealKevin/keiko/internal/tui"
	"github.com/LealKevin/keiko/internal/ui"
	tea "github.com/charmbracelet/bubbletea"

	hook "github.com/robotn/gohook"
)

var tuiMode = flag.Bool("tui", false, "Run in TUI mode")

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

	flag.Parse()
	if *tuiMode {
		runTui(c)
		return
	}

	db, err := db.Open(dbFilePath)
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	defer db.Close()

	err = db.Migrate()
	if err != nil {
		fmt.Println(err)
		panic(err)
	}

	allJLPTLevels := []int{1, 2, 3, 4, 5}

	count, err := db.GetWordsCount(allJLPTLevels)
	if err != nil {
		panic(err)
	}

	if count == 0 {
		words, err := data.FetchWords()
		if err != nil {
			panic(err)
		}

		err = db.SeedVocab(words)
		if err != nil {
			fmt.Println(err)
			panic(err)
		}
	}

	service := service.New(db)
	statusBar := ui.NewStatusBar(service)
	statusBar.Init()
	statusBar.Refresh(c.UserConfig.JLPTLevel)

	fmt.Println("Setup complete!")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	c.Watch()

	go keyboardListener()

	go func() {
		ticker := time.NewTicker(time.Minute * time.Duration(c.UserConfig.LoopInterval))
		for {
			select {
			case <-ticker.C:
				fmt.Println("Interval is", c.UserConfig.LoopInterval)
				fmt.Printf("Interval from viper: %d", c.Viper.GetInt("loop_interval"))
				statusBar.Refresh(c.UserConfig.JLPTLevel)
				ticker.Reset(time.Minute * time.Duration(c.UserConfig.LoopInterval))
			case <-c.Updated:
				fmt.Println("Config changed! Reloading...")
				fmt.Println("New interval:", c.UserConfig.LoopInterval)
				ticker.Reset(time.Minute * time.Duration(c.UserConfig.LoopInterval))
			case <-sigChan:
				fmt.Println("Exiting...")
				statusBar.Close()
				os.Exit(0)
			}
		}
	}()
	<-sigChan

	fmt.Println("Exiting...")
	statusBar.Close()
	os.Exit(0)
}

func runTui(config *config.Config) {
	tuiModel := tui.New(config)
	if _, err := tea.NewProgram(tuiModel, tea.WithAltScreen()).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}

func openTui() {
	path, err := os.Executable()
	if err != nil {
		fmt.Println("tmux not found")
		return
	}
	exec.Command("tmux", "display-popup", "-w", "80%", "-h", "80%", "-E", path, "--tui").Run()
}

func keyboardListener() {
	evChan := hook.Start()
	defer hook.End()

	for ev := range evChan {
		if ev.Kind == hook.KeyDown && ev.Rawcode == 120 {
			openTui()
		}
	}
}
