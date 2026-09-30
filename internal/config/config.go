// Package config 从环境变量与多层 .env 加载 OpenAI 兼容接口配置。
//
// 优先级（高 → 低）：进程已有环境变量 > 工作目录向上最近的 .env > 用户全局 ~/.hi-agent/.env。
// 工具读写均相对进程当前工作目录（cwd），因此可在任意项目路径下启动 hi-agent。
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

// Config 模型与接口配置，供 chat.New 使用。
type Config struct {
	BaseURL string // API 根路径，例如 https://api.deepseek.com
	APIKey  string // Bearer / SDK 密钥（必填）
	Model   string // Chat Completions 的 model 字段

	// WorkDir 启动时的绝对工作目录；ls/read/write/shell/会话落盘均相对此目录。
	WorkDir string
	// EnvFiles 实际加载成功的 .env 路径（低优先级在前），便于排查。
	EnvFiles []string
}

// 启动时就会被 dotenv 改写的键；加载文件前快照，最后强制还原进程原值。
var dotenvKeys = []string{
	"OPENAI_BASE_URL",
	"OPENAI_API_KEY",
	"OPENAI_MODEL",
	"GEEKAGENT_TUI",
	"GEEKAGENT_STREAM_USAGE",
	"GEEKAGENT_CONTEXT_WINDOW",
	"GEEKAGENT_MAX_HISTORY",
	"GEEKAGENT_MAX_TOOL_TURNS",
	"GEEKAGENT_MAX_DUP_TOOLS",
	"GEEKAGENT_DUP_WINDOW",
	"GEEKAGENT_MAX_REPLY_CHARS",
	"GEEKAGENT_SHRINK_TOOLS_AFTER",
}

// Load 读取配置。
//
// 1. 解析 cwd 绝对路径
// 2. 快照进程中已存在的相关环境变量
// 3. 依次 Overload：全局 ~/.hi-agent/.env → 从磁盘根到 cwd 路径上的各 .env（越靠近 cwd 越后，覆盖前者）
// 4. 还原步骤 2 的快照（保证 shell 里 export 的值最高优先）
// 5. 填默认值并校验 API Key
func Load() Config {
	workDir, err := os.Getwd()
	if err != nil {
		workDir = "."
	}
	if abs, absErr := filepath.Abs(workDir); absErr == nil {
		workDir = abs
	}

	preserved := snapshotEnv(dotenvKeys)
	loaded := loadDotEnvCascade(workDir)
	restoreEnv(preserved)

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
		fmt.Fprintln(os.Stderr, "缺少 OPENAI_API_KEY。")
		fmt.Fprintln(os.Stderr, "可任选其一：")
		fmt.Fprintln(os.Stderr, "  1) 在当前项目放 .env（可参考仓库 .env.example）")
		fmt.Fprintln(os.Stderr, "  2) 写入用户全局配置：~/.hi-agent/.env")
		fmt.Fprintln(os.Stderr, "  3) 在 shell 中 export OPENAI_API_KEY=...")
		os.Exit(1)
	}

	return Config{
		BaseURL:  baseURL,
		APIKey:   apiKey,
		Model:    model,
		WorkDir:  workDir,
		EnvFiles: loaded,
	}
}

// GlobalEnvPath 返回用户全局配置文件路径（~/.hi-agent/.env）。
func GlobalEnvPath() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".hi-agent", ".env")
}

// loadDotEnvCascade 按优先级从低到高 Overload .env，返回成功加载的路径列表。
func loadDotEnvCascade(workDir string) []string {
	var files []string

	if g := GlobalEnvPath(); g != "" {
		if st, err := os.Stat(g); err == nil && !st.IsDir() {
			if err := godotenv.Overload(g); err == nil {
				files = append(files, g)
			}
		}
	}

	// 从根到 cwd：先远后近，近处覆盖远处。
	chain := envChainFromRoot(workDir)
	for _, dir := range chain {
		p := filepath.Join(dir, ".env")
		st, err := os.Stat(p)
		if err != nil || st.IsDir() {
			continue
		}
		if err := godotenv.Overload(p); err == nil {
			files = append(files, p)
		}
	}
	return files
}

// envChainFromRoot 返回 [祖先..., workDir]，文件系统根在前。
func envChainFromRoot(workDir string) []string {
	var up []string
	dir := workDir
	for {
		up = append(up, dir)
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	// up 目前是 cwd → root，反转为 root → cwd
	for i, j := 0, len(up)-1; i < j; i, j = i+1, j-1 {
		up[i], up[j] = up[j], up[i]
	}
	return up
}

func snapshotEnv(keys []string) map[string]envSnap {
	out := make(map[string]envSnap, len(keys))
	for _, k := range keys {
		v, ok := os.LookupEnv(k)
		out[k] = envSnap{ok: ok, val: v}
	}
	return out
}

type envSnap struct {
	ok  bool
	val string
}

func restoreEnv(snaps map[string]envSnap) {
	// 只写回「启动前进程里本来就有」的键，保留 dotenv 新写入的值。
	for k, s := range snaps {
		if s.ok {
			_ = os.Setenv(k, s.val)
		}
	}
}
