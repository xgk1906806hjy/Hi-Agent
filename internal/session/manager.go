package session

import (
	"fmt"
	"strings"

	"hi-agent/internal/chat"
)

// Manager 绑定一个 Chat 与多会话 Data。
type Manager struct {
	data *Data
	chat *chat.Chat
}

// NewManager 使用已有 Data 与 Chat；若 data 为 nil 则 NewData。
func NewManager(c *chat.Chat, data *Data) *Manager {
	if data == nil {
		data = NewData()
	}
	m := &Manager{data: data, chat: c}
	m.applyCurrentToChat()
	return m
}

// Current 当前会话 ID。
func (m *Manager) Current() string {
	return m.data.Current
}

// SyncFromChat 把 Chat.history 写回当前会话记录。
func (m *Manager) SyncFromChat() {
	m.data.PutMessages(m.data.Current, m.chat.History())
}

func (m *Manager) applyCurrentToChat() {
	rec := m.data.Get(m.data.Current)
	if rec == nil {
		m.data.Ensure(m.data.Current)
		rec = m.data.Get(m.data.Current)
	}
	m.chat.SetHistory(rec.Messages)
}

// NewSession 保存当前 → 创建空会话并切换。可选自定义 id；空则生成 8 位 ID。
func (m *Manager) NewSession(id string) (string, error) {
	m.SyncFromChat()
	id = strings.TrimSpace(id)
	if id == "" {
		var err error
		id, err = NewID()
		if err != nil {
			return "", err
		}
	}
	if !ValidID(id) {
		return "", fmt.Errorf("非法会话 ID：%s", id)
	}
	if _, exists := m.data.Sessions[id]; exists {
		return "", fmt.Errorf("会话已存在：%s", id)
	}
	m.data.Sessions[id] = &Record{ID: id}
	m.data.Current = id
	m.chat.SetHistory(nil)
	m.data.PutMessages(id, nil)
	return id, nil
}

// Open 保存当前 → 切换到已有会话。
func (m *Manager) Open(id string) error {
	id = strings.TrimSpace(id)
	if !ValidID(id) {
		return fmt.Errorf("非法会话 ID：%s", id)
	}
	if _, ok := m.data.Sessions[id]; !ok {
		return fmt.Errorf("会话不存在：%s", id)
	}
	m.SyncFromChat()
	m.data.Current = id
	m.applyCurrentToChat()
	return nil
}

// ListLines 返回供展示的会话列表行。
func (m *Manager) ListLines() []string {
	m.SyncFromChat()
	ids := m.data.IDs()
	lines := make([]string, 0, len(ids))
	for _, id := range ids {
		rec := m.data.Get(id)
		n := 0
		if rec != nil {
			n = len(rec.Messages)
		}
		mark := " "
		if id == m.data.Current {
			mark = "*"
		}
		lines = append(lines, fmt.Sprintf("%s %s  (%d 条消息)", mark, id, n))
	}
	return lines
}

// Save 同步当前 Chat 后落盘。
func (m *Manager) Save() error {
	m.SyncFromChat()
	return m.data.Save()
}

// Load 从磁盘重载并应用到 Chat（丢弃未保存的内存改动）。
func (m *Manager) Load() (string, error) {
	d, err := Load()
	if err != nil {
		return "", err
	}
	m.data = d
	m.applyCurrentToChat()
	return m.data.Current, nil
}

// PrepareExit 退出前：同步；若当前为 default 且有消息，改名为 8 位 ID；再落盘。
func (m *Manager) PrepareExit() (renamed string, err error) {
	m.SyncFromChat()
	if m.data.Current == DefaultID {
		rec := m.data.Get(DefaultID)
		if rec != nil && len(rec.Messages) > 0 {
			id, genErr := NewID()
			if genErr != nil {
				return "", genErr
			}
			if err := m.data.Rename(DefaultID, id); err != nil {
				return "", err
			}
			renamed = id
		}
	}
	if err := m.data.Save(); err != nil {
		return renamed, err
	}
	return renamed, nil
}
