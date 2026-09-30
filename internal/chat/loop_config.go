package chat

import (
	"os"
	"strconv"
	"strings"
)

const (
	defaultMaxToolTurns  = 8
	defaultMaxDupTools   = 3
	defaultDupWindow     = 12
	defaultMaxReplyChars = 120_000 // 单次 StreamReply 模型正文+工具结果字符预算；0=关闭
)

// wrapUpSystem 强制收尾时的临时 system（不写入持久 history）。
const wrapUpSystem = `工具调用轮次、重复调用或回复预算已达上限，禁止再调用任何工具。
请根据对话里已经得到的工具结果，用中文直接回复用户：
1. 已经完成或已经确认的结论；
2. 尚未完成的部分；
3. 建议用户下一步怎么拆分或继续提问。
不要寒暄，不要再提出要调用工具。`

// maxToolTurns 工具循环上限；GEEKAGENT_MAX_TOOL_TURNS，<1 → 默认。
func maxToolTurns() int {
	return envInt("GEEKAGENT_MAX_TOOL_TURNS", defaultMaxToolTurns)
}

// maxDupTools 窗口内同参出现次数达到此值则熔断。
func maxDupTools() int {
	return envInt("GEEKAGENT_MAX_DUP_TOOLS", defaultMaxDupTools)
}

// dupWindow 重复检测滑动窗口长度；若小于熔断次数则抬高到熔断次数。
func dupWindow() int {
	w := envInt("GEEKAGENT_DUP_WINDOW", defaultDupWindow)
	limit := maxDupTools()
	if w < limit {
		return limit
	}
	return w
}

// maxReplyChars 单次回复字符预算；环境变量为 0 时关闭。
func maxReplyChars() int {
	return envIntAllowZero("GEEKAGENT_MAX_REPLY_CHARS", defaultMaxReplyChars)
}

// shrinkToolsAfter 从第几个工具轮次（0-based）起只暴露只读工具。
// 未设置时默认 max(1, maxToolTurns*3/4)；显式 0 关闭收缩。
func shrinkToolsAfter() int {
	v := strings.TrimSpace(os.Getenv("GEEKAGENT_SHRINK_TOOLS_AFTER"))
	if v == "" {
		n := maxToolTurns() * 3 / 4
		if n < 1 {
			n = 1
		}
		return n
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		n := maxToolTurns() * 3 / 4
		if n < 1 {
			n = 1
		}
		return n
	}
	return n
}

// envInt 解析正整数环境变量；空、非法或 <1 时用 fallback。
func envInt(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 {
		return fallback
	}
	return n
}

// envIntAllowZero 同 envInt，但允许 0（用于显式关闭某护栏）。
func envIntAllowZero(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return fallback
	}
	return n
}

// toolCallKey 重复检测键：工具名 + 去空白后的参数。
func toolCallKey(name, args string) string {
	return name + "\x00" + strings.TrimSpace(args)
}
