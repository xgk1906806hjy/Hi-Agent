//go:build desktop

package gui

import (
	"context"
	"embed"
	"sync"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/xgk1906806hjy/Hi-Agent/internal/plan"
	"github.com/xgk1906806hjy/Hi-Agent/internal/session"
	"github.com/xgk1906806hjy/Hi-Agent/internal/workspace"
)

//go:embed all:frontend/dist
var assets embed.FS

// App 是绑定给前端的对象：前端以 window.go.gui.App.<方法> 调用。
// 项目/会话操作统一翻译成斜杠命令送进 REPL，引擎侧无需新分支。
type App struct {
	adapter *Adapter

	mu  sync.Mutex
	ctx context.Context
}

type wailsEmitter struct{ ctx context.Context }

func (w wailsEmitter) Emit(event string, payload any) {
	if payload == nil {
		runtime.EventsEmit(w.ctx, event)
		return
	}
	runtime.EventsEmit(w.ctx, event, payload)
}

func newApp(adapter *Adapter) *App { return &App{adapter: adapter} }

// run 阻塞运行窗口，直到关闭。
func (a *App) run() {
	err := wails.Run(&options.App{
		Title:            "Hi-agent",
		Width:            1180,
		Height:           780,
		MinWidth:         760,
		MinHeight:        520,
		Frameless:        true,
		BackgroundColour: windowBackground(),
		AssetServer:      &assetserver.Options{Assets: assets},
		OnStartup:        a.startup,
		OnShutdown:       func(context.Context) { a.adapter.Closed() },
		Bind:             []interface{}{a},
		Windows:          &windows.Options{Theme: windows.SystemDefault},
	})
	if err != nil {
		a.adapter.Error("窗口启动失败：" + err.Error())
	}
}

func (a *App) startup(ctx context.Context) {
	a.mu.Lock()
	a.ctx = ctx
	a.mu.Unlock()
}

func (a *App) quit() {
	a.mu.Lock()
	ctx := a.ctx
	a.mu.Unlock()
	if ctx != nil {
		runtime.Quit(ctx)
	}
}

// Ready 前端注册完事件监听后调用，之后才开始推送事件。
func (a *App) Ready() Snapshot {
	a.mu.Lock()
	ctx := a.ctx
	a.mu.Unlock()
	if ctx != nil {
		a.adapter.Attach(wailsEmitter{ctx})
	}
	return a.adapter.Snapshot()
}

// Snapshot 拉取最近状态。
func (a *App) Snapshot() Snapshot { return a.adapter.Snapshot() }

// Submit 发送消息或斜杠命令。
func (a *App) Submit(text string) { a.adapter.Submit(text) }

// Stop 中断当前回复。
func (a *App) Stop() { a.adapter.Stop() }

// Answer 回复确认/选择对话框。
func (a *App) Answer(id string, value int) { a.adapter.Answer(id, value) }

// requireIdle 侧栏操作只在 REPL 空闲于主提示时执行，避免被当成某条命令的参数输入。
func (a *App) requireIdle() bool {
	if a.adapter.Idle() {
		return true
	}
	a.adapter.Notice("请先完成当前输入或等待回复结束")
	return false
}

// AddProject 弹出系统文件夹对话框：无论是否当前目录都写入注册表并刷新侧栏；
// 与当前工作区不同时再提交 /project 切换。取消返回空串。
func (a *App) AddProject() string {
	if !a.requireIdle() {
		return ""
	}
	a.mu.Lock()
	ctx := a.ctx
	a.mu.Unlock()
	if ctx == nil {
		a.adapter.Error("窗口尚未就绪，无法选择项目文件夹")
		return ""
	}
	dir, err := runtime.OpenDirectoryDialog(ctx, runtime.OpenDialogOptions{Title: "选择项目文件夹"})
	if err != nil {
		a.adapter.Error("打开文件夹对话框失败：" + err.Error())
		return ""
	}
	if dir == "" || !a.requireIdle() {
		return ""
	}
	if err := workspace.Remember(dir); err != nil {
		a.adapter.Error("记录项目失败：" + err.Error())
		return ""
	}
	a.adapter.RefreshNav()
	cur := a.adapter.Snapshot().Status.WorkDir
	if !workspace.SamePath(dir, cur) {
		a.adapter.Submit("/project " + dir)
	}
	return dir
}

