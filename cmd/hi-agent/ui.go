package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/xgk1906806hjy/Hi-Agent/internal/color"
	"github.com/xgk1906806hjy/Hi-Agent/internal/tui"
)

// ui 抽象 REPL 的输出与输入，TUI 与纯文本两种实现。
type ui interface {
	Info(s string)
	Notice(s string)
	Error(s string)
	UserEcho(s string)
	Progress(s string)
	Model(s string)
	BeginReply()
	EndReply()
	Refresh()
	ReadLine(prompt string) (string, error)
	Confirm(prompt string) bool
	// Select ↑/↓ 选择；取消返回 -1。selected 为初始高亮下标。
	Select(title string, items []string, selected int) (int, error)
	Close()
}

// plainUI 非 TUI 时的着色行输出。
type plainUI struct {
	in         *bufio.Reader
	foldOn     bool
	think      tui.ThinkStream
	thinkNoted bool
}

func newPlainUI(in *bufio.Reader) *plainUI {
	return &plainUI{in: in, foldOn: tui.FoldEnabled()}
}

func (p *plainUI) Info(s string)     { color.Out(color.Sys, s, true) }
func (p *plainUI) Notice(s string)   { color.Out(color.Tool, s, true) }
func (p *plainUI) Error(s string)    { color.Err(color.Sys, s) }
func (p *plainUI) UserEcho(string)   {}
func (p *plainUI) Progress(s string) {
	if p.foldOn {
		color.Out(color.Tool, tui.ShortProgressLine(s), true)
		return
	}
	color.Out(color.Tool, s, false)
}
func (p *plainUI) Model(s string) {
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
func (p *plainUI) BeginReply() {
	p.think = tui.ThinkStream{}
	p.thinkNoted = false
	color.Out(color.Sys, "", true)
}
func (p *plainUI) EndReply() { color.Out(color.Sys, "", true) }
func (p *plainUI) Refresh()  {}
func (p *plainUI) Close()    {}

func (p *plainUI) ReadLine(prompt string) (string, error) {
	fmt.Fprint(os.Stdout, color.Paint(color.User, prompt))
	line, err := p.in.ReadString('\n')
	return strings.TrimSpace(line), err
}

func (p *plainUI) Confirm(prompt string) bool {
	color.Out(color.Tool, prompt+" [y/N] ", false)
	line, err := p.in.ReadString('\n')
	if err != nil {
		return false
	}
	ans := strings.ToLower(strings.TrimSpace(line))
	return ans == "y" || ans == "yes"
}

// Select 纯文本回退：打印列表并用序号选择。
func (p *plainUI) Select(title string, items []string, selected int) (int, error) {
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
