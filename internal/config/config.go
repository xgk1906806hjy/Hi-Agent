package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config 模型与接口配置。
type Config struct {
	BaseURL string
	APIKey  string
	Model   string
}

// Load 从环境变量读取配置；缺少 API Key 时退出并提示。
func Load() Config {
	_ = godotenv.Load() // 忽略「文件不存在」；已导出的环境变量优先

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
