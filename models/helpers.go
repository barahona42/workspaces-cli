package models

import (
	"strings"
	"time"
	"workspaces-cli/pkg/workspaces"
)

func isWorkspaceNameMatch(w *workspaces.Workspace, s string) bool {
	return strings.EqualFold(w.DirEntry.Name(), s) ||
		strings.Contains(w.DirEntry.Name(), s)
}

func isWorkspaceModTimeMatch(w *workspaces.Workspace, s string) bool {
	return strings.Contains(w.ModTime().Format(time.DateOnly), s)
}

func isFuzzyWorkspaceMatch(w *workspaces.Workspace, s string) bool {
	return isWorkspaceNameMatch(w, s) || isWorkspaceModTimeMatch(w, s)
}
