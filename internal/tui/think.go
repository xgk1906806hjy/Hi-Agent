package tui

import "strings"

// 支持的思考标签对（流式可能跨 chunk 切开）。
var thinkTagPairs = [][2]string{
	{"<think>", "</think>"},
	{"<thinking>", "</thinking>"},
	{"<reasoning>", "</reasoning>"},
}

// ThinkStream 从模型流中拆出思考段与正文段。
type ThinkStream struct {
	inThink bool
	hold    string // 可能跨 chunk 的半截标签
}

// Feed 喂入一段模型 delta，返回应走 Progress 的思考片段与走 Model 的正文。
func (t *ThinkStream) Feed(s string) (think, model string) {
	if s == "" {
		return "", ""
	}
	data := t.hold + s
	t.hold = ""

	var thinkB, modelB strings.Builder
	i := 0
	for i < len(data) {
		if t.inThink {
			endTag, endAt := findCloseTag(data[i:])
			if endAt < 0 {
				// 可能末尾是半截结束标签
				if n := partialSuffix(data[i:], closeTags()); n > 0 {
					thinkB.WriteString(data[i : len(data)-n])
					t.hold = data[len(data)-n:]
					return thinkB.String(), modelB.String()
				}
				thinkB.WriteString(data[i:])
				return thinkB.String(), modelB.String()
			}
			thinkB.WriteString(data[i : i+endAt])
			i += endAt + len(endTag)
			t.inThink = false
			continue
		}

		openTag, openAt := findOpenTag(data[i:])
		if openAt < 0 {
			if n := partialSuffix(data[i:], openTags()); n > 0 {
				modelB.WriteString(data[i : len(data)-n])
				t.hold = data[len(data)-n:]
				return thinkB.String(), modelB.String()
			}
			modelB.WriteString(data[i:])
			return thinkB.String(), modelB.String()
		}
		modelB.WriteString(data[i : i+openAt])
		i += openAt + len(openTag)
		t.inThink = true
	}
	return thinkB.String(), modelB.String()
}

func openTags() []string {
	out := make([]string, len(thinkTagPairs))
	for i, p := range thinkTagPairs {
		out[i] = p[0]
	}
	return out
}

func closeTags() []string {
	out := make([]string, len(thinkTagPairs))
	for i, p := range thinkTagPairs {
		out[i] = p[1]
	}
	return out
}

func findOpenTag(s string) (tag string, at int) {
	at = -1
	for _, p := range thinkTagPairs {
		if i := strings.Index(s, p[0]); i >= 0 && (at < 0 || i < at) {
			at = i
			tag = p[0]
		}
	}
	return tag, at
}

func findCloseTag(s string) (tag string, at int) {
	at = -1
	for _, p := range thinkTagPairs {
		if i := strings.Index(s, p[1]); i >= 0 && (at < 0 || i < at) {
			at = i
			tag = p[1]
		}
	}
	return tag, at
}

// partialSuffix 若 s 后缀是某个 tag 的真前缀（还未完整），返回该后缀长度。
func partialSuffix(s string, tags []string) int {
	best := 0
	for _, tag := range tags {
		max := len(tag) - 1
		if max > len(s) {
			max = len(s)
		}
		for n := max; n > 0; n-- {
			if strings.HasSuffix(s, tag[:n]) {
				if n > best {
					best = n
				}
				break
			}
		}
	}
	return best
}
