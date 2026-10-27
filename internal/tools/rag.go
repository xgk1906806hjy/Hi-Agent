package tools

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/xgk1906806hjy/Hi-Agent/internal/rag"
)

// registerRAG 注册轻 RAG 的采集与检索工具。
func registerRAG() {
	Register(Tool{
		Name: "rag_add",
		Description: "把工作区内的文件或目录加入项目轻 RAG 索引（" + rag.Path() + "）。" +
			"目录会递归采集；同一文件/目录重复添加会重建，避免重复段。" +
			"只返回来源和段数，不把采集全文放进当前上下文。免确认。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{"type": "string", "description": "工作区内的文件或目录"},
			},
			"required":             []string{"path"},
			"additionalProperties": false,
		},
		Run: runRAGAdd,
	})
	Register(Tool{
		Name: "rag_search",
		Description: "在当前项目的轻 RAG 索引中检索相关片段，返回 [来源#段号] 和片段内容。" +
			"用于回答已采集文档的问题；回答时保留来源段号引用。一次最多返回 top_k 段。免确认。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{"type": "string", "description": "检索问题或关键词"},
				"top_k": map[string]any{
					"type":        "integer",
					"description": "返回片段数，默认 5，最大 20",
					"minimum":     1,
					"maximum":     rag.MaxTopK,
				},
			},
			"required":             []string{"query"},
			"additionalProperties": false,
		},
		Run: runRAGSearch,
	})
}

// runRAGAdd 采集文件或目录；输出只含统计，不含正文。
func runRAGAdd(argsJSON string) (string, error) {
	var args struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", err
	}
	args.Path = strings.TrimSpace(args.Path)
	if args.Path == "" {
		return "", fmt.Errorf("缺少参数 path")
	}
	safe, err := guard(args.Path)
	if err != nil {
		return "", err
	}
	res, err := rag.Add(safe)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	if len(res.Sources) == 0 {
		b.WriteString("没有可索引的文本")
	} else {
		fmt.Fprintf(&b, "已索引 %d 个来源，共 %d 段：", len(res.Sources), res.Chunks)
		for i, source := range res.Sources {
			if i >= 20 {
				fmt.Fprintf(&b, "\n- …（还有 %d 个来源）", len(res.Sources)-i)
				break
			}
			fmt.Fprintf(&b, "\n- %s（%d 段）", clipRunes(source.Source, 160), source.Chunks)
		}
	}
	if res.Skipped > 0 {
		fmt.Fprintf(&b, "\n（跳过 %d 个文件：二进制、空文件、超过 1MB 或读取失败）", res.Skipped)
	}
	return b.String(), nil
}

// runRAGSearch 检索索引；片段带 [来源#段号]，便于回答时引用。
func runRAGSearch(argsJSON string) (string, error) {
	var args struct {
		Query string `json:"query"`
		TopK  int    `json:"top_k"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", err
	}
	args.Query = strings.TrimSpace(args.Query)
	if args.Query == "" {
		return "", fmt.Errorf("缺少参数 query")
	}
	topK := args.TopK
	if topK <= 0 {
		topK = rag.DefaultTopK
	}
	hits, err := rag.Search(args.Query, topK)
	if err != nil {
		return "", err
	}
	if len(hits) == 0 {
		return fmt.Sprintf("没有匹配 %q 的 RAG 片段（可先用 rag_add 采集文档）", args.Query), nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "找到 %d 段：", len(hits))
	for _, hit := range hits {
		fmt.Fprintf(&b, "\n[%s#%d] %s", hit.Source, hit.Seq, clipRunes(oneLine(hit.Content), 600))
	}
	return b.String(), nil
}

func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
