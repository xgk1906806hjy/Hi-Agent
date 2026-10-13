package workspace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/xgk1906806hjy/Hi-Agent/internal/session"
)

// SessionInfo 某项目里一个会话的摘要。
type SessionInfo struct {
	ID        string
	Title     string
	Messages  int
	UpdatedAt time.Time
	Current   bool
}

// ListSessions 只读解析 dir 下的会话文件，按 ID 排序；文件缺失或损坏返回空。
func ListSessions(dir string) []SessionInfo {
	b, err := os.ReadFile(filepath.Join(dir, session.Path()))
	if err != nil {
		return nil
	}
	var f struct {
		Current  string `json:"current"`
		Sessions map[string]struct {
			UpdatedAt time.Time `json:"updatedAt"`
			Messages  []struct {
				Role    string          `json:"role"`
				Content json.RawMessage `json:"content"`
			} `json:"messages"`
		} `json:"sessions"`
	}
	if json.Unmarshal(b, &f) != nil {
		return nil
	}
	out := make([]SessionInfo, 0, len(f.Sessions))
	for id, rec := range f.Sessions {
		info := SessionInfo{ID: id, Messages: len(rec.Messages), UpdatedAt: rec.UpdatedAt, Current: id == f.Current}
		for _, m := range rec.Messages {
			var text string
			if m.Role == "user" && json.Unmarshal(m.Content, &text) == nil {
				info.Title = session.TitleFromText(text)
				break
			}
		}
		out = append(out, info)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
