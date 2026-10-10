package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ApplyEndpoint 把端点写入当前进程环境；persistGlobal 时同步 upsert 到 ~/.hi-agent/.env。
func ApplyEndpoint(ep Endpoint, persistGlobal bool) error {
	if ep.Provider == "" || ep.BaseURL == "" || ep.APIKey == "" || ep.Model == "" {
		return fmt.Errorf("端点不完整：需要 provider/baseURL/apiKey/model")
	}
	_ = os.Setenv("GEEKAGENT_PROVIDER", ep.Provider)
	_ = os.Setenv("HI_AGENT_PROVIDER", ep.Provider)
	_ = os.Setenv("OPENAI_BASE_URL", ep.BaseURL)
	_ = os.Setenv("OPENAI_API_KEY", ep.APIKey)
	_ = os.Setenv("OPENAI_MODEL", ep.Model)
	_ = os.Setenv("HI_AGENT_BASE_URL", ep.BaseURL)
	_ = os.Setenv("HI_AGENT_API_KEY", ep.APIKey)
	_ = os.Setenv("HI_AGENT_MODEL", ep.Model)

	if !persistGlobal {
		return nil
	}
	path := GlobalEnvPath()
	if path == "" {
		return fmt.Errorf("无法解析全局配置路径")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return UpsertEnvFile(path, map[string]string{
		"GEEKAGENT_PROVIDER": ep.Provider,
		"OPENAI_BASE_URL":    ep.BaseURL,
		"OPENAI_API_KEY":     ep.APIKey,
		"OPENAI_MODEL":       ep.Model,
	})
}

// UpsertEnvFile 更新或追加 KEY=VALUE；保留其它行与注释。
func UpsertEnvFile(path string, kv map[string]string) error {
	if len(kv) == 0 {
		return nil
	}
	var lines []string
	if raw, err := os.ReadFile(path); err == nil {
		lines = strings.Split(string(raw), "\n")
		// 去掉末尾因 Split 产生的空串可保留
	} else if !os.IsNotExist(err) {
		return err
	}

	seen := map[string]bool{}
	for i, line := range lines {
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "#") {
			continue
		}
		body := strings.TrimSpace(strings.TrimPrefix(trim, "export "))
		eq := strings.IndexByte(body, '=')
		if eq <= 0 {
			continue
		}
		key := strings.TrimSpace(body[:eq])
		if val, ok := kv[key]; ok {
			lines[i] = key + "=" + val
			seen[key] = true
		}
	}
	for k, v := range kv {
		if seen[k] {
			continue
		}
		lines = append(lines, k+"="+v)
	}
	out := strings.Join(lines, "\n")
	if !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	return os.WriteFile(path, []byte(out), 0o600)
}

// MaskSecret 脱敏展示密钥（保留末 4 位）。
func MaskSecret(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "(空)"
	}
	r := []rune(s)
	if len(r) <= 4 {
		return "****"
	}
	return "****" + string(r[len(r)-4:])
}
