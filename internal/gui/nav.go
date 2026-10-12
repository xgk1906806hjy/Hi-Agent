package gui

import (
	"fmt"
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
)

// NavItem 左侧项目或会话一行。
type NavItem struct {
	Label   string
	Path    string // 项目目录
	ID      string // 会话 ID
	Current bool
}

// NavSnapshot 左侧「项目 + 当前项目的会话」。
type NavSnapshot struct {
	Projects []NavItem
	Sessions []NavItem
}

func (n NavSnapshot) signature() string {
	var b strings.Builder
	for _, p := range n.Projects {
		fmt.Fprintf(&b, "p:%s:%v\n", p.Path, p.Current)
	}
	for _, s := range n.Sessions {
		fmt.Fprintf(&b, "s:%s:%v\n", s.ID, s.Current)
	}
	return b.String()
}

func (s *Shell) renderNav(nav NavSnapshot) {
	sig := nav.signature()
	if sig == s.navSig && s.navBox != nil {
		return
	}
	s.navSig = sig

	rows := []fyne.CanvasObject{
		s.navHeader("项目", s.pickProject),
	}
	if len(nav.Projects) == 0 {
		rows = append(rows, quietHint("添加一个文件夹"))
	}
	for _, p := range nav.Projects {
		path := p.Path
		rows = append(rows, newTapRow(p.Label, p.Current, func() {
			s.pushLine("/project " + path)
		}))
	}

	rows = append(rows, spacer(10), s.navHeader("会话", func() {
		s.pushLine("/new")
	}))
	if len(nav.Sessions) == 0 {
		rows = append(rows, quietHint("新建一段对话"))
	}
	for _, ses := range nav.Sessions {
		id := ses.ID
		rows = append(rows, newTapRow(ses.Label, ses.Current, func() {
			s.pushLine("/open " + id)
		}))
	}

	s.navBox.Objects = rows
	s.navBox.Refresh()
}

func (s *Shell) navHeader(title string, onAdd func()) fyne.CanvasObject {
	t := canvas.NewText(title, muted)
	t.TextSize = 11
	t.TextStyle = fyne.TextStyle{Bold: true}
	add := newTapRow("+", false, onAdd)
	return container.NewBorder(nil, nil, nil, add, container.NewPadded(t))
}

func quietHint(s string) fyne.CanvasObject {
	t := canvas.NewText(s, muted)
	t.TextSize = 12
	return container.NewPadded(t)
}

func spacer(h float32) fyne.CanvasObject {
	r := canvas.NewRectangle(color.NRGBA{})
	r.SetMinSize(fyne.NewSize(0, h))
	return r
}

func (s *Shell) pickProject() {
	dlg := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
		if err != nil || uri == nil {
			return
		}
		path := uriFilePath(uri)
		if path == "" {
			return
		}
		s.pushLine("/project " + path)
	}, s.win)
	dlg.Resize(fyne.NewSize(720, 480))
	dlg.Show()
}

func uriFilePath(uri fyne.URI) string {
	if uri == nil {
		return ""
	}
	p := uri.Path()
	// Fyne 在 Windows 上常返回 /C:/foo
	if len(p) >= 3 && p[0] == '/' && p[2] == ':' {
		p = p[1:]
	}
	return p
}
