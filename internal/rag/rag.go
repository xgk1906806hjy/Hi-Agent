// Package rag 实现项目级轻量 RAG：把工作区文本切块后写入
// .geekagent/rag/index.json，并用 BM25 返回带来源段号的片段。
package rag

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const fileRel = ".geekagent/rag/index.json"

const (
	indexVersion = 1
	// DefaultTopK 是未指定 top_k 时返回的片段数。
	DefaultTopK = 5
	// MaxTopK 限制单次检索返回的片段数。
	MaxTopK = 20

	chunkMaxRunes = 800     // 单个索引块的最大 rune 数
	maxFileBytes  = 1 << 20 // 超过 1MB 的文档跳过
	bm25K1        = 1.2
	bm25B         = 0.75
)

var skipDirs = map[string]struct{}{
	".git":         {},
	".geekagent":   {},
	"node_modules": {},
	"dist":         {},
	"build":        {},
	".next":        {},
	".venv":        {},
	"venv":         {},
	"__pycache__":  {},
}

// Chunk 索引中的一个来源片段。
type Chunk struct {
	Source  string `json:"source"` // 相对工作区、/ 分隔
	Seq     int    `json:"seq"`    // 来源内段号，从 1 开始
	Content string `json:"content"`
}

// Hit 一次 BM25 检索结果。
type Hit struct {
	Source  string
	Seq     int
	Content string
	Score   float64
}

// SourceStat 一个来源的采集结果。
type SourceStat struct {
	Source string `json:"source"`
	Chunks int    `json:"chunks"`
}

// AddResult 采集结果；不包含全文。
type AddResult struct {
	Sources []SourceStat
	Chunks  int
	Skipped int
}

// IndexStats 索引概况。
type IndexStats struct {
	Sources   int
	Chunks    int
	UpdatedAt time.Time
}

type indexFile struct {
	Version   int       `json:"version"`
	UpdatedAt time.Time `json:"updatedAt"`
	Chunks    []Chunk   `json:"chunks"`
}

// Path 返回索引文件的相对路径。
func Path() string { return fileRel }

// Add 采集文件或目录并重建该范围在索引中的片段。
// 同一文件/目录重复添加不会重复；目录会清掉其下已不存在的旧来源。
func Add(path string) (AddResult, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return AddResult{}, fmt.Errorf("路径为空")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return AddResult{}, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return AddResult{}, err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return AddResult{}, err
	}
	root, err := filepath.Abs(cwd)
	if err != nil {
		return AddResult{}, err
	}

	files, skipped, err := collectFiles(abs, info)
	if err != nil {
		return AddResult{}, err
	}
	idx, err := load()
	if err != nil {
		return AddResult{}, err
	}

	scope := sourceScope(root, abs, info.IsDir())
	added := make([]Chunk, 0, len(files))
	counts := make(map[string]int, len(files))
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			skipped++
			continue
		}
		if len(data) > maxFileBytes || bytes.IndexByte(data, 0) >= 0 || !utf8.Valid(data) {
			skipped++
			continue
		}
		chunks := chunkText(normalizeText(string(data)))
		if len(chunks) == 0 {
			skipped++
			continue
		}
		source := sourcePath(root, file)
		for i, content := range chunks {
			added = append(added, Chunk{Source: source, Seq: i + 1, Content: content})
		}
		counts[source] = len(chunks)
	}

	kept := idx.Chunks[:0]
	for _, c := range idx.Chunks {
		if !inScope(c.Source, scope) {
			kept = append(kept, c)
		}
	}
	idx.Chunks = append(kept, added...)
	idx.Version = indexVersion
	idx.UpdatedAt = time.Now()
	if err := save(idx); err != nil {
		return AddResult{}, err
	}

	sources := make([]SourceStat, 0, len(counts))
	total := 0
	for source, n := range counts {
		sources = append(sources, SourceStat{Source: source, Chunks: n})
		total += n
	}
	sort.Slice(sources, func(i, j int) bool { return sources[i].Source < sources[j].Source })
	return AddResult{Sources: sources, Chunks: total, Skipped: skipped}, nil
}

// Search 在索引中做 BM25 检索，返回最多 topK 个相关片段。
// 没有索引时返回空结果；索引损坏时返回错误。
func Search(query string, topK int) ([]Hit, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	terms := uniqueStrings(tokenizeQuery(query))
	if len(terms) == 0 {
		return nil, nil
	}
	idx, err := load()
	if err != nil {
		return nil, err
	}
	if len(idx.Chunks) == 0 {
		return nil, nil
	}
	if topK <= 0 {
		topK = DefaultTopK
	}
	if topK > MaxTopK {
		topK = MaxTopK
	}

	type doc struct {
		chunk  Chunk
		tf     map[string]int
		length int
	}
	docs := make([]doc, 0, len(idx.Chunks))
	for _, c := range idx.Chunks {
		tokens := tokenize(c.Content)
		if len(tokens) == 0 {
			continue
		}
		tf := make(map[string]int, len(tokens))
		for _, t := range tokens {
			tf[t]++
		}
		docs = append(docs, doc{chunk: c, tf: tf, length: len(tokens)})
	}
	if len(docs) == 0 {
		return nil, nil
	}

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
		hit Hit
	}
	hits := make([]scoredHit, 0, len(docs))
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
		hits = append(hits, scoredHit{hit: Hit{
			Source:  d.chunk.Source,
			Seq:     d.chunk.Seq,
			Content: d.chunk.Content,
			Score:   score,
		}})
	}
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].hit.Score != hits[j].hit.Score {
			return hits[i].hit.Score > hits[j].hit.Score
		}
		if hits[i].hit.Source != hits[j].hit.Source {
			return hits[i].hit.Source < hits[j].hit.Source
		}
		return hits[i].hit.Seq < hits[j].hit.Seq
	})
	if len(hits) > topK {
		hits = hits[:topK]
	}
	out := make([]Hit, 0, len(hits))
	for _, h := range hits {
		out = append(out, h.hit)
	}
	return out, nil
}

