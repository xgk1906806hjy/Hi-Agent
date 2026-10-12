package ui

import (
	"io"
	"strings"

	"github.com/xgk1906806hjy/Hi-Agent/internal/gui"
)

func init() {
	Register(KindGUI, createGUI)
}

type guiAdapter struct {
	shell *gui.Shell
}

func createGUI(opts Options) (UI, error) {
	var statusFn func() gui.StatusSnapshot
	if opts.StatusFn != nil {
		fn, ok := opts.StatusFn.(func() gui.StatusSnapshot)
		if !ok {
			return nil, errf("Options.StatusFn 须为 func() gui.StatusSnapshot")
		}
		statusFn = fn
	}
	slash := make([]gui.SlashItem, 0, len(opts.SlashCommands))
	for _, c := range opts.SlashCommands {
		slash = append(slash, gui.SlashItem{Name: c.Name, Desc: c.Desc})
	}
	sh := gui.New(gui.Options{
		Title:         "Hi-agent",
		StatusFn:      statusFn,
		SlashCommands: slash,
	})
	return &guiAdapter{shell: sh}, nil
}

func errf(msg string) error { return &simpleError{msg} }

type simpleError struct{ s string }

func (e *simpleError) Error() string { return e.s }

func (a *guiAdapter) Run() { a.shell.Run() }

func (a *guiAdapter) Info(s string) {
	a.shell.AppendSys(s)
}

func (a *guiAdapter) Notice(s string) {
	a.shell.AppendTool(strings.TrimSpace(s))
}

func (a *guiAdapter) Error(s string) {
	a.shell.AppendErr(s)
}

func (a *guiAdapter) UserEcho(s string) {
	a.shell.AppendUser(s)
}

func (a *guiAdapter) Progress(s string) {
	a.shell.AppendTool(strings.Trim(s, "\n"))
}

func (a *guiAdapter) Model(s string) {
	a.shell.AppendModel(s)
}

func (a *guiAdapter) BeginReply() {
	a.shell.SetBusy(true)
	a.shell.RefreshStatus()
}

func (a *guiAdapter) EndReply() {
	a.shell.EndModel()
	a.shell.SetBusy(false)
	a.shell.RefreshStatus()
}

func (a *guiAdapter) Refresh() {
	a.shell.RefreshStatus()
}

func (a *guiAdapter) ReadLine(prompt string) (string, error) {
	a.shell.SetPrompt(prompt)
	select {
	case line := <-a.shell.Lines():
		return strings.TrimSpace(line), nil
	case <-a.shell.Done():
		return "", io.EOF
	}
}

func (a *guiAdapter) Confirm(prompt string) bool {
	return a.shell.Confirm(prompt)
}

func (a *guiAdapter) Select(title string, items []string, selected int) (int, error) {
	return a.shell.Select(title, items, selected)
}

func (a *guiAdapter) Close() {
	a.shell.Quit()
}
