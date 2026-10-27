// Package memory 保存项目级长期记忆（Day 10）。
// 文件为工作区 `.geekagent/memory.json`，切换目录后即换成新项目的文件。
// 只追加和搜索，不提供修改与删除。
package memory

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const fileRel = ".geekagent/memory.json"

const (
	searchLimit   = 5   // 显式 memory_search 最多返回条数
	recallLimit   = 5   // 每轮主动召回最多条数
	chunkMaxRunes = 240 // 单个记忆块的最大 rune 数
	bm25K1        = 1.2
	bm25B         = 0.75
)

// Entry 一条记忆。
type Entry struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
}

// Hit 一个 BM25 召回的记忆块。
type Hit struct {
	EntryID string
	Content string
	Score   float64
}

// Path 返回记忆文件的相对路径。
func Path() string { return fileRel }

// Write 追加一条记忆并落盘。
func Write(content string) (Entry, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return Entry{}, fmt.Errorf("记忆内容为空")
	}
	entries, err := load()
	if err != nil {
		return Entry{}, err
	}
	id, err := newID()
	if err != nil {
		return Entry{}, err
	}
	e := Entry{ID: id, Content: content, CreatedAt: time.Now()}
	entries = append(entries, e)
	if err := save(entries); err != nil {
		return Entry{}, err
	}
	return e, nil
}

// Search 不区分大小写的子串匹配，最多 5 条，按写入顺序。
func Search(query string) ([]Entry, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("缺少搜索词")
	}
	entries, err := load()
	if err != nil {
		return nil, err
	}
	q := strings.ToLower(query)
	var out []Entry
	for _, e := range entries {
		if strings.Contains(strings.ToLower(e.Content), q) || strings.Contains(strings.ToLower(e.ID), q) {
			out = append(out, e)
			if len(out) >= searchLimit {
				break
			}
		}
	}
	return out, nil
}

// Recall 把记忆按内容切块后做 BM25 检索，返回得分最高的 topN 个块。
// 只保留至少命中一个查询词项的块；topN<=0 时使用 recallLimit。
func Recall(query string, topN int) ([]Hit, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	terms := uniqueStrings(tokenizeQuery(query))
	if len(terms) == 0 {
		return nil, nil
	}
	entries, err := load()
	if err != nil {
		return nil, err
	}
	if topN <= 0 {
		topN = recallLimit
	}

	type doc struct {
		entryID   string
		createdAt time.Time
		content   string
		tf        map[string]int
		length    int
	}
	var docs []doc
	for _, e := range entries {
		for _, chunk := range chunkText(e.Content) {
			tokens := tokenize(chunk)
			if len(tokens) == 0 {
				continue
			}
			tf := make(map[string]int, len(tokens))
			for _, t := range tokens {
				tf[t]++
			}
			docs = append(docs, doc{
				entryID:   e.ID,
				createdAt: e.CreatedAt,
				content:   chunk,
				tf:        tf,
				length:    len(tokens),
			})
		}
	}
	if len(docs) == 0 {
		return nil, nil
	}

	// 只统计查询词对应的文档频率。
	df := make(map[string]int, len(terms))
	for _, t := range terms {
		for _, d := range docs {
			if d.tf[t] > 0 {
				df[t]++
			}
		}
	}
	totalLen := 0
	for _, d := range docs {
		totalLen += d.length
	}
	avgLen := float64(totalLen) / float64(len(docs))

	type scoredHit struct {
		hit     Hit
		created time.Time
	}
	var hits []scoredHit
	for _, d := range docs {
		score := 0.0
		for _, t := range terms {
			tf := d.tf[t]
			if tf == 0 {
				continue
			}
			n := float64(len(docs))
			freq := float64(df[t])
			idf := math.Log(1 + (n-freq+0.5)/(freq+0.5))
			denom := float64(tf) + bm25K1*(1-bm25B+bm25B*float64(d.length)/avgLen)
			score += idf * (float64(tf) * (bm25K1 + 1)) / denom
		}
		if score <= 0 {
			continue
		}
		hits = append(hits, scoredHit{
			hit:     Hit{EntryID: d.entryID, Content: d.content, Score: score},
			created: d.createdAt,
		})
	}
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].hit.Score != hits[j].hit.Score {
			return hits[i].hit.Score > hits[j].hit.Score
		}
		if !hits[i].created.Equal(hits[j].created) {
			return hits[i].created.After(hits[j].created)
		}
		return hits[i].hit.Content < hits[j].hit.Content
	})
	if len(hits) > topN {
		hits = hits[:topN]
	}
	out := make([]Hit, 0, len(hits))
	for _, h := range hits {
		out = append(out, h.hit)
	}
	return out, nil
}

