// Package skills 扫描工作区 skills/*/SKILL.md，解析技能说明与工具白名单。
//
// 技能只影响发给模型的 system 片段与可见工具集，不注册新工具，也不改变权限策略。
package skills

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

// Dir 技能目录（相对进程工作目录）；每个子目录放一个 SKILL.md。
const Dir = "skills"

// FileName 技能说明文件名。
const FileName = "SKILL.md"

// bodyLimit 单个技能正文的字符上限，避免 system 片段无限膨胀。
const bodyLimit = 8000

// Skill 一个已解析的技能。
type Skill struct {
	Name        string   // 展示名，默认取目录名
	Description string   // 一句话说明
	Tools       []string // 工具白名单；空表示不限制
	Body        string   // SKILL.md 正文（指令）
	Path        string   // 文件路径，供提示与排查
}

// Scan 读取工作区 skills/*/SKILL.md，按名称排序；目录不存在返回 nil。
func Scan() ([]Skill, error) {
	return scanDir(Dir)
}

func scanDir(root string) ([]Skill, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	out := make([]Skill, 0, len(entries))
	seen := map[string]struct{}{}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		path := filepath.Join(root, e.Name(), FileName)
		b, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		s := parse(e.Name(), string(b))
		s.Path = path
		key := strings.ToLower(s.Name)
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// parse 解析 SKILL.md：可选的 YAML 风格 frontmatter（name / description / tools）+ 正文。
// tools 支持 `tools: read, glob`、`tools: [read, glob]` 与多行 `- read` 三种写法。
func parse(dirName, text string) Skill {
	text = strings.TrimPrefix(text, "\ufeff")
	text = strings.ReplaceAll(text, "\r\n", "\n")
	s := Skill{Name: dirName}

	body := text
	lines := strings.Split(text, "\n")
	if len(lines) > 0 && strings.TrimSpace(lines[0]) == "---" {
		end := -1
		for i := 1; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) == "---" {
				end = i
				break
			}
		}
		if end > 0 {
			parseMeta(&s, lines[1:end])
			body = strings.Join(lines[end+1:], "\n")
		}
	}
	s.Name = strings.TrimSpace(s.Name)
	if s.Name == "" {
		s.Name = dirName
	}
	s.Body = strings.TrimSpace(body)
	if utf8.RuneCountInString(s.Body) > bodyLimit {
		s.Body = string([]rune(s.Body)[:bodyLimit]) + "\n\n[技能正文已截断]"
	}
	return s
}

func parseMeta(s *Skill, lines []string) {
	key := ""
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if key == "tools" && strings.HasPrefix(line, "- ") {
			s.Tools = appendTools(s.Tools, line[2:])
			continue
		}
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.ToLower(strings.TrimSpace(k))
		v = cleanValue(v)
		switch key {
		case "name":
			if v != "" {
				s.Name = v
			}
		case "description":
			s.Description = v
		case "tools":
			s.Tools = parseTools(v)
		}
	}
}

func parseTools(v string) []string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "[")
	v = strings.TrimSuffix(v, "]")
	if v == "" {
		return nil
	}
	var out []string
	for _, part := range strings.Split(v, ",") {
		out = appendTools(out, part)
	}
	return out
}

func appendTools(out []string, name string) []string {
	name = cleanValue(name)
	if name == "" {
		return out
	}
	for _, e := range out {
		if e == name {
			return out
		}
	}
	return append(out, name)
}

func cleanValue(v string) string {
	v = strings.TrimSpace(v)
	if len(v) >= 2 {
		if (v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'') {
			v = v[1 : len(v)-1]
		}
	}
	return strings.TrimSpace(v)
}

// Find 按名称查找技能（忽略大小写）。
func Find(list []Skill, name string) (Skill, bool) {
	for _, s := range list {
		if strings.EqualFold(s.Name, name) {
			return s, true
		}
	}
	return Skill{}, false
}

// Names 返回技能名列表（保持传入顺序）。
func Names(list []Skill) []string {
	out := make([]string, len(list))
	for i, s := range list {
		out[i] = s.Name
	}
	return out
}

// Note 把已加载技能拼成一段 system 正文；没有技能时返回空串。
func Note(active []Skill) string {
	if len(active) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("以下是当前项目已加载的技能，请遵循其中的工作方式与约束：\n")
	for _, s := range active {
		b.WriteString("\n## " + s.Name + "\n")
		if s.Description != "" {
			b.WriteString(s.Description + "\n")
		}
		if s.Body != "" {
			b.WriteString(s.Body + "\n")
		}
	}
	return strings.TrimSpace(b.String())
}

// Tools 返回已加载技能声明的工具白名单（去重，保持声明顺序）。
// 没有任何技能声明 tools 时返回 nil，表示不限制工具。
func Tools(active []Skill) []string {
	var out []string
	for _, s := range active {
		for _, t := range s.Tools {
			out = appendTools(out, t)
		}
	}
	return out
}
