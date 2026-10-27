// hi-agent 进程入口与 REPL：加载配置/会话，经 ui 工厂创建界面，驱动 chat.StreamReply。
// UI 可插拔（plain / tui / gui；web 预留）；退出时保存会话并 Close UI。
package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sashabaranov/go-openai"

	"github.com/xgk1906806hjy/Hi-Agent/internal/chat"
	"github.com/xgk1906806hjy/Hi-Agent/internal/color"
	"github.com/xgk1906806hjy/Hi-Agent/internal/config"
	"github.com/xgk1906806hjy/Hi-Agent/internal/memory"
	"github.com/xgk1906806hjy/Hi-Agent/internal/permission"
	"github.com/xgk1906806hjy/Hi-Agent/internal/plan"
	"github.com/xgk1906806hjy/Hi-Agent/internal/session"
	"github.com/xgk1906806hjy/Hi-Agent/internal/skills"
	"github.com/xgk1906806hjy/Hi-Agent/internal/tools"
	"github.com/xgk1906806hjy/Hi-Agent/internal/tui"
	"github.com/xgk1906806hjy/Hi-Agent/internal/ui"
	"github.com/xgk1906806hjy/Hi-Agent/internal/workspace"
)

// app 持有对话、会话管理与当前 UI，以及空闲中断与退出一次化状态。
type app struct {
	chat             *chat.Chat
	mgr              *session.Manager
	ui               ui.UI
	workDir          string
	provider         string
	baseURL          string
	apiKey           string
	policy           *permission.Policy
	policyAt         time.Time
	policyExists     bool
	userPolicyAt     time.Time
	userPolicyExists bool
	policyErr        string

	skillMu      sync.RWMutex
	skillList    []skills.Skill
	activeSkills map[string]bool // key 为小写技能名；仅在当前进程内有效

	busy     atomic.Bool
	exitOnce sync.Once
}

// defaultUI 构建时注入（-ldflags "-X main.defaultUI=gui"），用于双击即开窗口的 GUI 版 exe；
// 进程环境里已设 GEEKAGENT_UI / HI_AGENT_UI 时以环境为准。
var defaultUI string

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

	applyLaunchDefaults()

	// —— 启动：配置、会话、UI、用量钩子、空闲中断 ——
	// GUI 可能无控制台：配置缺失时不退出，进窗口后提示 /model。
	var cfg config.Config
	cfgErr := ""
	if ui.ResolveKind() == ui.KindGUI {
		var err error
		cfg, err = config.LoadE()
		if err != nil {
			cfgErr = "模型未配置：" + err.Error() + "\n请点底部 /model →「切换供应商 / 中转站」填写，并选择写入全局配置。"
		}
	} else {
		cfg = config.Load()
	}
	// .env Overload 可能改掉界面变量；GUI 专用包（ldflags 注入了 defaultUI）始终强制进桌面窗。
	forcePackagedUI()
	c := chat.New(cfg.BaseURL, cfg.APIKey, cfg.Model)
	c.SetRecall(memory.RecallNote)

	data, err := session.Load()
	loadErr := ""
	if err != nil {
		loadErr = "加载会话失败：" + err.Error() + "，将使用空会话"
		data = session.NewData()
	}
	if cfgErr != "" {
		loadErr = strings.TrimSpace(cfgErr + "\n" + loadErr)
	}
	a := &app{
		chat: c, mgr: session.NewManager(c, data), workDir: cfg.WorkDir,
		provider: cfg.Provider, baseURL: cfg.BaseURL, apiKey: cfg.APIKey,
		activeSkills: map[string]bool{},
	}

	in := bufio.NewReader(os.Stdin)
	a.ui = a.newUI(in)
	a.reloadSkills()
	c.SetUsageHook(a.ui.Refresh)
	tools.SetConfirm(a.ui.Confirm)
	tools.SetDelegator(func(ctx context.Context, task string) (string, error) {
		return a.chat.Delegate(ctx, task, func(delta string) {
			a.ui.Progress(delta)
		})
	})
	if err := a.installPermissions(); err != nil {
		loadErr = strings.TrimSpace(loadErr + "\n权限配置：" + err.Error())
	}
	if err := plan.Activate(a.workDir, a.mgr.Current()); err != nil {
		loadErr = strings.TrimSpace(loadErr + "\n任务清单：" + err.Error())
	}
	a.watchIdleInterrupt()

	// GUI 须占主线程 Run；REPL 放到 goroutine。
	if runner, ok := a.ui.(ui.DesktopRunner); ok {
		go func() {
			a.bannerAndLoop(cfg, loadErr)
			a.exit()
		}()
		runner.Run()
		a.exit()
		return
	}

	a.bannerAndLoop(cfg, loadErr)
	a.exit()
}