// RecallNote 生成可直接放进 system 的主动记忆说明；没有命中时返回空串。
// 读取失败按「没有可注入记忆」处理，不阻断本轮对话；显式 /memory 仍会报告错误。
func RecallNote(query string) string {
	hits, err := Recall(query, recallLimit)
	if err != nil || len(hits) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("【项目长期记忆·自动召回】\n")
	b.WriteString("以下内容可能与当前问题相关，仅在确实相关时参考，不要编造：\n")
	for _, h := range hits {
		b.WriteString("- ")
		b.WriteString(h.Content)
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// chunkText 按段落和句子把记忆切成不超过 chunkMaxRunes 的块。
// 过长的单句会硬切；空行会结束当前块。
func chunkText(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}

	var chunks []string
	var current strings.Builder
	currentLen := 0
	flush := func() {
		if text := strings.TrimSpace(current.String()); text != "" {
			chunks = append(chunks, text)
		}
		current.Reset()
		currentLen = 0
	}
	appendPiece := func(piece string) {
		piece = strings.TrimSpace(piece)
		if piece == "" {
			return
		}
		n := utf8.RuneCountInString(piece)
		if n > chunkMaxRunes {
			flush()
			runes := []rune(piece)
			for len(runes) > 0 {
				take := chunkMaxRunes
				if len(runes) < take {
					take = len(runes)
				}
				chunks = append(chunks, strings.TrimSpace(string(runes[:take])))
				runes = runes[take:]
			}
			return
		}
		if currentLen > 0 && currentLen+1+n > chunkMaxRunes {
			flush()
		}
		if currentLen > 0 {
			current.WriteByte(' ')
			currentLen++
		}
		current.WriteString(piece)
		currentLen += n
	}

	for _, paragraph := range strings.Split(s, "\n") {
		if strings.TrimSpace(paragraph) == "" {
			flush()
			continue
		}
		for _, sentence := range splitSentences(paragraph) {
			appendPiece(sentence)
		}
	}
	flush()
	return chunks
}

// splitSentences 按中英文句末标点切句，保留标点。
func splitSentences(s string) []string {
	var out []string
	var b strings.Builder
	for _, r := range s {
		b.WriteRune(r)
		switch r {
		case '。', '！', '？', '!', '?', '；', ';':
			if text := strings.TrimSpace(b.String()); text != "" {
				out = append(out, text)
			}
			b.Reset()
		}
	}
	if text := strings.TrimSpace(b.String()); text != "" {
		out = append(out, text)
	}
	return out
}

// tokenize 做文档侧轻量分词：拉丁字母/数字按连续词，CJK 生成单字与双字 gram。
// 这样无需词典也能让中文查询与记忆产生稳定的词项重叠。
func tokenize(s string) []string {
	return tokenizeText(s, true)
}

// tokenizeQuery 是查询侧分词：CJK 主要用双字 gram，只有单字查询才保留单字，
// 避免「我」「的」这类高频单字把无关记忆带进 system。
func tokenizeQuery(s string) []string {
	return tokenizeText(s, false)
}

func tokenizeText(s string, cjkUnigrams bool) []string {
	s = strings.ToLower(s)
	var out []string
	var word strings.Builder
	var cjk []rune
	flushWord := func() {
		if word.Len() > 0 {
			out = append(out, word.String())
			word.Reset()
		}
	}
	flushCJK := func() {
		if cjkUnigrams {
			for _, r := range cjk {
				out = append(out, string(r))
			}
		} else if len(cjk) == 1 {
			out = append(out, string(cjk[0]))
		}
		for i := 0; i+1 < len(cjk); i++ {
			out = append(out, string(cjk[i:i+2]))
		}
		cjk = cjk[:0]
	}
	for _, r := range s {
		switch {
		case isCJK(r):
			flushWord()
			cjk = append(cjk, r)
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_':
			flushCJK()
			word.WriteRune(r)
		default:
			flushWord()
			flushCJK()
		}
	}
	flushWord()
	flushCJK()
	return out
}

func isCJK(r rune) bool {
	return unicode.Is(unicode.Han, r) ||
		unicode.Is(unicode.Hiragana, r) ||
		unicode.Is(unicode.Katakana, r) ||
		unicode.Is(unicode.Hangul, r)
}

func uniqueStrings(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

// List 返回全部记忆。
func List() ([]Entry, error) {
	return load()
}

// Lines 供 /memory 展示。
func Lines() ([]string, error) {
	entries, err := List()
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return []string{"（暂无长期记忆）"}, nil
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, fmt.Sprintf("%s  %s  %s", e.CreatedAt.Format("2006-01-02 15:04"), e.ID, e.Content))
	}
	return out, nil
}

func load() ([]Entry, error) {
	b, err := os.ReadFile(filepath.FromSlash(fileRel))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(strings.TrimSpace(string(b))) == 0 {
		return nil, nil
	}
	var entries []Entry
	if err := json.Unmarshal(b, &entries); err != nil {
		return nil, fmt.Errorf("解析记忆文件失败：%w", err)
	}
	return entries, nil
}

func save(entries []Entry) error {
	path := filepath.FromSlash(fileRel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if entries == nil {
		entries = []Entry{}
	}
	b, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func newID() (string, error) {
	var buf [4]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf[:]), nil
}
