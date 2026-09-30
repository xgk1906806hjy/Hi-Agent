package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"hi-agent/internal/color"
)

// ui 抽象 REPL 的输出与输入，TUI 与纯文本两种实现。
type ui interface {
	Info(s string)     // 系统信息（灰）
	Notice(s string)   // 提示（黄）
	Error(s string)    // 错误
	UserEcho(s string) // 回显用户输入（纯文本模式终端已回显，不重复）
	Progress(s string) // 工具 / 压缩等进度行
	Model(s string)    // 模型流式正文
	BeginReply()
	EndReply()
	Refresh() // 用量更新等外部状态变化
	ReadLine(prompt string) (string, error)
	Confirm(prompt string) bool
	Close()
}

// plainUI 非 TUI 时的着色行输出；Confirm/ReadLine 走同一 bufio.Reader。
type plainUI struct {
	in *bufio.Reader
}

func (p *plainUI) Info(s string)     { color.Out(color.Sys, s, true) }
func (p *plainUI) Notice(s string)   { color.Out(color.Tool, s, true) }
func (p *plainUI) Error(s string)    { color.Err(color.Sys, s) }
func (p *plainUI) UserEcho(string)   {} // 终端已回显，避免重复
func (p *plainUI) Progress(s string) { color.Out(color.Tool, s, false) }
func (p *plainUI) Model(s string)    { color.Out(color.Model, s, false) }
func (p *plainUI) BeginReply()       { color.Out(color.Sys, "", true) }
func (p *plainUI) EndReply()         { color.Out(color.Sys, "", true) }
func (p *plainUI) Refresh()          {}
func (p *plainUI) Close()            {}

// ReadLine 在 stdout 打印 prompt 后读一行。
func (p *plainUI) ReadLine(prompt string) (string, error) {
	fmt.Fprint(os.Stdout, color.Paint(color.User, prompt))
	line, err := p.in.ReadString('\n')
	return strings.TrimSpace(line), err
}

// Confirm 打印提示并以 y/yes 为确认。
func (p *plainUI) Confirm(prompt string) bool {
	color.Out(color.Tool, prompt+" [y/N] ", false)
	line, err := p.in.ReadString('\n')
	if err != nil {
		return false
	}
	ans := strings.ToLower(strings.TrimSpace(line))
	return ans == "y" || ans == "yes"
}
