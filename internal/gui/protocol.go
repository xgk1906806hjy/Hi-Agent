// Package gui 是基于 Wails 的桌面界面：Go 侧把 ui.UI 调用翻译成事件推给 React 前端，
// 前端通过绑定方法把用户操作送回来。
//
// protocol.go / adapter.go 不依赖 Wails，可在普通 go test 下测试；
// Wails 绑定与窗口启动只在 desktop 构建标签下编译（app.go / register.go）。
// 事件名与载荷须与 frontend/src/bridge/types.ts 保持一致。
package gui

import "github.com/xgk1906806hjy/Hi-Agent/internal/ui"

// Go → 前端事件名。
const (
	EvUser     = "chat:user"      // TextPayload：用户输入回显
	EvDelta    = "chat:delta"     // TextPayload：助手正文增量
	EvProgress = "chat:progress"  // TextPayload：工具/压缩等过程
	EvNotice   = "chat:notice"    // TextPayload：轻提示
	EvInfo     = "chat:info"      // TextPayload：系统信息（命令输出等）
	EvError    = "chat:error"     // TextPayload
	EvReset    = "chat:reset"     // ResetPayload：切换项目/会话后整体重绘
	EvTurnBeg  = "turn:begin"     // 无载荷
	EvTurnEnd  = "turn:end"       // 无载荷
	EvStatus   = "status"         // StatusPayload
	EvNav      = "nav"            // NavPayload
	EvConfirm  = "dialog:confirm" // ConfirmPayload
	EvSelect   = "dialog:select"  // SelectPayload
	EvDismiss  = "dialog:dismiss" // DismissPayload：对话框被取消（窗口关闭）
	EvPrompt   = "input:prompt"   // PromptPayload：REPL 正在等待输入
)

// TextPayload 文本类事件。
type TextPayload struct {
	Text string `json:"text"`
}

// TurnPayload 一条历史消息。
type TurnPayload struct {
	Role string `json:"role"`
	Text string `json:"text"`
}

// ResetPayload 重绘对话流。
type ResetPayload struct {
	Turns []TurnPayload `json:"turns"`
}

// StatusPayload 镜像 ui.Status。
type StatusPayload struct {
	Model          string        `json:"model"`
	Provider       string        `json:"provider"`
	Session        string        `json:"session"`
	WorkDir        string        `json:"workDir"`
	Busy           bool          `json:"busy"`
	CtxTokens      int           `json:"ctxTokens"`
	CtxEstimated   bool          `json:"ctxEstimated"`
	CtxWindow      int           `json:"ctxWindow"`
	TurnPrompt     int           `json:"turnPrompt"`
	TurnCompletion int           `json:"turnCompletion"`
	Messages       int           `json:"messages"`
	Policy         []string      `json:"policy"`
	Todos          []TodoPayload `json:"todos"`
}

// TodoPayload 镜像 ui.Todo。
type TodoPayload struct {
	ID      string `json:"id"`
	Content string `json:"content"`
	Status  string `json:"status"`
}

// NavSessionPayload 镜像 ui.NavSession。
type NavSessionPayload struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	UpdatedAt int64  `json:"updatedAt"` // Unix 毫秒；0 表示未落盘
	Messages  int    `json:"messages"`
	Current   bool   `json:"current"`
}

// SlashPayload 一条可补全的斜杠命令。
type SlashPayload struct {
	Name string `json:"name"`
	Desc string `json:"desc"`
}

// NavProjectPayload 镜像 ui.NavProject。
type NavProjectPayload struct {
	Name     string              `json:"name"`
	Path     string              `json:"path"`
	Current  bool                `json:"current"`
	Sessions []NavSessionPayload `json:"sessions"`
}

// NavPayload 镜像 ui.Nav。
type NavPayload struct {
	Projects []NavProjectPayload `json:"projects"`
}

// ConfirmPayload 请求是/否确认；前端以 Answer(id, 1|0) 回复。
type ConfirmPayload struct {
	ID     string `json:"id"`
	Prompt string `json:"prompt"`
}

// SelectPayload 请求单选；前端以 Answer(id, index) 回复，-1 表示取消。
type SelectPayload struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Items    []string `json:"items"`
	Selected int      `json:"selected"`
}

// DismissPayload 撤销尚未回答的对话框。
type DismissPayload struct {
	ID string `json:"id"`
}

// PromptPayload 输入框提示语。
type PromptPayload struct {
	Prompt string `json:"prompt"`
}

// Snapshot 前端启动或刷新时一次性拉取的全量状态。
type Snapshot struct {
	Status StatusPayload  `json:"status"`
	Nav    NavPayload     `json:"nav"`
	Prompt string         `json:"prompt"`
	Slash  []SlashPayload `json:"slash"`
}

func toStatus(s ui.Status) StatusPayload {
	return StatusPayload{
		Model: s.Model, Provider: s.Provider, Session: s.Session, WorkDir: s.WorkDir, Busy: s.Busy,
		CtxTokens: s.CtxTokens, CtxEstimated: s.CtxEstimated, CtxWindow: s.CtxWindow,
		TurnPrompt: s.TurnPrompt, TurnCompletion: s.TurnCompletion, Messages: s.Messages,
		Policy: s.Policy, Todos: toTodos(s.Todos),
	}
}

func toTodos(in []ui.Todo) []TodoPayload {
	if len(in) == 0 {
		return []TodoPayload{}
	}
	out := make([]TodoPayload, len(in))
	for i, t := range in {
		out[i] = TodoPayload{ID: t.ID, Content: t.Content, Status: t.Status}
	}
	return out
}

func toNav(n ui.Nav) NavPayload {
	out := NavPayload{Projects: make([]NavProjectPayload, 0, len(n.Projects))}
	for _, p := range n.Projects {
		pp := NavProjectPayload{Name: p.Name, Path: p.Path, Current: p.Current, Sessions: make([]NavSessionPayload, 0, len(p.Sessions))}
		for _, s := range p.Sessions {
			var ms int64
			if !s.UpdatedAt.IsZero() {
				ms = s.UpdatedAt.UnixMilli()
			}
			pp.Sessions = append(pp.Sessions, NavSessionPayload{
				ID: s.ID, Title: s.Title, UpdatedAt: ms, Messages: s.Messages, Current: s.Current,
			})
		}
		out.Projects = append(out.Projects, pp)
	}
	return out
}

func toTurns(in []ui.Turn) []TurnPayload {
	out := make([]TurnPayload, 0, len(in))
	for _, t := range in {
		out = append(out, TurnPayload{Role: t.Role, Text: t.Text})
	}
	return out
}
