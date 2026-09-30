package tools

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const (
	maxReadChars   = 8000 // read 单次最多返回的字符数（按 rune）
	maxDiffLines   = 100  // simpleDiff 最多展示的 +/- 行数
	maxGlobResults = 200  // glob 最多匹配路径数
)

// registerFiles 注册 ls / glob / read / write / patch。
func registerFiles() {
	Register(Tool{
		Name:        "ls",
		Description: "列出目录内容（名称与类型）。免确认。默认当前工作目录。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{"type": "string", "description": "相对或绝对目录路径，默认 ."},
			},
			"additionalProperties": false,
		},
		Run: runLs,
	})
	Register(Tool{
		Name:        "glob",
		Description: "按 glob 模式查找文件（支持 * 与 **）。免确认。返回相对路径列表。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"pattern": map[string]any{"type": "string", "description": "如 internal/**/*.go 或 *.md"},
			},
			"required":             []string{"pattern"},
			"additionalProperties": false,
		},
		Run: runGlob,
	})
	Register(Tool{
		Name:        "read",
		Description: "读取文本文件。可用 offset（字符偏移）分段续读。免确认。单次最多约 8000 字符。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path":   map[string]any{"type": "string", "description": "文件路径"},
				"offset": map[string]any{"type": "integer", "description": "从第几个字符开始读，默认 0"},
			},
			"required":             []string{"path"},
			"additionalProperties": false,
		},
		Run: runRead,
	})
	Register(Tool{
		Name:        "write",
		Description: "整体写入或覆盖文件。会先展示 diff，确认后落盘。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path":    map[string]any{"type": "string", "description": "文件路径"},
				"content": map[string]any{"type": "string", "description": "完整新内容"},
			},
			"required":             []string{"path", "content"},
			"additionalProperties": false,
		},
		Run: runWrite,
	})
	Register(Tool{
		Name:        "patch",
		Description: "用唯一旧文本片段定位并替换。每个 hunk 的 old 必须在文件中恰好出现一次。先展示 diff，确认后落盘。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{"type": "string", "description": "文件路径"},
				"hunks": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"old": map[string]any{"type": "string", "description": "要替换的原文（须唯一）"},
							"new": map[string]any{"type": "string", "description": "替换后的文本"},
						},
						"required":             []string{"old", "new"},
						"additionalProperties": false,
					},
					"description": "替换片段列表",
				},
			},
			"required":             []string{"path", "hunks"},
			"additionalProperties": false,
		},
		Run: runPatch,
	})
}

// runLs 列出目录项，每行「dir|file\t名称」。
func runLs(argsJSON string) (string, error) {
	var args struct {
		Path string `json:"path"`
	}
	_ = json.Unmarshal([]byte(argsJSON), &args)
	path := strings.TrimSpace(args.Path)
	if path == "" {
		path = "."
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "共 %d 项（%s）：\n", len(entries), path)
	for _, e := range entries {
		kind := "file"
		if e.IsDir() {
			kind = "dir"
		}
		fmt.Fprintf(&b, "%s\t%s\n", kind, e.Name())
	}
	return strings.TrimSpace(b.String()), nil
}

// runGlob 从当前目录 Walk，按 pattern 匹配相对路径。
func runGlob(argsJSON string) (string, error) {
	var args struct {
		Pattern string `json:"pattern"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", err
	}
	pattern := filepath.ToSlash(strings.TrimSpace(args.Pattern))
	if pattern == "" {
		return "缺少参数 pattern", nil
	}
	matches, err := globWalk(pattern)
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return fmt.Sprintf("没有匹配 %q", pattern), nil
	}
	hint := ""
	if len(matches) >= maxGlobResults {
		hint = fmt.Sprintf("\n...(已达上限 %d，请收窄 pattern)", maxGlobResults)
	}
	return fmt.Sprintf("匹配到 %d 个：\n%s%s", len(matches), strings.Join(matches, "\n"), hint), nil
}

// globWalk 遍历「.」下文件；跳过 .git；达上限后 SkipAll。
func globWalk(pattern string) ([]string, error) {
	var matches []string
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // 单个路径错误不中断整次搜索
		}
		if d.IsDir() {
			if filepath.Base(path) == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if len(matches) >= maxGlobResults {
			return fs.SkipAll
		}
		slash := filepath.ToSlash(path)
		ok, mErr := pathMatch(pattern, slash)
		if mErr == nil && ok {
			matches = append(matches, slash)
		}
		return nil
	})
	return matches, err
}

// pathMatch 支持 * 与 **（按 / 分段匹配）。
func pathMatch(pattern, name string) (bool, error) {
	return matchParts(strings.Split(pattern, "/"), strings.Split(name, "/"))
}

// matchParts 递归匹配路径段：普通段用 filepath.Match，** 可跨任意层目录。
func matchParts(pat, name []string) (bool, error) {
	for len(pat) > 0 && len(name) > 0 {
		p := pat[0]
		if p == "**" {
			if len(pat) == 1 {
				return true, nil // 尾部 ** 吃掉剩余路径
			}
			// 尝试让 ** 消费 0..len(name) 段后继续匹配后续 pattern
			for i := 0; i <= len(name); i++ {
				ok, err := matchParts(pat[1:], name[i:])
				if err != nil {
					return false, err
				}
				if ok {
					return true, nil
				}
			}
			return false, nil
		}
		ok, err := filepath.Match(p, name[0])
		if err != nil {
			return false, err
		}
		if !ok {
			return false, nil
		}
		pat = pat[1:]
		name = name[1:]
	}
	// 吃掉尾部多余的 **
	for len(pat) > 0 && pat[0] == "**" {
		pat = pat[1:]
	}
	return len(pat) == 0 && len(name) == 0, nil
}

// runRead 按字符 offset 分段读取；超出长度时给出下一 offset 提示。
func runRead(argsJSON string) (string, error) {
	var args struct {
		Path   string `json:"path"`
		Offset int    `json:"offset"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", err
	}
	path := strings.TrimSpace(args.Path)
	if path == "" {
		return "缺少参数 path", nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	runes := []rune(string(data))
	total := len(runes)
	off := args.Offset
	if off < 0 {
		off = 0
	}
	if off > total {
		return fmt.Sprintf("offset %d 超出文件长度（共 %d 字符）", off, total), nil
	}
	end := off + maxReadChars
	if end > total {
		end = total
	}
	chunk := string(runes[off:end])
	more := ""
	if end < total {
		more = fmt.Sprintf("；未读完，可用 offset=%d 继续", end)
	}
	return fmt.Sprintf("（共 %d 字符 / 读到第 %d-%d 段%s）\n%s", total, off, end, more, chunk), nil
}

// runWrite 整体覆盖写入，经 commitWrite（diff + 确认）。
func runWrite(argsJSON string) (string, error) {
	var args struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", err
	}
	path := strings.TrimSpace(args.Path)
	if path == "" {
		return "缺少参数 path", nil
	}
	ok, msg := commitWrite(path, args.Content)
	if !ok {
		return msg, nil
	}
	return fmt.Sprintf("已写入 %s（%d 字符）", path, len([]rune(args.Content))), nil
}

