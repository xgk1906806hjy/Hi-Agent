package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ApplyResult 是 setup-env 的执行结果摘要。
type ApplyResult struct {
	Source   string   // 读取的 .env 路径
	Global   string   // 写入的全局 ~/.hi-agent/.env
	Keys     []string // 解析到的键
	UserEnv  []string // 已写入操作系统「用户环境变量」的键
	ShellHint string  // 给用户的后续提示（如重开终端）
}

// ApplyEnvFile 读取 src（默认当前目录 .env），写入全局配置，并尽量设置用户级环境变量。
//
// 供 `hi-agent setup-env` 与安装脚本调用；不修改当前已运行的无关进程的环境。
func ApplyEnvFile(src string) (ApplyResult, error) {
	var out ApplyResult
	if strings.TrimSpace(src) == "" {
		src = ".env"
	}
	abs, err := filepath.Abs(src)
	if err != nil {
		return out, err
	}
	out.Source = abs

	kv, err := parseEnvFile(abs)
	if err != nil {
		return out, err
	}
	if len(kv) == 0 {
		return out, fmt.Errorf("文件中没有有效的 KEY=VALUE：%s", abs)
	}
	for k := range kv {
		out.Keys = append(out.Keys, k)
	}

	global := GlobalEnvPath()
	if global == "" {
		return out, fmt.Errorf("无法解析用户主目录，无法写入全局配置")
	}
	if err := os.MkdirAll(filepath.Dir(global), 0o755); err != nil {
		return out, err
	}
	// 原样复制（保留注释），便于用户日后编辑
	raw, err := os.ReadFile(abs)
	if err != nil {
		return out, err
	}
	if err := os.WriteFile(global, raw, 0o600); err != nil {
		return out, err
	}
	out.Global = global

	// 仅把常用键写入 OS 用户环境，避免把整份 .env 不受控地灌进系统
	persistKeys := []string{
		"OPENAI_API_KEY",
		"OPENAI_BASE_URL",
		"OPENAI_MODEL",
	}
	toSet := map[string]string{}
	for _, k := range persistKeys {
		if v, ok := kv[k]; ok && strings.TrimSpace(v) != "" {
			toSet[k] = v
		}
	}
	if len(toSet) == 0 {
		out.ShellHint = "全局 .env 已写入，但未找到非空的 OPENAI_API_KEY/BASE_URL/MODEL，跳过系统环境变量。"
		return out, nil
	}

	setKeys, hint, err := setUserEnv(toSet)
	if err != nil {
		// 全局文件已成功；环境变量失败仍返回错误，方便脚本判断
		out.ShellHint = hint
		return out, fmt.Errorf("全局配置已写入 %s，但设置用户环境变量失败：%w", global, err)
	}
	out.UserEnv = setKeys
	out.ShellHint = hint
	return out, nil
}

// parseEnvFile 解析简易 KEY=VALUE（支持 # 注释、可选引号）；不执行变量展开。
func parseEnvFile(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	out := make(map[string]string)
	sc := bufio.NewScanner(f)
	// 兼容较长行（密钥一般不长，给足余量）
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// 去掉可选的 export 前缀
		line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		i := strings.IndexByte(line, '=')
		if i <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:i])
		val := strings.TrimSpace(line[i+1:])
		if key == "" {
			continue
		}
		val = unquoteEnv(val)
		out[key] = val
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("读取 %s 第 %d 行附近失败：%w", path, lineNo, err)
	}
	return out, nil
}

func unquoteEnv(v string) string {
	if len(v) >= 2 {
		if (v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'') {
			return v[1 : len(v)-1]
		}
	}
	return v
}
