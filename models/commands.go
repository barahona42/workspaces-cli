package models

import (
	tea "github.com/charmbracelet/bubbletea"
)

// renderpanescmd: base command
type renderpanescmd struct {
	main   string
	footer string
}

// renderpaneswithcallbackcmd: provides a message for rendering and a post-rendering callback
type renderpaneswithcallbackcmd struct {
	renderpanescmd
	callback tea.Cmd
}

type addcheckpointcmd string

type viewcheckpointscmd string

type errormessage struct {
	err error
}
