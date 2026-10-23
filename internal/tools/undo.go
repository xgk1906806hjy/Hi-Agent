package tools

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// UndoPath 文件撤销栈（相对当前工作目录，随项目切换而切换）。最多 20 条。
const UndoPath = ".geekagent/undo.json"

const maxUndo = 20

// undoRecord 一次 write/patch 落盘前的状态。
type undoRecord struct {
	Path       string `json:"path"` // 绝对路径
	Existed    bool   `json:"existed"`
	OldContent string `json:"oldContent"`
}

type undoFile struct {
	Entries []undoRecord `json:"entries"`
}

// saveUndo 把一次写入前的状态压栈，超过 20 条时丢掉最旧的。
func saveUndo(path string, existed bool, old string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	entries, err := loadUndo()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		entries = nil
	}
	entries = append(entries, undoRecord{Path: abs, Existed: existed, OldContent: old})
	if len(entries) > maxUndo {
		entries = entries[len(entries)-maxUndo:]
	}
	return writeUndo(entries)
}

// UndoLast 弹出最近一次 write/patch：原本存在则写回旧内容，原本不存在则删除文件。
// 恢复前展示 diff 并确认。栈空时提示没有可撤销的写入。不撤销 shell 的副作用。
func UndoLast() (string, error) {
	entries, err := loadUndo()
	if errors.Is(err, os.ErrNotExist) || (err == nil && len(entries) == 0) {
		return "没有可撤销的写入", nil
	}
	if err != nil {
		_ = os.Remove(filepath.FromSlash(UndoPath))
		return "", err
	}
	rec := entries[len(entries)-1]
	cur := ""
	if data, err := os.ReadFile(rec.Path); err == nil {
		cur = string(data)
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	action := "恢复"
	if !rec.Existed {
		action = "删除（该文件由上次写入新建）"
	}
	prompt := fmt.Sprintf("\n%s\n确认撤销：%s %s？", simpleDiff(cur, rec.OldContent), action, rec.Path)
	if !confirm(redact(prompt)) {
		return "已取消撤销", nil
	}
	if rec.Existed {
		if err := os.WriteFile(rec.Path, []byte(rec.OldContent), 0o644); err != nil {
			return "", err
		}
	} else if err := os.Remove(rec.Path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	if err := writeUndo(entries[:len(entries)-1]); err != nil {
		return "", err
	}
	if rec.Existed {
		return "已恢复 " + rec.Path, nil
	}
	return "已删除 " + rec.Path, nil
}

func loadUndo() ([]undoRecord, error) {
	b, err := os.ReadFile(filepath.FromSlash(UndoPath))
	if err != nil {
		return nil, err
	}
	var stack undoFile
	if err := json.Unmarshal(b, &stack); err == nil && stack.Entries != nil {
		return stack.Entries, nil
	}
	var one undoRecord
	if err := json.Unmarshal(b, &one); err == nil && one.Path != "" {
		return []undoRecord{one}, nil
	}
	return nil, fmt.Errorf("撤销记录损坏，已清除")
}

func writeUndo(entries []undoRecord) error {
	file := filepath.FromSlash(UndoPath)
	if len(entries) == 0 {
		err := os.Remove(file)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(undoFile{Entries: entries}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(file, b, 0o644)
}
