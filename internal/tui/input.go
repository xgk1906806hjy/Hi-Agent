package tui

import (
	"errors"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"golang.org/x/term"
)

// ErrInterrupted 表示输入时按下 Ctrl+C（raw 模式下 ISIG 关闭，需自行处理）。
var ErrInterrupted = errors.New("interrupted")

// SlashCmd 斜杠命令提示项。
type SlashCmd struct {
	Name string // 如 "/help"
	Desc string
}

// key 表示一次按键解析结果。
type key struct {
	kind keyKind
	r    rune // kindRune 时有效
}

type keyKind int

const (
	keyRune keyKind = iota
	keyEnter
	keyBackspace
	keyTab
	keyEscape
	keyUp
	keyDown
	keyPgUp
	keyPgDn
	keyWheelUp
	keyWheelDown
	keyCtrlC
	keyCtrlD
	keyUnknown
)

const wheelScrollLines = 3 // 滚轮一格滚动行数

// readKey 从原始终端读一个按键（含 CSI、SGR/X10 鼠标滚轮）。
func readKey(r io.Reader) (key, error) {
	var buf [32]byte
	n, err := r.Read(buf[:1])
	if n == 0 {
		if err != nil {
			return key{}, err
		}
		return key{kind: keyUnknown}, nil
	}
	b := buf[0]
	switch b {
	case '\r', '\n':
		return key{kind: keyEnter}, nil
	case 0x7f, 0x08:
		return key{kind: keyBackspace}, nil
	case '\t':
		return key{kind: keyTab}, nil
	case 0x03:
		return key{kind: keyCtrlC}, nil
	case 0x04:
		return key{kind: keyCtrlD}, nil
	case 0x1b:
		return readEscapeKey(r)
	}

	// UTF-8 续读
	need := utf8RuneLen(b)
	if need == 1 {
		return key{kind: keyRune, r: rune(b)}, nil
	}
	if need > 1 {
		got, _ := readMore(r, buf[1:need])
		if got+1 < need {
			return key{kind: keyUnknown}, nil
		}
		rr, _ := utf8.DecodeRune(buf[:need])
		if rr == utf8.RuneError {
			return key{kind: keyUnknown}, nil
		}
		return key{kind: keyRune, r: rr}, nil
	}
	return key{kind: keyUnknown}, nil
}

// readEscapeKey 解析 ESC 开头序列：方向键、PgUp/PgDn、鼠标滚轮。
func readEscapeKey(r io.Reader) (key, error) {
	var b [1]byte
	if n, _ := r.Read(b[:]); n == 0 {
		return key{kind: keyEscape}, nil
	}
	if b[0] != '[' {
		return key{kind: keyUnknown}, nil
	}
	if n, _ := r.Read(b[:]); n == 0 {
		return key{kind: keyEscape}, nil
	}
	switch b[0] {
	case 'A':
		return key{kind: keyUp}, nil
	case 'B':
		return key{kind: keyDown}, nil
	case 'M':
		// X10 鼠标：再跟 3 字节 cb, cx, cy；滚轮 cb=64/65(+32 编码后为 96/97… 实际读到的是 button+32)
		var m [3]byte
		if got, _ := readMore(r, m[:]); got < 3 {
			return key{kind: keyUnknown}, nil
		}
		btn := int(m[0]) - 32
		switch btn {
		case 64:
			return key{kind: keyWheelUp}, nil
		case 65:
			return key{kind: keyWheelDown}, nil
		default:
			return key{kind: keyUnknown}, nil
		}
	case '<':
		// SGR 鼠标：ESC [ < btn ; x ; y M/m
		return readSGRMouse(r)
	case '5':
		if n, _ := r.Read(b[:]); n > 0 && b[0] == '~' {
			return key{kind: keyPgUp}, nil
		}
		return key{kind: keyUnknown}, nil
	case '6':
		if n, _ := r.Read(b[:]); n > 0 && b[0] == '~' {
			return key{kind: keyPgDn}, nil
		}
		return key{kind: keyUnknown}, nil
	default:
		// 吞掉未知 CSI 直到字母结束符，避免污染后续输入
		drainCSI(r, b[0])
		return key{kind: keyUnknown}, nil
	}
}

