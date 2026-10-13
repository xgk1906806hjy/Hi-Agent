//go:build desktop

package gui

import "github.com/xgk1906806hjy/Hi-Agent/internal/ui"

func init() {
	ui.Register(ui.KindGUI, func(opts ui.Options) (ui.UI, error) {
		var app *App
		ad := NewAdapter(Config{
			StatusFn:  opts.StatusFn,
			NavFn:     opts.NavFn,
			HistoryFn: opts.HistoryFn,
			Slash:     opts.SlashCommands,
			Run:       func() { app.run() },
			Quit:      func() { app.quit() },
		})
		app = newApp(ad)
		return ad, nil
	})
}
