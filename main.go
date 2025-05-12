package main

import (
	"context"
	"fmt"
	"os"
	"workspaces-cli/models"
	"workspaces-cli/pkg/editors"
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
	m, err := models.NewModel(
		context.Background(),
		w,
		os.ExpandEnv("$HOME/development/workspaces/workspaces.sql"),
		editors.Helix{})
	if err != nil {
		fatalf("new model: %w", err)
	}
	defer m.Cleanup()
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fatalf("run error: %w", err)
		os.Exit(1)
	}
}
