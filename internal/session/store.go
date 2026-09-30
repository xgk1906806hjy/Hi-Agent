// Package session 管理多会话内存与落盘（.geekagent/sessions.json），
// 与单个 chat.Chat 的 history 双向同步。
package session

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

const (
	// DefaultID 默认会话 ID；退出时若有消息会改名为 8 位 hex。
	DefaultID   = "default"
	fileRelPath = ".geekagent/sessions.json"
)

// Data 落盘结构：当前会话 ID + 会话 Map。
type Data struct {
	Current  string             `json:"current"`
	Sessions map[string]*Record `json:"sessions"`
}

// Record 单会话快照；Messages 即 OpenAI history，零转换。
type Record struct {
	ID        string                         `json:"id"`
	UpdatedAt time.Time                      `json:"updatedAt"`
	Messages  []openai.ChatCompletionMessage `json:"messages"`
}

// Path 返回会话文件路径（相对工作目录）。
func Path() string {
	return fileRelPath
}

// NewData 创建仅含 default 空会话的数据。
func NewData() *Data {
	return &Data{
		Current: DefaultID,
		Sessions: map[string]*Record{
			DefaultID: {
				ID:        DefaultID,
				UpdatedAt: time.Now(),
				Messages:  nil,
			},
		},
	}
}

// Load 从磁盘读取；文件不存在返回 NewData + nil error。
func Load() (*Data, error) {
	b, err := os.ReadFile(Path())
	if err != nil {
		if os.IsNotExist(err) {
			return NewData(), nil
		}
		return nil, err
	}
	var d Data
	if err := json.Unmarshal(b, &d); err != nil {
		return nil, fmt.Errorf("解析会话文件失败：%w", err)
	}
	if d.Sessions == nil {
		d.Sessions = map[string]*Record{}
	}
	if d.Current == "" {
		d.Current = DefaultID
	}
	if _, ok := d.Sessions[d.Current]; !ok {
		if len(d.Sessions) == 0 {
			d.Sessions[DefaultID] = &Record{ID: DefaultID, UpdatedAt: time.Now()}
			d.Current = DefaultID
		} else {
			// 当前 ID 丢失：取字典序第一个
			ids := d.IDs()
			d.Current = ids[0]
		}
	}
	for id, rec := range d.Sessions {
		if rec == nil {
			d.Sessions[id] = &Record{ID: id, UpdatedAt: time.Now()}
			continue
		}
		if rec.ID == "" {
			rec.ID = id
		}
	}
	return &d, nil
}

// Save 写入 `.geekagent/sessions.json`（自动建目录）。
func (d *Data) Save() error {
	if err := os.MkdirAll(filepath.Dir(Path()), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(Path(), b, 0o644)
}

// IDs 返回按字典序排列的会话 ID。
func (d *Data) IDs() []string {
	ids := make([]string, 0, len(d.Sessions))
	for id := range d.Sessions {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// Get 返回会话记录（可能为 nil）。
func (d *Data) Get(id string) *Record {
	return d.Sessions[id]
}

// PutMessages 写入指定会话消息并刷新时间戳（深拷贝 messages）。
func (d *Data) PutMessages(id string, msgs []openai.ChatCompletionMessage) {
	rec := d.Sessions[id]
	if rec == nil {
		rec = &Record{ID: id}
		d.Sessions[id] = rec
	}
	rec.ID = id
	rec.Messages = cloneMsgs(msgs)
	rec.UpdatedAt = time.Now()
}

// Ensure 保证 id 对应的空记录存在。
func (d *Data) Ensure(id string) {
	if _, ok := d.Sessions[id]; !ok {
		d.Sessions[id] = &Record{ID: id, UpdatedAt: time.Now()}
	}
}

// Rename 将 oldID 改为 newID（目标已存在则失败）；若 Current 是 oldID 则一并更新。
func (d *Data) Rename(oldID, newID string) error {
	if oldID == newID {
		return nil
	}
	rec := d.Sessions[oldID]
	if rec == nil {
		return fmt.Errorf("会话不存在：%s", oldID)
	}
	if _, exists := d.Sessions[newID]; exists {
		return fmt.Errorf("会话已存在：%s", newID)
	}
	delete(d.Sessions, oldID)
	rec.ID = newID
	rec.UpdatedAt = time.Now()
	d.Sessions[newID] = rec
	if d.Current == oldID {
		d.Current = newID
	}
	return nil
}

// NewID 生成 8 位十六进制 ID（小写）。
func NewID() (string, error) {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

// ValidID 校验会话 ID（非空、无空白、不含路径分隔）。
func ValidID(id string) bool {
	id = strings.TrimSpace(id)
	if id == "" || id != strings.TrimSpace(id) {
		return false
	}
	if strings.ContainsAny(id, "/\\ \t\n") {
		return false
	}
	return true
}

// cloneMsgs 深拷贝 messages（含 ToolCalls），避免与 Chat history 共享底层切片。
func cloneMsgs(in []openai.ChatCompletionMessage) []openai.ChatCompletionMessage {
	if len(in) == 0 {
		return nil
	}
	out := make([]openai.ChatCompletionMessage, len(in))
	copy(out, in)
	for i := range out {
		if len(out[i].ToolCalls) > 0 {
			out[i].ToolCalls = append([]openai.ToolCall(nil), out[i].ToolCalls...)
		}
	}
	return out
}
