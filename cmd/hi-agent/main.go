// hi-agent 进程入口与 REPL：加载配置/会话，接线确认与斜杠命令，驱动 chat.StreamReply。
// 终端且未设 GEEKAGENT_TUI=0 时启用 TUI，否则纯文本；退出（/exit、EOF、空闲 Ctrl+C）时保存会话并恢复终端。
package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/xgk1906806hjy/Hi-Agent/internal/chat"
	"github.com/xgk1906806hjy/Hi-Agent/internal/color"
	"github.com/xgk1906806hjy/Hi-Agent/internal/config"
	"github.com/xgk1906806hjy/Hi-Agent/internal/session"
	"github.com/xgk1906806hjy/Hi-Agent/internal/tools"
	"github.com/xgk1906806hjy/Hi-Agent/internal/tui"
)

// app 持有对话、会话管理与当前 UI，以及空闲中断与退出一次化状态。
type app struct {
	chat    *chat.Chat
	mgr     *session.Manager
	ui      ui
	workDir string

	busy     atomic.Bool
	exitOnce sync.Once
}

func main() {
	// 子命令：hi-agent setup-env [path/to/.env]
	if len(os.Args) > 1 && os.Args[1] == "setup-env" {
		src := ".env"
		if len(os.Args) > 2 {
			src = os.Args[2]
		}
		res, err := config.ApplyEnvFile(src)
		if err != nil {
			color.Err(color.Sys, err.Error())
			os.Exit(1)
		}
		color.Out(color.Sys, "已从 "+res.Source+" 配置环境：", true)
		color.Out(color.Sys, "  全局配置 → "+res.Global, true)
		if len(res.UserEnv) > 0 {
			color.Out(color.Sys, "  用户环境变量 → "+strings.Join(res.UserEnv, ", "), true)
		}
		if res.ShellHint != "" {
			color.Out(color.Tool, res.ShellHint, true)
		}
		color.Out(color.Sys, "之后可在任意目录执行：hi-agent", true)
		return
	}

	// —— 启动：配置、会话、UI、用量钩子、空闲中断 ——
	cfg := config.Load()
	c := chat.New(cfg.BaseURL, cfg.APIKey, cfg.Model)

	data, err := session.Load()
	loadErr := ""
	if err != nil {
		loadErr = "加载会话失败：" + err.Error() + "，将使用空会话"
		data = session.NewData()
	}
	a := &app{chat: c, mgr: session.NewManager(c, data), workDir: cfg.WorkDir}

	in := bufio.NewReader(os.Stdin)
	a.ui = a.newUI(in)
	c.SetUsageHook(a.ui.Refresh)
	tools.SetConfirm(a.ui.Confirm)
	a.watchIdleInterrupt()

	if loadErr != "" {
		a.ui.Error(loadErr)
	}
	a.ui.Info(fmt.Sprintf(
		"Hi-agent —— 工作目录：%s\n模型：%s · 会话：%s · 输入 /help 查看命令\n文件工具相对当前目录读写；会话落在 .geekagent/",
		cfg.WorkDir, cfg.Model, a.mgr.Current(),
	))
	if len(cfg.EnvFiles) > 0 {
		a.ui.Info("已加载配置：" + strings.Join(cfg.EnvFiles, " → "))
	}
	// —— 主循环：读行 → 斜杠命令 / 普通对话 ——
	for {
		line, err := a.ui.ReadLine("You › ")
		if err != nil {
			break
		}
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "/") {
			if a.handleCommand(line) {
				return
			}
			continue
		}
		a.reply(line)
	}
	// —— 退出：EOF 等走到此处 ——
	a.exit()
}

// newUI 默认在终端中启用 TUI；GEEKAGENT_TUI=0 或非终端时用纯文本输出。
func (a *app) newUI(in *bufio.Reader) ui {
	if strings.TrimSpace(os.Getenv("GEEKAGENT_TUI")) == "0" || !tui.Supported() {
		return &plainUI{in: in}
	}
	t := tui.New(in, a.panel)
	if err := t.Start(); err != nil {
		color.Err(color.Sys, err.Error()+"，改用纯文本模式")
		return &plainUI{in: in}
	}
	return t
}

