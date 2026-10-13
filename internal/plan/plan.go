// Package plan 保存当前进程的任务清单（Day 9）。只在内存中，不落盘；
// 同时最多一项 in_progress。切换项目时由 REPL 调用 Reset。
package plan

import (
	"fmt"
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
	ID      string
	Content string
	Status  string
}

var (
	mu    sync.Mutex
	items []Item
)

// List 返回清单副本。
func List() []Item {
	mu.Lock()
	defer mu.Unlock()
	out := make([]Item, len(items))
	copy(out, items)
	return out
}

// Reset 清空清单。
func Reset() {
	mu.Lock()
	items = nil
	mu.Unlock()
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
