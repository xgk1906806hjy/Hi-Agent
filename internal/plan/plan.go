// Package plan 保存当前会话的任务清单（Day 9）。
// 按会话写入工作区 `.geekagent/todos.json`；同时最多一项 in_progress。
package plan

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	Pending    = "pending"
	InProgress = "in_progress"
	Completed  = "completed"
	Cancelled  = "cancelled"
)

// Item 一条任务。
type Item struct {
	ID      string `json:"id"`
	Content string `json:"content"`
	Status  string `json:"status"`
}

const fileRel = ".geekagent/todos.json"

type fileData struct {
	Sessions map[string][]Item `json:"sessions"`
}

var (
	mu      sync.Mutex
	items   []Item
	root    string
	session string
	bound   bool
)

// List 返回清单副本。
func List() []Item {
	mu.Lock()
	defer mu.Unlock()
	out := make([]Item, len(items))
	copy(out, items)
	return out
}

// Reset 清空内存清单并解除落盘绑定。测试与未绑定场景使用。
func Reset() {
	mu.Lock()
	items = nil
	root, session = "", ""
	bound = false
	mu.Unlock()
}

// Activate 先保存当前会话的清单，再载入 dir 下目标会话的清单。首次调用只载入。
func Activate(dir, id string) error {
	mu.Lock()
	defer mu.Unlock()
	if bound {
		if err := saveLocked(); err != nil {
			return err
		}
	}
	root, session = dir, id
	items = loadSession(dir, id)
	bound = true
	return nil
}

// Move 把清单从旧会话 ID 挪到新 ID。当前绑定着旧 ID 时，内存清单跟过去。
func Move(dir, oldID, newID string) error {
	mu.Lock()
	defer mu.Unlock()
	if err := moveKey(dir, oldID, newID); err != nil {
		return err
	}
	if bound && sameDir(root, dir) && session == oldID {
		session = newID
	}
	return nil
}

// Drop 从文件里去掉该会话的清单。去掉的是当前绑定时，内存先清空，避免随后又写回去。
func Drop(dir, id string) error {
	mu.Lock()
	defer mu.Unlock()
	if err := dropKey(dir, id); err != nil {
		return err
	}
	if bound && sameDir(root, dir) && session == id {
		session = ""
		items = nil
	}
	return nil
}

// Apply 按 ID 合并更新。新任务必须有 content；同一次调用里多项 in_progress 时，最后一项保留，其余降为 pending。
// 返回更新后的清单文本。
func Apply(updates []Item) (string, error) {
	mu.Lock()
	defer mu.Unlock()
	index := map[string]int{}
	for i, it := range items {
		index[it.ID] = i
	}
	promoted := ""
	for _, u := range updates {
		id := strings.TrimSpace(u.ID)
		if id == "" || strings.ContainsAny(id, " \t\r\n/\\") {
			return "", fmt.Errorf("非法任务 ID：%q", u.ID)
		}
		st := strings.TrimSpace(u.Status)
		if st == "" {
			st = Pending
		}
		if !valid(st) {
			return "", fmt.Errorf("非法状态 %q（应为 pending / in_progress / completed / cancelled）", u.Status)
		}
		content := strings.TrimSpace(u.Content)
		if i, ok := index[id]; ok {
			if content != "" {
				items[i].Content = content
			}
			items[i].Status = st
		} else {
			if content == "" {
				return "", fmt.Errorf("新任务 %s 缺少 content", id)
			}
			items = append(items, Item{ID: id, Content: content, Status: st})
			index[id] = len(items) - 1
		}
		if st == InProgress {
			promoted = id
		}
	}
	if promoted != "" {
		for i := range items {
			if items[i].Status == InProgress && items[i].ID != promoted {
				items[i].Status = Pending
			}
		}
	}
	if err := saveLocked(); err != nil {
		return formatLocked(), err
	}
	return formatLocked(), nil
}

// Lines 供界面逐行展示。
func Lines() []string {
	mu.Lock()
	defer mu.Unlock()
	if len(items) == 0 {
		return nil
	}
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = mark(it.Status) + " " + it.Content
	}
	return out
}

func formatLocked() string {
	if len(items) == 0 {
		return "（任务清单为空）"
	}
	var b strings.Builder
	b.WriteString("当前任务：\n")
	for _, it := range items {
		fmt.Fprintf(&b, "%s [%s] %s\n", mark(it.Status), it.ID, it.Content)
	}
	return strings.TrimRight(b.String(), "\n")
}

func mark(status string) string {
	switch status {
	case InProgress:
		return "●"
	case Completed:
		return "✓"
	case Cancelled:
		return "–"
	default:
		return "○"
	}
}

func valid(s string) bool {
	switch s {
	case Pending, InProgress, Completed, Cancelled:
		return true
	default:
		return false
	}
}

func saveLocked() error {
	if !bound || root == "" || session == "" {
		return nil
	}
	data := readFile(root)
	if data.Sessions == nil {
		data.Sessions = map[string][]Item{}
	}
	data.Sessions[session] = append([]Item(nil), items...)
	return writeFile(root, data)
}

func loadSession(dir, id string) []Item {
	if dir == "" || id == "" {
		return nil
	}
	data := readFile(dir)
	src := data.Sessions[id]
	if len(src) == 0 {
		return nil
	}
	return append([]Item(nil), src...)
}

func moveKey(dir, oldID, newID string) error {
	if dir == "" || oldID == "" || oldID == newID {
		return nil
	}
	data := readFile(dir)
	src, ok := data.Sessions[oldID]
	if !ok {
		return nil
	}
	if data.Sessions == nil {
		data.Sessions = map[string][]Item{}
	}
	data.Sessions[newID] = src
	delete(data.Sessions, oldID)
	return writeFile(dir, data)
}

func dropKey(dir, id string) error {
	if dir == "" || id == "" {
		return nil
	}
	data := readFile(dir)
	if _, ok := data.Sessions[id]; !ok {
		return nil
	}
	delete(data.Sessions, id)
	return writeFile(dir, data)
}

func readFile(dir string) fileData {
	b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(fileRel)))
	if err != nil {
		return fileData{Sessions: map[string][]Item{}}
	}
	var data fileData
	if err := json.Unmarshal(b, &data); err != nil || data.Sessions == nil {
		return fileData{Sessions: map[string][]Item{}}
	}
	return data
}

func writeFile(dir string, data fileData) error {
	if data.Sessions == nil {
		data.Sessions = map[string][]Item{}
	}
	path := filepath.Join(dir, filepath.FromSlash(fileRel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func sameDir(a, b string) bool {
	if a == "" || b == "" {
		return a == b
	}
	aa, err1 := filepath.Abs(a)
	bb, err2 := filepath.Abs(b)
	if err1 != nil || err2 != nil {
		return filepath.Clean(a) == filepath.Clean(b)
	}
	return strings.EqualFold(filepath.Clean(aa), filepath.Clean(bb))
}