func readSGRMouse(r io.Reader) (key, error) {
	var raw strings.Builder
	var b [1]byte
	for {
		n, err := r.Read(b[:])
		if n == 0 {
			if err != nil {
				return key{}, err
			}
			return key{kind: keyUnknown}, nil
		}
		c := b[0]
		if c == 'M' || c == 'm' {
			break
		}
		raw.WriteByte(c)
		if raw.Len() > 24 {
			return key{kind: keyUnknown}, nil
		}
	}
	// btn;col;row
	parts := strings.Split(raw.String(), ";")
	if len(parts) < 1 {
		return key{kind: keyUnknown}, nil
	}
	btn := 0
	for _, ch := range parts[0] {
		if ch < '0' || ch > '9' {
			return key{kind: keyUnknown}, nil
		}
		btn = btn*10 + int(ch-'0')
	}
	switch btn {
	case 64:
		return key{kind: keyWheelUp}, nil
	case 65:
		return key{kind: keyWheelDown}, nil
	default:
		return key{kind: keyUnknown}, nil
	}
}

func drainCSI(r io.Reader, first byte) {
	if first >= 0x40 && first <= 0x7e {
		return
	}
	var b [1]byte
	for {
		n, _ := r.Read(b[:])
		if n == 0 {
			return
		}
		if b[0] >= 0x40 && b[0] <= 0x7e {
			return
		}
	}
}

func readMore(r io.Reader, dst []byte) (int, error) {
	total := 0
	for total < len(dst) {
		n, err := r.Read(dst[total:])
		total += n
		if err != nil || n == 0 {
			return total, err
		}
	}
	return total, nil
}

func utf8RuneLen(b byte) int {
	switch {
	case b < 0x80:
		return 1
	case b&0xe0 == 0xc0:
		return 2
	case b&0xf0 == 0xe0:
		return 3
	case b&0xf8 == 0xf0:
		return 4
	default:
		return 1
	}
}

// FilterSlashCmds 按已输入前缀过滤命令（input 通常以 / 开头）。
func FilterSlashCmds(cmds []SlashCmd, input string) []SlashCmd {
	if !strings.HasPrefix(input, "/") {
		return nil
	}
	var out []SlashCmd
	for _, c := range cmds {
		if strings.HasPrefix(c.Name, input) {
			out = append(out, c)
		}
	}
	return out
}

// CompleteSlash 返回 Tab 补全结果（唯一前缀或第一个匹配）。
func CompleteSlash(cmds []SlashCmd, input string) string {
	matches := FilterSlashCmds(cmds, input)
	if len(matches) == 0 {
		return input
	}
	if len(matches) == 1 {
		return matches[0].Name
	}
	// 公共前缀
	prefix := matches[0].Name
	for _, m := range matches[1:] {
		prefix = commonPrefix(prefix, m.Name)
		if prefix == input {
			break
		}
	}
	if len(prefix) > len(input) {
		return prefix
	}
	return input
}

func commonPrefix(a, b string) string {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	i := 0
	for i < n && a[i] == b[i] {
		i++
	}
	return a[:i]
}

// formatSlashHint 生成提示行文案（带简短说明）。
func formatSlashHint(matches []SlashCmd, maxWidth int) string {
	if len(matches) == 0 {
		return ""
	}
	var parts []string
	for _, m := range matches {
		item := m.Name
		if m.Desc != "" {
			item = m.Name + ":" + m.Desc
		}
		trial := parts
		trial = append(trial, item)
		if textWidth("命令 "+strings.Join(trial, " · ")+" · Tab") > maxWidth-2 {
			if len(parts) == 0 {
				parts = append(parts, truncateRunes(item, maxWidth-12))
			} else {
				parts = append(parts, "…")
			}
			break
		}
		parts = append(parts, item)
	}
	return "命令 " + strings.Join(parts, " · ") + " · Tab 补全"
}

