// Package webui 提供浏览器界面：HTTP + Server-Sent Events。
//
// 它复用 internal/gui 的 Adapter 与事件协议（chat:* / turn:* / status / nav /
// dialog:* / input:prompt），只把 Wails 绑定替换成 HTTP JSON + SSE，
// 因此 REPL 不需要为 Web 模式增加任何分支。
package webui

import (
	"embed"

	"github.com/xgk1906806hjy/Hi-Agent/internal/gui"
	"github.com/xgk1906806hjy/Hi-Agent/internal/ui"
)

//go:embed static/index.html
var staticFS embed.FS

func init() {
	ui.Register(ui.KindWeb, newUI)
}

func newUI(opts ui.Options) (ui.UI, error) {
	s := newServer()
	ad := gui.NewAdapter(gui.Config{
		StatusFn:  opts.StatusFn,
		NavFn:     opts.NavFn,
		HistoryFn: opts.HistoryFn,
		Slash:     opts.SlashCommands,
		Run:       s.Run,
		Quit:      s.Shutdown,
	})
	s.adapter = ad
	return ad, nil
}
