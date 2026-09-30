package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"

	"hi-agent/internal/chat"
	"hi-agent/internal/color"
	"hi-agent/internal/config"
	"hi-agent/internal/session"
	"hi-agent/internal/tools"
)

func main() {
	cfg := config.Load()
	c := chat.New(cfg.BaseURL, cfg.APIKey, cfg.Model)

	data, err := session.Load()
	if err != nil {
		color.Err(color.Sys, "加载会话失败："+err.Error()+"，将使用空会话")
		data = session.NewData()
	}
	mgr := session.NewManager(c, data)

	in := bufio.NewReader(os.Stdin)
	tools.SetConfirm(func(prompt string) bool {
		color.Out(color.Tool, prompt+" [y/N] ", false)
		line, err := in.ReadString('\n')
		if err != nil {
			return false
		}
		ans := strings.ToLower(strings.TrimSpace(line))
		return ans == "y" || ans == "yes"
	})

	color.Out(color.Sys, fmt.Sprintf(
		"Hi-agent —— 多会话（模型：%s，会话：%s，输入 /help 查看命令）",
		cfg.Model, mgr.Current(),
	), true)

	for {
		fmt.Fprint(os.Stdout, color.Paint(color.User, "You › "))
		line, err := in.ReadString('\n')
		if err != nil {
			break
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "/") {
			if handleCommand(line, c, mgr) {
				return
			}
			continue
		}

		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		color.Out(color.Sys, "", true)
		err = c.StreamReply(ctx, line, func(delta string) {
			if strings.HasPrefix(delta, "\n[调用工具") ||
				strings.HasPrefix(delta, "\n[历史压缩") ||
				strings.HasPrefix(delta, "\n[工具轮次") ||
				strings.HasPrefix(delta, "\n[重复工具") ||
				strings.HasPrefix(delta, "\n[回复预算") ||
				strings.HasPrefix(delta, "\n[工具集") {
				color.Out(color.Tool, delta, false)
				return
			}
			color.Out(color.Model, delta, false)
		})
		stop()
		color.Out(color.Sys, "", true)
		if errors.Is(err, chat.ErrInterrupted) {
			color.Out(color.Tool, "\n（已中断当前回复，可继续提问）", true)
			continue
		}
		if err != nil {
			color.Err(color.Sys, "\n请求失败："+err.Error())
		}
	}

	exitWithSave(mgr)
}

// handleCommand 处理斜杠命令；返回 true 表示应退出进程。
func handleCommand(line string, c *chat.Chat, mgr *session.Manager) bool {
	fields := strings.Fields(line)
	cmd := fields[0]
	args := fields[1:]

	switch cmd {
	case "/help":
		printHelp()
	case "/status":
		color.Out(color.Sys, "当前会话："+mgr.Current()+"\n"+c.Status(), true)
	case "/reset":
		c.Reset()
		mgr.SyncFromChat()
		color.Out(color.Sys, "（已清空当前会话记忆）", true)
	case "/compact":
		color.Out(color.Tool, "["+c.Compact(context.Background())+"]", true)
		mgr.SyncFromChat()
	case "/sessions":
		for _, row := range mgr.ListLines() {
			color.Out(color.Sys, row, true)
		}
	case "/new":
		id := ""
		if len(args) > 0 {
			id = args[0]
		}
		newID, err := mgr.NewSession(id)
		if err != nil {
			color.Err(color.Sys, err.Error())
			return false
		}
		color.Out(color.Sys, fmt.Sprintf("（已新建并切换到会话 %s）", newID), true)
	case "/open":
		if len(args) < 1 {
			color.Out(color.Sys, "用法：/open <会话ID>", true)
			return false
		}
		if err := mgr.Open(args[0]); err != nil {
			color.Err(color.Sys, err.Error())
			return false
		}
		color.Out(color.Sys, fmt.Sprintf("（已切换到会话 %s）", mgr.Current()), true)
	case "/save":
		if err := mgr.Save(); err != nil {
			color.Err(color.Sys, "保存失败："+err.Error())
			return false
		}
		color.Out(color.Sys, fmt.Sprintf("（已保存到 %s）", session.Path()), true)
	case "/load":
		cur, err := mgr.Load()
		if err != nil {
			color.Err(color.Sys, "加载失败："+err.Error())
			return false
		}
		color.Out(color.Sys, fmt.Sprintf("（已从 %s 加载，当前会话 %s）", session.Path(), cur), true)
	case "/exit":
		exitWithSave(mgr)
		return true
	default:
		color.Out(color.Sys, fmt.Sprintf("未知命令：%s（输入 /help 查看）", cmd), true)
	}
	return false
}

func exitWithSave(mgr *session.Manager) {
	renamed, err := mgr.PrepareExit()
	if err != nil {
		color.Err(color.Sys, "退出保存失败："+err.Error())
	} else if renamed != "" {
		color.Out(color.Sys, fmt.Sprintf("（default 已重命名为 %s 并保存）", renamed), true)
	} else {
		color.Out(color.Sys, fmt.Sprintf("（会话已保存到 %s）", session.Path()), true)
	}
	color.Out(color.Sys, "bye", true)
}

func printHelp() {
	color.Out(color.Sys, fmt.Sprintf(`可用命令：
  /help              显示帮助
  /status            显示当前会话与护栏状态
  /sessions          列出全部会话（* 为当前）
  /new [id]          新建空会话并切换（可省略 id，自动 8 位）
  /open <id>         切换到已有会话
  /save              保存全部会话到 %s
  /load              从磁盘重新加载（丢弃未保存改动）
  /reset             清空当前会话记忆
  /compact           立即压缩旧对话摘要
  /exit              保存并退出（default 有内容时改名为 8 位 ID）
回复进行中按 Ctrl+C 可中断当前轮（不退出程序）。
已注册工具：%s
历史超长时自动压缩（GEEKAGENT_MAX_HISTORY，默认 4000 字符）。
工具轮次上限 GEEKAGENT_MAX_TOOL_TURNS（默认 8）；
重复熔断 GEEKAGENT_MAX_DUP_TOOLS（默认 3）/ GEEKAGENT_DUP_WINDOW（默认 12）；
回复预算 GEEKAGENT_MAX_REPLY_CHARS（默认 120000，0=关闭）；
工具收缩 GEEKAGENT_SHRINK_TOOLS_AFTER（默认约 3/4 轮次起只读，0=关闭）。`, session.Path(), strings.Join(tools.Names(), ", ")), true)
}
