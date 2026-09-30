// Package config 从环境变量（及可选 .env）加载 OpenAI 兼容接口配置。
//
// 已导出的进程环境变量优先于 .env；缺少 OPENAI_API_KEY 时直接退出进程。
package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config 模型与接口配置，供 chat.New 使用。
type Config struct {
	BaseURL string // API 根路径，例如 https://api.deepseek.com
	APIKey  string // Bearer / SDK 密钥（必填）
	Model   string // Chat Completions 的 model 字段
}

// Load 读取配置。
//
// 流程：godotenv.Load（忽略缺文件）→ TrimSpace 读三项 → 填默认 BaseURL/Model →
// 校验 APIKey，为空则写 stderr 并 os.Exit(1)。无缓存，每次重新读环境。
func Load() Config {
	// 忽略「文件不存在」；已在环境中的变量不会被 .env 覆盖（godotenv 默认行为）。
	_ = godotenv.Load()

	baseURL := strings.TrimSpace(os.Getenv("OPENAI_BASE_URL"))
	if baseURL == "" {
		baseURL = "https://api.deepseek.com"
	}
	apiKey := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	model := strings.TrimSpace(os.Getenv("OPENAI_MODEL"))
	if model == "" {
		model = "deepseek-v4-flash"
	}

	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "缺少 OPENAI_API_KEY：请复制 .env.example 为 .env 并填写。")
		os.Exit(1)
	}

	return Config{BaseURL: baseURL, APIKey: apiKey, Model: model}
}
