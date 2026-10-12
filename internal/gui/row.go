package gui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
)

// tapRow 是无边框的可点行，用来代替侧栏/快捷里的默认按钮。
type tapRow struct {
	widget.BaseWidget
	label  string
	active bool
	onTap  func()
}

func newTapRow(label string, active bool, onTap func()) *tapRow {
	r := &tapRow{label: label, active: active, onTap: onTap}
	r.ExtendBaseWidget(r)
	return r
}

func (r *tapRow) Tapped(*fyne.PointEvent) {
	if r.onTap != nil {
		r.onTap()
	}
}

func (r *tapRow) CreateRenderer() fyne.WidgetRenderer {
	bg := canvas.NewRectangle(color.Transparent)
	tx := canvas.NewText(r.label, ink)
	tx.TextSize = 13
	rd := &tapRender{row: r, bg: bg, tx: tx}
	rd.Refresh()
	return rd
}

type tapRender struct {
	row *tapRow
	bg  *canvas.Rectangle
	tx  *canvas.Text
}

func (r *tapRender) Layout(sz fyne.Size) {
	r.bg.Resize(sz)
	r.bg.Move(fyne.NewPos(0, 0))
	th := r.tx.MinSize().Height
	r.tx.Move(fyne.NewPos(12, (sz.Height-th)/2))
	r.tx.Resize(fyne.NewSize(sz.Width-18, th))
}

func (r *tapRender) MinSize() fyne.Size {
	return fyne.NewSize(36, 32)
}

func (r *tapRender) Refresh() {
	r.tx.Text = r.row.label
	r.bg.CornerRadius = 8
	if r.row.active {
		r.bg.FillColor = rowActiveBG
		r.tx.Color = rowActiveFG
		r.tx.TextStyle = fyne.TextStyle{Bold: true}
	} else {
		r.bg.FillColor = color.NRGBA{}
		r.tx.Color = ink
		r.tx.TextStyle = fyne.TextStyle{}
	}
	r.bg.Refresh()
	r.tx.Refresh()
}

func (r *tapRender) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.bg, r.tx}
}

func (r *tapRender) Destroy() {}