// panel 每帧向 TUI 提供模型/会话/上下文与用量快照。
func (a *app) panel() tui.Panel {
	u := a.chat.Usage()
	return tui.Panel{
		Model:          a.chat.Model(),
		Session:        a.mgr.Current(),
		WorkDir:        a.workDir,
		CtxTokens:      u.ContextTokens,
		CtxEstimated:   u.ContextEstimated,
		CtxWindow:      u.ContextWindow,
		HistChars:      u.HistoryChars,
		HistLimit:      u.HistoryLimit,
		Messages:       u.Messages,
		TurnPrompt:     u.Turn.Prompt,
		TurnCompletion: u.Turn.Completion,
		TurnRequests:   u.Turn.Requests,
		CumPrompt:      u.Cumulative.Prompt,
		CumCompletion:  u.Cumulative.Completion,
		CumRequests:    u.Cumulative.Requests,
		ToolTurns:      u.LastToolTurns,
		ToolCalls:      u.LastToolCalls,
	}
}

// reply 一轮普通对话：回显 → StreamReply（进度/正文分流）→ 中断不退出。
func (a *app) reply(line string) {
	a.ui.UserEcho(line)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	a.busy.Store(true)
	a.ui.BeginReply()
	err := a.chat.StreamReply(ctx, line, func(delta string) {
		if isProgress(delta) {
			a.ui.Progress(delta)
			return
		}
		a.ui.Model(delta)
	})
	stop()
	a.busy.Store(false)
	a.ui.EndReply()
	if errors.Is(err, chat.ErrInterrupted) {
		a.ui.Notice("（已中断当前回复，可继续提问）")
		return
	}
	if err != nil {
		a.ui.Error("请求失败：" + err.Error())
	}
}

// isProgress 识别工具/压缩/预算等进度前缀，供 UI 与正文分流。
func isProgress(delta string) bool {
	for _, p := range []string{"\n[调用工具", "\n[历史压缩", "\n[工具轮次", "\n[重复工具", "\n[回复预算", "\n[工具集"} {
		if strings.HasPrefix(delta, p) {
			return true
		}
	}
	return false
}

// watchIdleInterrupt 空闲时 Ctrl+C 走正常退出（保存会话、恢复终端）；
// 回复中 busy=true，信号交给 NotifyContext 只中断当前轮。
func (a *app) watchIdleInterrupt() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	go func() {
		for range ch {
			if a.busy.Load() {
				continue
			}
			a.exit()
			os.Exit(0)
		}
	}()
}

