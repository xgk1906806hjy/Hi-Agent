package ui

import (
	"bufio"
	"fmt"
	"os"

	"github.com/xgk1906806hjy/Hi-Agent/internal/tui"
)

// ErrInterrupted 与 tui 共用：raw 输入时 Ctrl+C。
var ErrInterrupted = tui.ErrInterrupted

func init() {
	Register(KindTUI, createTUI)
}

func createTUI(opts Options) (UI, error) {
	in := opts.In
	if in == nil {
		in = bufio.NewReader(os.Stdin)
	}
	if !tui.Supported() {
		return Create(KindPlain, opts)
	}

	var panel func() tui.Panel
	if opts.PanelFn != nil {
		fn, ok := opts.PanelFn.(func() tui.Panel)
		if !ok {
			return nil, fmt.Errorf("Options.PanelFn 须为 func() tui.Panel")
		}
		panel = fn
	}

	t := tui.New(in, panel)
	if len(opts.SlashCommands) > 0 {
		cmds := make([]tui.SlashCmd, len(opts.SlashCommands))
		for i, c := range opts.SlashCommands {
			cmds[i] = tui.SlashCmd{Name: c.Name, Desc: c.Desc}
		}
		t.SetSlashCommands(cmds)
	}
	if err := t.Start(); err != nil {
		// 启动失败回退 plain
		return Create(KindPlain, opts)
	}
	return t, nil
}