// rawReadLine 在 raw 模式下编辑一行；失败时回退到 cooked bufio。
func (t *TUI) rawReadLine(prompt string) (string, error) {
	fd := int(os.Stdin.Fd())
	old, err := term.MakeRaw(fd)
	if err != nil {
		t.mu.Lock()
		t.prompt = prompt
		t.inputBuf = ""
		t.mu.Unlock()
		t.Flush()
		line, err2 := t.in.ReadString('\n')
		t.mu.Lock()
		t.prompt = ""
		t.inputBuf = ""
		t.mu.Unlock()
		return strings.TrimSpace(line), err2
	}
	defer func() {
		disableMouse()
		_ = term.Restore(fd, old)
	}()
	enableMouse()

	t.mu.Lock()
	t.prompt = prompt
	t.inputBuf = ""
	t.scrollOff = 0
	t.mu.Unlock()
	t.Flush()

	for {
		k, err := readKey(os.Stdin)
		if err != nil {
			t.clearInputState()
			return "", err
		}
		switch k.kind {
		case keyEnter:
			t.mu.Lock()
			line := t.inputBuf
			t.prompt = ""
			t.inputBuf = ""
			t.scrollOff = 0
			t.mu.Unlock()
			t.Flush()
			return strings.TrimSpace(line), nil
		case keyCtrlC:
			t.clearInputState()
			return "", ErrInterrupted
		case keyCtrlD:
			t.clearInputState()
			return "", io.EOF
		case keyBackspace:
			t.mu.Lock()
			t.inputBuf = trimLastRune(t.inputBuf)
			t.scrollOff = 0
			t.mu.Unlock()
			t.Flush()
		case keyTab:
			t.mu.Lock()
			if strings.HasPrefix(t.inputBuf, "/") {
				t.inputBuf = CompleteSlash(t.slashCmds, t.inputBuf)
			}
			t.mu.Unlock()
			t.Flush()
		case keyPgUp, keyUp, keyWheelUp:
			t.scrollBy(wheelDelta(k.kind))
		case keyPgDn, keyDown, keyWheelDown:
			t.scrollBy(-wheelDelta(k.kind))
		case keyRune:
			if k.r < 32 {
				break
			}
			t.mu.Lock()
			t.inputBuf += string(k.r)
			t.scrollOff = 0 // 输入时回到底部
			t.mu.Unlock()
			t.Flush()
		}
	}
}

func wheelDelta(k keyKind) int {
	switch k {
	case keyWheelUp, keyWheelDown:
		return wheelScrollLines
	default:
		return 1
	}
}

func (t *TUI) scrollBy(delta int) {
	t.mu.Lock()
	t.scrollOff += delta
	if t.scrollOff < 0 {
		t.scrollOff = 0
	}
	t.mu.Unlock()
	t.Flush()
}

func enableMouse() {
	// 1000=基本鼠标；1006=SGR 坐标（滚轮 btn 64/65）
	_, _ = os.Stdout.WriteString("\x1b[?1000h\x1b[?1006h")
}

func disableMouse() {
	_, _ = os.Stdout.WriteString("\x1b[?1006l\x1b[?1000l")
}

func (t *TUI) clearInputState() {
	t.mu.Lock()
	t.prompt = ""
	t.inputBuf = ""
	t.scrollOff = 0
	t.mu.Unlock()
	t.Flush()
}

func trimLastRune(s string) string {
	if s == "" {
		return ""
	}
	_, size := utf8.DecodeLastRuneInString(s)
	return s[:len(s)-size]
}