// handleCommand 处理斜杠命令；返回 true 表示应退出进程。
func (a *app) handleCommand(line string) bool {
	fields := strings.Fields(line)
	cmd := fields[0]
	args := fields[1:]
	c, mgr := a.chat, a.mgr

	switch cmd {
	case "/help":
		a.ui.Info(helpText())
	case "/pwd":
		a.ui.Info("工作目录：" + a.workDir + "\n（ls/read/write/run_shell 均相对此目录；会话：" + session.Path() + "）")
	case "/status":
		a.ui.Info("工作目录：" + a.workDir + "\n当前会话：" + mgr.Current() + "\n" + c.Status())
	case "/reset":
		c.Reset()
		mgr.SyncFromChat()
		a.ui.Info("（已清空当前会话记忆）")
	case "/compact":
		a.ui.Notice("[" + c.Compact(context.Background()) + "]")
		mgr.SyncFromChat()
	case "/sessions":
		a.ui.Info(strings.Join(mgr.ListLines(), "\n"))
	case "/new":
		id := ""
		if len(args) > 0 {
			id = args[0]
		}
		newID, err := mgr.NewSession(id)
		if err != nil {
			a.ui.Error(err.Error())
			break
		}
		a.ui.Info(fmt.Sprintf("（已新建并切换到会话 %s）", newID))
	case "/open":
		if len(args) < 1 {
			a.ui.Info("用法：/open <会话ID>")
			break
		}
		if err := mgr.Open(args[0]); err != nil {
			a.ui.Error(err.Error())
			break
		}
		a.ui.Info(fmt.Sprintf("（已切换到会话 %s）", mgr.Current()))
	case "/save":
		if err := mgr.Save(); err != nil {
			a.ui.Error("保存失败：" + err.Error())
			break
		}
		a.ui.Info(fmt.Sprintf("（已保存到 %s）", session.Path()))
	case "/load":
		cur, err := mgr.Load()
		if err != nil {
			a.ui.Error("加载失败：" + err.Error())
			break
		}
		a.ui.Info(fmt.Sprintf("（已从 %s 加载，当前会话 %s）", session.Path(), cur))
	case "/exit":
		a.exit()
		return true
	default:
		a.ui.Info(fmt.Sprintf("未知命令：%s（输入 /help 查看）", cmd))
	}
	a.ui.Refresh()
	return false
}

// exit 保存会话、关闭 UI（恢复终端）并打印结果；并发调用只执行一次。
func (a *app) exit() {
	a.exitOnce.Do(func() {
		renamed, err := a.mgr.PrepareExit()
		a.ui.Close()
		switch {
		case err != nil:
			color.Err(color.Sys, "退出保存失败："+err.Error())
		case renamed != "":
			color.Out(color.Sys, fmt.Sprintf("（default 已重命名为 %s 并保存）", renamed), true)
		default:
			color.Out(color.Sys, fmt.Sprintf("（会话已保存到 %s）", session.Path()), true)
		}
		color.Out(color.Sys, "bye", true)
	})
}

// helpText 返回 /help 文案（含会话路径、已注册工具与相关环境变量）。
func helpText() string {
	return fmt.Sprintf(`可用命令：
  /help              显示帮助
  /pwd               显示当前工作目录（工具读写基准）
  /status            显示工作目录、会话、tokens 与护栏状态
  /sessions          列出全部会话（* 为当前）
  /new [id]          新建空会话并切换（可省略 id，自动 8 位）
  /open <id>         切换到已有会话
  /save              保存全部会话到 %s
  /load              从磁盘重新加载（丢弃未保存改动）
  /reset             清空当前会话记忆
  /compact           立即压缩旧对话摘要
  /exit              保存并退出（default 有内容时改名为 8 位 ID）
在任意项目目录执行 hi-agent：读写相对当前目录；会话写入该目录下 .geekagent/。
API Key：项目 .env、上级目录 .env，或全局 %%USERPROFILE%%\\.hi-agent\\.env（Linux/macOS: ~/.hi-agent/.env）。
回复进行中按 Ctrl+C 中断当前轮；空闲时按 Ctrl+C 保存并退出。
已注册工具：%s
界面：终端中默认 TUI，GEEKAGENT_TUI=0 使用纯文本。
用量：GEEKAGENT_CONTEXT_WINDOW（默认 128000）；GEEKAGENT_STREAM_USAGE=0 关闭流式用量。
历史超长时自动压缩（GEEKAGENT_MAX_HISTORY，默认 4000 字符）。
工具轮次上限 GEEKAGENT_MAX_TOOL_TURNS（默认 8）；
重复熔断 GEEKAGENT_MAX_DUP_TOOLS（默认 3）/ GEEKAGENT_DUP_WINDOW（默认 12）；
回复预算 GEEKAGENT_MAX_REPLY_CHARS（默认 120000，0=关闭）；
工具收缩 GEEKAGENT_SHRINK_TOOLS_AFTER（默认约 3/4 轮次起只读，0=关闭）。`, session.Path(), strings.Join(tools.Names(), ", "))
}
