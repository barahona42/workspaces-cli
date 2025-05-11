package models

import tea "github.com/charmbracelet/bubbletea"

type workspacelistcmd string
type messagecallback struct {
	message  string
	callback tea.Cmd
}
