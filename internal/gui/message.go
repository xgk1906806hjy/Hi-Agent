package gui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// bubble 按消息类型生成对话气泡。
func bubble(m chatMsg) fyne.CanvasObject {
	switch m.kind {
	case msgUser:
		return container.New(&sideLayout{right: true}, userBubble(m.text))
	case msgModel:
		return container.New(&sideLayout{right: false}, agentBubble(m.text))
	case msgTool:
		return container.New(&sideLayout{right: false}, toolBubble(m.text))
	case msgErr:
		return container.New(&sideLayout{right: false}, errBubble(m.text))
	default:
		return sysBubble(m.text)
	}
}

type sideLayout struct {
	right bool
}

func (s *sideLayout) MinSize(objs []fyne.CanvasObject) fyne.Size {
	if len(objs) == 0 {
		return fyne.NewSize(0, 0)
	}
	ms := objs[0].MinSize()
	return fyne.NewSize(ms.Width, ms.Height+8)
}

func (s *sideLayout) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	if len(objs) == 0 {
		return
	}
	o := objs[0]
	maxW := size.Width * 0.72
	if maxW < 180 {
		maxW = size.Width
	}
	o.Resize(fyne.NewSize(maxW, o.MinSize().Height))
	h := o.MinSize().Height
	w := o.MinSize().Width
	if w > maxW {
		w = maxW
	}
	if w < 48 {
		w = 48
	}
	o.Resize(fyne.NewSize(w, h))
	x := float32(0)
	if s.right {
		x = size.Width - w
	}
	o.Move(fyne.NewPos(x, 4))
}

func userBubble(text string) fyne.CanvasObject {
	lbl := widget.NewLabel(text)
	lbl.Wrapping = fyne.TextWrapWord
	return paddedRound(lbl, bubbleUserBG, 14)
}

func agentBubble(text string) fyne.CanvasObject {
	cap := canvas.NewText("Agent", muted)
	cap.TextSize = 11
	body := widget.NewLabel(text)
	body.Wrapping = fyne.TextWrapWord
	col := container.NewVBox(cap, body)
	return paddedRound(col, bubbleAgentBG, 14)
}

func toolBubble(text string) fyne.CanvasObject {
	lbl := widget.NewLabel(text)
	lbl.Wrapping = fyne.TextWrapWord
	lbl.Importance = widget.LowImportance
	return paddedRound(lbl, bubbleToolBG, 10)
}

func errBubble(text string) fyne.CanvasObject {
	lbl := widget.NewLabel(text)
	lbl.Wrapping = fyne.TextWrapWord
	lbl.Importance = widget.DangerImportance
	return paddedRound(lbl, bubbleErrBG, 12)
}

func sysBubble(text string) fyne.CanvasObject {
	lbl := widget.NewLabel(text)
	lbl.Wrapping = fyne.TextWrapWord
	lbl.Importance = widget.LowImportance
	return container.NewPadded(lbl)
}

func paddedRound(content fyne.CanvasObject, bg color.Color, radius float32) fyne.CanvasObject {
	rect := canvas.NewRectangle(bg)
	rect.CornerRadius = radius
	return container.NewStack(rect, container.NewPadded(content))
}
