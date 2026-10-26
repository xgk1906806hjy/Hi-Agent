package tools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	maxSearchResults   = 200     // 单次最多返回的匹配行数
	maxSearchLineRunes = 240     // 单行展示上限（rune）
	maxSearchFileBytes = 2 << 20 // 超过 2MB 的文件跳过
)

// searchSkipDirs 搜索时跳过的噪声目录（按目录名匹配）。
var searchSkipDirs = map[string]struct{}{
	".git":         {},
	".geekagent":   {},
	"node_modules": {},
	"dist":         {},
	"build":        {},
	".next":        {},
	".venv":        {},
	"venv":         {},
	"__pycache__":  {},
}

// registerSearch 注册 search：工作区内按行搜索，返回 路径:行号: 内容。
func registerSearch() {
	Register(Tool{
		Name: "search",
		Description: "在工作区内按行搜索文本，返回 路径:行号: 行内容。默认字面匹配；regex=true 时按 Go 正则。" +
			"会跳过 .git、.geekagent、node_modules、dist 等目录和二进制文件。免确认。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"pattern": map[string]any{"type": "string", "description": "要搜索的文本或正则"},
				"path":    map[string]any{"type": "string", "description": "搜索范围（文件或目录），默认工作区根 ."},
				"regex":   map[string]any{"type": "boolean", "description": "true 时把 pattern 当作 Go 正则，默认 false（字面匹配）"},
			},
			"required":             []string{"pattern"},
			"additionalProperties": false,
		},
		Run: runSearch,
	})
}

// runSearch 遍历目标路径，按行匹配并汇总；达到上限后提前结束。
func runSearch(argsJSON string) (string, error) {
	var args struct {
		Pattern string `json:"pattern"`
		Path    string `json:"path"`
		Regex   bool   `json:"regex"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("参数不是合法 JSON：%w", err)
	}
	pattern := strings.TrimSpace(args.Pattern)
	if pattern == "" {
		return "缺少参数 pattern", nil
	}

	var re *regexp.Regexp
	if args.Regex {
		compiled, err := regexp.Compile(pattern)
		if err != nil {
			return "正则无效：" + err.Error(), nil
		}
		re = compiled
	}

	target := strings.TrimSpace(args.Path)
	if target == "" {
		target = "."
	}
	safe, err := guard(target)
	if err != nil {
		return "", err
	}
	root, err := guard(".")
	if err != nil {
		return "", err
	}
	safeAbs, err := filepath.Abs(safe)
	if err != nil {
		return "", err
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(safeAbs)
	if err != nil {
		return "", err
	}

	var hits []string
	truncated := false

	matchFile := func(path string) error {
		data, err := os.ReadFile(path)
		if err != nil || len(data) > maxSearchFileBytes {
			return nil
		}
		if bytes.IndexByte(data, 0) >= 0 {
			return nil // 含 NUL，按二进制跳过
		}
		text := strings.ReplaceAll(string(data), "\r\n", "\n")
		rel := displayPath(rootAbs, path)
		for i, line := range strings.Split(text, "\n") {
			matched := strings.Contains(line, pattern)
			if re != nil {
				matched = re.MatchString(line)
			}
			if !matched {
				continue
			}
			if len(hits) >= maxSearchResults {
				truncated = true
				return fs.SkipAll
			}
			hits = append(hits, fmt.Sprintf("%s:%d: %s", rel, i+1, shortenLine(line)))
		}
		return nil
	}

	if info.IsDir() {
		err = filepath.WalkDir(safeAbs, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return nil // 单个路径错误不中断整次搜索
			}
			if d.IsDir() {
				if path != safeAbs {
					if _, skip := searchSkipDirs[d.Name()]; skip {
						return filepath.SkipDir
					}
				}
				return nil
			}
			if !d.Type().IsRegular() {
				return nil
			}
			return matchFile(path)
		})
	} else {
		err = matchFile(safeAbs)
	}
	if err != nil && err != fs.SkipAll {
		return "", err
	}

	mode := "字面"
	if re != nil {
		mode = "正则"
	}
	scope := filepath.ToSlash(target)
	if len([]rune(scope)) > 80 {
		scope = string([]rune(scope)[:80]) + "…"
	}
	if len(hits) == 0 {
		return fmt.Sprintf("没有匹配 %q（范围 %s，%s）", clipRunes(pattern, 80), scope, mode), nil
	}

	var b strings.Builder
	head := fmt.Sprintf("匹配到 %d 处（范围 %s，%s）：", len(hits), scope, mode)
	if truncated {
		head = fmt.Sprintf("至少 %d 处（范围 %s，%s，已达上限 %d）：", len(hits), scope, mode, maxSearchResults)
	}
	b.WriteString(head)
	b.WriteString("\n")
	b.WriteString(strings.Join(hits, "\n"))
	if truncated {
		fmt.Fprintf(&b, "\n...(已达上限 %d，请收窄 pattern 或 path)", maxSearchResults)
	}
	return b.String(), nil
}

// displayPath 把绝对路径转成相对工作区的 / 分隔路径。
func displayPath(rootAbs, path string) string {
	rel, err := filepath.Rel(rootAbs, path)
	if err != nil || rel == "" {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

func shortenLine(s string) string {
	return clipRunes(strings.TrimSpace(s), maxSearchLineRunes)
}

func clipRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