// applyLaunchDefaults 处理双击 / 拖放启动：注入构建期默认界面；
// 首个参数是目录时切换到该目录作为工作区（把文件夹拖到 exe 上、或快捷方式带参数）。
func applyLaunchDefaults() {
	forcePackagedUI()
	if len(os.Args) > 1 {
		if st, err := os.Stat(os.Args[1]); err == nil && st.IsDir() {
			_ = os.Chdir(os.Args[1])
		}
	}
}

// forcePackagedUI：scripts/build-gui 注入了 defaultUI=gui 的 exe 始终开桌面窗，
// 不受 .env / 用户环境里的 GEEKAGENT_UI=tui 影响。普通 hi-agent（无注入）不受影响。
func forcePackagedUI() {
	if defaultUI == "" {
		return
	}
	_ = os.Setenv("GEEKAGENT_UI", defaultUI)
}

// bannerAndLoop 打印启动信息后进入读行主循环。
func (a *app) bannerAndLoop(cfg config.Config, loadErr string) {
	if loadErr != "" {
		a.ui.Error(loadErr)
	}
	// GUI：短欢迎，避免把帮助全文灌进聊天区；TUI/plain 仍给稍全的一行摘要。
	if _, isGUI := a.ui.(ui.DesktopRunner); isGUI {
		a.ui.Info(fmt.Sprintf(
			"工作目录 %s · %s / %s · 会话 %s\n输入消息开始对话；点 /model 配置模型，/help 查看帮助。",
			cfg.WorkDir, cfg.Provider, cfg.Model, a.mgr.Current(),
		))
	} else {
		a.ui.Info(fmt.Sprintf(
			"Hi-agent —— 工作目录：%s\n供应商：%s · 模型：%s · 会话：%s\n接口：%s\n输入 / 展开命令（含 /model 配置模型）；/help 查看帮助",
			cfg.WorkDir, cfg.Provider, cfg.Model, a.mgr.Current(), cfg.BaseURL,
		))
		if len(cfg.EnvFiles) > 0 {
			a.ui.Info("已加载配置：" + strings.Join(cfg.EnvFiles, " → "))
		}
	}
	for {
		line, err := a.ui.ReadLine(ui.MainPrompt)
		if err != nil {
			if errors.Is(err, ui.ErrInterrupted) {
				return
			}
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
}

// newUI 经抽象工厂创建 UI（GEEKAGENT_UI / HI_AGENT_UI，兼容 GEEKAGENT_TUI=0）。
func (a *app) newUI(in *bufio.Reader) ui.UI {
	kind := ui.ResolveKind()
	u, err := ui.Create(kind, ui.Options{
		In:            in,
		SlashCommands: slashCommands(),
		PanelFn:       a.panel,
		StatusFn:      a.statusSnapshot,
		NavFn:         a.navSnapshot,
		HistoryFn:     a.historySnapshot,
	})
	if err != nil {
		color.Err(color.Sys, err.Error()+"，回退纯文本")
		u, err = ui.Create(ui.KindPlain, ui.Options{In: in, SlashCommands: slashCommands()})
		if err != nil {
			panic(err)
		}
	}
	return u
}

// navSnapshot 供桌面侧栏「项目 > 会话」树。
func (a *app) navSnapshot() ui.Nav {
	paths := workspace.Load()
	cur := filepath.Clean(a.workDir)
	has := false
	for _, p := range paths {
		if workspace.SamePath(p, cur) {
			has = true
			break
		}
	}
	if cur != "" && cur != "." && !has {
		_ = workspace.Remember(a.workDir)
		paths = workspace.Load()
		if len(paths) == 0 {
			paths = []string{a.workDir}
		}
	}
	var nav ui.Nav
	for _, p := range paths {
		proj := ui.NavProject{
			Name:    workspace.Name(p),
			Path:    p,
			Current: workspace.SamePath(p, cur),
		}
		if proj.Current {
			for _, b := range a.mgr.SessionBriefs() {
				proj.Sessions = append(proj.Sessions, ui.NavSession{
					ID: b.ID, Title: b.Title, Messages: b.Messages, UpdatedAt: b.UpdatedAt, Current: b.Current,
				})
			}
		} else {
			for _, s := range workspace.ListSessions(p) {
				proj.Sessions = append(proj.Sessions, ui.NavSession{
					ID: s.ID, Title: s.Title, Messages: s.Messages, UpdatedAt: s.UpdatedAt, Current: s.Current,
				})
			}
		}
		nav.Projects = append(nav.Projects, proj)
	}
	return nav
}

// switchProject 保存当前会话后切换工作目录，并加载该目录下的会话。
// 返回 false 表示未切换（目录无效或保存失败）；已在该目录返回 true。
func (a *app) switchProject(path string) bool {
	path = strings.Trim(strings.TrimSpace(path), `"'`)
	abs, err := filepath.Abs(path)
	if err != nil {
		a.ui.Error(err.Error())
		return false
	}
	st, statErr := os.Stat(abs)
	if statErr != nil || !st.IsDir() {
		a.ui.Error("不是有效目录：" + abs)
		return false
	}
	if workspace.SamePath(abs, a.workDir) {
		a.ui.Info("已在该项目：" + abs)
		return true
	}
	if err := a.mgr.Save(); err != nil {
		a.ui.Error("保存当前会话失败：" + err.Error())
		return false
	}
	if err := os.Chdir(abs); err != nil {
		a.ui.Error("切换目录失败：" + err.Error())
		return false
	}
	a.workDir = abs
	data, loadErr := session.Load()
	if loadErr != nil {
		a.ui.Error("加载会话失败：" + loadErr.Error())
		data = session.NewData()
	}
	a.mgr = session.NewManager(a.chat, data)
	if err := workspace.Remember(abs); err != nil {
		a.ui.Error("记录项目失败：" + err.Error())
	}
	a.reloadSkills()
	if err := plan.Activate(abs, a.mgr.Current()); err != nil {
		a.ui.Error("任务清单：" + err.Error())
	}
	if err := a.reloadPolicy(); err != nil {
		a.ui.Error("权限配置：" + err.Error())
	}
	a.ui.Info("已切换项目：" + abs + "\n当前会话：" + a.mgr.Current())
	return true
}

// installPermissions 注入 Day 8 钩子：权限策略、工作区路径检查、输出脱敏。
// 钩子读取 a 的当前字段，/project 与 /model 之后无需重新注入（策略需 reloadPolicy）。
func (a *app) installPermissions() error {
	err := a.reloadPolicy()
	tools.SetPolicy(func(name, args string) tools.Decision {
		a.ensurePolicy()
		switch a.policy.Decide(name, args) {
		case permission.Allow:
			return tools.Allow
		case permission.Deny:
			return tools.Deny
		default:
			return tools.Ask
		}
	})
	tools.SetPathGuard(func(p string) (string, error) { return permission.SafePath(a.workDir, p) })
	tools.SetRedactor(func(s string) string { return permission.Redact(s, a.apiKey) })
	return err
}

// reloadPolicy 按当前工作目录重新读取权限配置；损坏时回退默认并返回错误。
func (a *app) reloadPolicy() error {
	p, err := permission.Load(a.workDir)
	a.policy = p
	a.notePolicyFile()
	return err
}

// ensurePolicy 在项目或用户配置的修改时间变化时重新加载。损坏时只提示一次。
func (a *app) ensurePolicy() {
	exists, at := policyStamp(filepath.Join(a.workDir, filepath.FromSlash(permission.ConfigPath)))
	userExists, userAt := policyStamp(permission.UserConfigPath())
	if a.policy != nil && exists == a.policyExists && at.Equal(a.policyAt) && userExists == a.userPolicyExists && userAt.Equal(a.userPolicyAt) {
		return
	}
	loadErr := a.reloadPolicy()
	if loadErr != nil {
		if loadErr.Error() != a.policyErr {
			a.policyErr = loadErr.Error()
			a.ui.Error("权限配置：" + loadErr.Error())
		}
		return
	}
	a.policyErr = ""
}

func (a *app) notePolicyFile() {
	a.policyExists, a.policyAt = policyStamp(filepath.Join(a.workDir, filepath.FromSlash(permission.ConfigPath)))
	a.userPolicyExists, a.userPolicyAt = policyStamp(permission.UserConfigPath())
}

func policyStamp(path string) (bool, time.Time) {
	if path == "" {
		return false, time.Time{}
	}
	st, err := os.Stat(path)
	if err != nil {
		return false, time.Time{}
	}
	return true, st.ModTime()
}

// statusSnapshot 供桌面顶栏/状态区刷新。
func (a *app) statusSnapshot() ui.Status {
	a.ensurePolicy()
	p := a.panel()
	todos := plan.List()
	uiTodos := make([]ui.Todo, len(todos))
	for i, t := range todos {
		uiTodos[i] = ui.Todo{ID: t.ID, Content: t.Content, Status: t.Status}
	}
	policyLines := []string{}
	if a.policy != nil {
		policyLines = a.policy.Lines()
	}
	return ui.Status{
		Model:          p.Model,
		Provider:       a.provider,
		Session:        p.Session,
		WorkDir:        p.WorkDir,
		CtxTokens:      p.CtxTokens,
		CtxEstimated:   p.CtxEstimated,
		CtxWindow:      p.CtxWindow,
		TurnPrompt:     p.TurnPrompt,
		TurnCompletion: p.TurnCompletion,
		Messages:       p.Messages,
		Policy:         policyLines,
		Todos:          uiTodos,
		Skills:         a.activeSkillNames(),
	}
}

// historySnapshot 把当前会话里可展示的消息交给桌面界面重绘。
func (a *app) historySnapshot() []ui.Turn {
	return historyTurns(a.chat.History())
}

// toolResultRunes 重绘时每条工具结果保留的字符数。
const toolResultRunes = 300

// historyTurns 把 messages 转成可展示消息；工具调用按 ToolCallID 配上结果摘要。
func historyTurns(msgs []openai.ChatCompletionMessage) []ui.Turn {
	results := map[string]string{}
	for _, m := range msgs {
		if m.Role == openai.ChatMessageRoleTool && m.ToolCallID != "" {
			results[m.ToolCallID] = m.Content
		}
	}
	out := make([]ui.Turn, 0, len(msgs))
	for _, m := range msgs {
		switch m.Role {
		case openai.ChatMessageRoleUser:
			if strings.TrimSpace(m.Content) != "" {
				out = append(out, ui.Turn{Role: "user", Text: m.Content})
			}
		case openai.ChatMessageRoleAssistant:
			if strings.TrimSpace(m.Content) != "" {
				out = append(out, ui.Turn{Role: "assistant", Text: m.Content})
			}
			for _, tc := range m.ToolCalls {
				name := tc.Function.Name
				if name == "" {
					name = "tool"
				}
				text := "[调用工具 " + name + "]"
				if res, ok := results[tc.ID]; ok {
					text = "[调用工具 " + name + " → " + clipRunes(strings.Join(strings.Fields(res), " "), toolResultRunes) + "]"
				}
				out = append(out, ui.Turn{Role: "tool", Text: text})
			}
		case openai.ChatMessageRoleTool:
			// 工具结果已由过程块概括，不重复灌进对话流。
		}
	}
	return out
}

func clipRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// —— Day 11 技能：扫描 skills/、按 /use 加载 ——

// reloadSkills 重新扫描 skills/，丢掉已不存在的已加载技能，并把结果应用到 chat。
// 技能状态只在进程内有效；切换项目后按新目录重新扫描。
func (a *app) reloadSkills() {
	list, err := skills.Scan()
	if err != nil {
		a.ui.Error("加载技能失败：" + err.Error())
		return
	}
	a.skillMu.Lock()
	a.skillList = list
	for name := range a.activeSkills {
		if _, ok := skills.Find(list, name); !ok {
			delete(a.activeSkills, name)
		}
	}
	active := a.activeSkillsLocked()
	a.skillMu.Unlock()
	a.applySkillContext(active)
}

// activeSkillsLocked 返回已加载技能（保持扫描顺序）；调用方需持有 skillMu。
func (a *app) activeSkillsLocked() []skills.Skill {
	out := make([]skills.Skill, 0, len(a.activeSkills))
	for _, s := range a.skillList {
		if a.activeSkills[strings.ToLower(s.Name)] {
			out = append(out, s)
		}
	}
	return out
}

func (a *app) activeSkillList() []skills.Skill {
	a.skillMu.RLock()
	defer a.skillMu.RUnlock()
	return a.activeSkillsLocked()
}

// activeSkillNames 供 /status 与桌面侧栏展示。
func (a *app) activeSkillNames() []string {
	return skills.Names(a.activeSkillList())
}

// applySkillContext 把已加载技能转成 chat 的 system 片段与工具白名单。
func (a *app) applySkillContext(active []skills.Skill) {
	a.chat.SetSkills(skills.Note(active), skills.Tools(active))
}

// setSkillActive 切换单个技能的加载状态并立即生效。
func (a *app) setSkillActive(name string, on bool) {
	key := strings.ToLower(name)
	a.skillMu.Lock()
	if on {
		a.activeSkills[key] = true
	} else {
		delete(a.activeSkills, key)
	}
	active := a.activeSkillsLocked()
	a.skillMu.Unlock()
	a.applySkillContext(active)
}

func (a *app) findSkill(name string) (skills.Skill, bool) {
	a.skillMu.RLock()
	defer a.skillMu.RUnlock()
	return skills.Find(a.skillList, name)
}

// showSkills 重新扫描并列出所有技能（* 表示已加载）。
func (a *app) showSkills() {
	a.reloadSkills()
	a.printSkills()
}

func (a *app) printSkills() {
	a.skillMu.RLock()
	list := append([]skills.Skill(nil), a.skillList...)
	active := make(map[string]bool, len(a.activeSkills))
	for k, v := range a.activeSkills {
		active[k] = v
	}
	a.skillMu.RUnlock()

	if len(list) == 0 {
		a.ui.Info("未找到技能。在 skills/<名称>/SKILL.md 写 frontmatter（name / description / tools）与正文，再用 /use <名称> 加载。")
		return
	}
	var b strings.Builder
	b.WriteString("技能（* 已加载）：\n")
	for _, s := range list {
		mark := "  "
		if active[strings.ToLower(s.Name)] {
			mark = "* "
		}
		b.WriteString(mark + s.Name)
		if s.Description != "" {
			b.WriteString(" — " + s.Description)
		}
		if len(s.Tools) > 0 {
			b.WriteString("（工具：" + strings.Join(s.Tools, ", ") + "）")
		}
		b.WriteString("\n")
	}
	a.ui.Info(strings.TrimRight(b.String(), "\n"))
}

// cmdSkillUse 处理 /use；无参数时等价于 /skills。
func (a *app) cmdSkillUse(args []string) {
	a.reloadSkills()
	if len(args) == 0 {
		a.printSkills()
		return
	}
	name := args[0]
	s, ok := a.findSkill(name)
	if !ok {
		a.ui.Error("未找到技能：" + name + "（用 /skills 查看）")
		return
	}
	a.setSkillActive(s.Name, true)
	if len(s.Tools) > 0 {
		a.ui.Info(fmt.Sprintf("已加载技能：%s（工具白名单：%s）", s.Name, strings.Join(s.Tools, ", ")))
	} else {
		a.ui.Info("已加载技能：" + s.Name + "（不限制工具）")
	}
}

// cmdSkillUnuse 处理 /unuse；无参数时卸载全部。
func (a *app) cmdSkillUnuse(args []string) {
	if len(args) == 0 {
		a.skillMu.Lock()
		n := len(a.activeSkills)
		a.activeSkills = map[string]bool{}
		a.skillMu.Unlock()
		a.applySkillContext(nil)
		a.ui.Info(fmt.Sprintf("已卸载全部技能（%d 个）", n))
		return
	}
	name := args[0]
	if _, ok := a.findSkill(name); !ok {
		a.ui.Error("未找到技能：" + name + "（用 /skills 查看）")
		return
	}
	a.skillMu.RLock()
	active := a.activeSkills[strings.ToLower(name)]
	a.skillMu.RUnlock()
	if !active {
		a.ui.Notice("技能未加载：" + name)
		return
	}
	a.setSkillActive(name, false)
	a.ui.Info("已卸载技能：" + name)
}

func slashCommands() []ui.SlashCmd {
	return []ui.SlashCmd{
		{Name: "/model", Desc: "配置供应商与模型"},
		{Name: "/help", Desc: "显示帮助"},
		{Name: "/pwd", Desc: "显示工作目录"},
		{Name: "/status", Desc: "状态与 tokens"},
		{Name: "/project", Desc: "切换项目目录"},
		{Name: "/skills", Desc: "列出技能"},
		{Name: "/use", Desc: "加载技能"},
		{Name: "/unuse", Desc: "卸载技能"},
		{Name: "/sessions", Desc: "列出会话"},
		{Name: "/new", Desc: "新建会话"},
		{Name: "/open", Desc: "打开会话"},
		{Name: "/rename", Desc: "重命名会话"},
		{Name: "/delete", Desc: "删除会话"},
		{Name: "/memory", Desc: "列出长期记忆"},
		{Name: "/undo", Desc: "撤销最近一次写文件"},
		{Name: "/permissions", Desc: "查看工具权限规则"},
		{Name: "/save", Desc: "保存会话"},
		{Name: "/load", Desc: "重新加载"},
		{Name: "/reset", Desc: "清空记忆"},
		{Name: "/compact", Desc: "压缩历史"},
		{Name: "/exit", Desc: "保存并退出"},
	}
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
		Todos:          plan.Lines(),
	}
}

// reply 一轮普通对话：回显 → StreamReply（进度/正文分流）→ 中断不退出。
func (a *app) reply(line string) {
	a.ui.UserEcho(line)
	sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	ctx, cancel := context.WithCancel(sigCtx)
	defer cancel()
	canceler, hasStop := a.ui.(ui.Canceler)
	if hasStop {
		canceler.SetCancel(cancel)
		defer canceler.SetCancel(nil)
	}
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
	for _, p := range []string{"\n[调用工具", "\n[历史压缩", "\n[工具轮次", "\n[重复工具", "\n[回复预算", "\n[工具集", "\n[主动记忆"} {
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
	case "/", "/commands":
		return a.cmdMenu()
	case "/model":
		a.cmdModel(args)
	case "/pwd":
		a.ui.Info("工作目录：" + a.workDir + "\n（ls/read/write/run_shell 均相对此目录；会话：" + session.Path() + "）")
	case "/status":
		loaded := "无"
		if names := a.activeSkillNames(); len(names) > 0 {
			loaded = strings.Join(names, ", ")
		}
		a.ui.Info(fmt.Sprintf(
			"工作目录：%s\n供应商：%s · 模型：%s\n接口：%s\n当前会话：%s\n已加载技能：%s\n%s",
			a.workDir, a.provider, c.Model(), a.baseURL, mgr.Current(), loaded, c.Status(),
		))
	case "/skills":
		a.showSkills()
	case "/use":
		a.cmdSkillUse(args)
	case "/unuse":
		a.cmdSkillUnuse(args)
	case "/reset":
		c.Reset()
		mgr.SyncFromChat()
		a.ui.Info("（已清空当前会话记忆）")
	case "/compact":
		a.ui.Notice("[" + c.Compact(context.Background()) + "]")
		mgr.SyncFromChat()
	case "/project":
		rest := strings.TrimSpace(strings.TrimPrefix(line, "/project"))
		if rest == "" {
			a.ui.Info("用法：/project <目录>\nGUI 左侧「项目」旁点 + 选择文件夹。")
			break
		}
		a.switchProject(rest)
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
		if err := plan.Activate(a.workDir, newID); err != nil {
			a.ui.Error("任务清单：" + err.Error())
		}
		a.ui.Info(fmt.Sprintf("（已新建并切换到会话 %s）", newID))
	case "/open":
		if len(args) < 1 {
			a.ui.Info("用法：/open <会话ID> [项目目录]")
			break
		}
		if dir := openDirArg(line, args[0]); dir != "" {
			if !a.switchProject(dir) {
				break
			}
			mgr = a.mgr
		}
		if err := mgr.Open(args[0]); err != nil {
			a.ui.Error(err.Error())
			break
		}
		if err := plan.Activate(a.workDir, mgr.Current()); err != nil {
			a.ui.Error("任务清单：" + err.Error())
		}
		a.ui.Info(fmt.Sprintf("（已切换到会话 %s）", mgr.Current()))
	case "/rename":
		oldID, newID := mgr.Current(), ""
		switch len(args) {
		case 1:
			newID = args[0]
		case 2:
			oldID, newID = args[0], args[1]
		default:
			a.ui.Info("用法：/rename <新ID> 或 /rename <旧ID> <新ID>")
		}
		if newID == "" {
			break
		}
		if err := mgr.Rename(oldID, newID); err != nil {
			a.ui.Error(err.Error())
			break
		}
		if err := mgr.Save(); err != nil {
			a.ui.Error("保存失败：" + err.Error())
			break
		}
		if err := plan.Move(a.workDir, oldID, newID); err != nil {
			a.ui.Error("任务清单：" + err.Error())
		}
		a.ui.Info(fmt.Sprintf("（会话 %s 已重命名为 %s）", oldID, newID))
	case "/delete":
		if len(args) != 1 {
			a.ui.Info("用法：/delete <会话ID>")
			break
		}
		if !a.ui.Confirm(fmt.Sprintf("确认删除会话 %s？此操作不可撤销。", args[0])) {
			a.ui.Info("（已取消）")
			break
		}
		if err := plan.Drop(a.workDir, args[0]); err != nil {
			a.ui.Error("任务清单：" + err.Error())
		}
		if err := mgr.Delete(args[0]); err != nil {
			a.ui.Error(err.Error())
			break
		}
		if err := mgr.Save(); err != nil {
			a.ui.Error("保存失败：" + err.Error())
			break
		}
		if err := plan.Activate(a.workDir, mgr.Current()); err != nil {
			a.ui.Error("任务清单：" + err.Error())
		}
		a.ui.Info(fmt.Sprintf("（已删除会话 %s，当前会话 %s）", args[0], mgr.Current()))
	case "/memory":
		lines, err := memory.Lines()
		if err != nil {
			a.ui.Error("记忆：" + err.Error())
			break
		}
		a.ui.Info(strings.Join(lines, "\n"))
	case "/undo":
		msg, err := tools.UndoLast()
		if err != nil {
			a.ui.Error("撤销失败：" + err.Error())
			break
		}
		a.ui.Info("（" + msg + "）")
	case "/permissions":
		if err := a.reloadPolicy(); err != nil {
			a.ui.Error("权限配置：" + err.Error())
		}
		a.ui.Info(strings.Join(a.policy.Lines(), "\n"))
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

// openDirArg 取 "/open <id> <目录>" 中 id 之后的整段（目录可含空格）。
func openDirArg(line, id string) string {
	rest := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "/open"))
	rest = strings.TrimSpace(strings.TrimPrefix(rest, id))
	return strings.Trim(rest, `"'`)
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
  /model             模型配置（↑/↓ 选菜单项，Enter 进入；也可 /model list）
  /help              显示帮助
  /pwd               显示当前工作目录（工具读写基准）
  /status            显示工作目录、会话、tokens 与护栏状态
  /project <目录>    切换项目（保存当前会话后以该目录为工作区）
  /skills            列出 skills/<名称>/SKILL.md 中的技能（* 为已加载）
  /use <名称>        加载技能：追加 system 片段，并按声明的 tools 限定工具
  /unuse [名称]      卸载技能（省略名称则全部卸载）
  /sessions          列出全部会话（* 为当前）
  /new [id]          新建空会话并切换（可省略 id，自动 8 位）
  /open <id> [目录]  切换到已有会话（带目录时先切项目再打开）
  /rename [旧] <新>  重命名会话（省略旧 ID 即当前会话）
  /delete <id>       删除会话（需确认；删当前会话会切到最近的另一个）
  /memory            列出当前项目的长期记忆（.geekagent/memory.json）
  /undo              撤销最近一次 write/patch（最多 20 步，不撤销 shell）
  /permissions       查看工具权限规则（项目与用户 GeekAgent.json，项目优先）
  /save              保存全部会话到 %s
  /load              从磁盘重新加载（丢弃未保存改动）
  /reset             清空当前会话记忆
  /compact           立即压缩旧对话摘要
  /exit              保存并退出（default 有内容时改名为 8 位 ID）
在任意项目目录执行 hi-agent：读写相对当前目录；会话写入该目录下 .geekagent/。
API Key：项目 .env、上级目录 .env，或全局 %%USERPROFILE%%\\.hi-agent\\.env（Linux/macOS: ~/.hi-agent/.env）。
%s
回复进行中按 Ctrl+C 中断当前轮；空闲时按 Ctrl+C 保存并退出。
已注册工具：%s
界面：GEEKAGENT_UI=tui|plain|gui（默认 tui；web 预留）；GEEKAGENT_TUI=0 兼容为 plain。
输入：敲 / 显示命令提示，Tab 补全；/model 配置模型；滚轮或 PgUp 上翻。
过程：工具调用与 <think> 思考默认折叠为一行摘要；GEEKAGENT_FOLD=0 展开全文。
用量：GEEKAGENT_CONTEXT_WINDOW（默认 128000）；GEEKAGENT_STREAM_USAGE=0 关闭流式用量。
历史超长时自动压缩（GEEKAGENT_MAX_HISTORY，默认 4000 字符）。
工具轮次上限 GEEKAGENT_MAX_TOOL_TURNS（默认 8）；
重复熔断 GEEKAGENT_MAX_DUP_TOOLS（默认 3）/ GEEKAGENT_DUP_WINDOW（默认 12）；
回复预算 GEEKAGENT_MAX_REPLY_CHARS（默认 120000，0=关闭）；
工具收缩 GEEKAGENT_SHRINK_TOOLS_AFTER（默认约 3/4 轮次起只读，0=关闭）。`,
		session.Path(), config.ProviderHelp(), strings.Join(tools.Names(), ", "))
}