// hunk 表示一处「唯一原文 → 新文」替换。
type hunk struct {
	Old string `json:"old"`
	New string `json:"new"`
}

// runPatch 顺序应用 hunk；每个 old 必须在当前全文恰好出现 1 次，再 commitWrite。
func runPatch(argsJSON string) (string, error) {
	var args struct {
		Path  string `json:"path"`
		Hunks []hunk `json:"hunks"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", err
	}
	path := strings.TrimSpace(args.Path)
	if path == "" {
		return "缺少参数 path", nil
	}
	if len(args.Hunks) == 0 {
		return "缺少参数 hunks", nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	content := string(data)
	for i, h := range args.Hunks {
		if h.Old == "" {
			return fmt.Sprintf("hunks[%d].old 不能为空", i), nil
		}
		count := strings.Count(content, h.Old)
		if count == 0 {
			return fmt.Sprintf("hunks[%d]：找不到唯一锚点（出现 0 次），请扩大 old 上下文", i), nil
		}
		if count > 1 {
			return fmt.Sprintf("hunks[%d]：锚点不唯一（出现 %d 次），请扩大 old 上下文", i, count), nil
		}
		content = strings.Replace(content, h.Old, h.New, 1)
	}
	ok, msg := commitWrite(path, content)
	if !ok {
		return msg, nil
	}
	return fmt.Sprintf("已应用 %d 处修改到 %s", len(args.Hunks), path), nil
}

// commitWrite：读旧内容 → 未变则跳过 → simpleDiff → confirm → MkdirAll → WriteFile。
func commitWrite(path, next string) (ok bool, message string) {
	var oldtxt string
	existing := true
	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return false, err.Error()
		}
		existing = false
		oldtxt = ""
	} else {
		oldtxt = string(data)
	}
	if existing && oldtxt == next {
		return true, "内容未变化"
	}
	diff := simpleDiff(oldtxt, next)
	prompt := fmt.Sprintf("\n%s\n确认写入 %s？", diff, path)
	if !confirm(prompt) {
		return false, "已取消写入"
	}
	// 父目录为「.」时 MkdirAll 失败可忽略（当前目录已存在）。
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil && filepath.Dir(path) != "." {
		return false, err.Error()
	}
	if err := os.WriteFile(path, []byte(next), 0o644); err != nil {
		return false, err.Error()
	}
	return true, ""
}

// simpleDiff 去掉公共前后缀行后，输出 -旧 / +新；超过 maxDiffLines 截断。
func simpleDiff(oldtxt, newtxt string) string {
	oldLines := strings.Split(strings.ReplaceAll(oldtxt, "\r\n", "\n"), "\n")
	newLines := strings.Split(strings.ReplaceAll(newtxt, "\r\n", "\n"), "\n")
	// 去公共前缀
	for len(oldLines) > 0 && len(newLines) > 0 && oldLines[0] == newLines[0] {
		oldLines = oldLines[1:]
		newLines = newLines[1:]
	}
	// 去公共后缀
	for len(oldLines) > 0 && len(newLines) > 0 && oldLines[len(oldLines)-1] == newLines[len(newLines)-1] {
		oldLines = oldLines[:len(oldLines)-1]
		newLines = newLines[:len(newLines)-1]
	}
	var b strings.Builder
	b.WriteString("--- diff ---\n")
	n := 0
	for _, line := range oldLines {
		if n >= maxDiffLines {
			b.WriteString("...(diff 已截断)\n")
			return b.String()
		}
		fmt.Fprintf(&b, "-%s\n", line)
		n++
	}
	for _, line := range newLines {
		if n >= maxDiffLines {
			b.WriteString("...(diff 已截断)\n")
			return b.String()
		}
		fmt.Fprintf(&b, "+%s\n", line)
		n++
	}
	if n == 0 {
		b.WriteString("(无行级差异或仅空白变化)\n")
	}
	return b.String()
}
