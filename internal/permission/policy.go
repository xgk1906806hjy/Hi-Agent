// Package permission 实现 Day 8 的工具权限策略、工作区路径越界检查与输出脱敏。
//
// 本包不依赖 tools：REPL 把 Policy.Decide / SafePath / Redact 包成钩子注入 tools。
package permission

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Decision 一次工具调用的权限结论。
type Decision int

const (
	Ask   Decision = iota // 执行前向用户确认
	Allow                 // 直接执行（免确认）
	Deny                  // 拒绝执行
)

func (d Decision) String() string {
	switch d {
	case Allow:
		return "allow"
	case Deny:
		return "deny"
	default:
		return "ask"
	}
}

// ConfigPath 项目内权限配置文件（相对工作区根）。
const ConfigPath = ".geekagent/GeekAgent.json"

// Rule 一条规则：Tool 为工具名；Pattern 非空时对参数主体（命令或路径）做通配匹配。
type Rule struct {
	Tool    string
	Pattern string
}

func (r Rule) String() string {
	if r.Pattern == "" {
		return r.Tool
	}
	return r.Tool + ":" + r.Pattern
}

// ParseRule 解析 "tool" 或 "tool:pattern"。
func ParseRule(s string) Rule {
	s = strings.TrimSpace(s)
	if i := strings.Index(s, ":"); i > 0 {
		return Rule{Tool: strings.TrimSpace(s[:i]), Pattern: strings.TrimSpace(s[i+1:])}
	}
	return Rule{Tool: s}
}

// Policy 项目规则、用户规则与内置默认。
// 判定顺序：项目 deny、项目 ask、项目 allow、用户 deny、用户 ask、用户 allow、内置默认。
type Policy struct {
	Allow  []Rule
	AskR   []Rule
	DenyR  []Rule
	Source string // 项目配置绝对路径；空表示该层没有文件

	UserAllow  []Rule
	UserAsk    []Rule
	UserDeny   []Rule
	UserSource string // 用户配置绝对路径
}

// resolveUserConfig 测试可替换，避免读到本机 ~/.hi-agent。
var resolveUserConfig = UserConfigPath

// UserConfigPath 用户级权限文件 ~/.hi-agent/GeekAgent.json。取不到主目录时为空。
func UserConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".hi-agent", "GeekAgent.json")
}

// 内置默认：只读工具 allow，会改动环境/联网的工具 ask，其余（含未知工具）deny。
var (
	defaultAllow = []string{"get_current_time", "ls", "glob", "read", "search", "todo_write", "delegate_task", "memory_write", "memory_search"}
	defaultAsk   = []string{"run_shell", "write", "patch", "fetch"}
)

// Default 只含内置默认的策略。
func Default() *Policy { return &Policy{} }

type fileConfig struct {
	Permissions struct {
		Allow []string `json:"allow"`
		Ask   []string `json:"ask"`
		Deny  []string `json:"deny"`
	} `json:"permissions"`
}

// Load 合并项目 `.geekagent/GeekAgent.json` 与用户 `~/.hi-agent/GeekAgent.json`。
// 某一层文件不存在则该层为空；JSON 损坏则该层回退为空并返回错误，另一层仍生效。
func Load(root string) (*Policy, error) {
	p := Default()
	var errs []error
	allow, ask, deny, src, err := loadLayer(filepath.Join(root, filepath.FromSlash(ConfigPath)))
	if err != nil {
		errs = append(errs, err)
	}
	p.Allow, p.AskR, p.DenyR, p.Source = allow, ask, deny, src
	uAllow, uAsk, uDeny, uSrc, uErr := loadLayer(resolveUserConfig())
	if uErr != nil {
		errs = append(errs, uErr)
	}
	p.UserAllow, p.UserAsk, p.UserDeny, p.UserSource = uAllow, uAsk, uDeny, uSrc
	return p, errors.Join(errs...)
}

func loadLayer(path string) (allow, ask, deny []Rule, source string, err error) {
	if path == "" {
		return nil, nil, nil, "", nil
	}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil, nil, "", nil
	}
	if err != nil {
		return nil, nil, nil, "", err
	}
	var cfg fileConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		return nil, nil, nil, "", fmt.Errorf("解析 %s 失败：%w（该层已忽略）", path, err)
	}
	return parseRules(cfg.Permissions.Allow), parseRules(cfg.Permissions.Ask), parseRules(cfg.Permissions.Deny), path, nil
}

