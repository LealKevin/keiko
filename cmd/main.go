package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/LealKevin/keiko/internal/data"
	"github.com/LealKevin/keiko/internal/db"
	"github.com/LealKevin/keiko/internal/service"
	"github.com/LealKevin/keiko/internal/ui"

	hook "github.com/robotn/gohook"
)

func main() {
	db, err := db.Open("keiko.db")
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

	count, err := db.GetWordsCount()
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

	fmt.Println("Setup complete!")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go keyboardListener()

	go func() {
		for {
			statusBar.Refresh([]int{5})
			time.Sleep(time.Minute * 1)
		}
	}()
	<-sigChan

	fmt.Println("Exiting...")
	statusBar.Close()
	os.Exit(0)
}

// tmux display-popup -w 80% -h 80% -E ~kevin/repos/keiko/tui
func openTui() {
	path, err := os.Executable()
	exeDir := filepath.Dir(path)
	path = filepath.Join(exeDir, "tui")
	fmt.Println("Heres the path", path)

	if err != nil {
		fmt.Println("tmux not found")
		return
	}
	exec.Command("tmux", "display-popup", "-w", "80%", "-h", "80%", "-E", path).Run()
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
