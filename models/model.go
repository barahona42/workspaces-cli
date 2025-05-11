package models

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
	db "workspaces-cli/models/database"
	"workspaces-cli/pkg/workspaces"

	tea "github.com/charmbracelet/bubbletea"
	"golang.design/x/clipboard"
)

type Model struct {
	mainPane   string
	footerPane string

	workspaces []workspaces.Workspace
	maxnamelen int
	cursor     int
	maxrows    int

	filterValue        string
	isFilterActive     bool
	filteredWorkspaces []workspaces.Workspace
}

func (m *Model) resetFilter() {
	m.isFilterActive = false
	m.filterValue = ""
	clear(m.filteredWorkspaces)
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
		index   string = ""
		modtime string = modtimeColorize(w.ModTime())
	)
	if selected {
		// cursor = "\033[34m>\033[0m"
		cursor = "👉"
		name = fmt.Sprintf("\033[1;34m%-*s\033[0m", namepadding, w.DirEntry.Name())
		path = fmt.Sprintf("\033[90m%s\033[0m", w.Path())
		index = fmt.Sprintf("\033[1;34m%-3d\033[0m", pos)
	} else {
		cursor = " "
		name = fmt.Sprintf("%-*s", namepadding, w.DirEntry.Name())
		index = fmt.Sprintf("\033[00m%-3d\033[0m", pos)
	}
	return fmt.Sprintf("%s\t%s   %s   %s   %s", cursor, index, name, modtime, path)
}

func (m *Model) generateFooter() string {
	b := strings.Builder{}
	if m.isFilterActive || len(m.filterValue) > 0 {
		b.WriteString(fmt.Sprintf("\033[33m↳ FILTER > %s\033[0m\n", m.filterValue))
	} else {
		b.WriteString("\n")
	}
	b.WriteString("\033[90m")
	b.WriteString("   type 'c' to copy selected path to clipboard\n")
	b.WriteString("   type 'o' to open selected workspace in vscode\n")
	b.WriteString("\033[0m\n")
	b.WriteString("\n")
	return b.String()
}

func (m *Model) generateWorkspacesString() string {
	ws := m.getWorkspaces()
	strs := make([]func(int) string, len(ws))
	for i := range ws {
		if l := len(ws[i].DirEntry.Name()); l > m.maxnamelen {
			m.maxnamelen = l
		}
		strs[i] = func(namepadding int) string {
			return m.generateWorkspaceString(i+1, namepadding, m.cursor == i, ws[i])
		}

	}
	b := strings.Builder{}
	for i := 0; i < m.maxrows; i++ {
		if i+m.cursor < len(strs) {
			b.WriteString(strs[i+m.cursor](m.maxnamelen) + "\n")
		} else {
			b.WriteString(".\n")
		}
	}
	return b.String()
}

func (m *Model) renderWorkspaces() tea.Msg {
	return renderpanescmd{main: m.generateWorkspacesString(), footer: m.generateFooter()}
}

func (m *Model) setFilteredWorkspaces(filter string) {
	matchName := func(name string) bool {
		return strings.Contains(strings.ToLower(name), strings.ToLower(filter))
	}
	matchModTime := func(modtime time.Time) bool {
		return strings.Contains(modtime.Format(time.DateOnly), filter)
	}
	m.filteredWorkspaces = make([]workspaces.Workspace, 0, len(m.workspaces))
	for i := range m.workspaces {
		if matchName(m.workspaces[i].DirEntry.Name()) || matchModTime(m.workspaces[i].ModTime()) {
			m.filteredWorkspaces = append(m.filteredWorkspaces, m.workspaces[i])
		}
	}
}

func (m *Model) handleCommandInput(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.Type {
	case tea.KeyLeft, tea.KeyRight: // noop
	case tea.KeyEnter:
		var cursor int = -1
		for i := 0; i < len(m.workspaces) && cursor == -1; i++ {
			if m.workspaces[i] == m.filteredWorkspaces[m.cursor] {
				cursor = i
			}
		}
		m.cursor = cursor
		m.resetFilter()
		return m, func() tea.Msg {
			return m.renderWorkspaces()
		}
	case tea.KeyEsc: // exit filter mode
		m.resetFilter()
		m.cursor = 0
		return m, func() tea.Msg {
			return m.renderWorkspaces()
		}
	case tea.KeyBackspace:
		if len(m.filterValue) > 0 {
			m.filterValue = m.filterValue[:len(m.filterValue)-1]
		}
		m.cursor = 0
		m.setFilteredWorkspaces(m.filterValue)
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
			return renderpaneswithcallbackcmd{
				renderpanescmd: renderpanescmd{
					main:   "📋 copied message to clipboard",
					footer: m.generateFooter(),
				},
				callback: func() tea.Msg {
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
			return renderpaneswithcallbackcmd{
				renderpanescmd: renderpanescmd{
					main:   "💻 opening workspace",
					footer: m.generateFooter(),
				},
				callback: func() tea.Msg {
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

func (m *Model) formatView() string {
	b := strings.Builder{}
	b.WriteString(m.mainPane)
	b.WriteString("\n----------\n")
	b.WriteString(m.footerPane)
	return b.String()
}

func (m *Model) Cleanup() error {
	return errors.Join(db.Close())
}

// interface
func (m Model) Update(rawmsg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := rawmsg.(type) {
	case renderpanescmd:
		m.mainPane = msg.main
		m.footerPane = msg.footer
	case renderpaneswithcallbackcmd:
		m.mainPane = msg.main
		m.footerPane = msg.footer
		return m, msg.callback
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	case string:
		panic(fmt.Sprintf("string type deprecated: '%s'", msg))
	}
	return m, nil
}
func (m Model) View() string {
	return m.formatView()
}
func (m Model) Init() tea.Cmd {
	return func() tea.Msg { return m.renderWorkspaces() }
}
