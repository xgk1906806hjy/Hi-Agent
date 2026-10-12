package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// Confirm 模态确认；在 UI 线程调度，阻塞调用方直到选择。
func (s *Shell) Confirm(prompt string) bool {
	ch := make(chan bool, 1)
	s.do(func() {
		d := dialog.NewConfirm("确认", prompt, func(ok bool) {
			select {
			case ch <- ok:
			default:
			}
		}, s.win)
		d.SetDismissText("取消")
		d.SetConfirmText("确认")
		d.Resize(fyne.NewSize(480, 240))
		d.Show()
	})
	select {
	case ok := <-ch:
		return ok
	case <-s.done:
		return false
	}
}

// Select 列表选择；取消返回 -1。
func (s *Shell) Select(title string, items []string, selected int) (int, error) {
	if len(items) == 0 {
		return -1, nil
	}
	if selected < 0 || selected >= len(items) {
		selected = 0
	}
	ch := make(chan int, 1)
	s.do(func() {
		idx := selected
		list := widget.NewList(
			func() int { return len(items) },
			func() fyne.CanvasObject { return widget.NewLabel("template") },
			func(id widget.ListItemID, obj fyne.CanvasObject) {
				obj.(*widget.Label).SetText(items[id])
			},
		)
		list.Select(selected)
		list.OnSelected = func(id widget.ListItemID) { idx = int(id) }
		wrap := container.NewBorder(nil, nil, nil, nil, list)
		wrap.Resize(fyne.NewSize(440, 300))

		d := dialog.NewCustomConfirm(title, "选择", "取消", wrap, func(ok bool) {
			if !ok {
				ch <- -1
				return
			}
			ch <- idx
		}, s.win)
		d.Resize(fyne.NewSize(480, 400))
		d.Show()
	})
	select {
	case v := <-ch:
		return v, nil
	case <-s.done:
		return -1, nil
	}
}
