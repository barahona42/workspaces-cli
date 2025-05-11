package main

import (
	"fmt"
	"os"
	"workspaces-cli/models"
	"workspaces-cli/pkg/workspaces"

	tea "github.com/charmbracelet/bubbletea"
)

func fatalf(format string, a ...any) {
	fmt.Printf("\033[31m%s\033[0m", fmt.Errorf(format, a...))
	os.Exit(1)
}

func main() {
	w, err := workspaces.Load(os.ExpandEnv("$HOME/development/workspaces"))
	if err != nil {
		fatalf("load workspaces: %w", err)
	}
	m, err := models.NewModel(w)
	if err != nil {
		fatalf("new model: %w", err)
	}

	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fatalf("run error: %w", err)
		os.Exit(1)
	}
}
