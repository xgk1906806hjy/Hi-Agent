package ui

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/xgk1906806hjy/Hi-Agent/internal/color"
	"github.com/xgk1906806hjy/Hi-Agent/internal/tui"
)

func init() {
	Register(KindPlain, func(opts Options) (UI, error) {
		in := opts.In
		if in == nil {
			in = bufio.NewReader(os.Stdin)
		}
		return newPlain(in), nil
	})
}

// plain 纯文本终端 UI（无备用屏）。
type plain struct {
	in         *bufio.Reader
	foldOn     bool
	think      tui.ThinkStream
	thinkNoted bool
}

func newPlain(in *bufio.Reader) *plain {
	return &plain{in: in, foldOn: tui.FoldEnabled()}
}

func (p *plain) Info(s string)   { color.Out(color.Sys, s, true) }
func (p *plain) Notice(s string) { color.Out(color.Tool, s, true) }
func (p *plain) Error(s string)  { color.Err(color.Sys, s) }
func (p *plain) UserEcho(string) {}
func (p *plain) Progress(s string) {
	if p.foldOn {
		color.Out(color.Tool, tui.ShortProgressLine(s), true)
		return
	}
	color.Out(color.Tool, s, false)
}
func (p *plain) Model(s string) {
	if !p.foldOn {
		color.Out(color.Model, s, false)
		return
	}
	think, model := p.think.Feed(s)
	if think != "" && !p.thinkNoted {
		color.Out(color.Tool, "▸ 思考（已折叠）", true)
		p.thinkNoted = true
	}
	if model != "" {
		color.Out(color.Model, model, false)
	}
}
func (p *plain) BeginReply() {
	p.think = tui.ThinkStream{}
	p.thinkNoted = false
	color.Out(color.Sys, "", true)
}
func (p *plain) EndReply() { color.Out(color.Sys, "", true) }
func (p *plain) Refresh()  {}
func (p *plain) Close()    {}

func (p *plain) ReadLine(prompt string) (string, error) {
	fmt.Fprint(os.Stdout, color.Paint(color.User, prompt))
	line, err := p.in.ReadString('\n')
	return strings.TrimSpace(line), err
}

func (p *plain) Confirm(prompt string) bool {
	color.Out(color.Tool, prompt+" [y/N] ", false)
	line, err := p.in.ReadString('\n')
	if err != nil {
		return false
	}
	ans := strings.ToLower(strings.TrimSpace(line))
	return ans == "y" || ans == "yes"
}

func (p *plain) Select(title string, items []string, selected int) (int, error) {
	if len(items) == 0 {
		return -1, fmt.Errorf("没有可选项")
	}
	color.Out(color.Sys, title, true)
	for i, it := range items {
		mark := " "
		if i == selected {
			mark = "*"
		}
		color.Out(color.Sys, fmt.Sprintf("  %s %d) %s", mark, i+1, it), true)
	}
	ans, err := p.ReadLine("序号（回车取消）› ")
	if err != nil || strings.TrimSpace(ans) == "" {
		return -1, nil
	}
	n, err := strconv.Atoi(strings.TrimSpace(ans))
	if err != nil || n < 1 || n > len(items) {
		return -1, fmt.Errorf("无效序号")
	}
	return n - 1, nil
}
