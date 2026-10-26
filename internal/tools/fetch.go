package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const (
	fetchTimeout   = 15 * time.Second
	maxFetchBytes  = 2 << 20 // 响应体上限 2MB
	maxFetchRunes  = 8000    // 返回给模型的正文上限
	fetchUserAgent = "Hi-agent/0.1 (+https://github.com/xgk1906806hjy/Hi-Agent)"
)

var (
	htmlTitleRe   = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title\s*>`)
	htmlCommentRe = regexp.MustCompile(`(?s)<!--.*?-->`)
	htmlTagRe     = regexp.MustCompile(`(?s)<[^>]*>`)
	htmlBlockRe   = regexp.MustCompile(`(?i)</?(?:br|p|div|li|tr|h[1-6]|section|article|header|footer|nav|table|ul|ol|pre|blockquote|hr|head|body)\b[^>]*>`)
	htmlSpaceRe   = regexp.MustCompile(`[ \t\f\v]+`)
	htmlNLRe      = regexp.MustCompile(` *\n *`)
	htmlBlankRe   = regexp.MustCompile(`\n{3,}`)
)

// htmlDropBlocks 整段丢弃的标签（内容不是正文）。
var htmlDropBlocks = []*regexp.Regexp{
	regexp.MustCompile(`(?is)<head\b[^>]*>.*?</head\s*>`),
	regexp.MustCompile(`(?is)<script\b[^>]*>.*?</script\s*>`),
	regexp.MustCompile(`(?is)<style\b[^>]*>.*?</style\s*>`),
	regexp.MustCompile(`(?is)<noscript\b[^>]*>.*?</noscript\s*>`),
	regexp.MustCompile(`(?is)<svg\b[^>]*>.*?</svg\s*>`),
	regexp.MustCompile(`(?is)<template\b[^>]*>.*?</template\s*>`),
	regexp.MustCompile(`(?is)<iframe\b[^>]*>.*?</iframe\s*>`),
}

// fetchClient 单独实例，便于测试替换/复用连接。
var fetchClient = &http.Client{Timeout: fetchTimeout}

// registerFetch 注册 fetch：确认后抓取 http/https 页面并转纯文本。
func registerFetch() {
	Register(Tool{
		Name: "fetch",
		Description: "抓取一个 http/https 网页并转成纯文本（去脚本/样式/标签）。执行前会向用户确认。" +
			"超时 15 秒，响应上限 2MB，正文最多返回约 8000 字。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"url": map[string]any{"type": "string", "description": "要抓取的完整 http/https URL"},
			},
			"required":             []string{"url"},
			"additionalProperties": false,
		},
		Run: runFetch,
	})
}

// runFetch 校验 URL → 确认 → 请求 → 限流读取 → HTML 转文本 → 截断。
func runFetch(argsJSON string) (string, error) {
	var args struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("参数不是合法 JSON：%w", err)
	}
	raw := strings.TrimSpace(args.URL)
	if raw == "" {
		return "缺少参数 url", nil
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return "只支持 http/https URL：" + raw, nil
	}

	if !ask(fmt.Sprintf("即将抓取网页：%s", raw)) {
		return "已取消抓取", nil
	}

	parent := callCtx
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithTimeout(parent, fetchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "抓取失败：" + err.Error(), nil
	}
	req.Header.Set("User-Agent", fetchUserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,text/plain;q=0.9,*/*;q=0.5")

	resp, err := fetchClient.Do(req)
	if err != nil {
		if parent.Err() != nil {
			return "抓取已取消", nil
		}
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Sprintf("抓取失败：超时（%s）", fetchTimeout), nil
		}
		return "抓取失败：" + err.Error(), nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxFetchBytes+1))
	if err != nil {
		return "读取响应失败：" + err.Error(), nil
	}
	bodyTruncated := len(body) > maxFetchBytes
	if bodyTruncated {
		body = body[:maxFetchBytes]
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Sprintf("抓取失败：HTTP %d %s", resp.StatusCode, http.StatusText(resp.StatusCode)), nil
	}

	ct := strings.TrimSpace(resp.Header.Get("Content-Type"))
	rawText := string(body)
	var title, text string
	if looksLikeHTML(ct, rawText) {
		title, text = htmlToText(rawText)
	} else {
		text = strings.TrimSpace(rawText)
	}

	finalURL := raw
	if resp.Request != nil && resp.Request.URL != nil {
		finalURL = resp.Request.URL.String()
	}
	var b strings.Builder
	fmt.Fprintf(&b, "抓取 %s（HTTP %d", finalURL, resp.StatusCode)
	if ct != "" {
		b.WriteString("，")
		b.WriteString(ct)
	}
	b.WriteString("）")
	if title != "" {
		b.WriteString("\n标题：" + clipRunes(title, 200))
	}
	if text == "" {
		b.WriteString("\n(没有可提取的正文)")
	} else {
		b.WriteString("\n\n")
		b.WriteString(clipFetchText(text))
	}
	if bodyTruncated {
		fmt.Fprintf(&b, "\n...(响应超过 %d 字节，已截断)", maxFetchBytes)
	}
	return b.String(), nil
}

// clipFetchText 限制返回正文长度并附提示。
func clipFetchText(s string) string {
	r := []rune(s)
	if len(r) <= maxFetchRunes {
		return s
	}
	return fmt.Sprintf("%s\n...(正文已截断，原共 %d 字符)", string(r[:maxFetchRunes]), len(r))
}

// looksLikeHTML 优先看 Content-Type；缺失时用正文前缀判断。
func looksLikeHTML(contentType, body string) bool {
	ct := strings.ToLower(contentType)
	if ct != "" {
		return strings.Contains(ct, "html") || strings.Contains(ct, "xml")
	}
	head := strings.ToLower(body)
	if len(head) > 512 {
		head = head[:512]
	}
	return strings.Contains(head, "<html") || strings.Contains(head, "<!doctype html")
}

// htmlToText 提取标题并去掉脚本/样式/注释/标签，解码实体后压缩空白。
func htmlToText(src string) (title, text string) {
	if m := htmlTitleRe.FindStringSubmatch(src); len(m) == 2 {
		title = strings.TrimSpace(html.UnescapeString(htmlTagRe.ReplaceAllString(m[1], " ")))
	}
	for _, re := range htmlDropBlocks {
		src = re.ReplaceAllString(src, " ")
	}
	src = htmlCommentRe.ReplaceAllString(src, " ")
	src = htmlBlockRe.ReplaceAllString(src, "\n")
	src = htmlTagRe.ReplaceAllString(src, " ")
	src = html.UnescapeString(src)
	src = strings.ReplaceAll(src, "\u00a0", " ")
	src = htmlSpaceRe.ReplaceAllString(src, " ")
	src = htmlNLRe.ReplaceAllString(src, "\n")
	src = htmlBlankRe.ReplaceAllString(src, "\n\n")
	return title, strings.TrimSpace(src)
}
