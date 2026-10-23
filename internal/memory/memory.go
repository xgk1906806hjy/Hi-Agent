// Package memory 保存项目级长期记忆（Day 10）。
// 文件为工作区 `.geekagent/memory.json`，切换目录后即换成新项目的文件。
// 只追加和搜索，不提供修改与删除。
package memory

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const fileRel = ".geekagent/memory.json"

const searchLimit = 5

// Entry 一条记忆。
type Entry struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
}

// Path 返回记忆文件的相对路径。
func Path() string { return fileRel }

// Write 追加一条记忆并落盘。
func Write(content string) (Entry, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return Entry{}, fmt.Errorf("记忆内容为空")
	}
	entries, err := load()
	if err != nil {
		return Entry{}, err
	}
	id, err := newID()
	if err != nil {
		return Entry{}, err
	}
	e := Entry{ID: id, Content: content, CreatedAt: time.Now()}
	entries = append(entries, e)
	if err := save(entries); err != nil {
		return Entry{}, err
	}
	return e, nil
}

// Search 不区分大小写的子串匹配，最多 5 条，按写入顺序。
func Search(query string) ([]Entry, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("缺少搜索词")
	}
	entries, err := load()
	if err != nil {
		return nil, err
	}
	q := strings.ToLower(query)
	var out []Entry
	for _, e := range entries {
		if strings.Contains(strings.ToLower(e.Content), q) || strings.Contains(strings.ToLower(e.ID), q) {
			out = append(out, e)
			if len(out) >= searchLimit {
				break
			}
		}
	}
	return out, nil
}

// List 返回全部记忆。
func List() ([]Entry, error) {
	return load()
}

// Lines 供 /memory 展示。
func Lines() ([]string, error) {
	entries, err := List()
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return []string{"（暂无长期记忆）"}, nil
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, fmt.Sprintf("%s  %s  %s", e.CreatedAt.Format("2006-01-02 15:04"), e.ID, e.Content))
	}
	return out, nil
}

func load() ([]Entry, error) {
	b, err := os.ReadFile(filepath.FromSlash(fileRel))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(strings.TrimSpace(string(b))) == 0 {
		return nil, nil
	}
	var entries []Entry
	if err := json.Unmarshal(b, &entries); err != nil {
		return nil, fmt.Errorf("解析记忆文件失败：%w", err)
	}
	return entries, nil
}

func save(entries []Entry) error {
	path := filepath.FromSlash(fileRel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if entries == nil {
		entries = []Entry{}
	}
	b, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func newID() (string, error) {
	var buf [4]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf[:]), nil
}