// Stats 返回索引概况；没有索引时为零值。
func Stats() (IndexStats, error) {
	idx, err := load()
	if err != nil {
		return IndexStats{}, err
	}
	counts := sourceCounts(idx.Chunks)
	return IndexStats{Sources: len(counts), Chunks: len(idx.Chunks), UpdatedAt: idx.UpdatedAt}, nil
}

// Lines 生成 /rag 展示文本。
func Lines() ([]string, error) {
	idx, err := load()
	if err != nil {
		return nil, err
	}
	if len(idx.Chunks) == 0 {
		return []string{"（暂无 RAG 索引，先用 rag_add 采集文件或目录）"}, nil
	}
	counts := sourceCounts(idx.Chunks)
	sources := make([]string, 0, len(counts))
	for source := range counts {
		sources = append(sources, source)
	}
	sort.Strings(sources)

	head := fmt.Sprintf("RAG 索引：%d 段 / %d 个来源", len(idx.Chunks), len(sources))
	if !idx.UpdatedAt.IsZero() {
		head += "；更新于 " + idx.UpdatedAt.Format("2006-01-02 15:04")
	}
	lines := []string{head}
	for i, source := range sources {
		if i >= 50 {
			lines = append(lines, fmt.Sprintf("- …（还有 %d 个来源）", len(sources)-i))
			break
		}
		lines = append(lines, fmt.Sprintf("- %s（%d 段）", source, counts[source]))
	}
	return lines, nil
}

// Clear 删除当前项目的 RAG 索引；索引不存在时视为成功。
func Clear() error {
	err := os.Remove(filepath.FromSlash(fileRel))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func load() (indexFile, error) {
	b, err := os.ReadFile(filepath.FromSlash(fileRel))
	if errors.Is(err, os.ErrNotExist) {
		return indexFile{}, nil
	}
	if err != nil {
		return indexFile{}, err
	}
	if len(strings.TrimSpace(string(b))) == 0 {
		return indexFile{}, nil
	}
	var idx indexFile
	if err := json.Unmarshal(b, &idx); err != nil {
		return indexFile{}, fmt.Errorf("解析 RAG 索引失败：%w", err)
	}
	return idx, nil
}

func save(idx indexFile) error {
	path := filepath.FromSlash(fileRel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if idx.Chunks == nil {
		idx.Chunks = []Chunk{}
	}
	b, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func sourceCounts(chunks []Chunk) map[string]int {
	out := make(map[string]int)
	for _, c := range chunks {
		out[c.Source]++
	}
	return out
}

func collectFiles(abs string, info fs.FileInfo) ([]string, int, error) {
	if !info.IsDir() {
		return []string{abs}, 0, nil
	}
	var files []string
	skipped := 0
	err := filepath.WalkDir(abs, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			skipped++
			return nil
		}
		if d.IsDir() {
			if _, skip := skipDirs[d.Name()]; skip {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			skipped++
			return nil
		}
		files = append(files, path)
		return nil
	})
	return files, skipped, err
}

func sourceScope(root, target string, isDir bool) string {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return filepath.ToSlash(target)
	}
	rel = filepath.ToSlash(rel)
	if !isDir {
		return rel
	}
	if rel == "." {
		return ""
	}
	return strings.TrimSuffix(rel, "/") + "/"
}

func sourcePath(root, file string) string {
	rel, err := filepath.Rel(root, file)
	if err != nil {
		return filepath.ToSlash(file)
	}
	return filepath.ToSlash(rel)
}

func inScope(source, scope string) bool {
	if scope == "" {
		return true
	}
	if strings.HasSuffix(scope, "/") {
		base := strings.TrimSuffix(scope, "/")
		return source == base || strings.HasPrefix(source, scope)
	}
	return source == scope
}

func normalizeText(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.TrimSpace(s)
}

// chunkText 按段落和句子把文档切成不超过 chunkMaxRunes 的块。
func chunkText(s string) []string {
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

// tokenize 是文档侧分词：拉丁词小写，CJK 生成单字与双字 gram。
func tokenize(s string) []string {
	return tokenizeText(s, true)
}

// tokenizeQuery 是查询侧分词：CJK 主要用双字 gram，单字查询才保留单字。
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
