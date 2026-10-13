package tools

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// UndoPath 单步撤销记录（相对当前工作目录，随项目切换而切换）。
const UndoPath = ".geekagent/undo.json"

// undoRecord 最近一次 write/patch 落盘前的状态。
type undoRecord struct {
	Path       string `json:"path"` // 绝对路径
	Existed    bool   `json:"existed"`
	OldContent string `json:"oldContent"`
}

// saveUndo 覆盖记录最近一次写入前的状态（只保留一步）。
func saveUndo(path string, existed bool, old string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(undoRecord{Path: abs, Existed: existed, OldContent: old}, "", "  ")
	if err != nil {
		return err
	}
	file := filepath.FromSlash(UndoPath)
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return err
	}
	return os.WriteFile(file, b, 0o644)
}

// UndoLast 撤销最近一次 write/patch：原本存在则写回旧内容，原本不存在则删除文件。
// 恢复前展示 diff 并确认；成功后清空记录。不撤销 shell 的副作用，也不支持多步。
func UndoLast() (string, error) {
	file := filepath.FromSlash(UndoPath)
	b, err := os.ReadFile(file)
	if errors.Is(err, os.ErrNotExist) {
		return "没有可撤销的写入", nil
	}
	if err != nil {
		return "", err
	}
	var rec undoRecord
	if err := json.Unmarshal(b, &rec); err != nil || rec.Path == "" {
		_ = os.Remove(file)
		return "", fmt.Errorf("撤销记录损坏，已清除")
	}
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
	_ = os.Remove(file)
	if rec.Existed {
		return "已恢复 " + rec.Path, nil
	}
	return "已删除 " + rec.Path, nil
}