func parseRules(in []string) []Rule {
	out := make([]Rule, 0, len(in))
	for _, s := range in {
		if r := ParseRule(s); r.Tool != "" {
			out = append(out, r)
		}
	}
	return out
}

// Decide 判定一次工具调用。argsJSON 用于取规则匹配的参数主体。
func (p *Policy) Decide(tool, argsJSON string) Decision {
	subject := Subject(tool, argsJSON)
	if p != nil {
		if matchAny(p.DenyR, tool, subject) {
			return Deny
		}
		if matchAny(p.AskR, tool, subject) {
			return Ask
		}
		if matchAny(p.Allow, tool, subject) {
			return Allow
		}
		if matchAny(p.UserDeny, tool, subject) {
			return Deny
		}
		if matchAny(p.UserAsk, tool, subject) {
			return Ask
		}
		if matchAny(p.UserAllow, tool, subject) {
			return Allow
		}
	}
	return defaultDecision(tool)
}

func defaultDecision(tool string) Decision {
	for _, n := range defaultAllow {
		if n == tool {
			return Allow
		}
	}
	for _, n := range defaultAsk {
		if n == tool {
			return Ask
		}
	}
	return Deny
}

// Subject 取规则匹配用的参数：run_shell 取 command，glob/search 取 pattern，fetch 取 url，文件类取 path。
func Subject(tool, argsJSON string) string {
	var args struct {
		Command string `json:"command"`
		Path    string `json:"path"`
		Pattern string `json:"pattern"`
		URL     string `json:"url"`
	}
	_ = json.Unmarshal([]byte(argsJSON), &args)
	switch tool {
	case "run_shell":
		return strings.TrimSpace(args.Command)
	case "glob", "search":
		return filepath.ToSlash(strings.TrimSpace(args.Pattern))
	case "fetch":
		return strings.TrimSpace(args.URL)
	default:
		return filepath.ToSlash(strings.TrimSpace(args.Path))
	}
}

func matchAny(rules []Rule, tool, subject string) bool {
	for _, r := range rules {
		if r.Tool != tool && r.Tool != "*" {
			continue
		}
		if r.Pattern == "" || Wildcard(filepath.ToSlash(r.Pattern), subject) {
			return true
		}
	}
	return false
}

// Wildcard 简单通配：* 匹配任意串（可跨 /），? 匹配单个字符。
func Wildcard(pattern, s string) bool {
	p, t := []rune(pattern), []rune(s)
	pi, ti, star, mark := 0, 0, -1, 0
	for ti < len(t) {
		switch {
		case pi < len(p) && (p[pi] == '?' || p[pi] == t[ti]):
			pi++
			ti++
		case pi < len(p) && p[pi] == '*':
			star, mark = pi, ti
			pi++
		case star >= 0:
			pi = star + 1
			mark++
			ti = mark
		default:
			return false
		}
	}
	for pi < len(p) && p[pi] == '*' {
		pi++
	}
	return pi == len(p)
}

// Lines 供 /permissions 展示当前生效规则与来源。
func (p *Policy) Lines() []string {
	var out []string
	if p == nil {
		p = Default()
	}
	out = append(out, layerLines("项目", p.Source, ConfigPath, p.DenyR, p.AskR, p.Allow)...)
	userHint := "~/.hi-agent/GeekAgent.json"
	out = append(out, layerLines("用户", p.UserSource, userHint, p.UserDeny, p.UserAsk, p.UserAllow)...)
	out = append(out,
		"内置默认（两层都未命中时）：",
		"  allow："+strings.Join(defaultAllow, ", "),
		"  ask：  "+strings.Join(defaultAsk, ", "),
		"  其余工具（含未知工具）：deny",
		"优先级：项目 deny > 项目 ask > 项目 allow > 用户 deny > 用户 ask > 用户 allow > 内置默认",
		"规则写法：工具名 或 工具名:通配（* 任意串，? 单字符）",
	)
	return out
}

func layerLines(title, source, hint string, deny, ask, allow []Rule) []string {
	if source == "" {
		return []string{title + "配置：无（可写 " + hint + "）"}
	}
	return []string{
		title + "配置：" + source,
		"  deny： " + joinRules(deny),
		"  ask：  " + joinRules(ask),
		"  allow：" + joinRules(allow),
	}
}

func joinRules(rs []Rule) string {
	if len(rs) == 0 {
		return "（无）"
	}
	s := make([]string, len(rs))
	for i, r := range rs {
		s[i] = r.String()
	}
	return strings.Join(s, ", ")
}
