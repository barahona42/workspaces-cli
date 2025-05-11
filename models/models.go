package models

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
	"workspaces-cli/pkg/workspaces"

	tea "github.com/charmbracelet/bubbletea"
	"golang.design/x/clipboard"
)

const (
	MESSAGE_TIMEOUT time.Duration = 2 * time.Second
)

type StringWithCallback struct {
	String   string
	Callback tea.Cmd
}

type Model struct {
	workspaces []workspaces.Workspace
	cursor     int
	display    string
	maxrows    int

	filterValue        string
	isFilterActive     bool
	filteredWorkspaces []workspaces.Workspace
}

func (m *Model) getCursorMax() int {
	if m.isFilterActive {
		if len(m.filteredWorkspaces) > 0 {
			return len(m.filteredWorkspaces) - 1
		}
		return 0
	}
	if len(m.workspaces) > 0 {
		return len(m.workspaces) - 1
	}
	return 0
}

func (m *Model) cursorUp() {
	if m.cursor > 0 {
		m.cursor--
	} else {
		m.cursor = m.getCursorMax()
	}
}
func (m *Model) cursorDown() {
	if m.cursor < m.getCursorMax() {
		m.cursor++
	} else {
		m.cursor = 0
	}
}

func (m *Model) getWorkspaces() []workspaces.Workspace {
	if m.isFilterActive && len(m.filterValue) > 0 {
		return m.filteredWorkspaces
	}
	return m.workspaces
}

func (m *Model) generateWorkspaceString(pos, namepadding int, selected bool, w workspaces.Workspace) string {
	var (
		cursor  string = ""
		name    string = ""
		path    string = ""
		modtime string = modtimeColorize(w.ModTime())
	)
	if selected {
		cursor = "\033[34m>\033[0m"
		name = fmt.Sprintf("\033[34m%-*s\033[0m", namepadding, w.DirEntry.Name())
		path = fmt.Sprintf("\033[90m%s\033[0m", w.Path())
	} else {
		cursor = " "
		name = fmt.Sprintf("%-*s", namepadding, w.DirEntry.Name())
	}
	return fmt.Sprintf("%s %3d   %s   %s   %s", cursor, pos, name, modtime, path)
}

func (m *Model) generateWorkspacesString() string {
	ws := m.getWorkspaces()
	var maxnamelen *int = new(int)
	*maxnamelen = 0
	strs := make([]func(int) string, len(ws))
	for i := range ws {
		if l := len(ws[i].DirEntry.Name()); l > *maxnamelen {
			*maxnamelen = l
		}
		strs[i] = func(namepadding int) string {
			return m.generateWorkspaceString(i+1, namepadding, m.cursor == i, ws[i])
		}

	}
	b := strings.Builder{}
	for i := 0; i < m.maxrows && i+m.cursor < len(strs); i++ {
		b.WriteString(strs[i+m.cursor](*maxnamelen) + "\n")
	}
	b.WriteString("\n\n----------\n")
	if m.isFilterActive || len(m.filterValue) > 0 {
		b.WriteString(fmt.Sprintf("FILTER > %s", m.filterValue))
	}
	b.WriteString("\033[90m\n")
	b.WriteString("   type 'c' to copy selected path to clipboard\n")
	b.WriteString("   type 'o' to open selected workspace in vscode\n")
	b.WriteString("\033[0m\n")
	b.WriteString("\n")
	return b.String()
}

func (m *Model) renderWorkspaces() tea.Msg {
	return workspaceList(m.generateWorkspacesString())
}

func (m *Model) setFilteredWorkspaces(filter string) {
	m.filteredWorkspaces = make([]workspaces.Workspace, 0, len(m.workspaces))
	for i := range m.workspaces {
		if strings.Contains(strings.ToLower(m.workspaces[i].DirEntry.Name()), strings.ToLower(filter)) {
			m.filteredWorkspaces = append(m.filteredWorkspaces, m.workspaces[i])
		}
	}
}

func (m *Model) handleCommandInput(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.Type {
	case tea.KeyBackspace:
		if len(m.filterValue) > 0 {
			m.filterValue = m.filterValue[:len(m.filterValue)-1]
		}
		return m, func() tea.Msg { return m.renderWorkspaces() }
	case tea.KeyEsc:
		m.isFilterActive = false
		m.filterValue = ""
		clear(m.filteredWorkspaces)
		return m, func() tea.Msg {
			return m.renderWorkspaces()
		}
	case tea.KeyEnter, tea.KeyLeft, tea.KeyRight:
	default:
		m.filterValue += key.String()
		m.cursor = 0
		m.setFilteredWorkspaces(m.filterValue)
	}
	return m, func() tea.Msg { return m.renderWorkspaces() }
}

func (m *Model) handleKeyString(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "q":
		return m, tea.Quit
	case "c": // clip workspace path
		return m, func() tea.Msg {
			clipboard.Write(clipboard.FmtText, []byte(m.workspaces[m.cursor].Path()))
			return StringWithCallback{
				String: "(i) copied message to clipboard",
				Callback: func() tea.Msg {
					time.Sleep(MESSAGE_TIMEOUT)
					return m.renderWorkspaces()
				},
			}
		}
	case "o": // open workspace path
		return m, func() tea.Msg {
			done := make(chan struct{}, 1)
			go func() {
				exec.Command("code", m.workspaces[m.cursor].Path()).Run()
				done <- struct{}{}
			}()
			return StringWithCallback{
				String: "(i) opening workspace",
				Callback: func() tea.Msg {
					time.Sleep(MESSAGE_TIMEOUT)
					<-done
					return m.renderWorkspaces()
				},
			}
		}
	case "/": // filter command
		m.cursor = 0
		m.filterValue = ""
		m.isFilterActive = true
		return m, func() tea.Msg { return m.renderWorkspaces() }
	}
	return m, nil
}

func (m *Model) handleKeyMsg(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyUp:
		m.cursorUp()
		return m, func() tea.Msg { return m.renderWorkspaces() }
	case tea.KeyDown:
		m.cursorDown()
		return m, func() tea.Msg { return m.renderWorkspaces() }
	default:
		if m.isFilterActive {
			return m.handleCommandInput(key)
		}
		return m.handleKeyString(key.String())
	}
}

// interface
func (m Model) Update(rawmsg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := rawmsg.(type) {
	case workspaceList:
		m.display = string(msg)
	case string:
		m.display = msg
	case StringWithCallback:
		m.display = msg.String
		return m, msg.Callback
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	}
	return m, nil
}
func (m Model) View() string {
	return m.display
}
func (m Model) Init() tea.Cmd {
	return func() tea.Msg { return m.renderWorkspaces() }
}

func NewModel(w []workspaces.Workspace) (*Model, error) {
	if err := clipboard.Init(); err != nil {
		return nil, fmt.Errorf("initialize clipboard: %w", err)
	}
	// TODO: terminal height for maxrows
	return &Model{workspaces: w, maxrows: 10}, nil
}

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
