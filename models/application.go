package models

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
	"workspaces-cli/models/db"
	"workspaces-cli/pkg/editors"
	"workspaces-cli/pkg/textcolor"
	"workspaces-cli/pkg/workspaces"

	tea "github.com/charmbracelet/bubbletea"
	"golang.design/x/clipboard"
)

type inputmode = int

const (
	mode_default       = iota // the base mode where the rendered items are workspaces
	mode_filter               // user is inputting a filter and workspaces are being updated
	mode_commandselect        // user is selecting from the command menu
)

type Application struct {
	mode inputmode // user input mode. determines what's rendered and how input is handled

	editor editors.Editor
	// TODO: mainPane needs to enforce persistent height throughout execution to prevent ghosting
	mainPane   string // main pane display
	footerPane string // footer display

	// workspace fields
	workspaces []workspaces.Workspace
	maxnamelen int
	cursor     int
	maxrows    int

	// filter mode fields
	filterCursor       int
	filterValue        string
	isFilterActive     bool
	filteredWorkspaces []workspaces.Workspace

	// command mode fields
	commands      []string
	commandCursor int
}

func (m *Application) resetMode() {
	switch m.mode {
	case mode_filter:
		m.cursor = m.filterCursor
		m.filterCursor = 0
		m.filterValue = ""
		m.isFilterActive = false
		clear(m.filteredWorkspaces)
	case mode_commandselect:
		m.commandCursor = 0
	}
	m.mode = mode_default
}

func (m *Application) startMode(mode inputmode) {
	switch mode {
	case mode_filter:
		m.filterCursor = 0
		m.filterValue = ""
		m.isFilterActive = true
	case mode_commandselect:
		m.commandCursor = 0
	}
	m.mode = mode
}

func (m *Application) getCommandCursorMax() int {
	if len(m.commands) > 0 {
		return len(m.commands) - 1
	}
	return 0
}

func (m *Application) commandCursorUp() {
	if m.commandCursor > 0 {
		m.commandCursor--
	} else {
		m.commandCursor = m.getCommandCursorMax()
	}
}
func (m *Application) commandCursorDown() {
	if m.commandCursor < m.getCommandCursorMax() {
		m.commandCursor++
	} else {
		m.commandCursor = 0
	}
}

func (m *Application) getCursorMax() int {
	if len(m.workspaces) > 0 {
		return len(m.workspaces) - 1
	}
	return 0
}
func (m *Application) getFilterCursorMax() int {
	if len(m.filteredWorkspaces) > 0 {
		return len(m.filteredWorkspaces) - 1
	}
	return 0
}

func (m *Application) filterCursorUp() {
	if m.filterCursor > 0 {
		m.filterCursor--
	} else {
		m.filterCursor = m.getFilterCursorMax()
	}
}
func (m *Application) filterCursorDown() {
	if m.filterCursor < m.getFilterCursorMax() {
		m.filterCursor++
	} else {
		m.filterCursor = 0
	}
}
func (m *Application) cursorUp() {
	if m.cursor > 0 {
		m.cursor--
	} else {
		m.cursor = m.getCursorMax()
	}
}
func (m *Application) cursorDown() {
	if m.cursor < m.getCursorMax() {
		m.cursor++
	} else {
		m.cursor = 0
	}
}

func (m *Application) defaultRenderer() tea.Msg {
	return renderpanescmd{main: m.generateWorkspacesString(), footer: m.generateFooter()}
}

func (m *Application) filterRenderer() tea.Msg {
	return renderpanescmd{main: m.generateFilterWorkspacesString(), footer: m.generateFooter()}
}

func (m *Application) commandSelectRenderer() tea.Msg {
	// TODO: could set the main dynamically off the active command
	return renderpanescmd{main: m.generateCommandSelectString(), footer: m.generateFooter()}
}

