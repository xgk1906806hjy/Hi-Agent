package session

import (
	"fmt"
	"sort"
	"strings"
	"time"

	openai "github.com/sashabaranov/go-openai"

	"github.com/xgk1906806hjy/Hi-Agent/internal/chat"
)

// Manager 绑定一个 Chat 与多会话 Data；切换靠替换 history，不并行多 Client。
type Manager struct {
	data *Data
	chat *chat.Chat
}

// NewManager 使用已有 Data 与 Chat；若 data 为 nil 则 NewData，并立即把当前会话应用到 Chat。
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

// applyCurrentToChat 将 Data 中当前会话的 messages 灌入 Chat（缺失则 Ensure 空记录）。
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

// SessionBrief 侧栏用的会话摘要（不从 Chat 回写，避免与流式写入抢 history）。
type SessionBrief struct {
	ID        string
	Title     string // 第一条用户消息的开头；无则为空
	Messages  int
	UpdatedAt time.Time
	Current   bool
}

// titleRunes 会话标题最多取的字符数。
const titleRunes = 24

// SessionBriefs 列出会话摘要。当前会话的消息数与标题取自 Chat（只读），其余取自 Data。
func (m *Manager) SessionBriefs() []SessionBrief {
	ids := m.data.IDs()
	out := make([]SessionBrief, 0, len(ids))
	for _, id := range ids {
		b := SessionBrief{ID: id, Current: id == m.data.Current}
		var msgs []openai.ChatCompletionMessage
		if rec := m.data.Get(id); rec != nil {
			msgs = rec.Messages
			b.UpdatedAt = rec.UpdatedAt
		}
		if b.Current {
			msgs = m.chat.History()
		}
		b.Messages = len(msgs)
		b.Title = Title(msgs)
		out = append(out, b)
	}
	return out
}

// Title 取第一条非空用户消息的前 titleRunes 个字符（换行压成空格）。
func Title(msgs []openai.ChatCompletionMessage) string {
	for _, m := range msgs {
		if m.Role != openai.ChatMessageRoleUser {
			continue
		}
		return TitleFromText(m.Content)
	}
	return ""
}

// TitleFromText 把一段用户输入压成侧栏标题。
func TitleFromText(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	r := []rune(s)
	if len(r) > titleRunes {
		return string(r[:titleRunes]) + "…"
	}
	return s
}

// Rename 重命名会话（当前会话也可改）；新 ID 须合法且未被占用。
func (m *Manager) Rename(oldID, newID string) error {
	oldID, newID = strings.TrimSpace(oldID), strings.TrimSpace(newID)
	if !ValidID(newID) {
		return fmt.Errorf("非法会话 ID：%s", newID)
	}
	m.SyncFromChat()
	return m.data.Rename(oldID, newID)
}

// Delete 删除会话。删的是当前会话时切到最近更新的另一个；删完为空则补一个空的 default。
func (m *Manager) Delete(id string) error {
	id = strings.TrimSpace(id)
	if _, ok := m.data.Sessions[id]; !ok {
		return fmt.Errorf("会话不存在：%s", id)
	}
	m.SyncFromChat()
	delete(m.data.Sessions, id)
	if m.data.Current != id {
		return nil
	}
	next := ""
	var latest time.Time
	for sid, rec := range m.data.Sessions {
		if next == "" || rec.UpdatedAt.After(latest) {
			next, latest = sid, rec.UpdatedAt
		}
	}
	if next == "" {
		next = DefaultID
		m.data.Ensure(next)
	}
	m.data.Current = next
	m.applyCurrentToChat()
	return nil
}

// ListLines 返回供展示的会话列表行（当前会话前缀 *）。
// 含标题与更新时间；未保存的排在最前，其余按更新时间倒序。
func (m *Manager) ListLines() []string {
	m.SyncFromChat()
	type row struct {
		line string
		at   time.Time
	}
	rows := make([]row, 0, len(m.data.Sessions))
	for _, id := range m.data.IDs() {
		rec := m.data.Get(id)
		n := 0
		title := ""
		var at time.Time
		if rec != nil {
			n = len(rec.Messages)
			title = Title(rec.Messages)
			at = rec.UpdatedAt
		}
		if title == "" {
			title = "（无标题）"
		}
		when := "未保存"
		if !at.IsZero() {
			when = at.Format("01-02 15:04")
		}
		mark := " "
		if id == m.data.Current {
			mark = "*"
		}
		rows = append(rows, row{
			line: fmt.Sprintf("%s %s  %s  (%d 条 · %s)", mark, id, title, n, when),
			at:   at,
		})
	}
	sort.SliceStable(rows, func(i, j int) bool {
		zi, zj := rows[i].at.IsZero(), rows[j].at.IsZero()
		if zi != zj {
			return zi
		}
		return rows[i].at.After(rows[j].at)
	})
	lines := make([]string, len(rows))
	for i, r := range rows {
		lines[i] = r.line
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

	// default 有内容时改名为随机 ID，避免下次启动覆盖有意义的历史
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
