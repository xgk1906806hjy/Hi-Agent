package config

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
)

// Provider 描述一个 OpenAI 兼容的模型供应商（含自定义中转站）。
// 所有内置厂商均走 Chat Completions 兼容协议；差异主要是默认 BaseURL / Model / 密钥环境变量名。
type Provider struct {
	ID           string // 稳定标识，如 deepseek、custom
	Name         string // 展示名
	DefaultURL   string // 默认 API 根；custom 可为空（必须由环境变量给出）
	DefaultModel string
	KeyEnv       string // 厂商专用密钥环境变量（可选，仍回退 OPENAI_API_KEY）
}

// Endpoint 是工厂解析后的可连接端点，供 chat.New 使用。
type Endpoint struct {
	Provider string
	BaseURL  string
	APIKey   string
	Model    string
}

var (
	providerMu sync.RWMutex
	providers  = map[string]Provider{}
)

func init() {
	registerBuiltins()
}

func registerBuiltins() {
	builtins := []Provider{
		{
			ID: "deepseek", Name: "DeepSeek",
			DefaultURL: "https://api.deepseek.com", DefaultModel: "deepseek-v4-flash",
			KeyEnv: "DEEPSEEK_API_KEY",
		},
		{
			ID: "openai", Name: "OpenAI",
			DefaultURL: "https://api.openai.com/v1", DefaultModel: "gpt-4o-mini",
			KeyEnv: "OPENAI_API_KEY",
		},
		{
			ID: "moonshot", Name: "Moonshot / Kimi",
			DefaultURL: "https://api.moonshot.cn/v1", DefaultModel: "moonshot-v1-8k",
			KeyEnv: "MOONSHOT_API_KEY",
		},
		{
			ID: "qwen", Name: "通义千问（兼容模式）",
			DefaultURL: "https://dashscope.aliyuncs.com/compatible-mode/v1", DefaultModel: "qwen-plus",
			KeyEnv: "DASHSCOPE_API_KEY",
		},
		{
			ID: "zhipu", Name: "智谱 GLM",
			DefaultURL: "https://open.bigmodel.cn/api/paas/v4", DefaultModel: "glm-4-flash",
			KeyEnv: "ZHIPU_API_KEY",
		},
		{
			ID: "siliconflow", Name: "硅基流动",
			DefaultURL: "https://api.siliconflow.cn/v1", DefaultModel: "deepseek-ai/DeepSeek-V3",
			KeyEnv: "SILICONFLOW_API_KEY",
		},
		{
			ID: "openrouter", Name: "OpenRouter",
			DefaultURL: "https://openrouter.ai/api/v1", DefaultModel: "openai/gpt-4o-mini",
			KeyEnv: "OPENROUTER_API_KEY",
		},
		{
			ID: "groq", Name: "Groq",
			DefaultURL: "https://api.groq.com/openai/v1", DefaultModel: "llama-3.3-70b-versatile",
			KeyEnv: "GROQ_API_KEY",
		},
		{
			ID: "ollama", Name: "Ollama（本地）",
			DefaultURL: "http://127.0.0.1:11434/v1", DefaultModel: "llama3.2",
			KeyEnv: "OLLAMA_API_KEY", // 本地常可填任意非空或 ollama
		},
		{
			ID: "custom", Name: "自定义 / 中转站",
			DefaultURL: "", DefaultModel: "",
			KeyEnv: "OPENAI_API_KEY",
		},
	}
	for _, p := range builtins {
		providers[p.ID] = p
	}
}

// RegisterProvider 注册或覆盖供应商（供扩展 / 测试）。id 会规范化为小写。
func RegisterProvider(p Provider) error {
	id := normalizeProviderID(p.ID)
	if id == "" {
		return fmt.Errorf("provider id 不能为空")
	}
	p.ID = id
	providerMu.Lock()
	providers[id] = p
	providerMu.Unlock()
	return nil
}

// LookupProvider 按 id 查找已注册供应商。
func LookupProvider(id string) (Provider, bool) {
	providerMu.RLock()
	defer providerMu.RUnlock()
	p, ok := providers[normalizeProviderID(id)]
	return p, ok
}