func (m *Application) generateWorkspaceString(pos, namepadding int, selected bool, w workspaces.Workspace) string {
	var (
		cursor  string = ""
		name    string = ""
		path    string = ""
		index   string = ""
		modtime string = modtimeColorize(w.ModTime())
	)
	if selected {
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

func (m *Application) generateFooter() string {
	b := strings.Builder{}
	switch m.mode {
	case mode_commandselect:
		for i := range m.commands {
			if m.commandCursor == i {
				b.WriteString(" > " + m.commands[i] + "\n")
			} else {
				b.WriteString("   " + m.commands[i] + "\n")
			}
		}
	case mode_filter:
		b.WriteString(textcolor.Colorize(textcolor.YELLOW, fmt.Sprintf("↳ FILTER > %s", m.filterValue)) + "\n")
		fallthrough
	default:
		b.WriteString(textcolor.Colorize(textcolor.LIGHT_GRAY, "   type 'c' to copy selected path to clipboard\n"))
		b.WriteString(textcolor.Colorize(textcolor.LIGHT_GRAY, "   type 'o' to open selected workspace in vscode\n"))
	}
	return b.String()
}
func (m *Application) generateCommandSelectString() string {
	b := strings.Builder{}
	for range m.maxrows {
		b.WriteString(".\n")
	}
	return b.String()
}

func (m *Application) generateFilterWorkspacesString() string {
	matchName := func(name string) bool {
		return strings.EqualFold(name, strings.ToLower(m.filterValue)) ||
			strings.Contains(strings.ToLower(name), strings.ToLower(m.filterValue))
	}
	matchModTime := func(modtime time.Time) bool {
		return strings.Contains(modtime.Format(time.DateOnly), m.filterValue)
	}
	m.filteredWorkspaces = make([]workspaces.Workspace, 0, len(m.workspaces))
	for i := range m.workspaces {
		if matchName(m.workspaces[i].DirEntry.Name()) || matchModTime(m.workspaces[i].ModTime()) {
			m.filteredWorkspaces = append(m.filteredWorkspaces, m.workspaces[i])
		}
	}
	strs := make([]func(int) string, len(m.filteredWorkspaces))
	for i := range m.filteredWorkspaces {
		if l := len(m.filteredWorkspaces[i].DirEntry.Name()); l > m.maxnamelen {
			m.maxnamelen = l
		}
		strs[i] = func(namepadding int) string {
			return m.generateWorkspaceString(i+1, namepadding, m.filterCursor == i, m.filteredWorkspaces[i])
		}

	}
	b := strings.Builder{}
	for i := range m.maxrows {
		if i < len(strs) {
			b.WriteString(strs[i](m.maxnamelen) + "\n")
		} else {
			b.WriteString(".\n")
		}
	}
	return b.String()
}

func (m *Application) generateWorkspacesString() string {
	ws := m.workspaces
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

func (m *Application) activeCommandHandler(ctx context.Context) tea.Cmd {
	switch m.commandCursor {
	case 0:
		f, err := m.editor.CreateTemp()
		if err != nil {
			return func() tea.Msg { return errormessage{err} }
		}
		c := exec.Command(m.editor.Command(), m.editor.OpenFileArgs(f.Name())...)
		return tea.ExecProcess(c, func(err error) tea.Msg {
			// here we need to load the data and render it
			// then we'll need to ship it to the db
			if err != nil {
				return errormessage{err: fmt.Errorf("exec '%s': %w", m.editor.Command(), err)}
			}
			data, err := os.ReadFile(f.Name())
			if err != nil {
				return errormessage{fmt.Errorf("read file: %w", err)}
			}
			if len(data) == 0 {
				return renderpaneswithcallbackcmd{
					renderpanescmd: renderpanescmd{
						main:   "\t\t❎ no checkpoint data received",
						footer: m.generateFooter()},
					callback: func() tea.Msg {
						m.resetMode()
						time.Sleep(MESSAGE_TIMEOUT)
						return m.defaultRenderer()
					},
				}
			}
			if err := db.InsertCheckpoint(ctx, m.workspaces[m.cursor], data); err != nil {
				return errormessage{err}
			}
			// now we generate the checkpoint row and insert it to the db
			// the returned data will be the the result of the write op
			return renderpaneswithcallbackcmd{
				renderpanescmd: renderpanescmd{
					main:   "\t\t✅ checkpoint inserted",
					footer: m.generateFooter()},
				callback: func() tea.Msg {
					m.resetMode()
					time.Sleep(MESSAGE_TIMEOUT)
					return m.defaultRenderer()
				},
			}
		})
	case 1:
		return func() tea.Msg { return viewcheckpointscmd("mock message from view checkpoints") }
	default:
		return nil
	}
}

func (m *Application) commandMode_handleKeyMsg(ctx context.Context, key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.Type {
	case tea.KeyEsc:
		m.resetMode()
		return m, m.defaultRenderer
	case tea.KeyEnter:
		// to keep things uniform, we need the command to have a renderer function (tea.Cmd)
		return m, m.activeCommandHandler(ctx)
		// show data in the main pane
	case tea.KeyDown:
		m.commandCursorDown()
		return m, m.commandSelectRenderer
	case tea.KeyUp:
		m.commandCursorUp()
		return m, m.commandSelectRenderer
	}
	return m, nil
}

func (m *Application) filterMode_handleKeyMsg(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.Type {
	case tea.KeyEsc, tea.KeyEnter:
		m.resetMode()
		return m, m.defaultRenderer
	case tea.KeyBackspace:
		if l := len(m.filterValue); l > 0 {
			m.filterValue = m.filterValue[:l-1]
		}
		m.filterCursor = 0
		return m, m.filterRenderer
	case tea.KeyUp:
		m.filterCursorUp()
		return m, m.filterRenderer
	case tea.KeyDown:
		m.filterCursorDown()
		return m, m.filterRenderer
	case tea.KeyRunes:
		m.filterValue += key.String()
		m.filterCursor = 0
		return m, m.filterRenderer
	}
	return m, nil
}

func (m *Application) defaultMode_handleKeyMsg(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	// nav keys
	switch key.Type {
	case tea.KeyUp:
		m.cursorUp()
		return m, m.defaultRenderer
	case tea.KeyDown:
		m.cursorDown()
		return m, m.defaultRenderer
	}
	// action keys
	switch key.String() {
	case "q":
		return m, tea.Quit
	case "c": // clip workspace path
		return m, func() tea.Msg {
			b := strings.Builder{}
			for range m.maxrows {
				b.WriteString("\n")
			}
			b.WriteString("📋 copied message to clipboard")
			return renderpaneswithcallbackcmd{
				renderpanescmd: renderpanescmd{
					main:   b.String(),
					footer: m.generateFooter(),
				},
				callback: func() tea.Msg {
					clipboard.Write(clipboard.FmtText, []byte(m.workspaces[m.cursor].Path()))
					time.Sleep(MESSAGE_TIMEOUT)
					return m.defaultRenderer()
				},
			}
		}
	case "o": // open workspace path
		return m, func() tea.Msg {
			b := strings.Builder{}
			for range m.maxrows {
				b.WriteString("\n")
			}
			b.WriteString("💻 opening workspace")
			return renderpaneswithcallbackcmd{
				renderpanescmd: renderpanescmd{
					main:   b.String(),
					footer: m.generateFooter(),
				},
				callback: func() tea.Msg {
					done := make(chan struct{}, 1)
					go func() {
						exec.Command("code", m.workspaces[m.cursor].Path()).Run()
						done <- struct{}{}
					}()
					time.Sleep(MESSAGE_TIMEOUT)
					<-done
					return m.defaultRenderer()
				},
			}
		}
	case "/": // enable filter mode
		m.startMode(mode_filter)
		return m, m.filterRenderer
	case ":": // enable command mode
		m.startMode(mode_commandselect)
		return m, m.commandSelectRenderer
	}
	return m, nil
}
func (m *Application) handleGlobalKeyMsg(key tea.KeyMsg) tea.Cmd {
	switch key.Type {
	case tea.KeyCtrlC:
		return tea.Quit
	}
	return nil
}

func (m *Application) handleKeyMsg(ctx context.Context, key tea.KeyMsg) (tea.Model, tea.Cmd) {
	// first we handle global keys
	if cmd := m.handleGlobalKeyMsg(key); cmd != nil {
		return m, cmd
	}
	switch m.mode {
	case mode_commandselect:
		return m.commandMode_handleKeyMsg(ctx, key)
	case mode_filter:
		return m.filterMode_handleKeyMsg(key)
	default:
		return m.defaultMode_handleKeyMsg(key)
	}
}

func (m *Application) Cleanup() error {
	return errors.Join(db.Close())
}

// interface
func (m Application) Update(rawmsg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := rawmsg.(type) {
	case renderpanescmd:
		m.mainPane = msg.main
		m.footerPane = msg.footer
	case renderpaneswithcallbackcmd:
		m.mainPane = msg.main
		m.footerPane = msg.footer
		return m, msg.callback
	case addcheckpointcmd:
		m.mainPane = string(msg)
	case viewcheckpointscmd:
		m.mainPane = string(msg)
	case errormessage:
		if msg.err != nil {
			m.mainPane = msg.err.Error()
			return m, tea.Quit
		}
	case tea.KeyMsg:
		return m.handleKeyMsg(context.TODO(), msg)
	case string:
		panic(fmt.Sprintf("string type deprecated: '%s'", msg))
	}
	return m, nil
}
func (m Application) View() string {
	b := strings.Builder{}
	b.WriteString(m.mainPane)
	b.WriteString("\n----------\n")
	b.WriteString(m.footerPane)
	return b.String()
}
func (m Application) Init() tea.Cmd {
	return func() tea.Msg { return renderpanescmd{main: m.generateWorkspacesString(), footer: m.generateFooter()} }
}
