package ui

import "time"

// Status 界面右侧/底部状态的展示数据，由 REPL 组装，各界面只读。
type Status struct {
	Model    string
	Provider string
	Session  string
	WorkDir  string
	Busy     bool

	CtxTokens      int
	CtxEstimated   bool
	CtxWindow      int
	TurnPrompt     int
	TurnCompletion int
	Messages       int
}

// Nav 左侧「项目 > 会话」树。
type Nav struct {
	Projects []NavProject
}

// NavProject 一个已登记的项目目录及其会话。
type NavProject struct {
	Name     string
	Path     string
	Current  bool
	Sessions []NavSession
}

// Turn 当前会话中一条可展示的消息（用于切换会话后重绘对话流）。
// Role 取 user / assistant / tool。
type Turn struct {
	Role string
	Text string
}

// NavSession 项目下的一个会话。
type NavSession struct {
	ID        string
	Title     string // 第一条用户消息摘要；空表示尚无对话
	Messages  int
	UpdatedAt time.Time
	Current   bool
}
