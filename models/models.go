package models

import (
	"context"
	"fmt"
	"slices"
	"time"
	"workspaces-cli/models/db"
	"workspaces-cli/pkg/editors"
	"workspaces-cli/pkg/workspaces"

	"golang.design/x/clipboard"
)

const (
	MESSAGE_TIMEOUT time.Duration = 2 * time.Second
)
const (
	DURATION_ONE_DAY     time.Duration = 24 * time.Hour
	DURATION_ONE_WEEK    time.Duration = 7 * DURATION_ONE_DAY
	DURATION_THIRTY_DAYS time.Duration = 30 * DURATION_ONE_DAY
)

var (
	modtimeColors map[time.Duration]string = map[time.Duration]string{
		DURATION_ONE_DAY:     "34", // blue
		DURATION_ONE_WEEK:    "32", // green
		DURATION_THIRTY_DAYS: "33", // yello
	}
)

func modtimeColor(t time.Time) string {
	switch t := time.Since(t); {
	case t < DURATION_ONE_DAY:
		return modtimeColors[DURATION_ONE_DAY]
	case t < DURATION_ONE_WEEK:
		return modtimeColors[DURATION_ONE_WEEK]
	case t < DURATION_THIRTY_DAYS:
		return modtimeColors[DURATION_THIRTY_DAYS]
	}
	return "31"
}

func modtimeColorize(t time.Time) string {
	return fmt.Sprintf("\033[%sm%s\033[0m", modtimeColor(t), t.In(time.Local).Format(time.DateOnly))
}

func sortWorkspaces(w []workspaces.Workspace) []workspaces.Workspace {
	ww := append([]workspaces.Workspace{}, w...)
	slices.SortFunc(ww, func(a, b workspaces.Workspace) int {
		if a.DirEntry.Name() < b.DirEntry.Name() {
			return -1
		} else if a.DirEntry.Name() > b.DirEntry.Name() {
			return 1
		}
		return 0
	})
	return ww
}

func NewModel(ctx context.Context, w []workspaces.Workspace, dbfile string, editor editors.Editor) (*Application, error) {
	if err := clipboard.Init(); err != nil {
		return nil, fmt.Errorf("initialize clipboard: %w", err)
	}
	if err := db.Open(ctx, dbfile); err != nil {
		return nil, fmt.Errorf("connect db: %w", err)
	}
	// TODO: terminal height for maxrows
	return &Application{
		workspaces: sortWorkspaces(w),
		maxrows:    10,
		commands:   []string{"add_checkpoint", "view_checkpoints"},
		editor:     editor}, nil
}