// OpenProject 切换到已登记的项目（先 Remember 再 /project）。
func (a *App) OpenProject(path string) {
	if path == "" || !a.requireIdle() {
		return
	}
	_ = workspace.Remember(path)
	if workspace.SamePath(path, a.adapter.Snapshot().Status.WorkDir) {
		a.adapter.RefreshNav()
		return
	}
	a.adapter.Submit("/project " + path)
}

// RemoveProject 从登记列表移除项目（不删除磁盘文件）；前端自行更新侧栏，
// 当前项目不可移除（下次刷新会被重新登记）。
func (a *App) RemoveProject(path string) error {
	return workspace.Remove(path)
}

// OpenSession 打开某项目下的会话；跨项目时只发一条 /open <id> <目录>，
// 由 REPL 先切项目、失败即止。
func (a *App) OpenSession(path, id string) {
	if id == "" || !a.requireIdle() {
		return
	}
	if path == "" || workspace.SamePath(path, a.adapter.Snapshot().Status.WorkDir) {
		a.adapter.Submit("/open " + id)
		return
	}
	a.adapter.Submit("/open " + id + " " + path)
}

// NewSession 在当前项目新建会话。
func (a *App) NewSession() {
	if a.requireIdle() {
		a.adapter.Submit("/new")
	}
}

// RenameSession 重命名会话。当前项目走 REPL；其它项目直接改该目录的会话文件。
func (a *App) RenameSession(path, oldID, newID string) {
	if oldID == "" || newID == "" || oldID == newID || !a.requireIdle() {
		return
	}
	if path == "" || workspace.SamePath(path, a.adapter.Snapshot().Status.WorkDir) {
		a.adapter.Submit("/rename " + oldID + " " + newID)
		return
	}
	if err := session.RenameIn(path, oldID, newID); err != nil {
		a.adapter.Error(err.Error())
		return
	}
	if err := plan.Move(path, oldID, newID); err != nil {
		a.adapter.Error("任务清单：" + err.Error())
	}
	a.adapter.RefreshNav()
}

// SetTheme 把原生窗口背景改成与页面一致。mode 取 light 或 dark（跟随系统时由前端算好再传入）。
func (a *App) SetTheme(mode string) {
	a.mu.Lock()
	ctx := a.ctx
	a.mu.Unlock()
	if ctx == nil {
		return
	}
	if mode == "dark" {
		// 与 --color-bg 深色一致
		runtime.WindowSetBackgroundColour(ctx, 26, 25, 23, 255)
		return
	}
	// 与 --color-bg 浅色 #f6f5f2 一致
	runtime.WindowSetBackgroundColour(ctx, 246, 245, 242, 255)
}

// WindowMinimise 最小化无边框窗口。
func (a *App) WindowMinimise() {
	if ctx := a.runtimeCtx(); ctx != nil {
		runtime.WindowMinimise(ctx)
	}
}

// WindowToggleMaximise 最大化 / 还原。
func (a *App) WindowToggleMaximise() {
	if ctx := a.runtimeCtx(); ctx != nil {
		runtime.WindowToggleMaximise(ctx)
	}
}

// WindowClose 关闭窗口（随后 REPL 会走 exit 保存会话）。
func (a *App) WindowClose() {
	a.quit()
}

func (a *App) runtimeCtx() context.Context {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.ctx
}

// DeleteSession 删除会话。当前项目走 REPL（含确认）；其它项目确认后直接改文件。
func (a *App) DeleteSession(path, id string) {
	if id == "" || !a.requireIdle() {
		return
	}
	if path == "" || workspace.SamePath(path, a.adapter.Snapshot().Status.WorkDir) {
		a.adapter.Submit("/delete " + id)
		return
	}
	if !a.adapter.Confirm("确认删除会话 " + id + "？此操作不可撤销。") {
		return
	}
	if err := plan.Drop(path, id); err != nil {
		a.adapter.Error("任务清单：" + err.Error())
	}
	if err := session.DeleteIn(path, id); err != nil {
		a.adapter.Error(err.Error())
		return
	}
	a.adapter.RefreshNav()
}