// ListProviders 返回已注册供应商（按 id 排序）。
func ListProviders() []Provider {
	providerMu.RLock()
	defer providerMu.RUnlock()
	out := make([]Provider, 0, len(providers))
	for _, p := range providers {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// getenvFn 便于单测注入。
type getenvFn func(string) string

// ResolveEndpoint 根据供应商 id 与环境变量解析最终 BaseURL / APIKey / Model。
//
// 覆盖优先级（高 → 低）：
//   - BaseURL: OPENAI_BASE_URL / HI_AGENT_BASE_URL → 供应商默认
//   - Model:   OPENAI_MODEL / HI_AGENT_MODEL → 供应商默认
//   - APIKey:  OPENAI_API_KEY / HI_AGENT_API_KEY → 供应商 KeyEnv
func ResolveEndpoint(providerID string, getenv getenvFn) (Endpoint, error) {
	if getenv == nil {
		getenv = func(k string) string { return "" }
	}
	id := normalizeProviderID(providerID)
	if id == "" {
		id = inferProviderID(getenv)
	}

	p, ok := LookupProvider(id)
	if !ok {
		// 未知 id：按 custom 处理，保留用户填写的 id 便于展示
		p = Provider{ID: id, Name: id, KeyEnv: "OPENAI_API_KEY"}
	}

	baseURL := firstNonEmpty(getenv("OPENAI_BASE_URL"), getenv("HI_AGENT_BASE_URL"), p.DefaultURL)
	model := firstNonEmpty(getenv("OPENAI_MODEL"), getenv("HI_AGENT_MODEL"), p.DefaultModel)
	apiKey := firstNonEmpty(
		getenv("OPENAI_API_KEY"),
		getenv("HI_AGENT_API_KEY"),
		getenv("GEEKAGENT_API_KEY"),
	)
	if apiKey == "" && p.KeyEnv != "" && p.KeyEnv != "OPENAI_API_KEY" {
		apiKey = strings.TrimSpace(getenv(p.KeyEnv))
	}
	// openai 供应商 KeyEnv 也是 OPENAI_API_KEY，上面已覆盖；ollama 允许占位密钥
	if apiKey == "" && id == "ollama" {
		apiKey = "ollama"
	}

	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	model = strings.TrimSpace(model)
	apiKey = strings.TrimSpace(apiKey)

	if baseURL == "" {
		return Endpoint{}, fmt.Errorf("供应商 %s 未配置 BaseURL：请设置 OPENAI_BASE_URL，或选用内置厂商（如 deepseek）", id)
	}
	if apiKey == "" {
		hint := "OPENAI_API_KEY"
		if p.KeyEnv != "" && p.KeyEnv != "OPENAI_API_KEY" {
			hint = "OPENAI_API_KEY 或 " + p.KeyEnv
		}
		return Endpoint{}, fmt.Errorf("缺少 API Key：请设置 %s", hint)
	}
	if model == "" {
		return Endpoint{}, fmt.Errorf("缺少模型名：请设置 OPENAI_MODEL")
	}

	return Endpoint{
		Provider: id,
		BaseURL:  baseURL,
		APIKey:   apiKey,
		Model:    model,
	}, nil
}

// ResolveEndpointFromEnv 使用真实进程环境解析端点。
func ResolveEndpointFromEnv() (Endpoint, error) {
	id := firstNonEmpty(osGetenv("GEEKAGENT_PROVIDER"), osGetenv("HI_AGENT_PROVIDER"))
	return ResolveEndpoint(id, osGetenv)
}

func osGetenv(k string) string { return strings.TrimSpace(lookupEnv(k)) }

// lookupEnv 默认同 os.Getenv，测试可替换。
var lookupEnv = os.Getenv

func inferProviderID(getenv getenvFn) string {
	if u := firstNonEmpty(getenv("OPENAI_BASE_URL"), getenv("HI_AGENT_BASE_URL")); u != "" {
		// 已显式给了 URL：若能匹配已知默认 URL 则用对应厂商，否则 custom
		u = strings.TrimRight(strings.TrimSpace(u), "/")
		providerMu.RLock()
		defer providerMu.RUnlock()
		for _, p := range providers {
			if p.ID == "custom" || p.DefaultURL == "" {
				continue
			}
			if strings.TrimRight(p.DefaultURL, "/") == u {
				return p.ID
			}
		}
		return "custom"
	}
	return "deepseek" // 向后兼容：未配置时默认 DeepSeek
}

func normalizeProviderID(id string) string {
	id = strings.TrimSpace(strings.ToLower(id))
	id = strings.ReplaceAll(id, "-", "_")
	return id
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}

// ProviderHelp 返回内置供应商说明（供 /help）。
func ProviderHelp() string {
	list := ListProviders()
	var b strings.Builder
	b.WriteString("模型供应商（GEEKAGENT_PROVIDER / HI_AGENT_PROVIDER）：\n")
	for _, p := range list {
		url := p.DefaultURL
		if url == "" {
			url = "（须自设 OPENAI_BASE_URL）"
		}
		model := p.DefaultModel
		if model == "" {
			model = "（须自设 OPENAI_MODEL）"
		}
		b.WriteString(fmt.Sprintf("  %-12s %s\n             默认 %s · %s\n", p.ID, p.Name, url, model))
	}
	b.WriteString("覆盖：OPENAI_BASE_URL / OPENAI_API_KEY / OPENAI_MODEL（或 HI_AGENT_*）。\n")
	b.WriteString("自定义中转：GEEKAGENT_PROVIDER=custom 并填写 OPENAI_BASE_URL。")
	return b.String()
}
